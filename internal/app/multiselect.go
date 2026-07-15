package app

// Панельный мультивыбор (чекбоксы) в едином стиле с главным экраном.
// Строки-чекбоксы внутри рамки-панели; клик по строке / пробел — отметить;
// Enter — готово; Esc — отмена; колесо — курсор; список прокручивается.

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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

// move двигает курсор по кругу.
func (m *multiModel) move(delta int) {
	n := len(m.opts)
	if n > 0 {
		m.cursor = (m.cursor + delta + n) % n
	}
}

// toggle переключает отметку пункта.
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
		case "q", "esc", "ctrl+c":
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

// optionRows формирует строки-чекбоксы с указателем на выделенном.
func (m *multiModel) optionRows() []string {
	rows := make([]string, len(m.opts))
	for i, o := range m.opts {
		box := styDim.Render("[ ]")
		if m.checked[i] {
			box = styGreen.Render("[✓]")
		}
		rows[i] = listPointer(box+" "+o.Label, i == m.cursor, true)
	}
	return rows
}

// View рисует панель с чекбоксами и подсказку; заполняет m.regions.
func (m *multiModel) View() string {
	lay := computeFormLayout(m.width, m.height, 0, false)
	box, rowY := panelBox(m.title, m.optionRows(), lay.panelW, lay.panelH, true, m.cursor, &m.offset)

	var regions []hitRegion
	for i, ry := range rowY {
		if ry >= 0 {
			regions = append(regions, hitRegion{index: i, x0: 0, x1: lay.panelW - 1, y0: ry, y1: ry})
		}
	}
	m.regions = regions

	lines := append([]string{}, box...)
	lines = append(lines, "", styDim.Render(T.TuiHintMulti))
	return strings.Join(lines, "\n")
}

// runMulti показывает чекбоксы и возвращает выбранные значения и ok.
// Объявлено как var — тесты подменяют его, чтобы прогонять мастера без TTY.
var runMulti = func(title string, opts []multiOption) ([]string, bool) {
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
