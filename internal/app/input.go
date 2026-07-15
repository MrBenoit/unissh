package app

// Панельный ввод текста (bubbles/textinput) в едином стиле
// с главным экраном: поле внутри рамки-панели с заголовком. Поддержка пароля
// (маска) и валидации (ошибка внутри рамки, Enter блокируется до исправления).

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type inputModel struct {
	title    string
	ti       textinput.Model
	validate func(string) error
	err      string
	ok       bool
	width    int
}

func (m *inputModel) Init() tea.Cmd { return textinput.Blink }

func (m *inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.validate != nil {
				if err := m.validate(m.ti.Value()); err != nil {
					m.err = err.Error()
					return m, nil
				}
			}
			m.ok = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.ok = false
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	m.err = "" // очищаем ошибку при любом вводе
	return m, cmd
}

// boxWidth возвращает внутреннюю ширину рамки поля под размер терминала.
func (m *inputModel) boxWidth() int {
	w := m.width
	if w <= 0 {
		w = 100
	}
	inner := w - 4
	if inner > 72 {
		inner = 72
	}
	if inner < 24 {
		inner = 24
	}
	return inner
}

func (m *inputModel) View() string {
	innerW := m.boxWidth()
	m.ti.Width = innerW - 6
	titleSty := lipgloss.NewStyle().Foreground(lipgloss.Color(cPink)).Bold(true)

	content := []string{titleSty.Render(m.title), "", m.ti.View()}
	if m.err != "" {
		content = append(content, "", styRed.Render("✗ "+m.err))
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(cPink)).Padding(0, 1).Width(innerW).
		Render(strings.Join(content, "\n"))

	return "\n" + box + "\n\n" + styDim.Render(T.TuiHintInput)
}

// runInput показывает поле ввода и возвращает сырое значение (без trim) и ok.
// Объявлено как var — тесты подменяют его, чтобы прогонять мастера без TTY.
var runInput = func(title, def string, password bool, validate func(string) error) (string, bool) {
	ti := textinput.New()
	ti.Prompt = styPink.Render("❯ ")
	ti.SetValue(def)
	ti.Focus()
	ti.Width = 60
	if password {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}
	m := &inputModel{title: title, ti: ti, validate: validate}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		return "", false
	}
	if !m.ok {
		return "", false
	}
	return m.ti.Value(), true
}
