package app

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// jHome изолирует журнал во временном HOME и создаёт каталог конфига.
func jHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		t.Fatal(err)
	}
}

// TestJournalAppendRead: записи дописываются, читаются свежими вперёд, время
// проставляется само.
func TestJournalAppendRead(t *testing.T) {
	setLang("ru")
	jHome(t)

	journalAppend(journalEntry{Kind: journalConnect, Server: "a", Host: "1.1.1.1"})
	journalAppend(journalEntry{Kind: journalRun, Server: "b", Host: "2.2.2.2", Cmd: "uptime", Code: 0})
	journalAppend(journalEntry{Kind: journalRun, Server: "c", Host: "3.3.3.3", Cmd: "false", Code: 1})

	got := journalRead(10)
	if len(got) != 3 {
		t.Fatalf("записей: %d", len(got))
	}
	if got[0].Server != "c" || got[2].Server != "a" {
		t.Fatalf("свежие записи должны быть первыми: %v", got)
	}
	if got[0].Code != 1 || got[0].Cmd != "false" {
		t.Errorf("код и команда потеряны: %+v", got[0])
	}
	if _, err := time.Parse(time.RFC3339, got[0].At); err != nil {
		t.Errorf("время не RFC3339: %q", got[0].At)
	}

	// limit режет хвост, а не голову
	if got := journalRead(1); len(got) != 1 || got[0].Server != "c" {
		t.Fatalf("limit=1 должен вернуть самую свежую: %v", got)
	}
}

// TestJournalPerms: журнал может содержать команды с секретами — только 0600.
func TestJournalPerms(t *testing.T) {
	jHome(t)
	journalAppend(journalEntry{Kind: journalRun, Server: "a", Cmd: "mysql -psecret"})
	fi, err := os.Stat(historyPath())
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("права журнала = %v, ожидались 600", fi.Mode().Perm())
	}
}

// TestJournalSkipsGarbage: одна битая строка не должна прятать весь журнал.
func TestJournalSkipsGarbage(t *testing.T) {
	jHome(t)
	journalAppend(journalEntry{Kind: journalRun, Server: "ok", Cmd: "ls"})
	f, err := os.OpenFile(historyPath(), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{это не json\n\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	got := journalRead(10)
	if len(got) != 1 || got[0].Server != "ok" {
		t.Fatalf("хорошая запись потерялась среди мусора: %v", got)
	}
}

// TestJournalMissingFile: журнала ещё нет — читаем пусто, без паники.
func TestJournalMissingFile(t *testing.T) {
	jHome(t)
	if got := journalRead(10); got != nil {
		t.Fatalf("ожидался nil, получено %v", got)
	}
}

// TestJournalLegacyEntry: у старых записей нет поля kind — считаем их входами.
func TestJournalLegacyEntry(t *testing.T) {
	jHome(t)
	line, _ := json.Marshal(map[string]string{"at": time.Now().Format(time.RFC3339), "server": "old", "host": "h"})
	if err := os.WriteFile(historyPath(), append(line, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	got := journalRead(10)
	if len(got) != 1 || got[0].kindOf() != journalConnect {
		t.Fatalf("старая запись должна читаться как подключение: %+v", got)
	}
}

// TestJournalRotate: разросшийся журнал уезжает в .1, а не растёт вечно.
func TestJournalRotate(t *testing.T) {
	jHome(t)
	big := make([]byte, journalMaxBytes+1)
	for i := range big {
		big[i] = '\n'
	}
	if err := os.WriteFile(historyPath(), big, 0o600); err != nil {
		t.Fatal(err)
	}
	journalAppend(journalEntry{Kind: journalRun, Server: "after", Cmd: "ls"})

	if _, err := os.Stat(historyPath() + ".1"); err != nil {
		t.Fatalf("старый журнал не сохранён в .1: %v", err)
	}
	got := journalRead(10)
	if len(got) != 1 || got[0].Server != "after" {
		t.Fatalf("после ротации в журнале только новая запись: %v", got)
	}
}

// TestJournalClear: очистка убирает и журнал, и его ротацию; повторная — не ошибка.
func TestJournalClear(t *testing.T) {
	jHome(t)
	journalAppend(journalEntry{Kind: journalRun, Server: "a", Cmd: "ls"})
	if err := os.WriteFile(historyPath()+".1", []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := journalClear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(historyPath()); !os.IsNotExist(err) {
		t.Error("журнал не удалён")
	}
	if _, err := os.Stat(historyPath() + ".1"); !os.IsNotExist(err) {
		t.Error("ротация не удалена")
	}
	if err := journalClear(); err != nil {
		t.Fatalf("повторная очистка не должна быть ошибкой: %v", err)
	}
}

// TestJournalRecordHelpers: команда сжимается в одну строку, направление
// передачи отражается стрелкой.
func TestJournalRecordHelpers(t *testing.T) {
	jHome(t)
	s := &Server{Name: "web", Host: "1.2.3.4"}

	journalRecordRun(s, "echo one\necho two", 3)
	journalRecordTransfer(s, "up", "/tmp/a", "~/b")
	journalRecordTransfer(s, "down", "/var/log/syslog", "/tmp")

	got := journalRead(10)
	if len(got) != 3 {
		t.Fatalf("записей: %d", len(got))
	}
	if !strings.Contains(got[2].Cmd, "⏎") || strings.Contains(got[2].Cmd, "\n") {
		t.Errorf("многострочная команда должна схлопнуться в одну строку: %q", got[2].Cmd)
	}
	if got[2].Code != 3 {
		t.Errorf("код выхода: %d", got[2].Code)
	}
	if !strings.Contains(got[1].Cmd, "→") {
		t.Errorf("отправка — стрелка вправо: %q", got[1].Cmd)
	}
	if !strings.Contains(got[0].Cmd, "←") {
		t.Errorf("скачивание — стрелка влево: %q", got[0].Cmd)
	}
}

// TestJournalTable: таблица содержит серверы, коды и вывод сервера — очищенный
// от ANSI (команда могла прийти из сниппета с управляющими последовательностями).
func TestJournalTable(t *testing.T) {
	setLang("ru")
	entries := []journalEntry{
		{At: time.Now().Format(time.RFC3339), Kind: journalRun, Server: "web", Cmd: "\x1b[31mls\x1b[0m", Code: 2},
		{At: time.Now().Format(time.RFC3339), Server: "db"}, // старый формат → вход
		{At: "не-время", Kind: journalTransfer, Server: "cdn", Cmd: "a → b"},
	}
	out := journalTable(entries)
	for _, want := range []string{"web", "db", "cdn", "2"} {
		if !strings.Contains(out, want) {
			t.Errorf("в таблице нет %q", want)
		}
	}
	if strings.Contains(out, "\x1b[31m") {
		t.Error("ANSI из команды не должен доехать до терминала")
	}
	if !strings.Contains(out, "не-время") {
		t.Error("нераспознанная метка времени должна показываться как есть")
	}
}

// TestJournalCodeCell: код показывается только у команд.
func TestJournalCodeCell(t *testing.T) {
	if got := journalCodeCell(journalEntry{Kind: journalConnect}); got != "" {
		t.Errorf("у подключения кода нет: %q", got)
	}
	if got := journalCodeCell(journalEntry{Kind: journalRun, Code: 0}); !strings.Contains(got, "✓") {
		t.Errorf("успех: %q", got)
	}
	if got := journalCodeCell(journalEntry{Kind: journalRun, Code: 7}); !strings.Contains(got, "7") {
		t.Errorf("ненулевой код: %q", got)
	}
}
