package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// keyMsg строит tea.KeyMsg по короткому имени клавиши.
func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func clickAt(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
}

func demoStore() *Store {
	st := &Store{Config: Config{Servers: []*Server{
		{ID: "1", Name: "web-1", Host: "10.0.0.1", User: "root", Port: 22, Auth: "password", Favorite: true},
		{ID: "2", Name: "db-1", Host: "10.0.0.2", User: "deploy", Port: 22, Auth: "key"},
	}}}
	// валидный путь во временный каталог: MustSave (например по «f»/«s») не должен
	// писать stray .tmp в каталог пакета
	st.path = filepath.Join(os.TempDir(), "unissh_test_servers.json")
	return st
}

// TestPanelsParityDrive прогоняет перенесённые из v1 возможности: поиск,
// сортировку, избранное, справку, прыжки и прямой connect.
func TestPanelsParityDrive(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m := newPanelsModel(demoStore())
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 110, Height: 32})
	_ = mm.(*panelsModel).View()

	// сортировка по кругу, избранное, прыжки
	for _, k := range []string{"s", "s", "s", "s", "f", "g", "G", "home", "end"} {
		mm, _ = mm.Update(keyMsg(k))
		_ = mm.(*panelsModel).View()
	}

	// поиск: включить, напечатать, стереть, применить, сбросить по esc
	mm, _ = mm.Update(keyMsg("/"))
	mm, _ = mm.Update(keyMsg("w"))
	mm, _ = mm.Update(keyMsg("e"))
	_ = mm.(*panelsModel).View()
	if got := mm.(*panelsModel).filter; got != "we" {
		t.Fatalf("фильтр = %q, ожидалось we", got)
	}
	mm, _ = mm.Update(keyMsg("backspace"))
	mm, _ = mm.Update(keyMsg("enter"))
	mm, _ = mm.Update(keyMsg("esc"))
	if got := mm.(*panelsModel).filter; got != "" {
		t.Fatalf("esc должен сбросить фильтр, got %q", got)
	}

	// справка
	mm, _ = mm.Update(keyMsg("?"))
	_ = mm.(*panelsModel).View()
	mm, _ = mm.Update(keyMsg("x")) // любая клавиша закрывает справку
	if mm.(*panelsModel).showHelp {
		t.Fatal("справка должна закрываться любой клавишей")
	}

	// прямой connect по «c»
	m2 := newPanelsModel(demoStore())
	m2.focus = fpServers
	m2.connectSelected()
	if m2.result == nil || m2.result.action != "connect" {
		t.Fatalf("«c» должен давать действие connect, got %+v", m2.result)
	}
}

func TestPanelsModelDrive(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m := newPanelsModel(demoStore())
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 120, Height: 34})
	_ = mm.(*panelsModel).View()

	// навигация: серверы → действия → назад, tab к меню и обратно
	for _, k := range []string{"down", "up", "right", "down", "up", "left", "tab", "down", "shift+tab", "enter", "esc"} {
		mm, _ = mm.Update(keyMsg(k))
		_ = mm.(*panelsModel).View()
	}

	// сообщения статуса/статистики
	mm, _ = mm.Update(statusMsg{"1": {Online: true}})
	mm, _ = mm.Update(refreshTickMsg{})
	mm, _ = mm.Update(statsTickMsg{})
	mm, _ = mm.Update(panelSampleMsg{id: "1", sample: parseMonSampleForTest()})
	_ = mm.(*panelsModel).View()

	// колесо и клики по регионам
	mm, _ = mm.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	mm.(*panelsModel).View()
	mm, _ = mm.Update(clickAt(2, 2)) // клик по серверу
	mm.(*panelsModel).View()
}

func TestPanelsModelMenuAndEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// фокус на меню → правая панель описания
	m := newPanelsModel(demoStore())
	m.width, m.height = 110, 30
	m.focus = fpMenu
	for i := 0; i < 9; i++ {
		m.handleKey(keyMsg("down"))
	}
	_ = m.View()
	// без серверов
	e := newPanelsModel(&Store{})
	e.width, e.height = 100, 28
	_ = e.View()
	e.handleKey(keyMsg("tab"))
	e.handleKey(keyMsg("enter"))
}

func parseMonSampleForTest() *monSample {
	return parseMonSample("0.1 0.2 0.3\n@\nMem: 1000 400 600\n@\n/dev/sda1 100 50 50 50% /\n@\ncpu 1 2 3 4\n@\neth0: 1 0 0 0 0 0 0 0 2 0 0 0 0 0 0 0\n@\n2\n@\nup 1 hour\n@\n10.0.0.9", time.Now())
}
