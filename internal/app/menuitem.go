package app

// Общие типы главного экрана и фоновых сообщений: пункт меню (результат выбора
// в TUI), карта статусов, тик обновления, кликабельная область.

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// menuItem — результат выбора на главном экране: действие и его цель. Для
// действий над одним сервером заполнен server, для групповых («batch_*») —
// servers. Папки редактируются инлайново, поэтому контекста папки тут нет.
type menuItem struct {
	server  *Server
	servers []*Server
	action  string
}

type statusMsg map[string]probeResult
type refreshTickMsg struct{}

// hitRegion — прямоугольная кликабельная область пункта на экране.
// Клик внутри [x0..x1]×[y0..y1] выбирает пункт.
type hitRegion struct {
	index          int
	x0, x1, y0, y1 int
}

// probeCmd запускает фоновую проверку доступности всех серверов.
func probeCmd(st *Store) tea.Cmd {
	servers := st.Servers
	to := statusTimeout(st)
	return func() tea.Msg { return statusMsg(probeAll(servers, to)) }
}

// refreshTick планирует периодическое обновление статусов (каждые 5 секунд).
func refreshTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return refreshTickMsg{} })
}

// statusDotPing возвращает индикатор статуса и строку пинга. Статусы различаются
// не только цветом, но и ФОРМОЙ (для дальтоников и монохромных терминалов):
// онлайн — ●, офлайн — ✗, ещё не проверен — ○.
func statusDotPing(pr probeResult, checked bool) (string, string) {
	if !checked {
		return styDim.Render("○"), styDim.Render("…")
	}
	if pr.Online {
		sty := styGreen
		if pr.Ping >= 80*time.Millisecond {
			sty = styYellow
		}
		return styGreen.Render("●"), sty.Render(fmt.Sprintf(T.MsFmt, formatPing(pr.Ping)))
	}
	return styRed.Render("✗"), styDim.Render("—")
}
