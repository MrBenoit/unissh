package app

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// pageWithSize отдаёт модель страницы, уже знающую размер терминала.
func pageWithSize(m *pageModel) *pageModel {
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return m
}

// TestPageRendersTitleAndFooter: страница рисует рамку с заголовком, содержимое
// и подсказку — то есть выглядит как остальные экраны, а не как вывод в консоль.
func TestPageRendersTitleAndFooter(t *testing.T) {
	setLang("ru")
	applyTheme("blue")
	m := pageWithSize(&pageModel{title: "ЗАГОЛОВОК", lines: []string{"строка один", "строка два"}})

	view := m.View()
	for _, want := range []string{"ЗАГОЛОВОК", "строка один", T.PageBack, T.PageScroll} {
		if !strings.Contains(view, want) {
			t.Errorf("во View нет %q", want)
		}
	}
	if !strings.Contains(view, "╭") {
		t.Error("содержимое должно быть в рамке, как панели главного экрана")
	}
}

// TestPageCloseKeys: страница закрывается esc/q/enter и не возвращает действия.
func TestPageCloseKeys(t *testing.T) {
	for _, key := range []string{"esc", "q", "enter", "left", "backspace"} {
		m := pageWithSize(&pageModel{title: "t", lines: []string{"x"}})
		_, cmd := m.Update(keyMsg(key))
		if cmd == nil {
			t.Errorf("клавиша %q должна закрывать страницу", key)
		}
		if m.result != pageBack {
			t.Errorf("клавиша %q не должна возвращать действие: %q", key, m.result)
		}
	}
}

// TestPageExtraKey: дополнительное действие закрывает страницу со своим id,
// а неизвестная клавиша ничего не делает.
func TestPageExtraKey(t *testing.T) {
	m := pageWithSize(&pageModel{title: "t", lines: []string{"x"},
		keys: []pageKey{{"c", "очистить", "clear"}}})

	if _, cmd := m.Update(keyMsg("z")); cmd != nil {
		t.Fatal("неизвестная клавиша не должна закрывать страницу")
	}
	_, cmd := m.Update(keyMsg("c"))
	if cmd == nil || m.result != "clear" {
		t.Fatalf("«c» должна вернуть clear, получено %q", m.result)
	}
	if !strings.Contains(m.footer(), "очистить") {
		t.Errorf("подсказка не упоминает доп. действие: %q", m.footer())
	}
}

// TestPageScroll: прокрутка не уезжает выше начала; окончательный кламп снизу
// делает panelBox, поэтому здесь проверяем только нижнюю границу off.
func TestPageScroll(t *testing.T) {
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "строка"
	}
	m := pageWithSize(&pageModel{title: "t", lines: lines})

	m.Update(keyMsg("up"))
	if m.off != 0 {
		t.Fatalf("выше начала прокручивать нельзя: off=%d", m.off)
	}
	m.Update(keyMsg("down"))
	m.Update(keyMsg("down"))
	if m.off != 2 {
		t.Fatalf("off = %d, ожидалось 2", m.off)
	}
	m.Update(keyMsg("pgdown"))
	if m.off <= 2 {
		t.Fatalf("pgdown должен листать страницами: off=%d", m.off)
	}
	m.Update(keyMsg("home"))
	if m.off != 0 {
		t.Fatalf("home → в начало: off=%d", m.off)
	}
	// колесо мыши работает так же, как стрелки
	m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	if m.off != 1 {
		t.Fatalf("колесо вниз: off=%d", m.off)
	}
}

// TestPageLoading: пока идёт сбор данных, показывается спиннер со статусом,
// клавиши игнорируются (кроме Ctrl+C), а после pageLoadedMsg видно содержимое.
func TestPageLoading(t *testing.T) {
	setLang("ru")
	m := &pageModel{title: "t", status: "Опрашиваю серверы…",
		load: func() []string { return []string{"готово"} }}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Init()

	if !m.loading {
		t.Fatal("после Init страница с load должна быть в состоянии загрузки")
	}
	if !strings.Contains(m.View(), "Опрашиваю серверы…") {
		t.Errorf("не видно статуса: %q", m.View())
	}
	if _, cmd := m.Update(keyMsg("esc")); cmd != nil {
		t.Fatal("esc во время загрузки не должен закрывать страницу")
	}
	if _, cmd := m.Update(keyMsg("ctrl+c")); cmd == nil {
		t.Fatal("ctrl+c должен прерывать загрузку")
	}

	m.Update(pageLoadedMsg([]string{"готово"}))
	if m.loading {
		t.Fatal("после pageLoadedMsg загрузка завершена")
	}
	if !strings.Contains(m.View(), "готово") {
		t.Errorf("содержимое не показано: %q", m.View())
	}
}

// TestPageSpinnerAnimates: тик крутит кадр и планирует следующий, но только
// пока идёт загрузка — иначе тикер жил бы вечно.
func TestPageSpinnerAnimates(t *testing.T) {
	m := &pageModel{load: func() []string { return nil }}
	m.Init()
	_, cmd := m.Update(pageTickMsg{})
	if m.frame != 1 || cmd == nil {
		t.Fatalf("кадр=%d, cmd=%v", m.frame, cmd)
	}
	m.loading = false
	if _, cmd := m.Update(pageTickMsg{}); cmd != nil {
		t.Fatal("после загрузки тикер должен остановиться")
	}
}

// TestPageToast: тост центрируется, не рисует рамку и закрывается сам —
// или по любой клавише, не дожидаясь таймера.
func TestPageToast(t *testing.T) {
	m := pageWithSize(&pageModel{lines: []string{"✓ Сделано"}, ttl: 10 * time.Millisecond})
	if cmd := m.Init(); cmd == nil {
		t.Fatal("тост обязан запланировать самозакрытие")
	}
	view := m.View()
	if !strings.Contains(view, "✓ Сделано") {
		t.Errorf("нет текста: %q", view)
	}
	if strings.Contains(view, "╭") {
		t.Error("тост рисуется без рамки")
	}
	if _, cmd := m.Update(keyMsg("x")); cmd == nil {
		t.Fatal("любая клавиша закрывает тост")
	}
	m2 := pageWithSize(&pageModel{lines: []string{"x"}, ttl: time.Millisecond})
	if _, cmd := m2.Update(pageCloseMsg{}); cmd == nil {
		t.Fatal("истёкший таймер закрывает тост")
	}
}

// TestPageLines: блоки склеиваются через пустую строку и режутся на строки.
func TestPageLines(t *testing.T) {
	got := pageLines("a\nb", "c")
	want := []string{"a", "b", "", "c"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("pageLines = %v", got)
	}
	if pageLines() != nil {
		t.Fatal("без блоков — nil")
	}
}

// TestPageZeroSizeView: до прихода WindowSizeMsg страница всё равно рисуется
// (bubbletea успевает вызвать View до первого замера).
func TestPageZeroSizeView(t *testing.T) {
	m := &pageModel{title: "t", lines: []string{"x"}}
	if m.View() == "" {
		t.Fatal("View без размера не должен быть пустым")
	}
	if m.visibleH() < 1 {
		t.Fatal("visibleH всегда положительная")
	}
}
