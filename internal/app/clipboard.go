package app

// Буфер обмена — кроссплатформенно.
//
//	macOS  — pbcopy / pbpaste (есть всегда)
//	Wayland — wl-copy / wl-paste
//	X11    — xclip либо xsel
//
// Если ничего не нашлось, копирование просто не работает: это удобство, а не
// функциональность, и падать из-за него нельзя. Все вызывающие проверяют bool.

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/MrBenoit/unissh/internal/core"
)

// Швы для тестов: подменяются, чтобы прогонять детект без реальных утилит.
var (
	lookPath = exec.LookPath
	getenv   = os.Getenv
	goos     = runtime.GOOS
)

// clipTool — набор команд конкретной утилиты буфера обмена.
type clipTool struct {
	name  string
	copy  []string // куда пишем секрет через stdin
	paste string   // shell-команда, печатающая содержимое буфера
	clear string   // shell-команда, очищающая буфер
}

// hasBin сообщает, есть ли исполняемый файл в PATH.
func hasBin(name string) bool {
	_, err := lookPath(name)
	return err == nil
}

// detectClipTool выбирает утилиту буфера обмена под текущее окружение.
// На Wayland предпочитаем wl-clipboard: xclip там работает только через Xwayland
// и не видит буфер нативных приложений.
func detectClipTool() *clipTool {
	if goos == "darwin" && hasBin("pbcopy") && hasBin("pbpaste") {
		return &clipTool{
			name:  "pbcopy",
			copy:  []string{"pbcopy"},
			paste: "pbpaste",
			clear: `printf "" | pbcopy`,
		}
	}
	wayland := func() *clipTool {
		if !hasBin("wl-copy") || !hasBin("wl-paste") {
			return nil
		}
		return &clipTool{
			name: "wl-copy",
			copy: []string{"wl-copy"},
			// без --no-newline wl-paste дописывает \n и хеш не сойдётся
			paste: "wl-paste --no-newline",
			clear: "wl-copy --clear",
		}
	}
	if getenv("WAYLAND_DISPLAY") != "" {
		if t := wayland(); t != nil {
			return t
		}
	}
	if hasBin("xclip") {
		return &clipTool{
			name:  "xclip",
			copy:  []string{"xclip", "-selection", "clipboard"},
			paste: "xclip -selection clipboard -o",
			clear: `printf "" | xclip -selection clipboard`,
		}
	}
	if hasBin("xsel") {
		return &clipTool{
			name:  "xsel",
			copy:  []string{"xsel", "--clipboard", "--input"},
			paste: "xsel --clipboard --output",
			clear: "xsel --clipboard --clear",
		}
	}
	return wayland() // Wayland без WAYLAND_DISPLAY (tmux, ssh) — последний шанс
}

// clipboardTool кеширует результат детекта: PATH за время работы не меняется.
var clipboardTool = sync.OnceValue(detectClipTool)

// hashCmd — команда, печатающая SHA-256 своего stdin. На macOS это `shasum`
// (из перла), на большинстве Linux — `sha256sum`.
func hashCmd() string {
	if hasBin("sha256sum") {
		return "sha256sum"
	}
	if hasBin("shasum") {
		return "shasum -a 256"
	}
	return ""
}

// clipCopy кладёт текст в буфер обмена. false — утилиты нет или она упала.
func clipCopy(text string) bool {
	t := clipboardTool()
	if t == nil {
		return false
	}
	cmd := exec.Command(t.copy[0], t.copy[1:]...) // #nosec G204 — команда из белого списка
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run() == nil
}

// clipboardTTL — через сколько секунд секрет вычищается из буфера обмена.
const clipboardTTL = 45

// clipCopyEphemeral кладёт секрет в буфер и планирует его автоочистку.
// Фоновый шелл сверяет SHA-256 текущего содержимого буфера с ожидаемым и чистит
// только если пользователь ничего не скопировал поверх. Сам секрет не попадает
// ни в argv, ни на диск — в argv уходит только его хеш.
func clipCopyEphemeral(secret string) bool {
	if !clipCopy(secret) {
		return false
	}
	t, hash := clipboardTool(), hashCmd()
	if hash == "" {
		return true // в буфер положили, но автоочистку не завести
	}
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(secret)))
	script := core.ClipClearScript(t.paste, hash, t.clear, sum, clipboardTTL)

	cmd := exec.Command("sh", "-c", script) // #nosec G204 — секрета в скрипте нет, только хеш
	if err := cmd.Start(); err != nil {
		return true
	}
	go func() { _ = cmd.Wait() }() // не оставляем зомби, если процесс ещё жив
	return true
}
