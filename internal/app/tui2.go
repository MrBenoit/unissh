package app

// Главный экран (панели, в стиле lazygit): слева-вверху — серверы, сгруппированные
// по папкам, слева-внизу — меню, справа-вверху — действия выбранного сервера,
// справа-внизу — живая статистика (или описание пункта меню / сводка по папке).
// Фокус — Tab/клик. Поиск «/», сортировка «s», избранное «f», справка «?».
// Папки: «n» создать, «r» переименовать, «d» удалить, shift+↑↓ — порядок,
// ←/→ свернуть/развернуть; «m» включает режим перемещения — схваченный сервер
// едет по папкам за курсором (Enter/Esc). Все диалоги папок — инлайновые.
// «space» отмечает серверы (на строке папки — всю группу): правая-верхняя панель
// переключается на групповые действия — команда, перенос, удаление.

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MrBenoit/unissh/internal/core"
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

// statsConnMsg — результат открытия постоянного соединения для статистики.
type statsConnMsg struct {
	id  string
	r   *Remote
	err error
}

// statsTickMsg — тик периодического обновления живой статистики.
type statsTickMsg struct{}

type panelRegion struct {
	panel          focusPanel
	index          int
	x0, x1, y0, y1 int
}

type panelsModel struct {
	store    *Store
	statuses map[string]probeResult
	live     map[string]*liveStat // живая статистика по ID сервера
	width    int
	height   int
	focus    focusPanel

	// одно постоянное SSH-соединение на выделенный сервер: dial выполняется
	// один раз, дальше по нему гоняется monCmd (иначе на каждый тик уходил бы
	// полный TCP+KEX+auth — до бана fail2ban)
	statsRemote   *Remote
	statsID       string // сервер, к которому открыт statsRemote
	statsConnID   string // сервер, к которому прямо сейчас идёт подключение
	statsSampling bool   // опрос по открытому соединению уже выполняется

	rows   []listRow // видимые строки: папки и серверы
	rowCur int       // курсор по строкам
	rowOff int       // прокрутка списка строк

	menuCur   int
	actionCur int
	menuOff   int
	actionOff int

	filter    string // текст поиска
	filtering bool   // активен режим ввода поиска
	showHelp  bool   // показан оверлей справки

	// мультивыбор: отмеченные ID серверов. Пока карта пуста, правая-верхняя
	// панель показывает действия одного сервера; как только что-то отмечено —
	// групповые действия.
	selected map[string]bool

	// режим перемещения: «схваченный» сервер едет по папкам вместе с курсором,
	// Enter фиксирует, Esc возвращает в исходную папку
	moving     *Server
	movingFrom string

	prompt *folderPrompt // активный инлайновый промпт (создать/переименовать/удалить папку)

	regions []panelRegion
	result  *menuItem
}

// panelMenu возвращает пункты левого-нижнего меню.
func panelMenu() []panelMenuItem {
	return []panelMenuItem{
		{"add", T.ItemAdd, T.MenuAddDesc},
		{"health", T.ToolHealth, T.MenuHealthDesc},
		{"multi", T.ToolMulti, T.MenuMultiDesc},
		{"import", T.ToolImport, T.MenuImportDesc},
		{"export", T.ToolExport, T.MenuExportDesc},
		{"backups", T.ToolBackups, T.MenuBackupsDesc},
		{"snippets", T.ToolSnippets, T.MenuSnippetsDesc},
		{"journal", T.ToolJournal, T.MenuJournalDesc},
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
		menuItemAction{"tunnels", T.MenuTunnels},
		menuItemAction{"transfer", T.MenuTransfer},
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
		live: map[string]*liveStat{}, selected: map[string]bool{},
	}
	m.rebuildRows()
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

// statsVisible сообщает, видна ли панель статистики: при фокусе на меню справа
// показывается описание пункта, и опрашивать сервер незачем.
func (m *panelsModel) statsVisible() bool { return m.focus != fpMenu }

// serverOffline сообщает, что сервер проверен и не отвечает. Такой сервер не
// опрашиваем: каждый dial висел бы до таймаута и тут же начинался заново.
func (m *panelsModel) serverOffline(s *Server) bool {
	pr, ok := m.statuses[s.ID]
	return ok && !pr.Online
}

// dropStatsConn закрывает постоянное соединение статистики.
func (m *panelsModel) dropStatsConn() {
	if m.statsRemote != nil {
		m.statsRemote.Close()
		m.statsRemote = nil
	}
	m.statsID = ""
	m.statsSampling = false
}

// onServerChanged вызывается при смене выделенного сервера: соединение к прежнему
// закрывается, новое откроется на ближайшем тике. Это естественный дебаунс —
// быстрая прокрутка списка не порождает шторм подключений.
func (m *panelsModel) onServerChanged() tea.Cmd {
	s := m.curServer()
	if s == nil || m.statsID != s.ID {
		m.dropStatsConn()
	}
	return nil
}

// refreshStats — шаг живой статистики (по тикеру и при старте): держим одно
// соединение на выделенный сервер и гоняем по нему monCmd.
func (m *panelsModel) refreshStats() tea.Cmd {
	s := m.curServer()
	if s == nil {
		m.dropStatsConn()
		return nil
	}
	if m.statsID != "" && m.statsID != s.ID {
		m.dropStatsConn()
	}
	if !m.statsVisible() || m.serverOffline(s) {
		return nil
	}
	if m.statsRemote == nil {
		if m.statsConnID != "" {
			return nil // подключение уже в процессе
		}
		m.statsConnID = s.ID
		return statsConnectCmd(s)
	}
	if m.statsSampling {
		return nil
	}
	m.statsSampling = true
	return statsSampleCmd(s.ID, m.statsRemote)
}

// statsConnectCmd асинхронно открывает постоянное соединение для статистики.
func statsConnectCmd(s *Server) tea.Cmd {
	id := s.ID
	return func() tea.Msg {
		r, err := openRemote(s, false)
		return statsConnMsg{id: id, r: r, err: err}
	}
}

// statsSampleCmd снимает срез метрик по уже открытому соединению (один RTT).
func statsSampleCmd(id string, r *Remote) tea.Cmd {
	return func() tea.Msg {
		_, out, err := r.Run(monCmd, "", nil, 10*time.Second)
		if err != nil {
			return panelSampleMsg{id: id, sample: &monSample{at: time.Now(), err: err}}
		}
		return panelSampleMsg{id: id, sample: parseMonSample(sanitizeRemote(out), time.Now())}
	}
}

// applySample сохраняет новый срез и пересчитывает дельты (CPU%, скорость сети).
func (m *panelsModel) applySample(id string, sample *monSample) {
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
		m.rebuildRows() // сортировка по пингу/доступности зависит от статусов
	case refreshTickMsg:
		return m, tea.Batch(probeCmd(m.store), refreshTick())
	case statsTickMsg:
		return m, tea.Batch(m.refreshStats(), statsTick())

	case statsConnMsg:
		if msg.id == m.statsConnID {
			m.statsConnID = ""
		}
		if msg.err != nil {
			m.applySample(msg.id, &monSample{at: time.Now(), err: msg.err})
			return m, nil
		}
		cur := m.curServer()
		if cur == nil || cur.ID != msg.id { // курсор уже ушёл — соединение не нужно
			msg.r.Close()
			return m, nil
		}
		m.statsRemote, m.statsID = msg.r, msg.id
		m.statsSampling = true
		return m, statsSampleCmd(msg.id, msg.r)

	case panelSampleMsg:
		// ответ мог прийти от соединения к уже покинутому серверу — тогда просто
		// сохраняем снимок и не трогаем текущее соединение
		if msg.id == m.statsID {
			m.statsSampling = false
		}
		m.applySample(msg.id, msg.sample)
		if msg.sample.err != nil && msg.id == m.statsID {
			m.dropStatsConn() // соединение мертво — переподключимся на следующем тике
		}

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

func (m *panelsModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false // любая клавиша закрывает справку
		return m, nil
	}
	if m.prompt != nil {
		return m.handlePromptKey(msg)
	}
	if m.moving != nil {
		return m.handleMoveKey(msg)
	}
	if m.filtering {
		return m.handleFilterKey(msg)
	}
	switch msg.String() {
	case "tab":
		m.cycleFocus(1)
	case "shift+tab":
		m.cycleFocus(-1)
	case "right", "l":
		return m, m.expandOrOpen()
	case "left", "h":
		m.collapseOrBack()
	case "up", "k":
		return m, m.moveCursor(-1)
	case "down", "j":
		return m, m.moveCursor(1)
	case "shift+up", "K": // порядок папок в списке
		m.reorderFolder(-1)
	case "shift+down", "J":
		m.reorderFolder(1)
	case "home", "g":
		return m, m.jump(0)
	case "end", "G":
		return m, m.jump(m.lastIndex())
	case "enter":
		return m.activate()
	case " ":
		// В списке серверов пробел отмечает строку; в остальных панелях он по
		// старой привычке работает как Enter.
		if m.focus == fpServers {
			m.toggleSelect()
			return m, nil
		}
		return m.activate()
	case "c":
		return m.connectSelected()
	case "t":
		return m.openAction("tunnels")
	case "f":
		return m, m.toggleFavorite()
	case "s":
		m.cycleSort()
	case "/":
		if m.focus == fpServers {
			m.filtering = true
		}
	case "n":
		if m.focus == fpServers {
			m.askFolderNew()
		}
	case "m":
		if m.focus == fpServers {
			m.startMove() // «захват» сервера — дальше он едет за курсором
		}
	case "r":
		if m.focus == fpServers {
			m.askFolderRename()
		}
	case "d":
		if m.focus == fpServers {
			m.askFolderDelete()
		}
	case "?":
		m.showHelp = true
	case "esc":
		if m.filter != "" { // сначала сбрасываем поиск
			m.filter = ""
			m.rebuildRows()
			return m, m.onServerChanged()
		}
		if m.selCount() > 0 { // затем — выделение: выход не должен быть внезапным
			m.clearSelection()
			m.focus = fpServers
			return m, nil
		}
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

// handleMoveKey — клавиши режима перемещения: ↑↓ ведут схваченный сервер по
// папкам, Enter/m фиксируют, Esc возвращает на место. Прочие клавиши игнорируются,
// чтобы случайное нажатие не оставило сервер «в воздухе».
func (m *panelsModel) handleMoveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "left", "h":
		m.shiftMove(-1)
	case "down", "j", "right", "l":
		m.shiftMove(1)
	case "enter", " ", "m":
		m.commitMove()
	case "esc":
		m.cancelMove()
	case "q", "ctrl+c":
		m.cancelMove()
		return m, tea.Quit
	}
	return m, nil
}

// handleFilterKey обрабатывает ввод в режиме поиска: печать сужает список,
// стрелки двигают курсор, Enter применяет, Esc сбрасывает.
func (m *panelsModel) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace:
		m.filter += string(msg.Runes)
		m.rebuildRows()
		return m, m.onServerChanged()
	case tea.KeyBackspace:
		if r := []rune(m.filter); len(r) > 0 {
			m.filter = string(r[:len(r)-1])
			m.rebuildRows()
			return m, m.onServerChanged()
		}
		return m, nil
	}
	switch msg.String() {
	case "up", "ctrl+k":
		return m, m.moveCursor(-1)
	case "down", "ctrl+j":
		return m, m.moveCursor(1)
	case "enter":
		m.filtering = false // применяем поиск, остаёмся в списке
	case "esc", "ctrl+c":
		m.filtering = false
		m.filter = ""
		m.rebuildRows()
		return m, m.onServerChanged()
	}
	return m, nil
}

// expandOrOpen: на папке — развернуть, на сервере — открыть панель действий.
func (m *panelsModel) expandOrOpen() tea.Cmd {
	if m.focus != fpServers {
		return nil
	}
	if name, ok := m.onFolderRow(); ok {
		if m.store.FolderCollapsed(name) {
			m.store.ToggleFolder(name)
			m.store.MustSave()
			m.rebuildRows()
		}
		return nil
	}
	if m.curServer() != nil {
		m.focus = fpActions
		m.actionCur = 0
		return m.onServerChanged()
	}
	return nil
}

// collapseOrBack: из действий — назад к серверам; на папке — свернуть.
func (m *panelsModel) collapseOrBack() {
	if m.focus == fpActions {
		m.focus = fpServers
		return
	}
	if name, ok := m.onFolderRow(); ok && !m.store.FolderCollapsed(name) {
		m.store.ToggleFolder(name)
		m.store.MustSave()
		m.rebuildRows()
	}
}

// lastIndex — индекс последней строки активной панели.
func (m *panelsModel) lastIndex() int {
	switch m.focus {
	case fpMenu:
		return len(panelMenu()) - 1
	case fpActions:
		if n := len(m.curActions()); n > 0 {
			return n - 1
		}
		return 0
	default:
		return len(m.rows) - 1
	}
}

// jump ставит курсор активной панели на конкретный индекс.
func (m *panelsModel) jump(idx int) tea.Cmd {
	if idx < 0 {
		idx = 0
	}
	switch m.focus {
	case fpMenu:
		m.menuCur = idx
	case fpActions:
		m.actionCur = idx
	default:
		if len(m.rows) > 0 {
			m.rowCur = idx
			m.actionCur = 0
			return m.onServerChanged()
		}
	}
	return nil
}

// connectSelected подключается к выделенному серверу (клавиша «c»), минуя
// панель действий. Попап подтверждения не показывается — выбор осознанный.
func (m *panelsModel) connectSelected() (tea.Model, tea.Cmd) {
	return m.openAction("connect")
}

// openAction выходит из панельного TUI с выбранным действием над текущим
// сервером: второй tea.Program поверх работающего запустить нельзя, поэтому
// диалоги выполняет dispatchMenu уже после Quit.
func (m *panelsModel) openAction(action string) (tea.Model, tea.Cmd) {
	if s := m.curServer(); s != nil {
		m.result = &menuItem{action: action, server: s}
		return m, tea.Quit
	}
	return m, nil
}

// toggleFavorite переключает «избранное» у выделенного сервера.
func (m *panelsModel) toggleFavorite() tea.Cmd {
	s := m.curServer()
	if s == nil {
		return nil
	}
	s.Favorite = !s.Favorite
	m.store.MustSave()
	m.rebuildRows()
	return m.onServerChanged()
}

// sortModes — порядок перебора режимов сортировки по клавише «s».
var sortModes = []string{"", "name", "ping", "online"}

// cycleSort переключает режим сортировки по кругу и сохраняет его.
func (m *panelsModel) cycleSort() {
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
	m.rebuildRows()
}

// moveCursor двигает курсор в активной панели; при смене сервера грузит статистику.
func (m *panelsModel) moveCursor(delta int) tea.Cmd {
	switch m.focus {
	case fpServers:
		if n := len(m.rows); n > 0 {
			m.rowCur = (m.rowCur + delta + n) % n
			m.actionCur = 0
			return m.onServerChanged()
		}
	case fpMenu:
		n := len(panelMenu())
		m.menuCur = (m.menuCur + delta + n) % n
	case fpActions:
		if n := len(m.curActions()); n > 0 {
			m.actionCur = (m.actionCur + delta + n) % n
		}
	}
	return nil
}

// activate выполняет выбранный пункт активной панели.
func (m *panelsModel) activate() (tea.Model, tea.Cmd) {
	switch m.focus {
	case fpServers:
		// Enter по папке сворачивает/разворачивает её; по серверу — открывает
		// панель действий справа (подключение — отдельная кнопка «Подключиться»).
		if name, ok := m.onFolderRow(); ok {
			m.store.ToggleFolder(name)
			m.store.MustSave()
			m.rebuildRows()
			return m, nil
		}
		if m.selCount() > 0 || m.curServer() != nil {
			m.focus = fpActions
			m.actionCur = 0
			return m, m.onServerChanged()
		}
	case fpMenu:
		mi := panelMenu()[m.menuCur]
		if mi.action == "exit" {
			return m, tea.Quit
		}
		m.result = &menuItem{action: mi.action}
		return m, tea.Quit
	case fpActions:
		acts := m.curActions()
		if m.actionCur >= len(acts) {
			return m, nil
		}
		action := acts[m.actionCur].action
		if m.selCount() > 0 {
			if action == "batch_clear" { // снять выделение можно не выходя из TUI
				m.clearSelection()
				m.focus = fpServers
				return m, nil
			}
			m.result = &menuItem{action: action, servers: m.selectedServers()}
			return m, tea.Quit
		}
		if s := m.curServer(); s != nil {
			m.result = &menuItem{action: action, server: s}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *panelsModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// в режиме перемещения колесо ведёт сервер по папкам, а клик кладёт его туда,
	// куда указали (в папку строки под курсором мыши)
	if m.moving != nil {
		switch {
		case msg.Button == tea.MouseButtonWheelUp:
			m.shiftMove(-1)
		case msg.Button == tea.MouseButtonWheelDown:
			m.shiftMove(1)
		case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
			for _, r := range m.regions {
				if r.panel == fpServers && msg.X >= r.x0 && msg.X <= r.x1 && msg.Y >= r.y0 && msg.Y <= r.y1 {
					if r.index < len(m.rows) {
						m.moving.Folder = m.rows[r.index].folder
						m.store.setCollapsed(m.moving.Folder, false)
					}
					m.commitMove()
					return m, nil
				}
			}
		}
		return m, nil
	}
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
					m.rowCur = r.index
					m.actionCur = 0
					if name, ok := m.onFolderRow(); ok { // клик по папке — свернуть/развернуть
						m.store.ToggleFolder(name)
						m.store.MustSave()
						m.rebuildRows()
						return m, nil
					}
					return m, m.onServerChanged() // клик по серверу — только выбрать
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

// serverItems формирует строки левой-верхней панели: заголовки папок и серверы.
func (m *panelsModel) serverItems() []string {
	items := make([]string, len(m.rows))
	grouped := len(m.store.Folders) > 0
	for i, r := range m.rows {
		ptr := "  "
		if i == m.rowCur {
			ptr = pointer(m.focus == fpServers)
		}
		if r.kind == rowFolder {
			arrow := "▾"
			if m.store.FolderCollapsed(r.folder) && strings.TrimSpace(m.filter) == "" {
				arrow = "▸"
			}
			count := len(m.groupServers(r.folder))
			items[i] = ptr + styViolet.Render(arrow+" "+folderLabel(r.folder)) +
				"  " + styDim.Render(strconv.Itoa(count))
			continue
		}
		s := r.server
		_, checked := m.statuses[s.ID]
		dot, _ := statusDotPing(m.statuses[s.ID], checked) // пинг теперь в панели статистики
		star := ""
		if s.Favorite {
			star = styYellow.Render("★")
		}
		indent := ""
		if grouped {
			indent = "  " // серверы внутри группы — со сдвигом
		}
		if s == m.moving { // «схваченный» сервер — акцентом и меткой перемещения
			items[i] = ptr + indent + styPink.Render("✥ ") + styPink.Bold(true).Render(s.Name) +
				"  " + styGrey.Render(s.User+"@"+s.Host)
			continue
		}
		items[i] = ptr + indent + m.selectMark(s) + dot + " " + star + styBold.Render(s.Name) + "  " +
			styGrey.Render(s.User+"@"+s.Host)
	}
	return items
}

// menuItems формирует строки меню (левая-нижняя панель): радио-кнопки [ ]/[•].
func (m *panelsModel) menuItems() []string {
	mm := panelMenu()
	items := make([]string, len(mm))
	for i, mi := range mm {
		items[i] = radioLabel(mi.label, i == m.menuCur)
	}
	return items
}

// actionItems формирует строки действий сервера (правая-верхняя панель).
func (m *panelsModel) actionItems() []string {
	acts := m.curActions()
	items := make([]string, len(acts))
	for i, a := range acts {
		items[i] = radioLabel(a.label, i == m.actionCur)
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

// statLines собирает содержимое правой-нижней панели: живая статистика сервера
// плюс список его туннелей. Туннели показываем всегда — это локальное знание, и
// оно нужно даже когда сервер офлайн или снимок ещё не пришёл.
func (m *panelsModel) statLines(s *Server, w int) []string {
	return append(m.liveLines(s, w), tunnelLines(s, w)...)
}

// liveLines формирует строки живой статистики сервера: пинг + CPU/RAM/диск/
// load/сеть/аптайм/IP. Панель работает как встроенный монитор, поэтому значения
// обновляются по тикеру каждые statsInterval.
func (m *panelsModel) liveLines(s *Server, w int) []string {
	var out []string
	add := func(label, val string) { out = append(out, kv(label, val)) }

	// пинг (TCP-проба) — вынесен из списка серверов сюда, в статистику
	pr, checked := m.statuses[s.ID]
	dot, ping := statusDotPing(pr, checked)
	add(T.ColPing, dot+" "+ping)

	if m.serverOffline(s) { // офлайн не опрашиваем — не держим «загружаю…» вечно
		out = append(out, styRed.Render("✗ ")+styDim.Render(T.StatsOffline))
		return out
	}
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
	if used, total, ok := core.MemUsage(smp.memLine); ok {
		add(T.LblRAM, meter(core.Frac(used, total), mw)+" "+
			styGrey.Render(humanBytes(used)+"/"+humanBytes(total)))
	}
	if used, total, ok := core.DiskUsage(smp.dfLine); ok {
		add(T.LblDisk, meter(core.Frac(used, total), mw)+" "+
			styGrey.Render(humanBytes(used)+"/"+humanBytes(total)))
	}
	if load, ok := core.LoadFields(smp.load); ok {
		add(T.LblLoad, styGrey.Render(load+fmt.Sprintf(T.CPUsFmt, smp.ncpu)))
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
// минимальный размер терминала: ниже него панели схлопываются (menuH → 0)
// и раскладка ломается, поэтому показываем понятную просьбу вместо мусора.
const (
	minTermW = 64
	minTermH = 18
)

func (m *panelsModel) render() string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 32
	}
	if w < minTermW || h < minTermH {
		return m.renderTooSmall(w, h)
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
	srvBox, srvRows := panelBox(m.serversTitle(), m.serverItems(), leftW, serversH, m.focus == fpServers, m.rowCur, &m.rowOff)
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
	} else if m.selCount() > 0 || s != nil {
		// В режиме выделения справа-вверху — групповые действия, а внизу вместо
		// живой статистики список отмеченных: снимок одного сервера про группу
		// ничего не говорит.
		actH := contentH * 55 / 100
		if actH < 5 {
			actH = 5
		}
		statH := contentH - actH
		title, bottom := T.PanelActions, T.PanelStats
		var lines []string
		if m.selCount() > 0 {
			title, bottom = m.batchTitle(), T.PanelSelected
			lines = m.batchSummary(rightW - 4)
		} else {
			lines = m.statLines(s, rightW-4)
		}
		actBox, actRows := panelBox(title, m.actionItems(), rightW, actH, m.focus == fpActions, m.actionCur, &m.actionOff)
		statBox, _ := panelBox(bottom, lines, rightW, statH, false, -1, new(int))
		rightLines = append(append([]string{}, actBox...), statBox...)
		for i, r := range actRows {
			if r >= 0 {
				regions = append(regions, panelRegion{fpActions, i, leftW, w - 1, r, r})
			}
		}
	} else if name, ok := m.onFolderRow(); ok {
		// курсор на папке — показываем сводку по группе
		info := []string{
			styBold.Render(folderLabel(name)),
			"",
			styViolet.Render(T.FolderServersFmt) + "  " + strconv.Itoa(len(m.groupServers(name))),
			"",
		}
		for _, l := range wrapText(T.FolderHint, rightW-4) {
			info = append(info, styDim.Render(l))
		}
		folderBox, _ := panelBox(T.PanelFolder, info, rightW, contentH, false, -1, new(int))
		rightLines = folderBox
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
	out = append(out, styDim.Render(m.footer()))

	m.regions = regions
	if m.showHelp {
		return m.renderHelp()
	}
	if m.prompt != nil {
		return m.renderPrompt()
	}
	return strings.Join(out, "\n")
}

// renderTooSmall просит увеличить окно вместо поломанной раскладки.
func (m *panelsModel) renderTooSmall(w, h int) string {
	m.regions = nil
	msg := fmt.Sprintf(T.TermTooSmall, minTermW, minTermH, w, h)
	body := strings.Join(wrapText(msg, w-2), "\n")
	return "\n" + styYellow.Render(body) + "\n"
}

// serversTitle — заголовок списка: «СЕРВЕРЫ» + активный поиск и режим сортировки.
func (m *panelsModel) serversTitle() string {
	title := T.PanelServers
	if m.filtering || m.filter != "" {
		title += "  " + T.FilterPrompt + m.filter
		if m.filtering {
			title += "▏"
		}
	}
	if lbl := m.sortLabel(); lbl != "" {
		title += "  " + lbl
	}
	return title
}

// sortLabel — подпись текущего режима сортировки (пусто для режима «по добавлению»).
func (m *panelsModel) sortLabel() string {
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

// footer — контекстная подсказка внизу экрана.
func (m *panelsModel) footer() string {
	switch {
	case m.moving != nil:
		return T.MoveHint
	case m.filtering:
		return T.FilterHint
	case m.selCount() > 0:
		return fmt.Sprintf(T.BatchFooterFmt, m.selCount())
	}
	return T.PanelHint
}

// reorderFolder двигает папку под курсором вверх/вниз в порядке отображения.
func (m *panelsModel) reorderFolder(delta int) {
	name, ok := m.onFolderRow()
	if !ok || name == "" || m.focus != fpServers {
		return
	}
	if !m.store.MoveFolder(name, delta) {
		return
	}
	m.store.MustSave()
	m.rebuildRows()
	if idx, ok := findFolderRowIndex(m.rows, name); ok {
		m.rowCur = idx
	}
}

// findFolderRowIndex ищет строку-заголовок папки.
func findFolderRowIndex(rows []listRow, name string) (int, bool) {
	for i, r := range rows {
		if r.kind == rowFolder && r.folder == name {
			return i, true
		}
	}
	return 0, false
}

// renderHelp рисует оверлей со списком горячих клавиш панельного экрана.
func (m *panelsModel) renderHelp() string {
	rows := [][2]string{
		{"tab", T.HelpPanels},
		{"↑↓ / j k / клик", T.HelpNavigate},
		{"g / G", T.HelpJump},
		{"enter / →", T.HelpOpenActions},
		{"← / esc", T.HelpBack},
		{"c", T.HelpConnect},
		{"space", T.HelpSelect},
		{"t", T.HelpTunnels},
		{"f", T.HelpFavorite},
		{"s", T.HelpSort},
		{"/", T.HelpFilter},
		{"n", T.HelpFolderNew},
		{"m", T.HelpFolderMove},
		{"r", T.HelpFolderRename},
		{"d", T.HelpFolderDel},
		{"shift+↑↓", T.HelpFolderOrder},
		{"?", T.HelpHelp},
		{"q", T.HelpQuit},
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

// runPanelsTUI показывает главный экран и возвращает выбранный пункт (nil — выход).
func runPanelsTUI(st *Store) *menuItem {
	m := newPanelsModel(st)
	defer m.dropStatsConn() // не оставляем открытым SSH-соединение статистики
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		return nil
	}
	return m.result
}
