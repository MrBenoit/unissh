package app

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

// hSrv — заготовка сервера для тестов дашборда.
func hSrv(name string) *Server {
	return &Server{ID: name, Name: name, Host: "127.0.0.1", Port: 1, User: "u"}
}

// TestHealthWorst: сортировочный вес — максимум из четырёх метрик, включая
// load на ядро (иначе перегруженный по load сервер уехал бы вниз).
func TestHealthWorst(t *testing.T) {
	h := healthRow{cpu: 0.2, mem: 0.5, disk: 0.1, loadPerCPU: 0.9}
	if got := h.worst(); got != 0.9 {
		t.Fatalf("worst = %v, ожидалось 0.9", got)
	}
	h = healthRow{cpu: 0.99}
	if got := h.worst(); got != 0.99 {
		t.Fatalf("worst = %v", got)
	}
}

// TestSortHealth: сверху — нагруженные, потом живые без метрик, внизу — офлайн;
// внутри группы порядок по имени, чтобы вывод не «плавал».
func TestSortHealth(t *testing.T) {
	rows := []healthRow{
		{srv: hSrv("offline-b")},
		{srv: hSrv("quiet"), online: true, hasData: true, cpu: 0.05},
		{srv: hSrv("noconn-b"), online: true, errText: "боль"},
		{srv: hSrv("hot"), online: true, hasData: true, cpu: 0.95},
		{srv: hSrv("noconn-a"), online: true, errText: "боль"},
		{srv: hSrv("offline-a")},
	}
	sortHealth(rows)
	var got []string
	for _, r := range rows {
		got = append(got, r.srv.Name)
	}
	want := []string{"hot", "quiet", "noconn-a", "noconn-b", "offline-a", "offline-b"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("порядок = %v, ожидался %v", got, want)
	}
}

// TestHealthFill: два снимка → CPU по дельте, остальное из последнего.
func TestHealthFill(t *testing.T) {
	t0 := time.Now()
	prev := &monSample{stat: []int64{100, 0, 0, 900, 0, 0, 0, 0}, at: t0}
	cur := &monSample{
		stat:    []int64{200, 0, 0, 1700, 0, 0, 0, 0}, // 100 занято из 900 тиков
		memLine: "Mem: 1000 250 750",
		dfLine:  "/dev/sda 4000 1000 3000 25% /",
		load:    "2.00 1.00 0.50 1/2 3",
		ncpu:    4,
		at:      t0.Add(time.Second),
	}
	var h healthRow
	h.fill(prev, cur)

	if !h.hasData {
		t.Fatal("hasData")
	}
	if h.cpu <= 0.10 || h.cpu >= 0.13 { // 100/(100+800) ≈ 0.111
		t.Errorf("cpu = %v", h.cpu)
	}
	if h.mem != 0.25 {
		t.Errorf("mem = %v", h.mem)
	}
	if h.disk != 0.25 {
		t.Errorf("disk = %v", h.disk)
	}
	if h.load != "2.00 1.00 0.50" {
		t.Errorf("load = %q", h.load)
	}
	if h.loadPerCPU != 0.5 { // 2.00 на 4 ядра
		t.Errorf("loadPerCPU = %v", h.loadPerCPU)
	}
}

// TestHealthStatus: три состояния колонки — офлайн, ошибка опроса, пинг.
// Текст ошибки приходит от сервера и обязан пройти через sanitizeRemote.
func TestHealthStatus(t *testing.T) {
	setLang("ru")
	if got := healthStatus(healthRow{srv: hSrv("a")}); !strings.Contains(got, T.StatsOffline) {
		t.Errorf("офлайн: %q", got)
	}
	got := healthStatus(healthRow{srv: hSrv("a"), online: true, errText: "\x1b[31mboom\x1b[0m"})
	if strings.Contains(got, "\x1b[31m") {
		t.Errorf("ANSI сервера доехал до терминала: %q", got)
	}
	if !strings.Contains(got, "boom") {
		t.Errorf("текст ошибки потерян: %q", got)
	}
	got = healthStatus(healthRow{srv: hSrv("a"), online: true, hasData: true, ping: 12 * time.Millisecond})
	if !strings.Contains(got, "12") {
		t.Errorf("пинг не показан: %q", got)
	}
}

// TestHealthCell: без данных — прочерк, с данными — шкала с процентами.
func TestHealthCell(t *testing.T) {
	if got := healthCell(0.5, false); !strings.Contains(got, "—") {
		t.Errorf("без данных: %q", got)
	}
	if got := healthCell(0.5, true); !strings.Contains(got, "50%") {
		t.Errorf("со шкалой: %q", got)
	}
}

// TestHealthTableAndSummary: таблица содержит все серверы, сводка — счётчики.
func TestHealthTableAndSummary(t *testing.T) {
	setLang("ru")
	rows := []healthRow{
		{srv: hSrv("hot"), online: true, hasData: true, cpu: 0.9, load: "1.00 1.00 1.00"},
		{srv: hSrv("mute"), online: true, errText: "нет доступа"},
		{srv: hSrv("dead")},
	}
	tbl := healthTable(rows)
	for _, name := range []string{"hot", "mute", "dead"} {
		if !strings.Contains(tbl, name) {
			t.Errorf("в таблице нет сервера %q", name)
		}
	}
	if !strings.Contains(tbl, "—") {
		t.Error("у сервера без метрик должен быть прочерк")
	}
	sum := healthSummary(rows)
	if !strings.Contains(sum, "2") || !strings.Contains(sum, "3") {
		t.Errorf("сводка «онлайн 2 из 3»: %q", sum)
	}
	if !strings.Contains(sum, "1") {
		t.Errorf("сводка должна упомянуть 1 сервер без метрик: %q", sum)
	}
}

// TestHealthSummaryAllGood: если метрики отдали все живые, про «без метрик» молчим.
func TestHealthSummaryAllGood(t *testing.T) {
	setLang("ru")
	sum := healthSummary([]healthRow{{srv: hSrv("a"), online: true, hasData: true}})
	if strings.Contains(sum, T.HealthNoDataFmt[:6]) {
		t.Errorf("лишнее предупреждение: %q", sum)
	}
}

// TestCollectHealthParallel: собирает все строки и сортирует их; сам опрос
// подменён швом fetchHealth.
func TestCollectHealth(t *testing.T) {
	orig := fetchHealth
	defer func() { fetchHealth = orig }()
	fetchHealth = func(s *Server, _ time.Duration) healthRow {
		switch s.Name {
		case "hot":
			return healthRow{srv: s, online: true, hasData: true, cpu: 0.8}
		case "cold":
			return healthRow{srv: s, online: true, hasData: true, cpu: 0.1}
		}
		return healthRow{srv: s}
	}
	rows := collectHealth([]*Server{hSrv("cold"), hSrv("dead"), hSrv("hot")}, time.Second)
	if len(rows) != 3 {
		t.Fatalf("строк: %d", len(rows))
	}
	if rows[0].srv.Name != "hot" || rows[2].srv.Name != "dead" {
		t.Fatalf("порядок: %s, %s, %s", rows[0].srv.Name, rows[1].srv.Name, rows[2].srv.Name)
	}
}

// TestFetchHealthOffline: к мёртвому серверу по SSH не ходим вовсе — иначе один
// недоступный VPS держал бы весь сбор до таймаута.
func TestFetchHealthOffline(t *testing.T) {
	orig := openRemote
	called := false
	openRemote = func(s *Server, ask bool) (*Remote, error) { called = true; return nil, errNoAuth }
	defer func() { openRemote = orig }()

	// порт 1 на localhost гарантированно закрыт
	row := fetchHealth(&Server{ID: "x", Name: "x", Host: "127.0.0.1", Port: 1}, 200*time.Millisecond)
	if row.online || row.hasData {
		t.Fatal("сервер должен считаться офлайн")
	}
	if called {
		t.Fatal("openRemote не должен вызываться для офлайн-сервера")
	}
}

// TestFetchHealthOverSSH: полный путь через настоящий (внутрипроцессный) SSH.
func TestFetchHealthOverSSH(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())

	calls := 0
	host, port := startTestSSHServer(t, true, func(cmd string) (string, int) {
		calls++
		idle := 900
		if calls > 1 {
			idle = 1700 // второй снимок: 100 тиков user, 800 idle → CPU ≈ 11%
		}
		user := 100 * calls
		return monSampleOut(user, idle), 0
	})

	origOpen := openRemote
	origDelay := healthCPUDelay
	healthCPUDelay = 0 // счётчики подставляем сами, ждать нечего
	openRemote = func(s *Server, ask bool) (*Remote, error) {
		return dialSSH(host, port, "u", "pw", "", "", 5*time.Second)
	}
	defer func() { openRemote, healthCPUDelay = origOpen, origDelay }()

	row := fetchHealth(&Server{ID: "x", Name: "x", Host: host, Port: port, User: "u"}, 2*time.Second)
	if !row.online {
		t.Fatal("сервер онлайн: SSH-сервер слушает")
	}
	if row.errText != "" {
		t.Fatalf("неожиданная ошибка: %s", row.errText)
	}
	if !row.hasData {
		t.Fatal("метрики должны собраться")
	}
	if row.mem != 0.25 {
		t.Errorf("mem = %v", row.mem)
	}
	if row.cpu <= 0 {
		t.Errorf("CPU считается по дельте двух снимков, получено %v", row.cpu)
	}
	if calls != 2 {
		t.Errorf("нужно ровно два снимка, сделано %d", calls)
	}
}

// TestFetchHealthConnectError: не смогли подключиться — сервер онлайн, но без
// метрик, и причина видна пользователю.
func TestFetchHealthConnectError(t *testing.T) {
	setLang("ru")
	host, port := startTestSSHServer(t, true, func(cmd string) (string, int) { return "", 0 })

	orig := openRemote
	openRemote = func(s *Server, ask bool) (*Remote, error) { return nil, errNoAuth }
	defer func() { openRemote = orig }()

	row := fetchHealth(&Server{ID: "x", Name: "x", Host: host, Port: port, User: "u"}, 2*time.Second)
	if !row.online {
		t.Fatal("TCP-проба прошла — сервер онлайн")
	}
	if row.hasData || row.errText == "" {
		t.Fatalf("ожидалась ошибка подключения: %+v", row)
	}
}

// monSampleOut собирает вывод monCmd с заданными счётчиками /proc/stat.
func monSampleOut(user, idle int) string {
	return strings.Join([]string{
		"2.00 1.00 0.50 1/2 3",
		"Mem: 1000 250 750",
		"/dev/sda 4000 1000 3000 25% /",
		"cpu " + strconv.Itoa(user) + " 0 0 " + strconv.Itoa(idle) + " 0 0 0 0",
		"  eth0: 100 0 0 0 0 0 0 0 200 0 0 0 0 0 0 0",
		"4",
		"up 1 hour",
		"203.0.113.7",
	}, "\n@\n")
}
