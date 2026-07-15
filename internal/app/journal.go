package app

// Журнал: что unissh делал на серверах — подключения, команды, передачи файлов.
// Формат — JSON Lines в ~/.config/unissh/history.jsonl, права 0600.
//
// ВАЖНО: тексты команд сохраняются как есть, и в них могут оказаться секреты
// (`mysql -pPASSWORD`). Поэтому файл только для владельца, а в просмотрщике есть
// «Очистить журнал». Записи дописываются под мьютексом: runOn выполняется
// параллельно на нескольких серверах.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/MrBenoit/unissh/internal/core"
)

// Виды записей журнала.
const (
	journalConnect  = "connect"
	journalRun      = "run"
	journalTransfer = "transfer"
)

// journalMaxBytes — после этого размера файл уезжает в history.jsonl.1.
// Без ротации журнал рос бы вечно, а читаем мы его целиком.
const journalMaxBytes = 1 << 20

// journalTail — сколько последних записей показывает просмотрщик.
const journalTail = 60

// journalEntry — одна запись журнала.
type journalEntry struct {
	At     string `json:"at"`             // RFC3339
	Kind   string `json:"kind,omitempty"` // пусто = запись старого формата (подключение)
	Server string `json:"server"`
	Host   string `json:"host"`
	Cmd    string `json:"cmd,omitempty"`  // команда (run) или описание передачи
	Code   int    `json:"code,omitempty"` // код выхода, если он был
}

// kindOf возвращает вид записи, подставляя «connect» для старого формата.
func (e journalEntry) kindOf() string {
	if e.Kind == "" {
		return journalConnect
	}
	return e.Kind
}

// journalMu защищает дописывание: команды на нескольких серверах выполняются
// параллельно, и их записи не должны перемешаться внутри строки.
var journalMu sync.Mutex

// journalAppend дописывает запись в журнал. Ошибки намеренно проглатываются:
// журнал — вспомогательный, из-за него нельзя ронять полезное действие.
func journalAppend(e journalEntry) {
	if e.At == "" {
		e.At = time.Now().Format(time.RFC3339)
	}
	line, err := json.Marshal(e)
	if err != nil {
		return
	}
	journalMu.Lock()
	defer journalMu.Unlock()

	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		return
	}
	journalRotate()
	f, err := os.OpenFile(historyPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.Write(append(line, '\n'))
}

// journalRotate переносит разросшийся журнал в history.jsonl.1. Вызывается под
// journalMu.
func journalRotate() {
	fi, err := os.Stat(historyPath())
	if err != nil || fi.Size() < journalMaxBytes {
		return
	}
	_ = os.Rename(historyPath(), historyPath()+".1")
}

// journalRead возвращает последние limit записей, самые свежие — первыми.
// Битые строки пропускаются: один испорченный байт не должен прятать журнал.
func journalRead(limit int) []journalEntry {
	raw, err := os.ReadFile(historyPath())
	if err != nil {
		return nil
	}
	lines := strings.Split(string(raw), "\n")
	out := make([]journalEntry, 0, limit)
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var e journalEntry
		if json.Unmarshal([]byte(line), &e) != nil {
			continue
		}
		out = append(out, e)
	}
	return out
}

// journalClear удаляет журнал вместе с его ротацией.
func journalClear() error {
	journalMu.Lock()
	defer journalMu.Unlock()
	if err := os.Remove(historyPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(historyPath() + ".1"); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// journalRecordRun пишет в журнал факт выполнения команды.
func journalRecordRun(s *Server, command string, code int) {
	journalAppend(journalEntry{
		Kind: journalRun, Server: s.Name, Host: s.Host,
		Cmd: core.OneLine(command, 300), Code: code,
	})
}

// journalRecordTransfer пишет в журнал факт передачи файла. direction — «up»/«down».
func journalRecordTransfer(s *Server, direction, from, to string) {
	arrow := "→"
	if direction == "down" {
		arrow = "←"
	}
	journalAppend(journalEntry{
		Kind: journalTransfer, Server: s.Name, Host: s.Host,
		Cmd: core.OneLine(from+" "+arrow+" "+to, 300),
	})
}

// journalKindCell — значок и подпись вида записи.
func journalKindCell(kind string) string {
	switch kind {
	case journalRun:
		return styCyan.Render("▶ " + T.JournalKindRun)
	case journalTransfer:
		return styViolet.Render("📦 " + T.JournalKindTransfer)
	}
	return styGreen.Render("🔌 " + T.JournalKindConnect)
}

// journalCodeCell — код выхода: пусто для подключений, ✓/цифра для команд.
func journalCodeCell(e journalEntry) string {
	if e.kindOf() != journalRun {
		return ""
	}
	if e.Code == 0 {
		return styGreen.Render("✓")
	}
	return styRed.Render(fmt.Sprint(e.Code))
}

// journalTable рендерит журнал таблицей.
func journalTable(entries []journalEntry) string {
	t := newTable(T.JournalColWhen, T.JournalColWhat, T.ColName, T.JournalColDetails, "")
	for _, e := range entries {
		when := relativeTime(e.At)
		if when == "" {
			when = e.At
		}
		t.Row(
			styGrey.Render(when),
			journalKindCell(e.kindOf()),
			styBold.Render(e.Server),
			styGrey.Render(core.OneLine(sanitizeRemote(e.Cmd), 48)),
			journalCodeCell(e),
		)
	}
	return t.Render()
}

// journalPage собирает содержимое страницы журнала.
func journalPage(entries []journalEntry) []string {
	return pageLines(journalTable(entries), styDim.Render(T.JournalSecretsHint))
}

// actionJournal — пункт меню «Журнал»: последние записи, «c» очищает.
func actionJournal() {
	for {
		entries := journalRead(journalTail)
		if len(entries) == 0 {
			showPage(T.JournalTitle, []string{styYellow.Render(T.JournalEmpty)})
			return
		}
		res := runPage(T.JournalTitle, "", journalPage(entries), nil,
			[]pageKey{{"c", T.JournalClearKey, "clear"}})
		if res != "clear" {
			return
		}
		yes, ok := askConfirm(T.JournalClearAsk, false)
		if !ok || !yes {
			continue
		}
		if err := journalClear(); err != nil {
			showError(T.JournalTitle, err)
			return
		}
		showPage(T.JournalTitle, []string{styGreen.Render(T.JournalCleared)})
		return
	}
}
