package app

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectModelDrive(t *testing.T) {
	m := &selectModel{
		title:  "Выбор",
		header: "шапка\nвторая строка",
		labels: []string{"one", "two", "three"},
		descs:  []string{"d1", "d2", "d3"},
		chosen: -1,
	}
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	_ = mm.(*selectModel).View()
	for _, k := range []string{"down", "down", "up", "home", "end", "j", "k"} {
		mm, _ = mm.Update(keyMsg(k))
	}
	mm.(*selectModel).View() // строим регионы
	mm, _ = mm.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	mm, _ = mm.Update(clickAt(2, 3))
	if mm.(*selectModel).chosen < 0 {
		// клик мог не попасть — тогда выбираем клавишей
		mm, _ = mm.Update(keyMsg("enter"))
	}
	if mm.(*selectModel).chosen < 0 {
		t.Fatal("после Enter должен быть выбран пункт")
	}

	// отмена по esc
	c := &selectModel{labels: []string{"a", "b"}, chosen: -1}
	c.Update(keyMsg("esc"))
	if c.chosen != -1 {
		t.Fatal("esc должен оставить chosen=-1")
	}
}

func TestSelectModelSmallScreen(t *testing.T) {
	m := &selectModel{title: "t", labels: []string{"a", "b", "c", "d", "e", "f"}, chosen: -1}
	m.width, m.height = 20, 8 // мало места → прокрутка
	m.cursor = 5
	_ = m.View()
}

func TestMultiModelDrive(t *testing.T) {
	m := &multiModel{
		title:   "Шаги",
		opts:    []multiOption{{Label: "a", Value: "a", Selected: true}, {Label: "b", Value: "b"}, {Label: "c", Value: "c"}},
		checked: []bool{true, false, false},
	}
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 90, Height: 20})
	for _, k := range []string{"down", "space", "x", "up", "home", "end", "j", "k"} {
		mm, _ = mm.Update(keyMsg(k))
	}
	mm.(*multiModel).View()
	mm, _ = mm.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	mm, _ = mm.Update(clickAt(2, 3)) // тумблер по клику
	mm, _ = mm.Update(keyMsg("enter"))
	if !mm.(*multiModel).ok {
		t.Fatal("enter должен завершить с ok")
	}

	// esc → отмена
	c := &multiModel{opts: []multiOption{{Label: "a"}}, checked: []bool{false}}
	c.Update(keyMsg("esc"))
	if c.ok {
		t.Fatal("esc не должен давать ok")
	}
}

func TestInputModelDrive(t *testing.T) {
	failOnce := true
	m := &inputModel{
		title: "Имя",
		width: 80,
		validate: func(s string) error {
			if failOnce && s == "bad" {
				return errors.New("плохо")
			}
			return nil
		},
	}
	m.ti.SetValue("bad")
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 80})
	// enter при ошибке валидации — не завершает
	mm, _ = mm.Update(keyMsg("enter"))
	if mm.(*inputModel).ok {
		t.Fatal("при ошибке валидации не должно быть ok")
	}
	if mm.(*inputModel).err == "" {
		t.Fatal("ошибка валидации должна отобразиться")
	}
	_ = mm.(*inputModel).View()
	// ввод очищает ошибку, меняем значение и подтверждаем
	mm, _ = mm.Update(keyMsg("x"))
	mm.(*inputModel).ti.SetValue("good")
	mm, _ = mm.Update(keyMsg("enter"))
	if !mm.(*inputModel).ok {
		t.Fatal("корректный ввод должен завершиться ok")
	}

	// esc → отмена
	c := &inputModel{title: "t", width: 60}
	c.Update(keyMsg("esc"))
	if c.ok {
		t.Fatal("esc не должен давать ok")
	}
}
