package app

import (
	"strings"
	"testing"
)

// TestSelectRenderDesc проверяет панельный рендер меню-выбора с описаниями:
// список слева, панель описания справа, кликабельные регионы построены.
func TestSelectRenderDesc(t *testing.T) {
	setLang("ru")
	applyTheme("blue")
	m := &selectModel{
		title:  "Настройки:",
		labels: []string{"Язык: Русский", "Тема: blue", "Оформление: панели"},
		descs:  []string{"Язык интерфейса.", "Цветовая тема.", "Карточки или панели."},
		chosen: -1,
	}
	m.width, m.height = 120, 30
	out := m.View()
	if !strings.Contains(out, "Настройки") || !strings.Contains(out, T.PanelDesc) {
		t.Fatalf("нет заголовка или панели описания:\n%s", out)
	}
	if len(m.regions) != 3 {
		t.Fatalf("ожидалось 3 региона, получили %d", len(m.regions))
	}
}

// TestSelectRenderNoDesc проверяет рендер меню без описаний (одна панель).
func TestSelectRenderNoDesc(t *testing.T) {
	setLang("ru")
	m := &selectModel{title: "Выбор", labels: []string{"Один", "Два"}, chosen: -1}
	m.width, m.height = 100, 24
	out := m.View()
	if !strings.Contains(out, "Один") || !strings.Contains(out, "Два") {
		t.Fatalf("нет пунктов:\n%s", out)
	}
	if len(m.regions) != 2 {
		t.Fatalf("ожидалось 2 региона, получили %d", len(m.regions))
	}
}

// TestMultiRender проверяет панельный рендер чекбоксов.
func TestMultiRender(t *testing.T) {
	setLang("ru")
	m := &multiModel{
		title:   "Шаги",
		opts:    []multiOption{{Label: "Обновление", Value: "upd"}, {Label: "Файрвол", Value: "fw"}},
		checked: []bool{true, false},
	}
	m.width, m.height = 100, 24
	out := m.View()
	if !strings.Contains(out, "Обновление") || !strings.Contains(out, "[✓]") {
		t.Fatalf("нет отмеченного пункта:\n%s", out)
	}
	if len(m.regions) != 2 {
		t.Fatalf("ожидалось 2 региона, получили %d", len(m.regions))
	}
}

// TestInputRender проверяет, что поле ввода рисуется в рамке-панели.
func TestInputRender(t *testing.T) {
	setLang("ru")
	m := &inputModel{title: "Имя сервера", width: 100}
	out := m.View()
	if !strings.Contains(out, "Имя сервера") {
		t.Fatalf("нет заголовка поля:\n%s", out)
	}
}
