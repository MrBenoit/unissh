package app

import (
	"strings"
	"testing"
)

func TestParseSections(t *testing.T) {
	raw := "===OS===\nUbuntu 22.04\n===HOST===\nweb-1\n===EMPTY===\n===END==="
	sec := parseSections(raw)
	if sec["OS"] != "Ubuntu 22.04" {
		t.Errorf("OS = %q", sec["OS"])
	}
	if sec["HOST"] != "web-1" {
		t.Errorf("HOST = %q", sec["HOST"])
	}
	if sec["EMPTY"] != "" {
		t.Errorf("EMPTY должно быть пустым, got %q", sec["EMPTY"])
	}
}

func TestParseCPUStatLine(t *testing.T) {
	v := parseCPUStatLine("cpu 100 200 300 400")
	if len(v) != 4 || v[0] != 100 || v[3] != 400 {
		t.Fatalf("parseCPUStatLine = %v", v)
	}
	if parseCPUStatLine("notcpu 1 2 3") != nil {
		t.Error("строка без префикса cpu → nil")
	}
	if parseCPUStatLine("cpu") != nil {
		t.Error("строка без чисел → nil")
	}
}

func TestOrDash(t *testing.T) {
	if orDash("") != "—" || orDash("  ") != "—" {
		t.Error("пустое → тире")
	}
	if orDash("  value  ") != "value" {
		t.Error("orDash должен тримить")
	}
}

func TestIPAddrLines(t *testing.T) {
	// из секции IPS (формат iface\taddr/mask)
	sec := map[string]string{"IPS": "eth0\t203.0.113.5/24\n-\t2001:db8::1/64"}
	lines := ipAddrLines(sec)
	if len(lines) != 2 || !strings.Contains(lines[0], "203.0.113.5") {
		t.Fatalf("IPS не разобран: %v", lines)
	}
	if !strings.Contains(lines[0], "eth0") {
		t.Errorf("должна быть метка интерфейса: %q", lines[0])
	}
	// запасной путь: hostname -I
	fb := ipAddrLines(map[string]string{"IP": "10.0.0.1 10.0.0.2"})
	if len(fb) != 2 {
		t.Fatalf("fallback hostname -I: %v", fb)
	}
	// пусто → тире
	if got := ipAddrLines(map[string]string{}); len(got) != 1 || got[0] != "—" {
		t.Fatalf("пусто должно давать тире: %v", got)
	}
}
