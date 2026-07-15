package app

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func statsStore() *Store {
	st := &Store{Config: Config{Servers: []*Server{
		{ID: "1", Name: "web-1", Host: "10.0.0.1", User: "root", Port: 22},
		{ID: "2", Name: "db-1", Host: "10.0.0.2", User: "root", Port: 22},
	}}}
	st.path = "/tmp/unissh_stats_test.json"
	return st
}

// TestStatsSkipsHiddenPanel: при фокусе на меню справа показывается описание —
// опрашивать сервер незачем.
func TestStatsSkipsHiddenPanel(t *testing.T) {
	m := newPanelsModel(statsStore())
	m.focus = fpMenu
	if cmd := m.refreshStats(); cmd != nil {
		t.Fatal("при невидимой панели статистики опрос не должен запускаться")
	}
	if m.statsConnID != "" {
		t.Fatal("подключение не должно начинаться")
	}
}

// TestStatsSkipsOfflineServer: заведомо офлайн-сервер не опрашиваем (иначе каждый
// цикл висел бы до таймаута dial).
func TestStatsSkipsOfflineServer(t *testing.T) {
	m := newPanelsModel(statsStore())
	m.focus = fpServers
	m.statuses = map[string]probeResult{"1": {Online: false}}
	if cmd := m.refreshStats(); cmd != nil {
		t.Fatal("офлайн-сервер не должен опрашиваться")
	}
	// а непроверенный (ещё нет статуса) — должен
	m.statuses = map[string]probeResult{}
	if cmd := m.refreshStats(); cmd == nil {
		t.Fatal("непроверенный сервер должен опрашиваться")
	}
	if m.statsConnID != "1" {
		t.Fatalf("statsConnID = %q, want 1", m.statsConnID)
	}
}

// TestStatsSingleConnectInFlight: пока идёт подключение, второй dial не стартует.
func TestStatsSingleConnectInFlight(t *testing.T) {
	m := newPanelsModel(statsStore())
	m.focus = fpServers
	if cmd := m.refreshStats(); cmd == nil {
		t.Fatal("первый вызов должен начать подключение")
	}
	if cmd := m.refreshStats(); cmd != nil {
		t.Fatal("второй вызов не должен начинать параллельное подключение")
	}
}

// TestStatsDebounceOnServerChange: смена сервера закрывает соединение и НЕ
// открывает новое сразу — это делает ближайший тик.
func TestStatsDebounceOnServerChange(t *testing.T) {
	m := newPanelsModel(statsStore())
	m.focus = fpServers
	m.statsID = "1" // как будто соединение к web-1 открыто

	m.rowCur = 1 // переехали на db-1
	if cmd := m.onServerChanged(); cmd != nil {
		t.Fatal("смена сервера не должна сразу открывать соединение")
	}
	if m.statsID != "" {
		t.Fatalf("старое соединение должно быть сброшено, statsID=%q", m.statsID)
	}
}

// TestStatsStaleConnClosed: соединение, приехавшее для покинутого сервера,
// закрывается и не подменяет текущее.
func TestStatsStaleConnClosed(t *testing.T) {
	m := newPanelsModel(statsStore())
	m.focus = fpServers
	m.statsConnID = "1"
	m.rowCur = 1 // курсор уже на db-1

	var mm tea.Model = m
	_, _ = mm.Update(statsConnMsg{id: "1", r: nil, err: errors.New("boom")})
	if m.statsConnID != "" {
		t.Fatal("флаг подключения должен сброситься")
	}
	if m.statsRemote != nil || m.statsID != "" {
		t.Fatal("устаревшее соединение не должно становиться текущим")
	}
	// ошибка подключения попадает в снимок сервера
	if ls := m.live["1"]; ls == nil || ls.errText == "" {
		t.Fatal("ошибка подключения должна сохраниться в статистике сервера")
	}
}

// TestStatsSampleErrorDropsConn: ошибка опроса роняет соединение, чтобы следующий
// тик переподключился.
func TestStatsSampleErrorDropsConn(t *testing.T) {
	m := newPanelsModel(statsStore())
	m.focus = fpServers
	m.statsID = "1"
	m.statsSampling = true

	var mm tea.Model = m
	_, _ = mm.Update(panelSampleMsg{id: "1", sample: &monSample{at: time.Now(), err: errors.New("EOF")}})
	if m.statsID != "" || m.statsSampling {
		t.Fatal("ошибка опроса должна сбросить соединение")
	}
}

// TestStatsOfflineLine: для офлайн-сервера панель показывает «недоступен», а не «загружаю…».
func TestStatsOfflineLine(t *testing.T) {
	setLang("ru")
	m := newPanelsModel(statsStore())
	m.statuses = map[string]probeResult{"1": {Online: false}}
	joined := strings.Join(m.statLines(m.store.Servers[0], 60), "\n")
	if !strings.Contains(joined, T.StatsOffline) {
		t.Fatalf("ожидалась строка «%s», получили:\n%s", T.StatsOffline, joined)
	}
}
