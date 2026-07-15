package core

import "testing"

// TestMemUsage: строка `free -b` — поля total(1) и used(2).
func TestMemUsage(t *testing.T) {
	used, total, ok := MemUsage("Mem: 8000 2000 6000 0 0 5900")
	if !ok || used != 2000 || total != 8000 {
		t.Fatalf("used=%v total=%v ok=%v", used, total, ok)
	}
	for _, bad := range []string{"", "Mem:", "Mem: 8000", "Mem: x y z", "Mem: 0 0 0"} {
		if _, _, ok := MemUsage(bad); ok {
			t.Errorf("строка %q не должна разбираться", bad)
		}
	}
}

// TestDiskUsage: `df -P -k` даёт килобайты, наружу отдаём байты.
func TestDiskUsage(t *testing.T) {
	used, total, ok := DiskUsage("/dev/sda1 1000 250 750 25% /")
	if !ok || used != 250*1024 || total != 1000*1024 {
		t.Fatalf("used=%v total=%v ok=%v", used, total, ok)
	}
	for _, bad := range []string{"", "a b c", "a x y z", "a 0 0 0"} {
		if _, _, ok := DiskUsage(bad); ok {
			t.Errorf("строка %q не должна разбираться", bad)
		}
	}
}

// TestLoadFields: берём ровно три первых числа loadavg.
func TestLoadFields(t *testing.T) {
	got, ok := LoadFields("0.10 0.20 0.30 1/234 5678")
	if !ok || got != "0.10 0.20 0.30" {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
	if _, ok := LoadFields("0.10 0.20"); ok {
		t.Error("двух чисел мало")
	}
}

// TestLoadPerCPU: нормировка на число ядер — чтобы 8-ядерный и 1-ядерный
// сервер можно было сравнивать одной шкалой.
func TestLoadPerCPU(t *testing.T) {
	if v, ok := LoadPerCPU("4.00 1 1", 4); !ok || v != 1.0 {
		t.Fatalf("v=%v ok=%v", v, ok)
	}
	if v, ok := LoadPerCPU("1.00 1 1", 4); !ok || v != 0.25 {
		t.Fatalf("v=%v ok=%v", v, ok)
	}
	if _, ok := LoadPerCPU("", 4); ok {
		t.Error("пустая строка")
	}
	if _, ok := LoadPerCPU("nan-ish", 4); ok {
		t.Error("нечисло")
	}
	if _, ok := LoadPerCPU("1.0", 0); ok {
		t.Error("ноль ядер: делить нельзя")
	}
}

// TestFrac: доля всегда в [0,1], деления на ноль нет.
func TestFrac(t *testing.T) {
	if Frac(1, 2) != 0.5 {
		t.Error("1/2")
	}
	if Frac(5, 0) != 0 {
		t.Error("total=0 → 0")
	}
	if Frac(-1, 2) != 0 {
		t.Error("отрицательное → 0")
	}
	if Frac(3, 2) != 1 {
		t.Error("больше единицы → 1")
	}
}

// TestOneLine: перевод строки заменяется на ⏎, обрезка идёт по рунам —
// кириллица не должна разрубаться пополам.
func TestOneLine(t *testing.T) {
	if got := OneLine("echo one\necho two", 100); got != "echo one ⏎ echo two" {
		t.Errorf("got %q", got)
	}
	if got := OneLine("a\r\nb", 100); got != "a ⏎ b" {
		t.Errorf("CRLF: %q", got)
	}
	if got := OneLine("  много   пробелов  ", 100); got != "много пробелов" {
		t.Errorf("пробелы: %q", got)
	}
	got := OneLine("привет мир", 6)
	if []rune(got)[len([]rune(got))-1] != '…' {
		t.Errorf("нет многоточия: %q", got)
	}
	if len([]rune(got)) != 6 {
		t.Errorf("длина в рунах = %d, ожидалось 6: %q", len([]rune(got)), got)
	}
	if OneLine("короткая", 100) != "короткая" {
		t.Error("короткую строку не трогаем")
	}
	if OneLine("abc", 1) != "abc" {
		t.Error("max<2 — обрезать нечем, отдаём как есть")
	}
}
