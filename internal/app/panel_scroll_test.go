package app

import (
	"strings"
	"testing"
)

// TestPanelBoxScrollIndicators: при прокрутке видно, сколько пунктов скрыто
// сверху и снизу, а координаты видимых строк остаются корректными.
func TestPanelBoxScrollIndicators(t *testing.T) {
	setLang("ru")
	applyTheme("blue")
	items := make([]string, 20)
	for i := range items {
		items[i] = "item"
	}
	off := 0
	// высота 8 → рамка(2) + заголовок(1) => окно ~5 строк, скрыто и сверху, и снизу
	box, rows := panelBox("T", items, 30, 8, true, 10, &off)
	joined := strings.Join(box, "\n")

	if !strings.Contains(joined, "▲") || !strings.Contains(joined, "▼") {
		t.Fatalf("ожидались индикаторы прокрутки:\n%s", joined)
	}
	if off == 0 {
		t.Fatal("окно должно было проскроллиться к курсору")
	}
	// у курсора должна быть валидная экранная строка, и она попадает внутрь бокса
	y := rows[10]
	if y < 0 || y >= len(box) {
		t.Fatalf("строка курсора вне бокса: y=%d, строк=%d", y, len(box))
	}
	// строка курсора не должна совпадать со строкой индикатора
	if strings.Contains(box[y], "▲") || strings.Contains(box[y], "▼") {
		t.Fatalf("курсор попал на строку индикатора: %q", box[y])
	}
}

// Без прокрутки индикаторов нет и все пункты видимы.
func TestPanelBoxNoScrollNoIndicators(t *testing.T) {
	setLang("ru")
	items := []string{"a", "b"}
	off := 0
	box, rows := panelBox("T", items, 30, 8, false, 0, &off)
	joined := strings.Join(box, "\n")
	if strings.Contains(joined, "▲") || strings.Contains(joined, "▼") {
		t.Fatalf("лишние индикаторы:\n%s", joined)
	}
	for i, y := range rows {
		if y < 0 {
			t.Fatalf("пункт %d должен быть видим", i)
		}
	}
}

// TestRenderTooSmall: на маленьком терминале показываем просьбу увеличить окно.
func TestRenderTooSmall(t *testing.T) {
	setLang("ru")
	m := newPanelsModel(statsStore())
	m.width, m.height = 40, 10
	out := m.render()
	if !strings.Contains(out, "минимум") && !strings.Contains(out, "Minimum") {
		t.Fatalf("ожидалась просьба увеличить окно:\n%s", out)
	}
	if len(m.regions) != 0 {
		t.Fatal("на маленьком экране кликабельных областей быть не должно")
	}
	// нормальный размер — обычная раскладка
	m.width, m.height = 100, 30
	if out := m.render(); !strings.Contains(out, T.PanelServers) {
		t.Fatal("при достаточном размере должна рисоваться раскладка")
	}
}
