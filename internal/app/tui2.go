package app

// Панельный лейаут главного экрана (v2, в стиле lazygit): слева-вверху — серверы,
// слева-внизу — меню, справа-вверху — действия выбранного сервера, справа-внизу —
// живая статистика (или описание пункта меню). Фокус — Tab/клик, «v» — вернуться
// к карточкам. Включается настройкой ui_mode=panels.

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type focusPanel int

const (
	fpServers focusPanel = iota
	fpMenu
	fpActions
)

// statsInterval — период опроса живой статистики выделенного сервера.
const statsInterval = 2 * time.Second

// panelMenuItem — пункт левого-нижнего меню (глобальные действия) с описанием.
type panelMenuItem struct {
	action, label, desc string
}

// liveStat — живая статистика одного сервера (панель = встроенный монитор):
// последний снимок, предыдущий (для дельт), вычисленные CPU% и скорости сети.
type liveStat struct {
	sample  *monSample
	prev    *monSample
	cpu     float64
	rxRate  float64
	txRate  float64
	hasRate bool   // дельты посчитаны (есть два снимка)
	errText string // ошибка последнего опроса
}

// panelSampleMsg — результат опроса статистики сервера (по тикеру).
type panelSampleMsg struct {
	id     string
	sample *monSample
}

// statsTickMsg — тик периодического обновления живой статистики.
type statsTickMsg struct{}

type panelRegion struct {
	panel          focusPanel
	index          int
	x0, x1, y0, y1 int
}

type panelsModel struct {
	store       *Store
	statuses    map[string]probeResult
	live        map[string]*liveStat // живая статистика по ID сервера
	liveLoading map[string]bool      // опрос статистики уже идёт
	width       int
	height      int
	focus       focusPanel
	serverCur   int
	menuCur     int
	actionCur   int
	serverOff   int
	menuOff     int
	actionOff   int
	regions     []panelRegion
	result      *menuItem
	switchToV1  bool // пользователь нажал «v» — переключить на карточки
}

// panelMenu возвращает пункты левого-нижнего меню.
func panelMenu() []panelMenuItem {
	return []panelMenuItem{
		{"add", T.ItemAdd, T.MenuAddDesc},
		{"import", T.ToolImport, T.MenuImportDesc},
		{"export", T.ToolExport, T.MenuExportDesc},
		{"backups", T.ToolBackups, T.MenuBackupsDesc},
		{"snippets", T.ToolSnippets, T.MenuSnippetsDesc},
		{"pubkey", T.ToolPubkey, T.MenuPubkeyDesc},
		{"settings", T.ToolSettings, T.MenuSettingsDesc},
		{"exit", T.ItemExit, T.MenuExitDesc},
	}
}

// serverActions возвращает действия для конкретного сервера (правая-верхняя панель).
func serverActions(s *Server) []menuItemAction {
	// «Монитор» здесь нет намеренно: живая статистика показана в панели справа-внизу.
	acts := []menuItemAction{
		{"connect", T.MenuConnect},
		{"info", T.MenuInfo},
		{"run", T.MenuRun},
		{"password", T.MenuPasswords},
	}
	if s.Auth == "password" {
		acts = append(acts, menuItemAction{"install_key", T.MenuInstallKey})
	}
	acts = append(acts,
		menuItemAction{"bootstrap", T.MenuBootstrap},
		menuItemAction{"copy", T.MenuCopyCmd},
		menuItemAction{"forget", T.MenuForget},
		menuItemAction{"reboot", T.MenuReboot},
		menuItemAction{"edit", T.MenuEdit},
		menuItemAction{"delete", T.MenuDelete},
	)
	return acts
}

type menuItemAction struct{ action, label string }

// newPanelsModel создаёт модель панельного лейаута.
func newPanelsModel(st *Store) *panelsModel {
	m := &panelsModel{
		store: st, statuses: map[string]probeResult{},
		live: map[string]*liveStat{}, liveLoading: map[string]bool{},
	}
	if len(st.Servers) == 0 {
		m.focus = fpMenu
	}
	return m
}

func (m *panelsModel) Init() tea.Cmd {
	return tea.Batch(probeCmd(m.store), refreshTick(), statsTick(), m.refreshStats())
}

// statsTick планирует периодическое обновление живой статистики.
func statsTick() tea.Cmd {
	return tea.Tick(statsInterval, func(time.Time) tea.Msg { return statsTickMsg{} })
}

// curServer возвращает выделенный сервер (или nil).
func (m *panelsModel) curServer() *Server {
	if m.serverCur >= 0 && m.serverCur < len(m.store.Servers) {
		return m.store.Servers[m.serverCur]
	}
	return nil
}

// refreshStats запускает опрос живой статистики выделенного сервера, если опрос
// ещё не идёт. Зовётся и по тикеру (реальное время), и при смене сервера — так
// панель статистики работает как встроенный live-монитор.
func (m *panelsModel) refreshStats() tea.Cmd {
	s := m.curServer()
	if s == nil || m.liveLoading[s.ID] {
		return nil
	}
	m.liveLoading[s.ID] = true
	return sampleCmd(s)
}

// sampleCmd асинхронно снимает один срез метрик сервера (тем же monCmd, что и
// полноэкранный монитор). Соединение открывается на каждый опрос и закрывается —
// счётчики /proc кумулятивные, поэтому дельты между опросами считаются корректно.
func sampleCmd(s *Server) tea.Cmd {
	return func() tea.Msg {
		r, err := openRemote(s, false)
		if err != nil {
			return panelSampleMsg{id: s.ID, sample: &monSample{at: time.Now(), err: err}}
		}
		defer r.Close()
		_, out, err := r.Run(monCmd, "", nil, 10*time.Second)
		if err != nil {
			return panelSampleMsg{id: s.ID, sample: &monSample{at: time.Now(), err: err}}
		}
		return panelSampleMsg{id: s.ID, sample: parseMonSample(sanitizeRemote(out), time.Now())}
	}
}

// applySample сохраняет новый срез и пересчитывает дельты (CPU%, скорость сети).
func (m *panelsModel) applySample(id string, sample *monSample) {
	m.liveLoading[id] = false
	ls := m.live[id]
	if ls == nil {
		ls = &liveStat{}
		m.live[id] = ls
	}
	if sample.err != nil {
		ls.errText = sample.err.Error()
		return
	}
	ls.errText = ""
	if cpu, rx, tx, ok := computeDeltas(ls.prev, sample); ok {
		ls.cpu, ls.rxRate, ls.txRate, ls.hasRate = cpu, rx, tx, true
	}
	ls.sample = sample
	ls.prev = sample
}

func (m *panelsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case statusMsg:
		m.statuses = msg
	case refreshTickMsg:
		return m, tea.Batch(probeCmd(m.store), refreshTick())
	case statsTickMsg:
		return m, tea.Batch(m.refreshStats(), statsTick())
	case panelSampleMsg:
		m.applySample(msg.id, msg.sample)
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	return m, nil
}

// cycleFocus переключает фокус только между двумя левыми окнами: серверы ↔ меню.
// Панель действий в обход по Tab не входит — она открывается по Enter на сервере.
func (m *panelsModel) cycleFocus(dir int) {
	_ = dir // два окна — направление роли не играет
	if len(m.store.Servers) == 0 {
		m.focus = fpMenu
		return
	}
	if m.focus == fpMenu {
		m.focus = fpServers
	} else { // серверы или (после Enter) действия — уводим к меню
		m.focus = fpMenu
	}
}

// closeActions возвращает фокус с панели действий обратно на список серверов.
func (m *panelsModel) closeActions() {
	if m.focus == fpActions {
		m.focus = fpServers
	}
}

func (m *panelsModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.cycleFocus(1)
	case "shift+tab":
		m.cycleFocus(-1)
	case "right", "l":
		// из списка серверов «вправо» — открыть панель действий выбранного сервера
		if m.focus == fpServers && m.curServer() != nil {
			m.focus = fpActions
			m.actionCur = 0
			return m, m.refreshStats()
		}
	case "left", "h":
		m.closeActions()
	case "up", "k":
		return m, m.moveCursor(-1)
	case "down", "j":
		return m, m.moveCursor(1)
	case "enter", " ":
		return m.activate()
	case "v":
		m.switchToV1 = true
		return m, tea.Quit
	case "esc":
		if m.focus == fpActions { // Esc из действий — назад к серверам, не выход
			m.focus = fpServers
			return m, nil
		}
		return m, tea.Quit
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// moveCursor двигает курсор в активной панели; при смене сервера грузит статистику.
func (m *panelsModel) moveCursor(delta int) tea.Cmd {
	switch m.focus {
	case fpServers:
		n := len(m.store.Servers)
		if n > 0 {
			m.serverCur = (m.serverCur + delta + n) % n
			m.actionCur = 0
			return m.refreshStats()
		}
	case fpMenu:
		n := len(panelMenu())
		m.menuCur = (m.menuCur + delta + n) % n
	case fpActions:
		if s := m.curServer(); s != nil {
			n := len(serverActions(s))
			m.actionCur = (m.actionCur + delta + n) % n
		}
	}
	return nil
}

// activate выполняет выбранный пункт активной панели.
func (m *panelsModel) activate() (tea.Model, tea.Cmd) {
	switch m.focus {
	case fpServers:
		// Enter по серверу не подключает сразу — открывает панель действий справа
		// (подключение — отдельная кнопка «Подключиться»).
		if s := m.curServer(); s != nil {
			m.focus = fpActions
			m.actionCur = 0
			return m, m.refreshStats()
		}
	case fpMenu:
		mi := panelMenu()[m.menuCur]
		if mi.action == "exit" {
			return m, tea.Quit
		}
		m.result = &menuItem{kind: kindAction, action: mi.action}
		return m, tea.Quit
	case fpActions:
		if s := m.curServer(); s != nil {
			acts := serverActions(s)
			if m.actionCur < len(acts) {
				m.result = &menuItem{kind: kindAction, action: acts[m.actionCur].action, server: s}
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m *panelsModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Button == tea.MouseButtonWheelUp:
		return m, m.moveCursor(-1)
	case msg.Button == tea.MouseButtonWheelDown:
		return m, m.moveCursor(1)
	case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
		for _, r := range m.regions {
			if msg.X >= r.x0 && msg.X <= r.x1 && msg.Y >= r.y0 && msg.Y <= r.y1 {
				m.focus = r.panel
				switch r.panel {
				case fpServers:
					m.serverCur = r.index
					m.actionCur = 0
					return m, m.refreshStats() // клик по серверу — только выбрать
				case fpMenu:
					m.menuCur = r.index
					return m.activate()
				case fpActions:
					m.actionCur = r.index
					return m.activate()
				}
			}
		}
	}
	return m, nil
}

// View возвращает текущий кадр панельного лейаута.
func (m *panelsModel) View() string { return m.render() }

// serverItems формирует строки серверов для левой-верхней панели.
func (m *panelsModel) serverItems() []string {
	items := make([]string, len(m.store.Servers))
	for i, s := range m.store.Servers {
		_, checked := m.statuses[s.ID]
		dot, _ := statusDotPing(m.statuses[s.ID], checked) // пинг теперь в панели статистики
		ptr := "  "
		if i == m.serverCur {
			ptr = pointer(m.focus == fpServers)
		}
		star := ""
		if s.Favorite {
			star = styYellow.Render("★")
		}
		items[i] = ptr + dot + " " + star + styBold.Render(s.Name) + "  " +
			styGrey.Render(s.User+"@"+s.Host)
	}
	return items
}

// menuItems формирует строки меню (левая-нижняя панель).
func (m *panelsModel) menuItems() []string {
	mm := panelMenu()
	items := make([]string, len(mm))
	for i, mi := range mm {
		ptr := "  "
		if i == m.menuCur {
			ptr = pointer(m.focus == fpMenu)
		}
		items[i] = ptr + mi.label
	}
	return items
}

// actionItems формирует строки действий сервера (правая-верхняя панель).
func (m *panelsModel) actionItems(s *Server) []string {
	acts := serverActions(s)
	items := make([]string, len(acts))
	for i, a := range acts {
		ptr := "  "
		if i == m.actionCur {
			ptr = pointer(m.focus == fpActions)
		}
		items[i] = ptr + a.label
	}
	return items
}

// pointer возвращает указатель: акцентный, если панель в фокусе, иначе тусклый.
func pointer(focused bool) string {
	if focused {
		return styPink.Render("❯ ")
	}
	return styDim.Render("· ")
}

// statLines формирует строки живой статистики сервера (правая-нижняя панель):
// пинг + CPU/RAM/диск/load/сеть/аптайм. Панель работает как встроенный монитор,
// поэтому значения обновляются по тикеру каждые statsInterval.
func (m *panelsModel) statLines(s *Server, w int) []string {
	var out []string
	add := func(label, val string) { out = append(out, kv(label, val)) }

	// пинг (TCP-проба) — вынесен из списка серверов сюда, в статистику
	pr, checked := m.statuses[s.ID]
	dot, ping := statusDotPing(pr, checked)
	add(T.ColPing, dot+" "+ping)

	ls := m.live[s.ID]
	if ls == nil || (ls.sample == nil && ls.errText == "") {
		out = append(out, styDim.Render(T.StatsLoading))
		return out
	}
	if ls.sample == nil { // была только ошибка, снимка ещё нет
		out = append(out, styRed.Render("✗ ")+styDim.Render(T.StatsError))
		return out
	}

	// ширина шкалы: оставляем место под подпись (метка ~11) и байты (~16)
	mw := w - 30
	if mw < 8 {
		mw = 8
	}
	if mw > 40 {
		mw = 40
	}
	smp := ls.sample
	// CPU (появляется после второго опроса — нужна дельта)
	if ls.hasRate {
		add(T.LblCPU, meter(ls.cpu, mw))
	} else {
		add(T.LblCPU, styDim.Render("…"))
	}
	if mem := strings.Fields(smp.memLine); len(mem) >= 3 {
		total, _ := strconv.ParseFloat(mem[1], 64)
		used, _ := strconv.ParseFloat(mem[2], 64)
		if total > 0 {
			add(T.LblRAM, meter(used/total, mw)+" "+
				styGrey.Render(humanBytes(used)+"/"+humanBytes(total)))
		}
	}
	if disk := strings.Fields(smp.dfLine); len(disk) >= 4 {
		total, _ := strconv.ParseFloat(disk[1], 64)
		used, _ := strconv.ParseFloat(disk[2], 64)
		if total > 0 {
			add(T.LblDisk, meter(used/total, mw)+" "+
				styGrey.Render(humanBytes(used*1024)+"/"+humanBytes(total*1024)))
		}
	}
	if load := strings.Fields(smp.load); len(load) >= 3 {
		add(T.LblLoad, styGrey.Render(strings.Join(load[:3], " ")+fmt.Sprintf(T.CPUsFmt, smp.ncpu)))
	}
	add(T.LblNet, styCyan.Render("↓ "+humanBytes(ls.rxRate)+"/s")+"  "+
		styPink.Render("↑ "+humanBytes(ls.txRate)+"/s"))
	if smp.uptime != "" {
		add(T.LblUptime, styGrey.Render(clip(smp.uptime, w-12)))
	}
	// внешние IP-адреса VPS — внизу (первый с меткой, остальные выровнены)
	for i, ip := range smp.ips {
		label := ""
		if i == 0 {
			label = T.LblIPs
		}
		sty := styGreen // IPv4
		if strings.Contains(ip, ":") {
			sty = styCyan // IPv6
		}
		add(label, sty.Render(clip(ip, w-12)))
	}
	return out
}

// render компонует четыре панели и заполняет m.regions.
func (m *panelsModel) render() string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 32
	}
	leftW := w * 40 / 100
	if leftW < 30 {
		leftW = 30
	}
	if leftW > 54 {
		leftW = 54
	}
	rightW := w - leftW
	contentH := h - 1 // строка подсказки
	serversH := contentH * 6 / 10
	if serversH < 5 {
		serversH = 5
	}
	menuH := contentH - serversH

	var regions []panelRegion

	// левая колонка
	srvBox, srvRows := panelBox(T.PanelServers, m.serverItems(), leftW, serversH, m.focus == fpServers, m.serverCur, &m.serverOff)
	menuBox, menuRows := panelBox(T.PanelMenu, m.menuItems(), leftW, menuH, m.focus == fpMenu, m.menuCur, &m.menuOff)
	leftLines := append(append([]string{}, srvBox...), menuBox...)
	for i, r := range srvRows {
		if r >= 0 {
			regions = append(regions, panelRegion{fpServers, i, 0, leftW - 1, r, r})
		}
	}
	for i, r := range menuRows {
		if r >= 0 {
			regions = append(regions, panelRegion{fpMenu, i, 0, leftW - 1, serversH + r, serversH + r})
		}
	}

	// правая колонка
	var rightLines []string
	s := m.curServer()
	if m.focus == fpMenu {
		// описание пункта меню на всю правую высоту
		desc := wrapText(panelMenu()[m.menuCur].desc, rightW-4)
		descBox, _ := panelBox(T.PanelDesc, desc, rightW, contentH, false, -1, new(int))
		rightLines = descBox
	} else if s != nil {
		actH := contentH * 55 / 100
		if actH < 5 {
			actH = 5
		}
		statH := contentH - actH
		actBox, actRows := panelBox(T.PanelActions, m.actionItems(s), rightW, actH, m.focus == fpActions, m.actionCur, &m.actionOff)
		statBox, _ := panelBox(T.PanelStats, m.statLines(s, rightW-4), rightW, statH, false, -1, new(int))
		rightLines = append(append([]string{}, actBox...), statBox...)
		for i, r := range actRows {
			if r >= 0 {
				regions = append(regions, panelRegion{fpActions, i, leftW, w - 1, r, r})
			}
		}
	} else {
		emptyBox, _ := panelBox(T.PanelStats, wrapText(T.NoServersHint, rightW-4), rightW, contentH, false, -1, new(int))
		rightLines = emptyBox
	}

	// компоновка колонок построчно
	var out []string
	for r := 0; r < contentH; r++ {
		l, rl := "", ""
		if r < len(leftLines) {
			l = leftLines[r]
		}
		if r < len(rightLines) {
			rl = rightLines[r]
		}
		out = append(out, l+rl)
	}
	out = append(out, styDim.Render(T.PanelHint))

	m.regions = regions
	return strings.Join(out, "\n")
}

// runPanelsTUI показывает панельный экран; возвращает (menuItem, переключитьНаV1).
func runPanelsTUI(st *Store) (*menuItem, bool) {
	m := newPanelsModel(st)
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		return nil, false
	}
	return m.result, m.switchToV1
}
