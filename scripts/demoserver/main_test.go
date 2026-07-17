package main

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// startServer поднимает демо-сервер на свободном порту и возвращает адрес.
func startServer(t *testing.T, load string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	p, err := parseSpec(ln.Addr().String() + "=" + load)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := newServerConfig()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go p.serve(conn, cfg)
		}
	}()
	return ln.Addr().String()
}

// runCmd подключается к серверу так же, как это делает unissh, и выполняет команду.
func runCmd(t *testing.T, addr, cmd string) string {
	t.Helper()
	cl, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User:            "deploy",
		Auth:            []ssh.AuthMethod{ssh.Password("x")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // тест против своего же сервера
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = cl.Close() }()

	sess, err := cl.NewSession()
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	defer func() { _ = sess.Close() }()

	out, err := sess.Output(cmd)
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	return string(out)
}

// TestServerAnswersMonCmd — главная проверка: сервер должен отвечать на monCmd
// ровно тем, что ждёт parseMonSample. Если формат разъедется, демо покажет
// «loading…» вместо шкал, и заметить это можно только глазами на гифке.
func TestServerAnswersMonCmd(t *testing.T) {
	addr := startServer(t, "42,31,24")
	// команда, как её шлёт unissh: обёрнута в sh -c, содержит /proc/loadavg
	out := runCmd(t, addr, "LANG=C LC_ALL=C sh -c 'cat /proc/loadavg; echo @; free -b'")

	parts := strings.Split(out, "\n@\n")
	if len(parts) != 8 {
		t.Fatalf("ожидалось 8 секций (столько ждёт parseMonSample), получено %d:\n%s",
			len(parts), out)
	}

	// память: поля [1]=total, [2]=used → 31%
	mem := strings.Fields(parts[1])
	total, _ := strconv.ParseFloat(mem[1], 64)
	used, _ := strconv.ParseFloat(mem[2], 64)
	if pct := used / total * 100; pct < 30.5 || pct > 31.5 {
		t.Errorf("память = %.1f%%, ожидалось 31%%", pct)
	}

	// диск: поля [1]=total, [2]=used → 24%
	df := strings.Fields(parts[2])
	dt, _ := strconv.ParseFloat(df[1], 64)
	du, _ := strconv.ParseFloat(df[2], 64)
	if pct := du / dt * 100; pct < 23.5 || pct > 24.5 {
		t.Errorf("диск = %.1f%%, ожидалось 24%%", pct)
	}

	// сеть: fields[1]=rx, fields[9]=tx — позиции критичны
	nd := strings.Fields(parts[4])
	if len(nd) < 10 {
		t.Fatalf("строка /proc/net/dev короче 10 полей: %q", parts[4])
	}
	if strings.HasPrefix(nd[0], "lo") {
		t.Errorf("интерфейс не должен начинаться с lo (его пропускают): %q", nd[0])
	}
}

// TestCPUDeltaGivesTargetPercent: unissh считает CPU по дельте двух снимков.
// Проверяем ровно эту арифметику — из одного снимка процент не получить.
func TestCPUDeltaGivesTargetPercent(t *testing.T) {
	addr := startServer(t, "42,31,24")
	cmd := "sh -c 'cat /proc/loadavg'"

	statOf := func(out string) []int64 {
		fields := strings.Fields(strings.Split(out, "\n@\n")[3])
		var v []int64
		for _, f := range fields[1:] { // как в parseMonSample: метку «cpu» пропускаем
			n, _ := strconv.ParseInt(f, 10, 64)
			v = append(v, n)
		}
		return v
	}
	a := statOf(runCmd(t, addr, cmd))
	b := statOf(runCmd(t, addr, cmd))

	// повторяем формулу core.CPUNetDeltas
	var total, idle int64
	for i := range 8 {
		total += b[i] - a[i]
	}
	idle = b[3] - a[3]
	if total <= 0 {
		t.Fatalf("счётчики не выросли между снимками: %v → %v", a, b)
	}
	pct := float64(total-idle) / float64(total) * 100
	if pct < 40 || pct > 44 {
		t.Errorf("CPU по дельте = %.1f%%, ожидалось ~42%%", pct)
	}
}

// TestNetDeltaGrows: счётчики сети должны расти, иначе скорость всегда 0.
func TestNetDeltaGrows(t *testing.T) {
	addr := startServer(t, "10,10,10")
	cmd := "sh -c 'cat /proc/loadavg'"
	rxOf := func(out string) int64 {
		f := strings.Fields(strings.Split(out, "\n@\n")[4])
		n, _ := strconv.ParseInt(f[1], 10, 64)
		return n
	}
	a, b := rxOf(runCmd(t, addr, cmd)), rxOf(runCmd(t, addr, cmd))
	if b <= a {
		t.Fatalf("rx не растёт: %d → %d", a, b)
	}
}

// TestParseSpec: разбор аргумента командной строки.
func TestParseSpec(t *testing.T) {
	p, err := parseSpec("203.0.113.10:2222=42,31,24")
	if err != nil {
		t.Fatal(err)
	}
	if p.addr != "203.0.113.10:2222" || p.cpu != 0.42 || p.mem != 0.31 || p.disk != 0.24 {
		t.Fatalf("разобрано неверно: %+v", p)
	}
	for _, bad := range []string{"нет-равно", "a:1=1,2", "a:1=x,y,z"} {
		if _, err := parseSpec(bad); err == nil {
			t.Errorf("%q должно быть ошибкой", bad)
		}
	}
}
