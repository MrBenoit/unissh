package app

// Панельное меню-выбор (bubbletea) в едином стиле с главным
// экраном v2. Пункты — строки в рамке-панели с указателем ❯; если заданы описания,
// справа показывается панель с описанием выделенного пункта (как в настройках).
// При нехватке высоты список прокручивается. Клик по строке — выбор.

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// selOption — пункт селектора: видимая подпись и возвращаемое значение.
// Раньше это был huh.Option; от всего huh нам нужны были ровно эти два поля,
// и ради них тянуть зависимость незачем.
type selOption[T comparable] struct {
	label string
	value T
}

// opt — конструктор пункта селектора.
func opt[T comparable](label string, value T) selOption[T] {
	return selOption[T]{label: label, value: value}
}

type selectModel struct {
	title   string
	header  string // необязательный блок сверху (например таблица сервера)
	labels  []string
	descs   []string // описания пунктов (опционально) → правая панель
	cursor  int
	offset  int         // индекс первой видимой строки (прокрутка)
	regions []hitRegion // кликабельные области видимых строк
	chosen  int         // выбранный индекс, -1 — отмена
	width   int
	height  int
}

func (m *selectModel) Init() tea.Cmd { return nil }

// move двигает курсор по кругу.
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
		case "enter", " ", "right", "l":
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

// optionRows формирует строки пунктов селектора: радио-кнопка [ ]/[•] + текст.
func (m *selectModel) optionRows() []string {
	rows := make([]string, len(m.labels))
	for i, l := range m.labels {
		rows[i] = radioLabel(l, i == m.cursor)
	}
	return rows
}

// View рисует форму: (шапка) + панель со списком + (панель описания) + подсказка;
// попутно заполняет m.regions кликабельными областями видимых строк.
func (m *selectModel) View() string {
	var top []string
	if m.header != "" {
		top = append(top, strings.Split(m.header, "\n")...)
		top = append(top, "")
	}
	hasDesc := len(m.descs) > 0 && m.cursor < len(m.descs)
	lay := computeFormLayout(m.width, m.height, len(top), hasDesc)

	listBox, rowY := panelBox(m.title, m.optionRows(), lay.panelW, lay.panelH, true, m.cursor, &m.offset)

	lines := append([]string{}, top...)
	if hasDesc {
		desc := wrapText(m.descs[m.cursor], lay.descW-4)
		descBox, _ := panelBox(T.PanelDesc, desc, lay.descW, lay.panelH, false, -1, new(int))
		lines = append(lines, joinPanelLines(listBox, descBox, lay.panelH)...)
	} else {
		lines = append(lines, listBox...)
	}

	var regions []hitRegion
	for i, ry := range rowY {
		if ry >= 0 {
			regions = append(regions, hitRegion{index: i, x0: 0, x1: lay.panelW - 1, y0: lay.baseY + ry, y1: lay.baseY + ry})
		}
	}
	m.regions = regions

	lines = append(lines, "", styDim.Render(T.TuiHintSelect))
	return strings.Join(lines, "\n")
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

// runSelect показывает панельное меню и возвращает индекс выбранного пункта
// (-1 — отмена). descs (если непустой) даёт правую панель описаний.
// Объявлено как var — тесты подменяют его, чтобы прогонять мастера без TTY.
var runSelect = func(title, header string, labels, descs []string, initial int) int {
	m := &selectModel{title: title, header: header, labels: labels, descs: descs, cursor: initial, chosen: -1}
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		return -1
	}
	return m.chosen
}
