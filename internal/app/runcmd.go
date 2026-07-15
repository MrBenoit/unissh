package app

// Выполнение команд на одном или нескольких серверах.

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// pickCommand выбирает команду: из сохранённых сниппетов или вводом вручную.
// После ручного ввода предлагает сохранить команду как сниппет.
func pickCommand(st *Store) (string, bool) {
	snips := st.Settings.Snippets
	if len(snips) == 0 {
		return askInput(T.AskCommand, "", nil)
	}
	opts := make([]selOption[string], 0, len(snips)+1)
	for i, sn := range snips {
		opts = append(opts, opt(sn.Name+"  —  "+sn.Cmd, strconv.Itoa(i)))
	}
	opts = append(opts, opt(T.SnippetManual, "manual"))
	choice, ok := askSelect(T.AskCommand, opts)
	if !ok {
		return "", false
	}
	if choice == "manual" {
		cmd, ok := askInput(T.AskCommand, "", nil)
		if ok && cmd != "" {
			offerSaveSnippet(st, cmd)
		}
		return cmd, ok
	}
	i, _ := strconv.Atoi(choice)
	return snips[i].Cmd, true
}

// manageSnippets — меню сниппетов: список + удаление выбранного.
func manageSnippets(st *Store) {
	for {
		if len(st.Settings.Snippets) == 0 {
			showPage(T.SnippetsTitle, []string{styYellow.Render(T.NoSnippetsYet)})
			return
		}
		opts := make([]selOption[string], 0, len(st.Settings.Snippets)+1)
		for i, sn := range st.Settings.Snippets {
			opts = append(opts, opt("🗑  "+sn.Name+"  —  "+sn.Cmd, strconv.Itoa(i)))
		}
		opts = append(opts, opt(T.Back, "back"))
		pick, ok := askSelect(T.SnippetsTitle, opts)
		if !ok || pick == "back" {
			return
		}
		i, _ := strconv.Atoi(pick)
		if i < 0 || i >= len(st.Settings.Snippets) {
			return
		}
		if yes, ok := askConfirm(fmt.Sprintf(T.AskDeleteSnippet, st.Settings.Snippets[i].Name), false); ok && yes {
			st.Settings.Snippets = append(st.Settings.Snippets[:i], st.Settings.Snippets[i+1:]...)
			st.MustSave()
			showToast(styGreen.Render(T.SnippetDeleted))
		}
	}
}

// offerSaveSnippet предлагает сохранить только что введённую команду как сниппет.
func offerSaveSnippet(st *Store, cmd string) {
	if yes, ok := askConfirm(T.SnippetSaveQ, false); !ok || !yes {
		return
	}
	name, ok := askInput(T.AskSnippetName, "", nil)
	if !ok || name == "" {
		return
	}
	st.Settings.Snippets = append(st.Settings.Snippets, Snippet{Name: name, Cmd: cmd})
	st.MustSave()
}

type runResult struct {
	name string
	code int
	out  string
}

// runOn выполняет команду на сервере и фиксирует факт запуска в журнале —
// единственная точка, через которую проходят и одиночный, и групповой запуск.
func runOn(s *Server, command string) runResult {
	r, err := openRemote(s, false)
	if err != nil {
		journalRecordRun(s, command, -1)
		return runResult{name: s.Name, code: -1, out: err.Error()}
	}
	defer r.Close()
	code, out, err := r.Run(command, "", nil, 120*time.Second)
	if err != nil {
		journalRecordRun(s, command, -1)
		return runResult{name: s.Name, code: -1, out: strings.TrimSpace(out + "\n" + err.Error())}
	}
	journalRecordRun(s, command, code)
	return runResult{name: s.Name, code: code, out: out}
}

// runResultLines рисует результат одного сервера панелью и отдаёт её строками —
// просмотрщик страницы прокручивает их сам, поэтому обрезать вывод больше не нужно.
func runResultLines(res runResult) []string {
	panel := panelGreen
	label := styGreen.Render(T.StatusOK)
	if res.code != 0 {
		panel = panelRed
		label = styRed.Render(fmt.Sprintf(T.StatusCode, res.code))
	}
	body := sanitizeRemote(strings.TrimSpace(res.out))
	if body == "" {
		body = styDim.Render(T.NoOutput)
	}
	return strings.Split(panel.Render(styBold.Render(res.name)+" · "+label+"\n"+body), "\n")
}

// actionRunSingle выполняет команду (из сниппета или введённую) на одном сервере
// и показывает вывод полноэкранной страницей.
func actionRunSingle(st *Store, s *Server) {
	command, ok := pickCommand(st)
	if !ok || command == "" {
		return
	}
	status := fmt.Sprintf(T.RunningOn, s.Name)
	runPage(s.Name, status, nil, func() []string { return runResultLines(runOn(s, command)) }, nil)
}

// actionRunMulti выполняет команду параллельно на нескольких выбранных серверах.
func actionRunMulti(st *Store) {
	if len(st.Servers) == 0 {
		showPage(T.ToolMulti, []string{styYellow.Render(T.NoServersYet)})
		return
	}
	var opts []multiOption
	for _, s := range st.Servers {
		opts = append(opts, multiOption{Label: fmt.Sprintf("%s  (%s)", s.Name, s.Addr()), Value: s.ID})
	}
	picked, ok := askMulti(T.AskPickServers, opts)
	if !ok || len(picked) == 0 {
		return
	}
	command, ok := pickCommand(st)
	if !ok || command == "" {
		return
	}
	pickedSet := map[string]bool{}
	for _, id := range picked {
		pickedSet[id] = true
	}
	var targets []*Server
	for _, s := range st.Servers {
		if pickedSet[s.ID] {
			targets = append(targets, s)
		}
	}
	runOnTargets(targets, command)
}

// runOnTargets параллельно выполняет команду на списке серверов и показывает
// результаты одной прокручиваемой страницей, в порядке списка. Вынесено
// отдельно, потому что цели приходят из двух мест: мастера «Выполнить на
// нескольких» и мультивыбора в списке.
func runOnTargets(targets []*Server, command string) {
	if len(targets) == 0 || command == "" {
		return
	}
	status := fmt.Sprintf(T.RunningOnMany, len(targets))
	runPage(T.ToolMulti, status, nil, func() []string {
		results := make([]runResult, len(targets))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 8)
		for i, s := range targets {
			wg.Add(1)
			go func(i int, s *Server) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				results[i] = runOn(s, command)
			}(i, s)
		}
		wg.Wait()
		var out []string
		for i, res := range results {
			if i > 0 {
				out = append(out, "")
			}
			out = append(out, runResultLines(res)...)
		}
		return out
	}, nil)
}
