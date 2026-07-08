package app

// Главный экран — интерактивный TUI (bubbletea) в духе lazygit: список серверов
// сверху, действия снизу. Навигация стрелками и мышью (клик = выбор+нажатие,
// колесо — прокрутка). Одна клавиша = одно действие (c/i/b/e/r/d/m), фильтр «/»,
// избранное «f», сортировка «s», помощь «?». Живой статус/пинг.

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuKind int

const (
	kindServer menuKind = iota
	kindAction
)

// menuItem — пункт списка: сервер или действие. Для действий над конкретным
// сервером (info/edit/reboot…) заполнены и action, и server.
type menuItem struct {
	kind   menuKind
	server *Server
	action string // "add" | "manage" | "manage-direct" | "tools" | "info" | "edit" | "delete" | "reboot" | "bootstrap"
	label  string
}

type statusMsg map[string]probeResult
type refreshTickMsg struct{}

// hitRegion — прямоугольная кликабельная область пункта на экране (карточка
// сервера или кнопка действия). Клик внутри [x0..x1]×[y0..y1] выбирает пункт.
type hitRegion struct {
	index          int
	x0, x1, y0, y1 int
}

type mainModel struct {
	store    *Store
	visible  []menuItem // видимые пункты (после сортировки и фильтра)
	statuses map[string]probeResult
	cursor   int
	width    int
	height   int
	regions  []hitRegion // кликабельные области пунктов (для попадания мышью)
	result   *menuItem

	filtering    bool   // активен режим ввода фильтра
	filter       string // текст фильтра
	showHelp     bool   // показан оверлей помощи
	serverOffset int    // индекс первой видимой карточки сервера (прокрутка)
}

var sectionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cViolet)).Bold(true)

// newMainModel создаёт модель главного экрана и строит список пунктов.
func newMainModel(st *Store) *mainModel {
	m := &mainModel{store: st, statuses: map[string]probeResult{}}
	m.rebuild()
	return m
}

// sortedServers возвращает серверы в порядке согласно режиму сортировки;
// избранные всегда идут первыми.
func (m *mainModel) sortedServers() []*Server {
	list := make([]*Server, len(m.store.Servers))
	copy(list, m.store.Servers)
	mode := m.store.Settings.SortMode
	// стабильная сортировка: сначала по режиму, поверх — избранное наверх
	switch mode {
	case "name":
		sort.SliceStable(list, func(i, j int) bool {
			return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
		})
	case "ping":
		sort.SliceStable(list, func(i, j int) bool {
			return m.pingKey(list[i]) < m.pingKey(list[j])
		})
	case "online":
		sort.SliceStable(list, func(i, j int) bool {
			return m.onlineKey(list[i]) < m.onlineKey(list[j])
		})
	}
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Favorite && !list[j].Favorite
	})
	return list
}

// pingKey — ключ сортировки по пингу (в микросекундах): онлайн с меньшим пингом
// раньше, офлайн в конце.
func (m *mainModel) pingKey(s *Server) int64 {
	if pr, ok := m.statuses[s.ID]; ok && pr.Online {
		return pr.Ping.Microseconds()
	}
	return 1 << 50
}

// onlineKey — ключ сортировки по доступности: онлайн (0) раньше офлайн (1).
func (m *mainModel) onlineKey(s *Server) int {
	if pr, ok := m.statuses[s.ID]; ok && pr.Online {
		return 0
	}
	return 1
}

// matchesFilter сообщает, подходит ли сервер под текущий текст фильтра
// (по имени, хосту, логину или тегам, без учёта регистра).
func (m *mainModel) matchesFilter(s *Server) bool {
	q := strings.ToLower(strings.TrimSpace(m.filter))
	if q == "" {
		return true
	}
	hay := strings.ToLower(s.Name + " " + s.Host + " " + s.User + " " + strings.Join(s.Tags, " "))
	return strings.Contains(hay, q)
}

// rebuild пересобирает видимый список: серверы (отсортированные и отфильтрованные)
// + действия; удерживает курсор в границах.
func (m *mainModel) rebuild() {
	var items []menuItem
	for _, s := range m.sortedServers() {
		if !m.matchesFilter(s) {
			continue
		}
		items = append(items, menuItem{kind: kindServer, server: s})
	}
	items = append(items, menuItem{kind: kindAction, action: "add", label: T.ItemAdd})
	if len(m.store.Servers) > 0 {
		items = append(items, menuItem{kind: kindAction, action: "manage", label: T.ItemManage})
	}
	items = append(items,
		menuItem{kind: kindAction, action: "tools", label: T.ItemTools},
		menuItem{kind: kindAction, action: "exit", label: T.ItemExit})
	m.visible = items
	if m.cursor >= len(items) {
		m.cursor = len(items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
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

// Init стартует первую проверку статусов и таймер обновления.
func (m *mainModel) Init() tea.Cmd {
	return tea.Batch(probeCmd(m.store), refreshTick())
}

// move сдвигает курсор по кругу на delta позиций.
func (m *mainModel) move(delta int) {
	n := len(m.visible)
	if n > 0 {
		m.cursor = (m.cursor + delta + n) % n
	}
}

// current возвращает пункт под курсором (или nil, если список пуст).
func (m *mainModel) current() *menuItem {
	if m.cursor >= 0 && m.cursor < len(m.visible) {
		return &m.visible[m.cursor]
	}
	return nil
}

// activate выполняет выбор пункта под курсором и завершает TUI (кроме «Выход»).
func (m *mainModel) activate() (tea.Model, tea.Cmd) {
	it := m.current()
	if it == nil {
		return m, nil
	}
	if it.kind == kindAction && it.action == "exit" {
		return m, tea.Quit // result остаётся nil → выход
	}
	res := *it
	m.result = &res
	return m, tea.Quit
}

// actOnServer завершает TUI с действием verb над выделенным сервером
// (работает только когда курсор на сервере).
func (m *mainModel) actOnServer(verb string) (tea.Model, tea.Cmd) {
	it := m.current()
	if it == nil || it.kind != kindServer {
		return m, nil
	}
	m.result = &menuItem{kind: kindAction, action: verb, server: it.server}
	return m, tea.Quit
}

// toggleFavorite переключает «избранное» у выделенного сервера и сохраняет конфиг
// (список пересобирается: избранные всплывают наверх).
func (m *mainModel) toggleFavorite() {
	it := m.current()
	if it == nil || it.kind != kindServer {
		return
	}
	it.server.Favorite = !it.server.Favorite
	m.store.MustSave()
	m.rebuild()
}

// sortModes — порядок перебора режимов сортировки по клавише «s».
var sortModes = []string{"", "name", "ping", "online"}

// cycleSort переключает режим сортировки на следующий по кругу и сохраняет его.
func (m *mainModel) cycleSort() {
	cur := m.store.Settings.SortMode
	next := 0
	for i, mode := range sortModes {
		if mode == cur {
			next = (i + 1) % len(sortModes)
			break
		}
	}
	m.store.Settings.SortMode = sortModes[next]
	m.store.MustSave()
	m.rebuild()
}

// Update обрабатывает клавиатуру, мышь и фоновые сообщения (статусы, тик).
func (m *mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case statusMsg:
		m.statuses = msg
		if m.store.Settings.SortMode == "ping" || m.store.Settings.SortMode == "online" {
			m.rebuild() // порядок зависит от статусов
		}
	case refreshTickMsg:
		return m, tea.Batch(probeCmd(m.store), refreshTick())
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	return m, nil
}

// handleKey разбирает нажатия клавиш в трёх режимах: помощь, фильтр, обычный.
func (m *mainModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false // любая клавиша закрывает помощь
		return m, nil
	}
	if m.filtering {
		return m.handleFilterKey(msg)
	}
	switch msg.String() {
	case "up", "k", "left", "h":
		m.move(-1)
	case "down", "j", "right", "l":
		m.move(1)
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = len(m.visible) - 1
	case "enter":
		return m.activate()
	case "c":
		return m.actOnServerConnect()
	case "i":
		return m.actOnServer("info")
	case "b":
		return m.actOnServer("bootstrap")
	case "e":
		return m.actOnServer("edit")
	case "r":
		return m.actOnServer("reboot")
	case "d":
		return m.actOnServer("delete")
	case "m":
		return m.actOnServer("manage-direct")
	case "f":
		m.toggleFavorite()
	case "s":
		m.cycleSort()
	case "/":
		m.filtering = true
	case "?":
		m.showHelp = true
	case "esc":
		if m.filter != "" { // сначала сбрасываем фильтр, потом выходим
			m.filter = ""
			m.rebuild()
			return m, nil
		}
		return m, tea.Quit
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// actOnServerConnect подключается к выделенному серверу (клавиша c / Enter).
func (m *mainModel) actOnServerConnect() (tea.Model, tea.Cmd) {
	it := m.current()
	if it == nil || it.kind != kindServer {
		return m, nil
	}
	res := *it
	m.result = &res
	return m, tea.Quit
}

// handleFilterKey обрабатывает ввод в режиме фильтра: печать сужает список,
// стрелки двигают курсор, Enter применяет, Esc сбрасывает.
func (m *mainModel) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace:
		m.filter += string(msg.Runes) // для KeySpace Runes уже содержит пробел
		m.rebuild()
		return m, nil
	case tea.KeyBackspace:
		if r := []rune(m.filter); len(r) > 0 {
			m.filter = string(r[:len(r)-1])
			m.rebuild()
		}
		return m, nil
	}
	switch msg.String() {
	case "up", "ctrl+k":
		m.move(-1)
	case "down", "ctrl+j":
		m.move(1)
	case "enter":
		m.filtering = false // применяем фильтр, остаёмся в списке
	case "esc", "ctrl+c":
		m.filtering = false
		m.filter = ""
		m.rebuild()
	}
	return m, nil
}

// handleMouse обрабатывает колесо (движение курсора) и левый клик (выбор+нажатие).
func (m *mainModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}
	switch {
	case msg.Button == tea.MouseButtonWheelUp:
		m.move(-1)
	case msg.Button == tea.MouseButtonWheelDown:
		m.move(1)
	case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
		if i := m.regionAt(msg.X, msg.Y); i >= 0 {
			m.cursor = i
			return m.activate()
		}
	}
	return m, nil
}

// regionAt возвращает индекс пункта, чья кликабельная область содержит точку (x,y).
func (m *mainModel) regionAt(x, y int) int {
	for _, r := range m.regions {
		if x >= r.x0 && x <= r.x1 && y >= r.y0 && y <= r.y1 {
			return r.index
		}
	}
	return -1
}

// statusDotPing возвращает цветной индикатор статуса и строку пинга.
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
	return styRed.Render("●"), styDim.Render("—")
}

// cardWidth возвращает ширину карточки сервера с учётом ширины терминала.
func (m *mainModel) cardWidth() int {
	w := m.width - 2
	if w > 66 {
		w = 66
	}
	if w < 34 {
		w = 34
	}
	return w
}

// serverCard рисует сервер как карточку-кнопку в рамке (3 строки). Выделенная —
// розовая жирная рамка; клик по любой части карточки выбирает сервер.
func (m *mainModel) serverCard(it menuItem, sel bool) string {
	s := it.server
	_, checked := m.statuses[s.ID]
	dot, ping := statusDotPing(m.statuses[s.ID], checked)

	star := ""
	if s.Favorite {
		star = styYellow.Render("★ ")
	}
	tags := ""
	if len(s.Tags) > 0 {
		tags = "  " + styViolet.Render(strings.Join(s.Tags, " "))
	}
	last := ""
	if rel := relativeTime(s.LastConnected); rel != "" {
		last = styDim.Render("  · " + rel)
	}
	nameSty := styBold
	if sel {
		nameSty = lipgloss.NewStyle().Foreground(lipgloss.Color(cPink)).Bold(true)
	}
	content := dot + " " + star + nameSty.Render(s.Name) + "  " +
		styGrey.Render(s.User+"@"+s.Host) + styDim.Render(":"+strconv.Itoa(s.Port)) +
		"  " + ping + tags + last

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(cDim)).
		Padding(0, 1).
		Width(m.cardWidth())
	if sel {
		box = box.BorderForeground(lipgloss.Color(cPink)) // только акцентная рамка, без заливки
	}
	return box.Render(content)
}

// actionButton рисует действие как кнопку в рамке (3 строки).
func (m *mainModel) actionButton(it menuItem, sel bool) string {
	btn := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(cDim)).
		Padding(0, 2)
	label := it.label
	if sel {
		btn = btn.BorderForeground(lipgloss.Color(cPink)) // только акцентная рамка
		label = lipgloss.NewStyle().Foreground(lipgloss.Color(cPink)).Bold(true).Render(label)
	}
	return btn.Render(label)
}

// footerHint возвращает контекстную строку подсказок для текущего выделения.
func (m *mainModel) footerHint() string {
	if it := m.current(); it != nil && it.kind == kindServer {
		return T.FooterServer
	}
	return T.FooterAction
}

// render строит экран из карточек серверов и кнопок действий, попутно заполняя
// m.regions кликабельными областями (для попадания мышью).
func (m *mainModel) render() string {
	if m.showHelp {
		return m.renderHelp()
	}
	var lines []string
	var regions []hitRegion
	// push добавляет блок (возможно многострочный) и возвращает индекс его первой строки.
	push := func(s string) int {
		start := len(lines)
		lines = append(lines, strings.Split(s, "\n")...)
		return start
	}

	for _, l := range strings.Split(bannerArt, "\n") {
		lines = append(lines, gradientLine(l))
	}
	push(styGrey.Italic(true).Render(T.Tagline))

	// строка фильтра / индикатор сортировки
	if m.filtering || m.filter != "" {
		cursor := ""
		if m.filtering {
			cursor = styPink.Render("▏")
		}
		push(styViolet.Render(T.FilterPrompt) + m.filter + cursor)
	} else if mode := m.sortLabel(); mode != "" {
		push(styDim.Render(mode))
	}
	push("")

	// индексы карточек серверов среди видимых пунктов
	var serverIdx []int
	for i, it := range m.visible {
		if it.kind == kindServer {
			serverIdx = append(serverIdx, i)
		}
	}

	// кнопки действий заранее (нужна их высота для расчёта окна серверов)
	var abtns []string
	var aidx []int
	for i, it := range m.visible {
		if it.kind == kindAction {
			abtns = append(abtns, m.actionButton(it, i == m.cursor))
			aidx = append(aidx, i)
		}
	}
	actionRows := wrapButtons(abtns, m.width)
	actionHeight := len(actionRows) * 3

	// секция серверов — с прокруткой, если не помещается по высоте
	if len(serverIdx) > 0 {
		push(sectionStyle.Render(T.SectionServers))
		vis := m.serverViewport(len(serverIdx), len(lines), actionHeight)
		m.clampServerOffset(serverIdx, vis)
		end := min(m.serverOffset+vis, len(serverIdx))
		if m.serverOffset > 0 {
			push(styDim.Render(fmt.Sprintf(T.MoreAbove, m.serverOffset)))
		}
		for p := m.serverOffset; p < end; p++ {
			gi := serverIdx[p]
			card := m.serverCard(m.visible[gi], gi == m.cursor)
			y0 := push(card)
			h := strings.Count(card, "\n") + 1
			regions = append(regions, hitRegion{index: gi, x0: 0, x1: lipgloss.Width(card) - 1, y0: y0, y1: y0 + h - 1})
		}
		if end < len(serverIdx) {
			push(styDim.Render(fmt.Sprintf(T.MoreBelow, len(serverIdx)-end)))
		}
	} else if m.filter != "" {
		push(styDim.Render(T.FilterNoMatch))
	} else {
		push(styBold.Render(T.EmptyTitle))
		push(styGrey.Render(T.EmptyBody))
	}

	push("")
	push(sectionStyle.Render(T.SectionActions))

	// секция действий — ряд(ы) кнопок с переносом по ширине
	for _, group := range actionRows {
		rowBtns := make([]string, len(group))
		for k, bi := range group {
			rowBtns[k] = abtns[bi]
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, joinWithGap(rowBtns)...)
		y0 := push(row)
		h := strings.Count(row, "\n") + 1
		x := 0
		for _, bi := range group {
			w := lipgloss.Width(abtns[bi])
			regions = append(regions, hitRegion{index: aidx[bi], x0: x, x1: x + w - 1, y0: y0, y1: y0 + h - 1})
			x += w + 1
		}
	}

	push("")
	push(styDim.Render(m.footerHint()))

	m.regions = regions
	return strings.Join(lines, "\n")
}

// serverViewport вычисляет, сколько карточек серверов (по 3 строки) помещается
// ниже уже занятых linesUsed строк, оставив место под секцию действий и футер.
func (m *mainModel) serverViewport(total, linesUsed, actionHeight int) int {
	if m.height <= 0 {
		return total // высота ещё не известна — показываем все (без прокрутки)
	}
	below := 1 /*blank*/ + 1 /*ДЕЙСТВИЯ*/ + actionHeight + 1 /*blank*/ + 1 /*footer*/ + 2 /*индикаторы ▲▼*/
	avail := m.height - linesUsed - below
	if avail < 3 {
		avail = 3
	}
	return max(1, avail/3)
}

// clampServerOffset держит выделенную карточку в видимом окне прокрутки.
func (m *mainModel) clampServerOffset(serverIdx []int, vis int) {
	n := len(serverIdx)
	if vis >= n {
		m.serverOffset = 0
		return
	}
	curPos := -1
	for p, gi := range serverIdx {
		if gi == m.cursor {
			curPos = p
			break
		}
	}
	if curPos >= 0 {
		if curPos < m.serverOffset {
			m.serverOffset = curPos
		}
		if curPos >= m.serverOffset+vis {
			m.serverOffset = curPos - vis + 1
		}
	}
	if m.serverOffset > n-vis {
		m.serverOffset = n - vis
	}
	if m.serverOffset < 0 {
		m.serverOffset = 0
	}
}

// wrapButtons раскладывает кнопки по рядам, влезающим в ширину maxW; возвращает
// группы индексов кнопок по рядам.
func wrapButtons(btns []string, maxW int) [][]int {
	if maxW <= 0 {
		maxW = 100
	}
	var rows [][]int
	var cur []int
	curW := 0
	for i, b := range btns {
		w := lipgloss.Width(b)
		add := w
		if len(cur) > 0 {
			add++ // пробел-разделитель
		}
		if len(cur) > 0 && curW+add > maxW {
			rows = append(rows, cur)
			cur, curW = nil, 0
			add = w
		}
		cur = append(cur, i)
		curW += add
	}
	if len(cur) > 0 {
		rows = append(rows, cur)
	}
	return rows
}

// joinWithGap вставляет пробел-разделитель между кнопками для JoinHorizontal.
func joinWithGap(btns []string) []string {
	out := make([]string, 0, len(btns)*2)
	for i, b := range btns {
		if i > 0 {
			out = append(out, " ")
		}
		out = append(out, b)
	}
	return out
}

// sortLabel возвращает подпись текущего режима сортировки (пусто для ручного).
func (m *mainModel) sortLabel() string {
	switch m.store.Settings.SortMode {
	case "name":
		return T.SortByName
	case "ping":
		return T.SortByPing
	case "online":
		return T.SortByOnline
	}
	return ""
}

// renderHelp рисует оверлей со списком горячих клавиш.
func (m *mainModel) renderHelp() string {
	rows := [][2]string{
		{"↑↓ / j k / клик", T.HelpNavigate},
		{"Enter / c", T.HelpConnect},
		{"i", T.HelpInfo},
		{"b", T.HelpBootstrap},
		{"m", T.HelpManage},
		{"e", T.HelpEdit},
		{"r", T.HelpReboot},
		{"d", T.HelpDelete},
		{"f", T.HelpFavorite},
		{"s", T.HelpSort},
		{"/", T.HelpFilter},
		{"?", T.HelpHelp},
		{"q / Esc", T.HelpQuit},
	}
	keySty := lipgloss.NewStyle().Foreground(lipgloss.Color(cPink)).Bold(true).Width(18)
	var body []string
	for _, r := range rows {
		body = append(body, keySty.Render(r[0])+styGrey.Render(r[1]))
	}
	panel := panelViolet.Render(
		styViolet.Bold(true).Render(T.HelpTitle) + "\n\n" + strings.Join(body, "\n"))
	return panel + "\n" + styDim.Render(T.HelpClose)
}

// View возвращает текущий кадр экрана.
func (m *mainModel) View() string { return m.render() }

// runMainTUI показывает главный экран и возвращает выбранный пункт (nil — выход).
func runMainTUI(st *Store) *menuItem {
	m := newMainModel(st)
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		return nil
	}
	return m.result
}
