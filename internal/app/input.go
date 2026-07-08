package app

// Полноэкранный ввод текста (bubbles/textinput) — замена huh-инпутов, чтобы весь
// интерфейс был в одном стиле (баннер + заголовок + поле). Поддержка пароля
// (маска), валидации (ошибка под полем, Enter блокируется до исправления).

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type inputModel struct {
	title    string
	ti       textinput.Model
	validate func(string) error
	err      string
	ok       bool
}

func (m *inputModel) Init() tea.Cmd { return textinput.Blink }

func (m *inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, isKey := msg.(tea.KeyMsg); isKey {
		switch key.String() {
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

func (m *inputModel) View() string {
	var b strings.Builder
	for _, l := range strings.Split(bannerArt, "\n") {
		b.WriteString(gradientLine(l) + "\n")
	}
	b.WriteString(styGrey.Italic(true).Render(T.Tagline) + "\n\n")
	b.WriteString(styViolet.Bold(true).Render(m.title) + "\n\n")
	b.WriteString(m.ti.View() + "\n")
	if m.err != "" {
		b.WriteString("\n" + styRed.Render("✗ "+m.err) + "\n")
	}
	b.WriteString("\n" + styDim.Render(T.TuiHintInput))
	return b.String()
}

// runInput показывает поле ввода и возвращает сырое значение (без trim) и ok.
func runInput(title, def string, password bool, validate func(string) error) (string, bool) {
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
