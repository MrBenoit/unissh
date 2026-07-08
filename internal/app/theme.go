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
	styBold   = lipgloss.NewStyle().Bold(true)
	styTitle  = lipgloss.NewStyle().Foreground(lipgloss.Color(cViolet)).Bold(true)

	panelViolet = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cViolet)).Padding(0, 1)
	panelDim = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cDim)).Padding(0, 1)
	panelGreen = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cGreen)).Padding(0, 1)
	panelRed = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cRed)).Padding(0, 1)
	panelYellow = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cYellow)).Padding(0, 1)
	panelCyan = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cCyan)).Padding(0, 1)
)

const bannerArt = ` ██╗   ██╗███╗   ██╗██╗███████╗███████╗██╗  ██╗
 ██║   ██║████╗  ██║██║██╔════╝██╔════╝██║  ██║
 ██║   ██║██╔██╗ ██║██║███████╗███████╗███████║
 ██║   ██║██║╚██╗██║██║╚════██║╚════██║██╔══██║
 ╚██████╔╝██║ ╚████║██║███████║███████║██║  ██║
  ╚═════╝ ╚═╝  ╚═══╝╚═╝╚══════╝╚══════╝╚═╝  ╚═╝`

var gradientStops = []string{cCyan, cViolet, cPink}

// themePresets — цвета: [cyan, violet, pink(акцент), green, yellow, red, grey, dim].
var themePresets = map[string][8]string{
	"blue":    {"#56b6ff", "#7aa2f7", "#6ea8fe", "#34d399", "#e0af68", "#f7768e", "#9aa5ce", "#4b5573"},
	"dark":    {"#22d3ee", "#a78bfa", "#c586e0", "#34d399", "#fbbf24", "#f87171", "#8b93a7", "#5c6370"},
	"light":   {"#0891b2", "#4f46e5", "#2563eb", "#059669", "#b45309", "#dc2626", "#4b5563", "#9ca3af"},
	"dracula": {"#8be9fd", "#bd93f9", "#bd93f9", "#50fa7b", "#f1fa8c", "#ff5555", "#a4a7b0", "#6272a4"},
	"gruvbox": {"#83a598", "#d3869b", "#fabd2f", "#b8bb26", "#fabd2f", "#fb4934", "#a89984", "#665c54"},
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
	cGrey, cDim = p[6], p[7]
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
	styBold = lipgloss.NewStyle().Bold(true)
	styTitle = fg(cViolet).Bold(true)
	panelViolet, panelDim, panelGreen = panel(cViolet), panel(cDim), panel(cGreen)
	panelRed, panelYellow, panelCyan = panel(cRed), panel(cYellow), panel(cCyan)
	sectionStyle = fg(cViolet).Bold(true)
	gradientStops = []string{cCyan, cViolet, cPink}
}

func hexToRGB(h string) (int, int, int) {
	var r, g, b int
	fmt.Sscanf(strings.TrimPrefix(h, "#"), "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

func blendHex(a, b string, f float64) string {
	ar, ag, ab := hexToRGB(a)
	br, bg, bb := hexToRGB(b)
	mix := func(x, y int) int { return x + int(float64(y-x)*f) }
	return fmt.Sprintf("#%02x%02x%02x", mix(ar, br), mix(ag, bg), mix(ab, bb))
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

func humanBytes(n float64) string {
	units := T.BytesUnits
	if len(units) == 0 {
		units = []string{"B", "KB", "MB", "GB", "TB", "PB"}
	}
	i := 0
	for n >= 1024 && i < len(units)-1 {
		n /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", n, units[i])
	}
	return fmt.Sprintf("%.1f %s", n, units[i])
}

// sanitizeRemote убирает управляющие символы (в т.ч. ANSI/OSC-последовательности)
// из вывода удалённого сервера, чтобы он не мог перекрасить/переписать локальный
// терминал. Сохраняются перевод строки и табуляция.
func sanitizeRemote(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == 0x1b { // ESC — пропускаем следующую управляющую последовательность
			i++
			if i < len(runes) && runes[i] == '[' { // CSI ...
				i++
				for i < len(runes) && !(runes[i] >= 0x40 && runes[i] <= 0x7e) {
					i++
				}
			} else if i < len(runes) && runes[i] == ']' { // OSC ... BEL/ST
				i++
				for i < len(runes) && runes[i] != 0x07 && runes[i] != 0x1b {
					i++
				}
			}
			continue
		}
		if r == '\n' || r == '\t' || r >= 0x20 {
			b.WriteRune(r)
		}
	}
	return b.String()
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
			if len(line) > 60 {
				line = line[:60] + "…"
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
	s.line = line
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
