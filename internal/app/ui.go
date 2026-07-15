package app

// Единый полноэкранный UI на bubbletea: меню-выбор (select.go), ввод текста
// (input.go), чекбоксы (multiselect.go), полноэкранные страницы и тосты (page.go).
// Все ask*-хелперы: (значение, ok); ok=false — отмена (Esc/Ctrl+C).

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func askInput(title, def string, validate func(string) error) (string, bool) {
	v, ok := runInput(title, def, false, validate)
	return strings.TrimSpace(v), ok
}

func askPasswordInput(title string) (string, bool) {
	return runInput(title, "", true, nil) // пароль не тримим
}

func askSelect[T comparable](title string, opts []selOption[T]) (T, bool) {
	return askSelectFull(title, "", opts, nil)
}

// askSelectDesc — панельное меню-выбор с описаниями пунктов: справа показывается
// описание выделенного пункта (используется в настройках).
func askSelectDesc[T comparable](title string, opts []selOption[T], descs []string) (T, bool) {
	return askSelectFull(title, "", opts, descs)
}

// askSelectFull — общая реализация: строит подписи и описания, зовёт runSelect.
func askSelectFull[T comparable](title, header string, opts []selOption[T], descs []string) (T, bool) {
	labels := make([]string, len(opts))
	for i, o := range opts {
		labels[i] = o.label
	}
	idx := runSelect(title, header, labels, descs, 0)
	if idx < 0 || idx >= len(opts) {
		var zero T
		return zero, false
	}
	return opts[idx].value, true
}

func askMulti(title string, opts []multiOption) ([]string, bool) {
	return runMulti(title, opts)
}

func askConfirm(title string, def bool) (bool, bool) {
	initial := 0
	if !def {
		initial = 1
	}
	idx := runSelect(title, "", []string{T.Yes, T.No}, nil, initial)
	if idx < 0 {
		return false, false
	}
	return idx == 0, true
}

func pressEnter() {
	fmt.Print(styDim.Render(T.PressEnter))
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

// newTable создаёт таблицу в общем стиле: скруглённая рамка, фиолетовая шапка,
// отступы в ячейках. Все таблицы приложения строятся только через неё.
func newTable(headers ...string) *table.Table {
	headerSty := lipgloss.NewStyle().Foreground(lipgloss.Color(cViolet)).Bold(true).Padding(0, 1)
	cellSty := lipgloss.NewStyle().Padding(0, 1)
	return table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(cDim))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerSty
			}
			return cellSty
		}).
		Headers(headers...)
}

// serversTable рисует таблицу серверов со статусами (используется в `unissh list`).
func serversTable(servers []*Server, statuses map[string]probeResult) string {
	t := newTable("", T.ColName, T.ColUser, T.ColAddr, T.ColPort, T.ColPing, T.ColNotes)
	for _, s := range servers {
		st, checked := statuses[s.ID]
		dot, ping := statusDotPing(st, checked) // единая форма значка (●/✗/○)
		notes := s.Notes
		if len([]rune(notes)) > 28 {
			notes = string([]rune(notes)[:27]) + "…"
		}
		t.Row(dot,
			styBold.Render(s.Name),
			styCyan.Render(s.User),
			styGrey.Render(s.Host),
			styGrey.Render(fmt.Sprint(s.Port)),
			ping,
			styDim.Render(notes))
	}
	return t.Render()
}
