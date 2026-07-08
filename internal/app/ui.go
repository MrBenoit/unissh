package app

// Единый полноэкранный UI на bubbletea: меню-выбор (select.go), ввод текста
// (input.go), чекбоксы (multiselect.go). huh.Option оставлен как контейнер
// данных для askSelect. Все ask*-хелперы: (значение, ok); ok=false — отмена (Esc/Ctrl+C).

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
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

func askSelect[T comparable](title string, opts []huh.Option[T]) (T, bool) {
	return askSelectHeader(title, "", opts)
}

// askSelectHeader — кликабельное меню-выбор с необязательной шапкой (например
// таблицей сервера) под баннером.
func askSelectHeader[T comparable](title, header string, opts []huh.Option[T]) (T, bool) {
	labels := make([]string, len(opts))
	for i, o := range opts {
		labels[i] = o.Key
	}
	idx := runSelect(title, header, labels, 0)
	if idx < 0 || idx >= len(opts) {
		var zero T
		return zero, false
	}
	return opts[idx].Value, true
}

func askMulti(title string, opts []multiOption) ([]string, bool) {
	return runMulti(title, opts)
}

func askConfirm(title string, def bool) (bool, bool) {
	initial := 0
	if !def {
		initial = 1
	}
	idx := runSelect(title, "", []string{T.Yes, T.No}, initial)
	if idx < 0 {
		return false, false
	}
	return idx == 0, true
}

func pressEnter() {
	fmt.Print(styDim.Render(T.PressEnter))
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func pbcopy(text string) bool {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run() == nil
}

// serversTable рисует таблицу серверов со статусами.
func serversTable(servers []*Server, statuses map[string]probeResult) string {
	headerSty := lipgloss.NewStyle().Foreground(lipgloss.Color(cViolet)).Bold(true).Padding(0, 1)
	cellSty := lipgloss.NewStyle().Padding(0, 1)

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(cDim))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerSty
			}
			return cellSty
		}).
		Headers("", T.ColName, T.ColUser, T.ColAddr, T.ColPort, T.ColTags, T.ColPing, T.ColNotes)

	for _, s := range servers {
		st, checked := statuses[s.ID]
		dot := styDim.Render("○")
		ping := styDim.Render("…")
		if checked {
			if st.Online {
				dot = styGreen.Render("●")
				pingSty := styGreen
				if st.Ping >= 80*time.Millisecond {
					pingSty = styYellow
				}
				ping = pingSty.Render(fmt.Sprintf(T.MsFmt, formatPing(st.Ping)))
			} else {
				dot = styRed.Render("●")
				ping = styDim.Render("—")
			}
		}
		notes := s.Notes
		if len([]rune(notes)) > 28 {
			notes = string([]rune(notes)[:27]) + "…"
		}
		t.Row(dot,
			styBold.Render(s.Name),
			styCyan.Render(s.User),
			styGrey.Render(s.Host),
			styGrey.Render(fmt.Sprint(s.Port)),
			styViolet.Render(strings.Join(s.Tags, " ")),
			ping,
			styDim.Render(notes))
	}
	return t.Render()
}
