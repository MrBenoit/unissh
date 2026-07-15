package app

import (
	"strings"
	"testing"
)

func TestHexToRGB(t *testing.T) {
	r, g, b := hexToRGB("#ff8800")
	if r != 255 || g != 136 || b != 0 {
		t.Fatalf("hexToRGB(#ff8800) = %d,%d,%d", r, g, b)
	}
	r, g, b = hexToRGB("00ff00") // без решётки
	if r != 0 || g != 255 || b != 0 {
		t.Fatalf("hexToRGB(00ff00) = %d,%d,%d", r, g, b)
	}
}

func TestBlendHex(t *testing.T) {
	if got := blendHex("#000000", "#ffffff", 0); got != "#000000" {
		t.Errorf("blend f=0 = %q", got)
	}
	if got := blendHex("#000000", "#ffffff", 1); got != "#ffffff" {
		t.Errorf("blend f=1 = %q", got)
	}
	if got := blendHex("#000000", "#ffffff", 0.5); got != "#7f7f7f" {
		t.Errorf("blend f=0.5 = %q", got)
	}
}

func TestMeter(t *testing.T) {
	// 50% из 10 → 5 закрашенных, 5 пустых, подпись 50%
	out := meter(0.5, 10)
	if strings.Count(out, "█") != 5 || strings.Count(out, "░") != 5 {
		t.Errorf("meter(0.5,10) заполнение неверно: %q", out)
	}
	if !strings.Contains(out, "50%") {
		t.Errorf("meter должен содержать проценты: %q", out)
	}
	// клэмпинг за границы
	if strings.Count(meter(-1, 8), "█") != 0 {
		t.Error("meter(-1) должен быть пустым")
	}
	if strings.Count(meter(2, 8), "█") != 8 || !strings.Contains(meter(2, 8), "100%") {
		t.Error("meter(2) должен быть заполнен на 100%")
	}
}

func TestHumanBytes(t *testing.T) {
	if !strings.HasPrefix(humanBytes(512), "512 ") {
		t.Errorf("humanBytes(512) = %q", humanBytes(512))
	}
	if !strings.HasPrefix(humanBytes(1024), "1.0 ") {
		t.Errorf("humanBytes(1024) = %q", humanBytes(1024))
	}
	if !strings.HasPrefix(humanBytes(1536), "1.5 ") {
		t.Errorf("humanBytes(1536) = %q", humanBytes(1536))
	}
	if !strings.HasPrefix(humanBytes(1024*1024), "1.0 ") {
		t.Errorf("humanBytes(1MiB) = %q", humanBytes(1024*1024))
	}
}

func TestGradientLine(t *testing.T) {
	// не должно паниковать на пустой/односимвольной строке
	_ = gradientLine("")
	_ = gradientLine("x")
	out := gradientLine("hello")
	if !strings.Contains(out, "h") || !strings.Contains(out, "o") {
		t.Errorf("gradientLine потерял символы: %q", out)
	}
}

func TestApplyThemeSetsStyles(t *testing.T) {
	// смена темы не должна паниковать и должна менять акцентный цвет
	applyTheme("dracula")
	a := cPink
	applyTheme("gruvbox")
	b := cPink
	applyTheme("blue") // вернуть по умолчанию для остальных тестов
	if a == "" || b == "" {
		t.Error("applyTheme должен задавать цвета")
	}
}
