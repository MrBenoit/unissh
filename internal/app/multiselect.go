package app

// Полноэкранный мультивыбор (чекбоксы) с мышью — замена huh.MultiSelect.
// Клик по строке / пробел — отметить; Enter — готово; Esc — отмена; колесо — курсор.

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type multiOption struct {
	Label    string
	Value    string
	Selected bool
}

type multiModel struct {
	title   string
	opts    []multiOption
	checked []bool
	cursor  int
	offset  int
	regions []hitRegion
	ok      bool
	width   int
	height  int
}

func (m *multiModel) Init() tea.Cmd { return nil }

func (m *multiModel) move(delta int) {
	n := len(m.opts)
	if n > 0 {
		m.cursor = (m.cursor + delta + n) % n
	}
}

func (m *multiModel) toggle(i int) {
	if i >= 0 && i < len(m.checked) {
		m.checked[i] = !m.checked[i]
	}
}

func (m *multiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.cursor = len(m.opts) - 1
		case " ", "x":
			m.toggle(m.cursor)
		case "enter":
			m.ok = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.ok = false
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
				m.toggle(i)
			}
		}
	}
	return m, nil
}

// visibleCount — сколько кнопок-чекбоксов помещается по высоте.
func (m *multiModel) visibleCount() int {
	h := m.height
	if h <= 0 {
		h = 40
	}
	if avail := h - 8; avail >= 3 {
		return avail / 3
	}
	return 1
}

func (m *multiModel) View() string {
	var lines []string
	var regions []hitRegion
	lines = append(lines, styViolet.Bold(true).Render(m.title), "")

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
	end := min(m.offset+vis, len(m.opts))

	if m.offset > 0 {
		lines = append(lines, styDim.Render(fmt.Sprintf(T.MoreAbove, m.offset)))
	}
	w := btnWidth(m.width)
	for i := m.offset; i < end; i++ {
		box := styDim.Render("[ ]")
		if m.checked[i] {
			box = styGreen.Render("[✓]")
		}
		btn := boxButton(box+" "+m.opts[i].Label, w, i == m.cursor)
		y0 := len(lines)
		lines = append(lines, strings.Split(btn, "\n")...)
		regions = append(regions, hitRegion{index: i, x0: 0, x1: lipgloss.Width(btn) - 1, y0: y0, y1: len(lines) - 1})
	}
	if end < len(m.opts) {
		lines = append(lines, styDim.Render(fmt.Sprintf(T.MoreBelow, len(m.opts)-end)))
	}

	lines = append(lines, "", styDim.Render(T.TuiHintMulti))
	m.regions = regions
	return strings.Join(lines, "\n")
}

// runMulti показывает чекбоксы и возвращает выбранные значения и ok.
func runMulti(title string, opts []multiOption) ([]string, bool) {
	checked := make([]bool, len(opts))
	for i, o := range opts {
		checked[i] = o.Selected
	}
	m := &multiModel{title: title, opts: opts, checked: checked}
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		return nil, false
	}
	if !m.ok {
		return nil, false
	}
	var out []string
	for i, o := range opts {
		if m.checked[i] {
			out = append(out, o.Value)
		}
	}
	return out, true
}
