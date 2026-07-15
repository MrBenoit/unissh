package app

// Общие панельные примитивы (рамка с заголовком, список строк с прокруткой,
// боковая панель описания). Используются и главным экраном v2 (tui2.go), и всеми
// формами (select/input/multiselect) — чтобы весь интерфейс был в одном стиле.

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// clip обрезает строку (с учётом ANSI) до w колонок.
func clip(s string, w int) string {
	return lipgloss.NewStyle().Inline(true).MaxWidth(w).Render(s)
}

// panelBox рисует панель в рамке с заголовком и списком; возвращает строки бокса
// и относительные (внутри бокса) экранные строки каждого видимого пункта (-1 — скрыт).
func panelBox(title string, items []string, w, h int, focused bool, cursor int, off *int) ([]string, []int) {
	innerW := w - 2
	innerH := h - 2
	if innerW < 4 {
		innerW = 4
	}
	if innerH < 2 {
		innerH = 2
	}
	listH := innerH - 1 // строка заголовка
	if listH < 1 {
		listH = 1
	}
	// прокрутка: держим курсор в окне
	if cursor >= 0 {
		if cursor < *off {
			*off = cursor
		}
		if cursor >= *off+listH {
			*off = cursor - listH + 1
		}
	}
	if *off > len(items)-listH {
		*off = len(items) - listH
	}
	if *off < 0 {
		*off = 0
	}

	titleSty := styDim
	if focused {
		titleSty = lipgloss.NewStyle().Foreground(lipgloss.Color(cPink)).Bold(true)
	}
	content := []string{titleSty.Render(title)}
	rows := make([]int, len(items))
	for i := range rows {
		rows[i] = -1
	}
	end := *off + listH
	if end > len(items) {
		end = len(items)
	}
	for i := *off; i < end; i++ {
		rows[i] = 1 + 1 + (i - *off) // рамка сверху + заголовок + позиция
		content = append(content, clip(items[i], innerW))
	}

	border := lipgloss.Color(cDim)
	if focused {
		border = lipgloss.Color(cPink)
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(border).Width(innerW).Height(innerH).
		Render(strings.Join(content, "\n"))
	return strings.Split(box, "\n"), rows
}

// wrapText разбивает текст на строки шириной не более w (перенос по словам).
func wrapText(s string, w int) []string {
	if w < 1 {
		w = 1
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		line := ""
		for _, word := range words {
			if line == "" {
				line = word
			} else if len(line)+1+len(word) <= w {
				line += " " + word
			} else {
				out = append(out, line)
				line = word
			}
		}
		out = append(out, line)
	}
	return out
}

// listPointer возвращает строку пункта списка с указателем: акцентный ❯ для
// выделенного пункта (когда панель в фокусе) и отступ для остальных.
func listPointer(label string, selected, focused bool) string {
	if selected {
		if focused {
			return styPink.Render("❯ ") + styBold.Render(label)
		}
		return styGrey.Render("· ") + label
	}
	return "  " + label
}

// joinPanelLines склеивает две колонки панелей построчно (левая уже нужной ширины).
func joinPanelLines(left, right []string, rowsN int) []string {
	out := make([]string, 0, rowsN)
	for r := 0; r < rowsN; r++ {
		l, rl := "", ""
		if r < len(left) {
			l = left[r]
		}
		if r < len(right) {
			rl = right[r]
		}
		out = append(out, l+rl)
	}
	return out
}

// formLayout — геометрия панельной формы (общая для select/multiselect).
type formLayout struct {
	panelW, descW, panelH, baseY int
	hasDesc                      bool
}

// computeFormLayout считает размеры центральной панели формы и боковой панели
// описания (если описания заданы) под текущий размер терминала.
func computeFormLayout(w, h, topLines int, hasDesc bool) formLayout {
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 32
	}
	panelH := h - topLines - 2 // -2 на пустую строку и подсказку
	if panelH < 5 {
		panelH = 5
	}
	panelW := w
	descW := 0
	if hasDesc {
		panelW = w * 45 / 100
		if panelW < 34 {
			panelW = 34
		}
		if panelW > 60 {
			panelW = 60
		}
		descW = w - panelW
	}
	return formLayout{panelW: panelW, descW: descW, panelH: panelH, baseY: topLines, hasDesc: hasDesc}
}
