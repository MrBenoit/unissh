package core

import (
	"strings"
	"testing"
)

// TestClipClearScript: скрипт сверяет ХЕШ буфера, а не сам секрет — иначе
// секрет уехал бы в argv фонового процесса и был бы виден в `ps`.
func TestClipClearScript(t *testing.T) {
	sum := "deadbeef"
	got := ClipClearScript("pbpaste", "shasum -a 256", `printf "" | pbcopy`, sum, 45)

	for _, want := range []string{
		"sleep 45;",
		"pbpaste | shasum -a 256 | cut",
		`= "deadbeef"`,
		`then printf "" | pbcopy; fi`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("в скрипте нет %q:\n%s", want, got)
		}
	}
	// одна строка: скрипт уходит в `sh -c`
	if strings.Contains(got, "\n") {
		t.Errorf("скрипт должен быть однострочным: %q", got)
	}
}

// TestClipClearScriptWayland: команды подставляются как есть, без предположений
// о том, что это pbcopy.
func TestClipClearScriptWayland(t *testing.T) {
	got := ClipClearScript("wl-paste --no-newline", "sha256sum", "wl-copy --clear", "abc", 10)
	if !strings.Contains(got, "wl-paste --no-newline | sha256sum") {
		t.Errorf("paste/hash: %q", got)
	}
	if !strings.Contains(got, "then wl-copy --clear; fi") {
		t.Errorf("clear: %q", got)
	}
	if !strings.HasPrefix(got, "sleep 10;") {
		t.Errorf("ttl: %q", got)
	}
}
