package app

import (
	"strings"
	"testing"
	"time"
)

// TestServerActionsNoMonitor проверяет, что «Монитор» убран из действий сервера
// (живая статистика теперь всегда в панели справа-внизу).
func TestServerActionsNoMonitor(t *testing.T) {
	setLang("ru")
	s := &Server{ID: "a", Name: "web-1", Auth: "key"}
	for _, a := range serverActions(s) {
		if a.action == "monitor" {
			t.Fatalf("«monitor» не должен быть в действиях сервера")
		}
	}
}

// TestLiveStatsDeltas проверяет живую статистику: после двух опросов считаются
// CPU% и скорости сети, а панель содержит строку пинга и сетевую строку.
func TestLiveStatsDeltas(t *testing.T) {
	setLang("ru")
	applyTheme("blue")
	st := &Store{Config: Config{Servers: []*Server{{ID: "a", Name: "web-1", Host: "192.0.2.10", User: "root", Port: 22}}}}
	m := newPanelsModel(st)
	m.width, m.height = 120, 34

	t0 := time.Now()
	prev := &monSample{
		load: "0.10 0.20 0.30", memLine: "Mem: 1000 400 600",
		dfLine: "/dev/sda1 1000 500 500 50% /", stat: []int64{100, 0, 50, 800},
		rx: 1000, tx: 2000, ncpu: 2, uptime: "up 1 hour",
		ips: []string{"203.0.113.7", "10.0.0.5"}, at: t0,
	}
	cur := &monSample{
		load: "0.10 0.20 0.30", memLine: "Mem: 1000 400 600",
		dfLine: "/dev/sda1 1000 500 500 50% /", stat: []int64{110, 0, 55, 880},
		rx: 3000, tx: 6000, ncpu: 2, uptime: "up 1 hour",
		ips: []string{"203.0.113.7", "10.0.0.5"}, at: t0.Add(2 * time.Second),
	}
	m.applySample("a", prev)
	m.applySample("a", cur)

	ls := m.live["a"]
	if ls == nil || !ls.hasRate {
		t.Fatalf("после двух опросов должны быть посчитаны дельты")
	}
	if ls.cpu <= 0 || ls.cpu >= 1 {
		t.Fatalf("CPU%% вне диапазона: %v", ls.cpu)
	}
	if ls.rxRate != 1000 || ls.txRate != 2000 { // (3000-1000)/2с, (6000-2000)/2с
		t.Fatalf("скорость сети посчитана неверно: rx=%v tx=%v", ls.rxRate, ls.txRate)
	}
	out := strings.Join(m.statLines(st.Servers[0], 60), "\n")
	if !strings.Contains(out, T.ColPing) {
		t.Fatalf("нет строки пинга в статистике:\n%s", out)
	}
	if !strings.Contains(out, T.LblNet) {
		t.Fatalf("нет сетевой строки в статистике:\n%s", out)
	}
	if !strings.Contains(out, T.LblIPs) || !strings.Contains(out, "203.0.113.7") {
		t.Fatalf("нет IP-адресов внизу статистики:\n%s", out)
	}
}

// TestPanelsRender проверяет, что панельный лейаут (v2) собирается без паники,
// строит кликабельные регионы и переключает правую панель по фокусу.
func TestPanelsRender(t *testing.T) {
	setLang("ru")
	applyTheme("blue")
	st := &Store{}
	st.Servers = []*Server{
		{ID: "a", Name: "web-1", Host: "192.0.2.10", User: "root", Port: 22, Auth: "password", Favorite: true},
		{ID: "b", Name: "db-1", Host: "192.0.2.20", User: "deploy", Port: 22, Auth: "key"},
	}
	m := newPanelsModel(st)
	m.width, m.height = 120, 34

	out := m.render()
	if !strings.Contains(out, "web-1") || !strings.Contains(out, T.PanelServers) {
		t.Fatalf("нет списка серверов в выводе:\n%s", out)
	}
	if len(m.regions) == 0 {
		t.Fatalf("кликабельные регионы не построены")
	}

	// фокус на меню → справа описание пункта, а не действия сервера
	m.focus = fpMenu
	if out = m.render(); !strings.Contains(out, T.PanelDesc) {
		t.Fatalf("при фокусе на меню ожидалась панель описания")
	}

	// фокус на действиях → справа-вверху действия выбранного сервера
	m.focus = fpActions
	if out = m.render(); !strings.Contains(out, T.PanelActions) {
		t.Fatalf("при фокусе на действиях ожидалась панель действий")
	}
}

// TestPanelsFocusCycle проверяет, что Tab ходит только между двумя окнами
// (серверы ↔ меню), а Enter на сервере открывает панель действий.
func TestPanelsFocusCycle(t *testing.T) {
	st := &Store{}
	st.Servers = []*Server{{ID: "a", Name: "web-1", Host: "192.0.2.10", User: "root", Port: 22}}
	m := newPanelsModel(st)

	// серверы ↔ меню
	m.focus = fpServers
	m.cycleFocus(1)
	if m.focus != fpMenu {
		t.Fatalf("после серверов ожидалось меню, получили %v", m.focus)
	}
	m.cycleFocus(1)
	if m.focus != fpServers {
		t.Fatalf("после меню ожидались серверы, получили %v", m.focus)
	}

	// Enter на сервере открывает действия (фокус fpActions), не подключает
	m.focus = fpServers
	m.activate()
	if m.focus != fpActions {
		t.Fatalf("Enter на сервере должен открыть действия, получили %v", m.focus)
	}
	if m.result != nil {
		t.Fatalf("Enter на сервере не должен сразу подключать (result=%v)", m.result)
	}
	// Tab из действий уводит обратно к меню/серверам (не остаётся на действиях)
	m.cycleFocus(1)
	if m.focus == fpActions {
		t.Fatalf("Tab должен уводить с панели действий")
	}

	// без серверов доступно только меню
	empty := newPanelsModel(&Store{})
	if empty.focus != fpMenu {
		t.Fatalf("без серверов фокус должен быть на меню")
	}
	empty.cycleFocus(1)
	if empty.focus != fpMenu {
		t.Fatalf("без серверов обход должен оставаться на меню")
	}
}
