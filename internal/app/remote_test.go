package app

import (
	"strings"
	"testing"
)

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"echo hi":     `'echo hi'`,
		"a'b":         `'a'"'"'b'`,
		"":            `''`,
		"no-quotes":   `'no-quotes'`,
		"it's a test": `'it'"'"'s a test'`,
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
	// экранированная строка должна безопасно оборачивать одинарные кавычки
	if strings.Count(shellQuote("awk '{print $1}'"), "'") < 4 {
		t.Error("одинарные кавычки должны экранироваться")
	}
}

func TestParseJump(t *testing.T) {
	tests := []struct {
		spec, defUser       string
		wUser, wHost, wPort string
	}{
		{"host", "deploy", "deploy", "host", "22"},
		{"user@host", "deploy", "user", "host", "22"},
		{"host:2200", "root", "root", "host", "2200"},
		{"user@host:2222", "deploy", "user", "host", "2222"},
	}
	for _, tt := range tests {
		u, h, p := parseJump(tt.spec, tt.defUser)
		if u != tt.wUser || h != tt.wHost || p != tt.wPort {
			t.Errorf("parseJump(%q,%q) = (%q,%q,%q), want (%q,%q,%q)",
				tt.spec, tt.defUser, u, h, p, tt.wUser, tt.wHost, tt.wPort)
		}
	}
}

func TestLineWriter(t *testing.T) {
	var lines []string
	w := &lineWriter{onLine: func(l string) { lines = append(lines, l) }}
	w.Write([]byte("first\nsec"))
	w.Write([]byte("ond\nthird\n"))
	if len(lines) != 3 || lines[0] != "first" || lines[1] != "second" || lines[2] != "third" {
		t.Fatalf("построчный вывод неверен: %v", lines)
	}
	if !strings.Contains(w.String(), "first") {
		t.Errorf("String() должен содержать весь вывод: %q", w.String())
	}
}
