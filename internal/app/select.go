package app

// Кликабельное меню-выбор (bubbletea): замена huh.Select. Каждый пункт —
// отдельная кнопка в рамке (клик по всей кнопке, промахнуться сложно). При
// нехватке высоты список прокручивается (окно вокруг курсора).

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type selectModel struct {
	title   string
	header  string // необязательный блок (например таблица сервера)
	labels  []string
	cursor  int
	offset  int         // индекс первой видимой кнопки (прокрутка)
	regions []hitRegion // кликабельные области видимых кнопок
	chosen  int         // выбранный индекс, -1 — отмена
	width   int
	height  int
}

func (m *selectModel) Init() tea.Cmd { return nil }

// btnWidth возвращает ширину кнопки пункта с учётом ширины терминала.
func btnWidth(termWidth int) int {
	w := termWidth - 6
	if w > 68 {
		w = 68
	}
	if w < 20 {
		w = 20
	}
	return w
}

func (m *selectModel) move(delta int) {
	n := len(m.labels)
	if n > 0 {
		m.cursor = (m.cursor + delta + n) % n
	}
}

func (m *selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.move(-1)
		case "down", "j":
			m.move(1)
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = len(m.labels) - 1
		case "enter", " ":
			m.chosen = m.cursor
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.chosen = -1
			return m, tea.Quit
		}
	case tea.MouseMsg:
		switch {
		case msg.Button == tea.MouseButtonWheelUp:
			m.move(-1)
		case msg.Button == tea.MouseButtonWheelDown:
			m.move(1)
		case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
			if i := hitIndex(m.regions, msg.X, msg.Y); i >= 0 {
				m.cursor = i
				m.chosen = i
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

// visibleCount — сколько кнопок (по 3 строки) помещается по высоте.
func (m *selectModel) visibleCount() int {
	h := m.height
	if h <= 0 {
		h = 40
	}
	reserved := 6 // заголовок, отступы, подсказка, индикаторы прокрутки
	if m.header != "" {
		reserved += strings.Count(m.header, "\n") + 2
	}
	if avail := h - reserved; avail >= 3 {
		return avail / 3
	}
	return 1
}

// View рисует заголовок, шапку и пункты-кнопки; попутно заполняет m.regions
// кликабельными областями видимых кнопок.
func (m *selectModel) View() string {
	var lines []string
	var regions []hitRegion

	if m.header != "" {
		lines = append(lines, strings.Split(m.header, "\n")...)
		lines = append(lines, "")
	}
	if m.title != "" {
		lines = append(lines, styViolet.Bold(true).Render(m.title), "")
	}

	// прокрутка: держим курсор в видимом окне
	vis := m.visibleCount()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vis {
		m.offset = m.cursor - vis + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
	end := min(m.offset+vis, len(m.labels))

	if m.offset > 0 {
		lines = append(lines, styDim.Render(fmt.Sprintf(T.MoreAbove, m.offset)))
	}
	w := btnWidth(m.width)
	for i := m.offset; i < end; i++ {
		box := boxButton(m.labels[i], w, i == m.cursor)
		y0 := len(lines)
		lines = append(lines, strings.Split(box, "\n")...)
		regions = append(regions, hitRegion{index: i, x0: 0, x1: lipgloss.Width(box) - 1, y0: y0, y1: len(lines) - 1})
	}
	if end < len(m.labels) {
		lines = append(lines, styDim.Render(fmt.Sprintf(T.MoreBelow, len(m.labels)-end)))
	}

	lines = append(lines, "", styDim.Render(T.TuiHintSelect))
	m.regions = regions
	return strings.Join(lines, "\n")
}

// boxButton рисует пункт как отдельную кнопку в рамке (rounded, 3 строки).
// Выделенная — акцентная (синяя) рамка, жирный текст и указатель ❯; без заливки строки.
func boxButton(label string, w int, selected bool) string {
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(w)
	if selected {
		return box.BorderForeground(lipgloss.Color(cPink)).Bold(true).
			Render(styPink.Render("❯ ") + label)
	}
	return box.BorderForeground(lipgloss.Color(cDim)).Render("  " + label)
}

// hitIndex возвращает index области, содержащей точку (x,y), или -1.
func hitIndex(regions []hitRegion, x, y int) int {
	for _, r := range regions {
		if x >= r.x0 && x <= r.x1 && y >= r.y0 && y <= r.y1 {
			return r.index
		}
	}
	return -1
}

// runSelect показывает меню и возвращает индекс выбранного пункта (-1 — отмена).
func runSelect(title, header string, labels []string, initial int) int {
	m := &selectModel{title: title, header: header, labels: labels, cursor: initial, chosen: -1}
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		return -1
	}
	return m.chosen
}
