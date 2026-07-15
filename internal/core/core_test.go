package core

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"echo hi":   `'echo hi'`,
		"a'b":       `'a'"'"'b'`,
		"":          `''`,
		"no-quotes": `'no-quotes'`,
	}
	for in, want := range cases {
		if got := ShellQuote(in); got != want {
			t.Errorf("ShellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseJump(t *testing.T) {
	tests := []struct{ spec, def, u, h, p string }{
		{"host", "deploy", "deploy", "host", "22"},
		{"user@host", "deploy", "user", "host", "22"},
		{"host:2200", "root", "root", "host", "2200"},
		{"user@host:2222", "deploy", "user", "host", "2222"},
	}
	for _, tt := range tests {
		u, h, p := ParseJump(tt.spec, tt.def)
		if u != tt.u || h != tt.h || p != tt.p {
			t.Errorf("ParseJump(%q) = (%q,%q,%q)", tt.spec, u, h, p)
		}
	}
}

func TestHexToRGB(t *testing.T) {
	if r, g, b := HexToRGB("#ff8800"); r != 255 || g != 136 || b != 0 {
		t.Errorf("#ff8800 = %d,%d,%d", r, g, b)
	}
	if r, g, b := HexToRGB("00ff00"); r != 0 || g != 255 || b != 0 {
		t.Errorf("00ff00 = %d,%d,%d", r, g, b)
	}
}

func TestBlendHex(t *testing.T) {
	if BlendHex("#000000", "#ffffff", 0) != "#000000" {
		t.Error("f=0")
	}
	if BlendHex("#000000", "#ffffff", 1) != "#ffffff" {
		t.Error("f=1")
	}
	if BlendHex("#000000", "#ffffff", 0.5) != "#7f7f7f" {
		t.Error("f=0.5")
	}
}

func TestFormatPing(t *testing.T) {
	if FormatPing(1200*time.Microsecond) != "1.2" {
		t.Errorf("1.2ms = %q", FormatPing(1200*time.Microsecond))
	}
	if FormatPing(50*time.Millisecond) != "50" {
		t.Errorf("50ms = %q", FormatPing(50*time.Millisecond))
	}
	if FormatPing(0) != "0.0" {
		t.Errorf("0 = %q", FormatPing(0))
	}
}

func TestHumanBytes(t *testing.T) {
	if HumanBytes(512, nil) != "512 B" {
		t.Errorf("512 = %q", HumanBytes(512, nil))
	}
	if HumanBytes(1024, nil) != "1.0 KB" {
		t.Errorf("1024 = %q", HumanBytes(1024, nil))
	}
	if HumanBytes(1536, nil) != "1.5 KB" {
		t.Errorf("1536 = %q", HumanBytes(1536, nil))
	}
	if HumanBytes(1024*1024*1024, nil) != "1.0 GB" {
		t.Errorf("1GiB = %q", HumanBytes(1024*1024*1024, nil))
	}
	// пользовательские единицы
	if HumanBytes(2048, []string{"Б", "КБ"}) != "2.0 КБ" {
		t.Errorf("custom units = %q", HumanBytes(2048, []string{"Б", "КБ"}))
	}
}

func TestSanitizeRemote(t *testing.T) {
	if SanitizeRemote("plain") != "plain" {
		t.Error("обычный текст")
	}
	// CSI-последовательность вырезается
	if got := SanitizeRemote("a\x1b[31mred\x1b[0mb"); got != "aredb" {
		t.Errorf("CSI не вырезан: %q", got)
	}
	// OSC с BEL
	if got := SanitizeRemote("x\x1b]0;title\x07y"); got != "xy" {
		t.Errorf("OSC(BEL) не вырезан: %q", got)
	}
	// OSC со ST (ESC \)
	if got := SanitizeRemote("x\x1b]0;t\x1b\\y"); !strings.HasPrefix(got, "x") {
		t.Errorf("OSC(ST): %q", got)
	}
	// \n и \t сохраняются, \r и прочий контроль отбрасываются
	if got := SanitizeRemote("a\tb\nc\r\x00d"); got != "a\tb\ncd" {
		t.Errorf("контрольные символы: %q", got)
	}
}

func TestParseSections(t *testing.T) {
	sec := ParseSections("===OS===\nUbuntu\n===HOST===\nweb\n===EMPTY===\n===END===")
	if sec["OS"] != "Ubuntu" || sec["HOST"] != "web" || sec["EMPTY"] != "" {
		t.Errorf("секции: %v", sec)
	}
}

func TestParseCPUStatLine(t *testing.T) {
	if v := ParseCPUStatLine("cpu 1 2 3"); len(v) != 3 || v[2] != 3 {
		t.Errorf("valid = %v", v)
	}
	if ParseCPUStatLine("intr 1 2") != nil {
		t.Error("не-cpu → nil")
	}
	if ParseCPUStatLine("cpu") != nil {
		t.Error("без чисел → nil")
	}
	if v := ParseCPUStatLine("cpu 1 x 3"); len(v) != 1 {
		t.Errorf("break на нечисле = %v", v)
	}
}

func TestCPUPercentFromStat(t *testing.T) {
	if _, ok := CPUPercentFromStat("cpu 1 2 3 4 5"); ok {
		t.Error("одна строка → false")
	}
	if _, ok := CPUPercentFromStat("cpu 1 2\ncpu 3 4"); ok {
		t.Error("мало полей → false")
	}
	// одинаковые замеры → total 0 → false
	if _, ok := CPUPercentFromStat("cpu 1 2 3 4 5\ncpu 1 2 3 4 5"); ok {
		t.Error("нулевая дельта → false")
	}
	frac, ok := CPUPercentFromStat("cpu 100 0 50 800 5\ncpu 110 0 55 880 5")
	if !ok || frac <= 0 || frac >= 1 {
		t.Errorf("frac=%v ok=%v", frac, ok)
	}
}

func TestCPUNetDeltas(t *testing.T) {
	if _, _, _, ok := CPUNetDeltas([]int64{1}, []int64{1}, 0, 0, 0, 0, 1); ok {
		t.Error("короткий stat → false")
	}
	cpu, rx, tx, ok := CPUNetDeltas(
		[]int64{100, 0, 50, 800}, []int64{110, 0, 55, 880}, 1000, 2000, 3000, 6000, 2)
	if !ok || cpu <= 0 || rx != 1000 || tx != 2000 {
		t.Errorf("cpu=%v rx=%v tx=%v ok=%v", cpu, rx, tx, ok)
	}
	// сброс счётчика → неотрицательные скорости
	_, rx, tx, _ = CPUNetDeltas([]int64{1, 1, 1, 1}, []int64{2, 2, 2, 2}, 9000, 9000, 1, 1, 1)
	if rx != 0 || tx != 0 {
		t.Errorf("клэмп: rx=%v tx=%v", rx, tx)
	}
	// dt=0 → скорости 0
	_, rx, tx, _ = CPUNetDeltas([]int64{1, 1, 1, 1}, []int64{2, 2, 2, 2}, 0, 0, 100, 100, 0)
	if rx != 0 || tx != 0 {
		t.Errorf("dt=0: rx=%v tx=%v", rx, tx)
	}
	// >8 полей (n усекается до 8)
	a := []int64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	b := []int64{2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	if _, _, _, ok := CPUNetDeltas(a, b, 0, 0, 0, 0, 1); !ok {
		t.Error("11 полей → ok")
	}

	// снимки разной длины (обрезанный /proc/stat) не должны ронять программу:
	// n берётся по короткому срезу
	short := []int64{1, 1, 1, 1}
	long := []int64{2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	if _, _, _, ok := CPUNetDeltas(short, long, 0, 0, 0, 0, 1); !ok {
		t.Error("короткий statA + длинный statB → ok без паники")
	}
	if _, _, _, ok := CPUNetDeltas(long, short, 0, 0, 0, 0, 1); !ok {
		t.Error("длинный statA + короткий statB → ok без паники")
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	if ExpandPath("~/x") != filepath.Join(home, "x") {
		t.Error("~/")
	}
	if ExpandPath("/abs") != "/abs" || ExpandPath("rel") != "rel" {
		t.Error("abs/rel")
	}
}

func TestNewID(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}$`)
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := NewID()
		if !re.MatchString(id) || seen[id] {
			t.Fatalf("id=%q", id)
		}
		seen[id] = true
	}
}

func TestGenPassword(t *testing.T) {
	pw := GenPassword(24)
	if len(pw) != 24 {
		t.Fatalf("длина = %d", len(pw))
	}
	if strings.ContainsAny(pw, "0O1lI") {
		t.Errorf("неоднозначные символы: %q", pw)
	}
	if !strings.ContainsAny(pw, "23456789") ||
		!strings.ContainsAny(pw, "ABCDEFGHJKLMNPQRSTUVWXYZ") ||
		!strings.ContainsAny(pw, "!@#%^*-_=+") {
		t.Errorf("не все классы символов: %q", pw)
	}
}

func TestFirstLine(t *testing.T) {
	if FirstLine("a\nb") != "a" || FirstLine("solo") != "solo" {
		t.Error("FirstLine")
	}
}

func TestSSHConfValue(t *testing.T) {
	if SSHConfValue("simple") != "simple" {
		t.Error("простое")
	}
	if !strings.HasPrefix(SSHConfValue("a b"), `"`) {
		t.Error("с пробелом → в кавычках")
	}
	if !strings.Contains(SSHConfValue(`a"b`), `\"`) {
		t.Error("кавычка экранируется")
	}
}

// Перевод строки не должен пролезать в ssh_config: иначе значение дописало бы
// произвольную директиву (ProxyCommand) → выполнение кода при следующем ssh.
func TestSSHConfValueNoNewlineInjection(t *testing.T) {
	evil := "host\n    ProxyCommand touch /tmp/pwned"
	got := SSHConfValue(evil)
	if strings.ContainsAny(got, "\n\r") {
		t.Fatalf("перевод строки не вырезан: %q", got)
	}
	if !strings.Contains(got, "ProxyCommand") {
		t.Fatalf("содержимое потеряно: %q", got)
	}
	// значение с пробелами обязано быть в кавычках — директива не станет отдельной
	if !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) {
		t.Fatalf("значение должно быть закавычено: %q", got)
	}
	if got := SSHConfValue("a\rb"); strings.Contains(got, "\r") {
		t.Fatalf("\\r не вырезан: %q", got)
	}
}
