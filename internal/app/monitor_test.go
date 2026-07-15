package app

import (
	"testing"
	"time"
)

func TestParseMonSample(t *testing.T) {
	out := "0.15 0.10 0.05 1/234 5678" +
		"\n@\nMem: 2000 800 1200 0 0 1100" +
		"\n@\n/dev/sda1 41943040 20971520 20971520 50% /" +
		"\n@\ncpu 100 0 50 800 5 0 2 0 0 0" +
		"\n@\nlo: 999 0 0 0 0 0 0 0 999 0 0 0 0 0 0 0\neth0: 5000 10 0 0 0 0 0 0 8000 20 0 0 0 0 0 0" +
		"\n@\n4" +
		"\n@\nup 2 hours" +
		"\n@\n203.0.113.9\n10.0.0.2\n203.0.113.9"
	s := parseMonSample(out, time.Now())
	if s.err != nil {
		t.Fatalf("неожиданная ошибка: %v", s.err)
	}
	if len(s.stat) != 10 || s.stat[0] != 100 || s.stat[3] != 800 {
		t.Fatalf("stat разобран неверно: %v", s.stat)
	}
	if s.rx != 5000 || s.tx != 8000 { // lo пропущен, взят eth0
		t.Fatalf("сеть разобрана неверно: rx=%d tx=%d", s.rx, s.tx)
	}
	if s.ncpu != 4 {
		t.Fatalf("ncpu = %d", s.ncpu)
	}
	if s.uptime != "up 2 hours" {
		t.Fatalf("uptime = %q", s.uptime)
	}
	// дубликат IP отфильтрован
	if len(s.ips) != 2 || s.ips[0] != "203.0.113.9" || s.ips[1] != "10.0.0.2" {
		t.Fatalf("ips разобраны неверно: %v", s.ips)
	}
}

func TestParseMonSampleTooShort(t *testing.T) {
	s := parseMonSample("only\n@\ntwo\n@\nthree", time.Now())
	if s.err == nil {
		t.Fatal("при недостатке секций должна быть ошибка")
	}
}

func TestComputeDeltas(t *testing.T) {
	t0 := time.Now()
	prev := &monSample{stat: []int64{100, 0, 50, 800}, rx: 1000, tx: 2000, at: t0}
	cur := &monSample{stat: []int64{110, 0, 55, 880}, rx: 3000, tx: 6000, at: t0.Add(2 * time.Second)}
	cpu, rx, tx, ok := computeDeltas(prev, cur)
	if !ok {
		t.Fatal("ожидались посчитанные дельты")
	}
	// total=(10+0+5+80)=95, idle=80 → cpu=15/95≈0.158
	if cpu <= 0 || cpu >= 1 {
		t.Fatalf("cpu вне диапазона: %v", cpu)
	}
	if rx != 1000 || tx != 2000 { // (3000-1000)/2, (6000-2000)/2
		t.Fatalf("скорости неверны: rx=%v tx=%v", rx, tx)
	}
}

func TestComputeDeltasNoPrev(t *testing.T) {
	if _, _, _, ok := computeDeltas(nil, &monSample{stat: []int64{1, 2, 3, 4}}); ok {
		t.Fatal("без предыдущего снимка ok должно быть false")
	}
	// отрицательная дельта счётчика (сброс) не даёт отрицательной скорости
	t0 := time.Now()
	prev := &monSample{stat: []int64{100, 0, 50, 800}, rx: 9000, tx: 9000, at: t0}
	cur := &monSample{stat: []int64{110, 0, 55, 880}, rx: 10, tx: 10, at: t0.Add(time.Second)}
	_, rx, tx, ok := computeDeltas(prev, cur)
	if !ok || rx != 0 || tx != 0 {
		t.Fatalf("сброс счётчика должен давать 0: rx=%v tx=%v", rx, tx)
	}
}
