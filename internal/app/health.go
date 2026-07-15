package app

// Дашборд здоровья парка: одним экраном — CPU/RAM/диск/load всех серверов.
//
// Метрики собираются параллельно (не больше healthConcurrency соединений сразу).
// Сначала TCP-проба: к офлайн-серверу мы не идём по SSH вообще, иначе один
// мёртвый VPS держал бы весь сбор до таймаута. Пароли в параллельном сборе не
// спрашиваем (openRemote(s, false)) — два tea.Program на одном tty не уживаются.

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/MrBenoit/unissh/internal/core"
)

// healthConcurrency — сколько SSH-сессий держим одновременно.
const healthConcurrency = 8

// healthCPUDelay — пауза между двумя снимками /proc/stat: CPU% считается по
// дельте счётчиков, из одного снимка его не получить.
var healthCPUDelay = 350 * time.Millisecond

// healthRow — строка дашборда: сервер и его метрики (или причина их отсутствия).
// Держим только доли: байты и топ процессов показывает «Информация о сервере»,
// дашборд отвечает на другой вопрос — «кому сейчас плохо».
type healthRow struct {
	srv        *Server
	online     bool
	ping       time.Duration
	hasData    bool
	cpu        float64 // доля 0..1
	mem        float64
	disk       float64
	load       string  // «0.10 0.20 0.30»
	loadPerCPU float64 // load за минуту на ядро: 1.0 = процессоры под завязку
	errText    string
}

// worst — самая высокая из загрузок строки. По ней сортируем дашборд, чтобы
// проблемный сервер оказался наверху и его не пришлось искать глазами.
func (h healthRow) worst() float64 {
	w := h.cpu
	for _, v := range []float64{h.mem, h.disk, h.loadPerCPU} {
		if v > w {
			w = v
		}
	}
	return w
}

// fetchHealth собирает метрики одного сервера: TCP-проба, затем два снимка
// monCmd подряд. Шов для тестов.
var fetchHealth = func(s *Server, timeout time.Duration) healthRow {
	row := healthRow{srv: s}
	pr := probe(s.Host, s.Port, timeout)
	if !pr.Online {
		return row
	}
	row.online, row.ping = true, pr.Ping

	r, err := openRemote(s, false)
	if err != nil {
		row.errText = err.Error()
		return row
	}
	defer r.Close()

	snap := func() (*monSample, error) {
		out, err := r.Sh(monCmd, "", nil, 10*time.Second)
		if err != nil {
			return nil, err
		}
		return parseMonSample(out, time.Now()), nil
	}
	first, err := snap()
	if err != nil {
		row.errText = err.Error()
		return row
	}
	time.Sleep(healthCPUDelay)
	second, err := snap()
	if err != nil {
		row.errText = err.Error()
		return row
	}
	if second.err != nil {
		row.errText = second.err.Error()
		return row
	}
	row.fill(first, second)
	return row
}

// fill переносит два снимка в строку дашборда: CPU — по дельте, остальное — из
// последнего снимка.
func (h *healthRow) fill(prev, cur *monSample) {
	h.hasData = true
	if cpu, _, _, ok := computeDeltas(prev, cur); ok {
		h.cpu = cpu
	}
	if used, total, ok := core.MemUsage(cur.memLine); ok {
		h.mem = core.Frac(used, total)
	}
	if used, total, ok := core.DiskUsage(cur.dfLine); ok {
		h.disk = core.Frac(used, total)
	}
	if l, ok := core.LoadFields(cur.load); ok {
		h.load = l
	}
	if lp, ok := core.LoadPerCPU(cur.load, cur.ncpu); ok {
		h.loadPerCPU = lp
	}
}

// collectHealth опрашивает весь парк параллельно и сортирует результат
// «самое горячее — наверх», офлайн-серверы — в конец.
func collectHealth(servers []*Server, timeout time.Duration) []healthRow {
	rows := make([]healthRow, len(servers))
	var wg sync.WaitGroup
	sem := make(chan struct{}, healthConcurrency)
	for i, s := range servers {
		wg.Add(1)
		go func(i int, s *Server) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			rows[i] = fetchHealth(s, timeout)
		}(i, s)
	}
	wg.Wait()
	sortHealth(rows)
	return rows
}

// sortHealth: живые с метриками — по убыванию нагрузки, затем живые без метрик,
// затем офлайн. Внутри групп — по имени, чтобы порядок не «плавал» между запусками.
func sortHealth(rows []healthRow) {
	rank := func(h healthRow) int {
		switch {
		case h.hasData:
			return 0
		case h.online:
			return 1
		}
		return 2
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if ri, rj := rank(rows[i]), rank(rows[j]); ri != rj {
			return ri < rj
		}
		if rows[i].hasData && rows[i].worst() != rows[j].worst() {
			return rows[i].worst() > rows[j].worst()
		}
		return rows[i].srv.Name < rows[j].srv.Name
	})
}

// healthMeterWidth — ширина шкал в таблице. Уже, чем в панели статистики:
// в строке четыре колонки-метрики, и им надо поместиться рядом.
const healthMeterWidth = 10

// healthCell рисует шкалу метрики либо прочерк, если данных нет.
func healthCell(frac float64, hasData bool) string {
	if !hasData {
		return styDim.Render("—")
	}
	return meter(frac, healthMeterWidth)
}

// healthStatus — колонка состояния: пинг, «офлайн» или текст ошибки опроса.
func healthStatus(h healthRow) string {
	switch {
	case !h.online:
		return styRed.Render("✗ " + T.StatsOffline)
	case h.errText != "":
		return styYellow.Render("! " + clip(sanitizeRemote(h.errText), 24))
	}
	return styGreen.Render("● ") + formatPing(h.ping)
}

// healthTable рендерит дашборд таблицей.
func healthTable(rows []healthRow) string {
	t := newTable(T.ColName, T.HealthColStatus, T.LblCPU, T.LblRAM, T.LblDisk, T.LblLoad)
	for _, h := range rows {
		load := styDim.Render("—")
		if h.load != "" {
			load = styGrey.Render(h.load)
		}
		t.Row(
			styBold.Render(h.srv.Name),
			healthStatus(h),
			healthCell(h.cpu, h.hasData),
			healthCell(h.mem, h.hasData),
			healthCell(h.disk, h.hasData),
			load,
		)
	}
	return t.Render()
}

// healthSummary — сводка под таблицей: сколько серверов онлайн и сколько отдали метрики.
func healthSummary(rows []healthRow) string {
	online, withData := 0, 0
	for _, h := range rows {
		if h.online {
			online++
		}
		if h.hasData {
			withData++
		}
	}
	s := styGrey.Render(fmt.Sprintf(T.HealthOnlineFmt, online, len(rows)))
	if withData < online {
		s += "  " + styYellow.Render(fmt.Sprintf(T.HealthNoDataFmt, online-withData))
	}
	return s
}

// actionHealth — пункт меню «Здоровье парка»: собирает метрики со всех серверов
// и показывает их одной таблицей. Сбор идёт внутри полноэкранной страницы, под
// спиннером: главный экран не схлопывается ни на кадр.
func actionHealth(st *Store) {
	if len(st.Servers) == 0 {
		showPage(T.HealthTitle, []string{styYellow.Render(T.NoServersYet)})
		return
	}
	servers, to := st.Servers, statusTimeout(st)
	runPage(T.HealthTitle, fmt.Sprintf(T.HealthCollectingFmt, len(servers)), nil,
		func() []string {
			rows := collectHealth(servers, to)
			return pageLines(healthTable(rows), healthSummary(rows))
		}, nil)
}
