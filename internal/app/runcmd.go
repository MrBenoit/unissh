package app

// Выполнение команд на одном или нескольких серверах.

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
)

// pickCommand выбирает команду: из сохранённых сниппетов или вводом вручную.
// После ручного ввода предлагает сохранить команду как сниппет.
func pickCommand() (string, bool) {
	snips := gStore.Settings.Snippets
	if len(snips) == 0 {
		return askInput(T.AskCommand, "", nil)
	}
	opts := make([]huh.Option[string], 0, len(snips)+1)
	for i, sn := range snips {
		opts = append(opts, huh.NewOption(sn.Name+"  —  "+sn.Cmd, strconv.Itoa(i)))
	}
	opts = append(opts, huh.NewOption(T.SnippetManual, "manual"))
	choice, ok := askSelect(T.AskCommand, opts)
	if !ok {
		return "", false
	}
	if choice == "manual" {
		cmd, ok := askInput(T.AskCommand, "", nil)
		if ok && cmd != "" {
			offerSaveSnippet(cmd)
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
			fmt.Println(styYellow.Render(T.NoSnippetsYet))
			time.Sleep(time.Second)
			return
		}
		opts := make([]huh.Option[string], 0, len(st.Settings.Snippets)+1)
		for i, sn := range st.Settings.Snippets {
			opts = append(opts, huh.NewOption("🗑  "+sn.Name+"  —  "+sn.Cmd, strconv.Itoa(i)))
		}
		opts = append(opts, huh.NewOption(T.Back, "back"))
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
			fmt.Println(styGreen.Render(T.SnippetDeleted))
			time.Sleep(700 * time.Millisecond)
		}
	}
}

// offerSaveSnippet предлагает сохранить только что введённую команду как сниппет.
func offerSaveSnippet(cmd string) {
	if yes, ok := askConfirm(T.SnippetSaveQ, false); !ok || !yes {
		return
	}
	name, ok := askInput(T.AskSnippetName, "", nil)
	if !ok || name == "" {
		return
	}
	gStore.Settings.Snippets = append(gStore.Settings.Snippets, Snippet{Name: name, Cmd: cmd})
	gStore.MustSave()
}

// withSpinner крутит спиннер, пока выполняется fn.
// Важно: huh/spinner запускает Action в отдельной горутине, а Ctrl+C (SIGINT)
// прерывает Run немедленно, НЕ дожидаясь Action. Поэтому мы всегда дожидаемся
// завершения fn через канал done — иначе была бы гонка данных с результатами,
// которые fn пишет уже после возврата. Возвращаем ошибку прерывания, чтобы
// вызыватель мог не печатать неполный результат.
func withSpinner(title string, fn func() error) error {
	var err error
	done := make(chan struct{})
	serr := spinner.New().Title(" " + title).
		Action(func() { err = fn(); close(done) }).Run()
	<-done // дожидаемся fn в любом случае (в т.ч. после SIGINT)
	if serr != nil {
		return serr
	}
	return err
}

type runResult struct {
	name string
	code int
	out  string
}

func runOn(s *Server, command string) runResult {
	r, err := openRemote(s, false)
	if err != nil {
		return runResult{name: s.Name, code: -1, out: err.Error()}
	}
	defer r.Close()
	code, out, err := r.Run(command, "", nil, 120*time.Second)
	if err != nil {
		return runResult{name: s.Name, code: -1, out: strings.TrimSpace(out + "\n" + err.Error())}
	}
	return runResult{name: s.Name, code: code, out: out}
}

func printRunResult(res runResult) {
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
	if len(body) > 4000 {
		body = "…" + body[len(body)-4000:]
	}
	fmt.Println(panel.Render(styBold.Render(res.name) + " · " + label + "\n" + body))
}

// actionRunSingle выполняет команду (из сниппета или введённую) на одном сервере.
func actionRunSingle(s *Server) {
	command, ok := pickCommand()
	if !ok || command == "" {
		return
	}
	var res runResult
	if err := withSpinner(fmt.Sprintf(T.RunningOn, s.Name), func() error {
		res = runOn(s, command)
		return nil
	}); err != nil {
		return // прервано (Ctrl+C)
	}
	printRunResult(res)
	pressEnter()
}

// actionRunMulti выполняет команду параллельно на нескольких выбранных серверах.
func actionRunMulti(st *Store) {
	if len(st.Servers) == 0 {
		fmt.Println(styYellow.Render(T.NoServersYet))
		time.Sleep(time.Second)
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
	command, ok := pickCommand()
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

	results := make([]runResult, len(targets))
	if err := withSpinner(fmt.Sprintf(T.RunningOnMany, len(targets)), func() error {
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
		return nil
	}); err != nil {
		return // прервано (Ctrl+C)
	}
	fmt.Println()
	for _, res := range results {
		printRunResult(res)
	}
	pressEnter()
}
