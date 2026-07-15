package app

import (
	"strings"
	"testing"
	"time"
)

func TestSortServersFavoritesFirst(t *testing.T) {
	st := &Store{Config: Config{Servers: []*Server{
		{ID: "1", Name: "zeta"},
		{ID: "2", Name: "alpha"},
		{ID: "3", Name: "omega", Favorite: true},
	}}}
	st.Settings.SortMode = "name"
	m := newPanelsModel(st)
	got := m.filterSort(st.Servers)
	if got[0].Name != "omega" {
		t.Fatalf("избранное должно быть первым, got %q", got[0].Name)
	}
	// среди неизбранных — алфавит
	if got[1].Name != "alpha" || got[2].Name != "zeta" {
		t.Fatalf("сортировка по имени неверна: %q %q", got[1].Name, got[2].Name)
	}
}

func TestSortServersByOnline(t *testing.T) {
	st := &Store{Config: Config{Servers: []*Server{{ID: "off"}, {ID: "on"}}}}
	st.Settings.SortMode = "online"
	m := newPanelsModel(st)
	m.statuses = map[string]probeResult{"on": {Online: true, Ping: time.Millisecond}}
	got := m.filterSort(st.Servers)
	if got[0].ID != "on" {
		t.Fatalf("онлайн-сервер должен идти первым, got %q", got[0].ID)
	}
}

func TestSortServersByPing(t *testing.T) {
	st := &Store{Config: Config{Servers: []*Server{{ID: "slow"}, {ID: "fast"}}}}
	st.Settings.SortMode = "ping"
	m := newPanelsModel(st)
	m.statuses = map[string]probeResult{
		"slow": {Online: true, Ping: 100 * time.Millisecond},
		"fast": {Online: true, Ping: time.Millisecond},
	}
	if got := m.filterSort(st.Servers); got[0].ID != "fast" {
		t.Fatalf("быстрый сервер должен идти первым, got %q", got[0].ID)
	}
}

func TestMatchesFilter(t *testing.T) {
	m := newPanelsModel(&Store{})
	s := &Server{Name: "web-1", Host: "10.0.0.1", User: "root"}
	m.filter = ""
	if !m.matchesFilter(s) {
		t.Error("пустой фильтр — всё подходит")
	}
	for _, q := range []string{"web", "10.0", "root", "WEB"} {
		m.filter = q
		if !m.matchesFilter(s) {
			t.Errorf("фильтр %q должен совпасть", q)
		}
	}
	m.filter = "nomatch"
	if m.matchesFilter(s) {
		t.Error("несовпадающий фильтр не должен проходить")
	}
}

func TestStatusDotPing(t *testing.T) {
	dot, ping := statusDotPing(probeResult{}, false)
	if !strings.Contains(dot, "○") || !strings.Contains(ping, "…") {
		t.Errorf("непроверенный: dot=%q ping=%q", dot, ping)
	}
	dot, ping = statusDotPing(probeResult{Online: true, Ping: 1200 * time.Microsecond}, true)
	if !strings.Contains(dot, "●") || !strings.Contains(ping, "1.2") { // без привязки к языку
		t.Errorf("онлайн: dot=%q ping=%q", dot, ping)
	}
	dot, ping = statusDotPing(probeResult{Online: false}, true)
	if !strings.Contains(dot, "✗") || !strings.Contains(ping, "—") {
		t.Errorf("офлайн: dot=%q ping=%q", dot, ping)
	}
}
