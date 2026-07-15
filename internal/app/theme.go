package app

// Палитра, баннер, стили и мелкие элементы отрисовки.

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Палитра текущей темы (по умолчанию — синяя). Меняется applyTheme().
// cPink — исторически «акцент» (указатели, рамки, выделенный текст),
// cSelBg — фон выделенной плашки/карточки.
var (
	cCyan   = "#56b6ff"
	cViolet = "#7aa2f7"
	cPink   = "#6ea8fe"
	cGreen  = "#34d399"
	cYellow = "#e0af68"
	cRed    = "#f7768e"
	cGrey   = "#9aa5ce"
	cDim    = "#4b5573"
	cText   = "#c0caf5" // основной цвет текста темы (для подписей селекторов)
)

var (
	styGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color(cGreen))
	styYellow = lipgloss.NewStyle().Foreground(lipgloss.Color(cYellow))
	styRed    = lipgloss.NewStyle().Foreground(lipgloss.Color(cRed))
	styCyan   = lipgloss.NewStyle().Foreground(lipgloss.Color(cCyan))
	styViolet = lipgloss.NewStyle().Foreground(lipgloss.Color(cViolet))
	styPink   = lipgloss.NewStyle().Foreground(lipgloss.Color(cPink))
	styGrey   = lipgloss.NewStyle().Foreground(lipgloss.Color(cGrey))
	styDim    = lipgloss.NewStyle().Foreground(lipgloss.Color(cDim))
	styText   = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	styBold   = lipgloss.NewStyle().Bold(true)
	styTitle  = lipgloss.NewStyle().Foreground(lipgloss.Color(cViolet)).Bold(true)

	panelViolet = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cViolet)).Padding(0, 1)
	panelGreen = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cGreen)).Padding(0, 1)
	panelRed = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cRed)).Padding(0, 1)
)

var gradientStops = []string{cCyan, cViolet, cPink}

// themePresets — цвета: [cyan, violet, pink(акцент), green, yellow, red, grey, dim, text].
var themePresets = map[string][9]string{
	"blue":    {"#56b6ff", "#7aa2f7", "#6ea8fe", "#34d399", "#e0af68", "#f7768e", "#9aa5ce", "#4b5573", "#c0caf5"},
	"dark":    {"#22d3ee", "#a78bfa", "#c586e0", "#34d399", "#fbbf24", "#f87171", "#8b93a7", "#5c6370", "#e5e7eb"},
	"light":   {"#0891b2", "#4f46e5", "#2563eb", "#059669", "#b45309", "#dc2626", "#4b5563", "#9ca3af", "#1f2937"},
	"dracula": {"#8be9fd", "#bd93f9", "#bd93f9", "#50fa7b", "#f1fa8c", "#ff5555", "#a4a7b0", "#6272a4", "#f8f8f2"},
	"gruvbox": {"#83a598", "#d3869b", "#fabd2f", "#b8bb26", "#fabd2f", "#fb4934", "#a89984", "#665c54", "#ebdbb2"},
}

// themeNames — порядок тем для выбора в настройках (синяя — по умолчанию).
var themeNames = []string{"blue", "dark", "dracula", "gruvbox", "light"}

// applyTheme применяет цветовую тему по имени (пустое/неизвестное → синяя)
// и пересобирает все стили.
func applyTheme(name string) {
	p, ok := themePresets[name]
	if !ok {
		p = themePresets["blue"]
	}
	cCyan, cViolet, cPink = p[0], p[1], p[2]
	cGreen, cYellow, cRed = p[3], p[4], p[5]
	cGrey, cDim, cText = p[6], p[7], p[8]
	rebuildStyles()
}

// rebuildStyles пересоздаёт все стили и градиент из текущей палитры c*.
func rebuildStyles() {
	fg := func(c string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(c)) }
	panel := func(c string) lipgloss.Style {
		return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(c)).Padding(0, 1)
	}
	styGreen, styYellow, styRed = fg(cGreen), fg(cYellow), fg(cRed)
	styCyan, styViolet, styPink = fg(cCyan), fg(cViolet), fg(cPink)
	styGrey, styDim = fg(cGrey), fg(cDim)
	styText = fg(cText)
	styBold = lipgloss.NewStyle().Bold(true)
	styTitle = fg(cViolet).Bold(true)
	panelViolet, panelGreen, panelRed = panel(cViolet), panel(cGreen), panel(cRed)
	gradientStops = []string{cCyan, cViolet, cPink}
}

// gradientLine раскрашивает строку горизонтальным градиентом по символам.
func gradientLine(s string) string {
	runes := []rune(s)
	n := len(runes) - 1
	if n < 1 {
		n = 1
	}
	var out strings.Builder
	for i, ch := range runes {
		if ch == ' ' {
			out.WriteRune(ch)
			continue
		}
		t := float64(i) / float64(n) * float64(len(gradientStops)-1)
		k := int(t)
		if k > len(gradientStops)-2 {
			k = len(gradientStops) - 2
		}
		c := blendHex(gradientStops[k], gradientStops[k+1], t-float64(k))
		out.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(string(ch)))
	}
	return out.String()
}

// meter рисует цветной прогресс-бар: зелёный → жёлтый → красный.
func meter(frac float64, width int) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac*float64(width) + 0.5)
	sty := styGreen
	if frac >= 0.9 {
		sty = styRed
	} else if frac >= 0.7 {
		sty = styYellow
	}
	return sty.Render(strings.Repeat("█", filled)) +
		styDim.Render(strings.Repeat("░", width-filled)) +
		sty.Render(fmt.Sprintf(" %3.0f%%", frac*100))
}

// stepSpinner — строка прогресса шага настройки: крутилка + последняя строка вывода.
type stepSpinner struct {
	mu    sync.Mutex
	title string
	line  string
	stop  chan struct{}
	done  chan struct{}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func newStepSpinner(title string) *stepSpinner {
	s := &stepSpinner{title: title, stop: make(chan struct{}), done: make(chan struct{})}
	go s.run()
	return s
}

func (s *stepSpinner) run() {
	defer close(s.done)
	t := time.NewTicker(90 * time.Millisecond)
	defer t.Stop()
	i := 0
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.mu.Lock()
			line := s.line
			s.mu.Unlock()
			if r := []rune(line); len(r) > 60 { // обрезаем по рунам, не по байтам
				line = string(r[:60]) + "…"
			}
			fmt.Printf("\r\033[K  %s %s  %s",
				styCyan.Render(spinnerFrames[i%len(spinnerFrames)]),
				styBold.Render(s.title), styDim.Render(line))
			i++
		}
	}
}

func (s *stepSpinner) setLine(line string) {
	s.mu.Lock()
	s.line = sanitizeRemote(line) // строка приходит из вывода сервера — санитайзим
	s.mu.Unlock()
}

func (s *stepSpinner) finish(ok bool) {
	close(s.stop)
	<-s.done
	mark := styGreen.Render("✓")
	if !ok {
		mark = styRed.Render("✗")
	}
	fmt.Printf("\r\033[K  %s %s\n", mark, s.title)
}
