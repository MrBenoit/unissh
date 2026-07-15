package app

// Инлайновые промпты папок: создание («n»), переименование («r») и подтверждение
// удаления («d») выполняются прямо в панельном экране — оверлеем поверх раскладки.
// Раньше это требовало выхода из TUI и отдельного tea.Program (экран мигал).

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type promptKind int

const (
	promptFolderNew promptKind = iota
	promptFolderRename
	promptFolderDelete
)

// folderPrompt — состояние активного промпта.
type folderPrompt struct {
	kind   promptKind
	value  string // вводимый текст (для new/rename)
	target string // папка, к которой относится rename/delete
	err    string // текст ошибки валидации
}

// askFolderNew открывает промпт создания папки.
func (m *panelsModel) askFolderNew() {
	m.prompt = &folderPrompt{kind: promptFolderNew}
}

// askFolderRename открывает промпт переименования папки под курсором.
func (m *panelsModel) askFolderRename() {
	name, ok := m.onFolderRow()
	if !ok || name == "" { // «Без папки» переименовать нельзя
		return
	}
	m.prompt = &folderPrompt{kind: promptFolderRename, target: name, value: name}
}

// askFolderDelete открывает подтверждение удаления папки под курсором.
func (m *panelsModel) askFolderDelete() {
	name, ok := m.onFolderRow()
	if !ok || name == "" {
		return
	}
	m.prompt = &folderPrompt{kind: promptFolderDelete, target: name}
}

// promptTitle — заголовок оверлея промпта.
func (m *panelsModel) promptTitle() string {
	switch m.prompt.kind {
	case promptFolderRename:
		return T.FolderRenameTitle
	case promptFolderDelete:
		return strings.Replace(T.AskDeleteFolder, "%s", m.prompt.target, 1)
	default:
		return T.FolderNewTitle
	}
}

// validatePrompt проверяет введённое имя папки.
func (m *panelsModel) validatePrompt() string {
	name := strings.TrimSpace(m.prompt.value)
	if name == "" {
		return T.ErrNameEmpty
	}
	if name != m.prompt.target && m.store.HasFolder(name) {
		return T.ErrFolderExists
	}
	return ""
}

// applyPrompt применяет промпт и сохраняет конфиг.
func (m *panelsModel) applyPrompt() {
	p := m.prompt
	switch p.kind {
	case promptFolderDelete:
		m.store.RemoveFolder(p.target)
	case promptFolderNew:
		if e := m.validatePrompt(); e != "" {
			p.err = e
			return
		}
		m.store.AddFolder(p.value)
	case promptFolderRename:
		if e := m.validatePrompt(); e != "" {
			p.err = e
			return
		}
		m.store.RenameFolder(p.target, p.value)
	}
	m.prompt = nil
	m.store.MustSave()
	m.rebuildRows()
}

// handlePromptKey — клавиши активного промпта. Остальной экран заблокирован.
func (m *panelsModel) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.prompt.kind == promptFolderDelete {
		switch msg.String() {
		case "enter", "y":
			m.applyPrompt()
		case "esc", "n", "q":
			m.prompt = nil
		}
		return m, nil
	}
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace:
		m.prompt.value += string(msg.Runes)
		m.prompt.err = ""
		return m, nil
	case tea.KeyBackspace:
		if r := []rune(m.prompt.value); len(r) > 0 {
			m.prompt.value = string(r[:len(r)-1])
			m.prompt.err = ""
		}
		return m, nil
	}
	switch msg.String() {
	case "enter":
		m.applyPrompt()
	case "esc":
		m.prompt = nil
	}
	return m, nil
}

// renderPrompt рисует оверлей промпта поверх экрана.
func (m *panelsModel) renderPrompt() string {
	var body []string
	if m.prompt.kind == promptFolderDelete {
		body = append(body, styGrey.Render(T.FolderDeleteBody), "", styDim.Render(T.ConfirmHint))
	} else {
		body = append(body, styText.Render(m.prompt.value)+styPink.Render("▏"))
		if m.prompt.err != "" {
			body = append(body, "", styRed.Render("✗ "+m.prompt.err))
		}
		body = append(body, "", styDim.Render(T.PromptHint))
	}
	w := m.width - 8
	if w > 60 {
		w = 60
	}
	if w < 24 {
		w = 24
	}
	panel := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(cPink)).Padding(0, 1).Width(w).
		Render(styViolet.Bold(true).Render(m.promptTitle()) + "\n\n" + strings.Join(body, "\n"))
	return "\n" + panel
}
