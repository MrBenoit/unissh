package app

// Полноэкранный просмотрщик результата действия (bubbletea, alt-screen).
//
// Зачем: без него «Здоровье парка», «Информация», «Журнал» и вывод команд
// печатались обычным fmt.Println уже ПОСЛЕ выхода из alt-screen — большое окно
// схлопывалось, и результат оказывался в куске обычного терминала. Теперь такие
// экраны открываются в том же полноэкранном режиме, что и главный, а Esc
// возвращает назад.
//
// Долгий сбор данных (SSH-опрос парка, info) выполняется прямо здесь, под
// спиннером внутри alt-screen, — поэтому мигания между экранами тоже нет.

import (
	"errors"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// pageKey — дополнительное действие просмотрщика (кроме «назад»). Нажатие key
// закрывает страницу и возвращает id вызывающему.
type pageKey struct{ key, label, id string }

// pageBack — что возвращает просмотрщик, если пользователь просто ушёл назад.
const pageBack = ""

// pageSpinInterval — период кадра спиннера загрузки.
const pageSpinInterval = 100 * time.Millisecond

// toastTTL — сколько живёт короткое сообщение («✓ Сервер удалён»), если его не
// закрыть клавишей.
const toastTTL = 1100 * time.Millisecond

type pageModel struct {
	title  string
	lines  []string
	keys   []pageKey
	load   func() []string // необязательный сбор данных под спиннером
	status string          // подпись под спиннером («Опрашиваю серверы…»)
	ttl    time.Duration   // >0 — страница закрывается сама (режим тоста)
	// auto = страница закрывается, как только load завершился: чистый спиннер
	// без результата (runSpinner).
	auto bool

	loading  bool
	canceled bool // Ctrl+C во время сбора данных
	frame    int
	off      int
	width    int
	height   int
	result   string
}

// pageCloseMsg — время тоста вышло.
type pageCloseMsg struct{}

// pageLoadedMsg — данные собраны, показываем их.
type pageLoadedMsg []string

// pageTickMsg — кадр спиннера.
type pageTickMsg struct{}

func pageTick() tea.Cmd {
	return tea.Tick(pageSpinInterval, func(time.Time) tea.Msg { return pageTickMsg{} })
}

func (m *pageModel) Init() tea.Cmd {
	if m.ttl > 0 {
		return tea.Tick(m.ttl, func(time.Time) tea.Msg { return pageCloseMsg{} })
	}
	if m.load == nil {
		return nil
	}
	m.loading = true
	load := m.load
	return tea.Batch(pageTick(), func() tea.Msg { return pageLoadedMsg(load()) })
}

// visibleH — сколько строк содержимого влезает в панель (без рамки, заголовка,
// пустой строки и подсказки внизу).
func (m *pageModel) visibleH() int {
	h := m.height - 6
	if h < 1 {
		h = 1
	}
	return h
}

// scroll двигает окно просмотра; окончательную обрезку делает panelBox.
func (m *pageModel) scroll(delta int) {
	m.off += delta
	if m.off < 0 {
		m.off = 0
	}
}

func (m *pageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case pageTickMsg:
		if m.loading {
			m.frame++
			return m, pageTick()
		}
	case pageCloseMsg:
		return m, tea.Quit
	case pageLoadedMsg:
		m.loading = false
		m.lines = msg
		if m.auto { // спиннер без результата — показывать больше нечего
			return m, tea.Quit
		}
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.scroll(-1)
		case tea.MouseButtonWheelDown:
			m.scroll(1)
		}
	case tea.KeyMsg:
		if m.loading { // во время сбора данных выйти можно только по Ctrl+C
			if msg.String() == "ctrl+c" {
				m.canceled = true
				return m, tea.Quit
			}
			return m, nil
		}
		if m.ttl > 0 { // тост закрывается любой клавишей, не дожидаясь таймера
			return m, tea.Quit
		}
		switch msg.String() {
		case "up", "k":
			m.scroll(-1)
		case "down", "j":
			m.scroll(1)
		case "pgup":
			m.scroll(-m.visibleH())
		case "pgdown":
			m.scroll(m.visibleH())
		case "home", "g":
			m.off = 0
		case "end", "G":
			m.off = len(m.lines)
		case "esc", "q", "enter", "ctrl+c", "backspace", "left":
			return m, tea.Quit
		default:
			for _, k := range m.keys {
				if msg.String() == k.key {
					m.result = k.id
					return m, tea.Quit
				}
			}
		}
	}
	return m, nil
}

// footer — подсказка внизу: прокрутка, дополнительные действия, выход.
func (m *pageModel) footer() string {
	parts := []string{T.PageScroll}
	for _, k := range m.keys {
		parts = append(parts, k.key+" — "+k.label)
	}
	parts = append(parts, T.PageBack)
	return styDim.Render(strings.Join(parts, "  ·  "))
}

func (m *pageModel) View() string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 32
	}
	if m.loading {
		return centered(w, h, styCyan.Render(spinnerFrames[m.frame%len(spinnerFrames)])+"  "+styBold.Render(m.status))
	}
	if m.ttl > 0 { // тост: сообщение по центру, без рамки и подсказок
		return centered(w, h, strings.Join(m.lines, "\n"))
	}
	box, _ := panelBox(m.title, m.lines, w, h-2, true, -1, &m.off)
	return strings.Join(box, "\n") + "\n" + m.footer()
}

// centered ставит блок текста в центр экрана.
func centered(w, h int, body string) string {
	return lipgloss.NewStyle().Width(w).Height(h).
		Align(lipgloss.Center, lipgloss.Center).Render(body)
}

// runPage показывает полноэкранную страницу и возвращает id нажатого
// дополнительного действия (или pageBack, если пользователь просто вышел).
// Если load != nil, данные собираются внутри страницы под спиннером, а status
// подписывает ожидание. Объявлено как var — тесты подменяют его, чтобы
// прогонять действия без TTY.
var runPage = func(title, status string, lines []string, load func() []string, keys []pageKey) string {
	m := &pageModel{title: title, status: status, lines: lines, load: load, keys: keys}
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		return pageBack
	}
	return m.result
}

// showPage — короткая форма для готового содержимого без доп. действий.
func showPage(title string, lines []string) {
	runPage(title, "", lines, nil, nil)
}

// runToast показывает короткое сообщение по центру экрана и закрывает его сам.
// Объявлено как var — тесты подменяют его вместе с runPage.
var runToast = func(lines []string) {
	m := &pageModel{lines: lines, ttl: toastTTL}
	_, _ = tea.NewProgram(m, tea.WithAltScreen()).Run()
}

// showToast — «✓ сделано»/«ничего нет»: сообщение показывается поверх, гаснет
// само и не роняет полноэкранный режим, как это делал fmt.Println + time.Sleep.
func showToast(msg string) { runToast(strings.Split(msg, "\n")) }

// runSpinner крутит полноэкранный спиннер, пока выполняется fn, и закрывается
// сам. Объявлено как var — тесты подменяют его вместе с runPage.
//
// Важно: fn стартует в отдельной горутине, а Ctrl+C завершает программу немедленно.
// Поэтому мы ВСЕГДА дожидаемся fn через канал done — иначе была бы гонка за
// результатами, которые fn пишет уже после возврата. Прерывание отдаётся как
// ошибка, чтобы вызыватель не печатал неполный результат.
var runSpinner = func(status string, fn func() error) error {
	var fnErr error
	done := make(chan struct{})
	m := &pageModel{status: status, auto: true, load: func() []string {
		fnErr = fn()
		close(done)
		return nil
	}}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	<-done // дожидаемся fn в любом случае (в т.ч. после Ctrl+C)
	if err != nil {
		return err
	}
	if m.canceled {
		return errors.New(T.Canceled)
	}
	return fnErr
}

// errLines — текст ошибки строками для страницы: одна и та же обёртка нужна и
// showError, и загрузчикам, которые возвращают ошибку вместо содержимого.
func errLines(err error) []string {
	return wrapText(styRed.Render(err.Error()), pageErrWidth)
}

// pageErrWidth — ширина переноса текста ошибки.
const pageErrWidth = 76

// showError показывает ошибку действия отдельной страницей: раньше она уходила
// в fmt.Println и мгновенно затиралась перерисовкой главного экрана.
func showError(title string, err error) {
	showPage(title, errLines(err))
}

// pageLines разбивает готовый блок (таблицу, панель) на строки для просмотрщика.
func pageLines(blocks ...string) []string {
	var out []string
	for i, b := range blocks {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, strings.Split(b, "\n")...)
	}
	return out
}
