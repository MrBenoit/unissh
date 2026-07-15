package app

// Точка входа и CLI:
//   unissh          — интерактивное меню
//   unissh <имя>    — быстрое подключение к серверу
//   unissh list     — таблица серверов
//   unissh add      — добавить сервер
//   unissh version  — версия

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// version — версия сборки; задаётся из main.version (ldflags) через Run().
var version = "dev"

func usage() {
	fmt.Println(gradientLine("unissh " + version))
	fmt.Println(styGrey.Render(T.UsageTagline))
	fmt.Println()
	fmt.Println(styBold.Render(T.UsageTitle))
	fmt.Println("  unissh              " + T.UsageMenu)
	fmt.Println("  unissh <name>       " + T.UsageConnect)
	fmt.Println("  unissh list         " + T.UsageList)
	fmt.Println("  unissh add          " + T.UsageAdd)
	fmt.Println("  unissh version      " + T.UsageVersion)
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func cmdList(st *Store) {
	if len(st.Servers) == 0 {
		fmt.Println(styYellow.Render(T.NoServersCLI))
		return
	}
	statuses := probeAll(st.Servers, statusTimeout(st))
	fmt.Println(serversTable(st.Servers, statuses))
}

func quickConnect(st *Store, query string) {
	s := st.Find(query)
	if s == nil {
		var names []string
		for _, srv := range st.Servers {
			names = append(names, srv.Name)
		}
		list := strings.Join(names, ", ")
		if list == "" {
			list = "—"
		}
		fmt.Println(styRed.Render(fmt.Sprintf(T.ErrNotFound, query)) + " " +
			styDim.Render(fmt.Sprintf(T.HaveList, list)))
		os.Exit(1)
	}
	actionConnect(st, s)
}

// Run — точка входа приложения (вызывается из корневого main.go). ver — версия
// сборки, подставленная линкером.
func Run(ver string) {
	if ver != "" {
		version = ver
	}
	st := loadStore()
	setLang(st.Settings.Language)
	applyTheme(st.Settings.Theme)
	if st.readOnly {
		// loadStore уже напечатал ошибку разбора и путь к .broken. Показываем,
		// что запись запрещена, и ждём Enter — иначе alt-screen сотрёт сообщение.
		fmt.Println(styRed.Render(T.ErrConfigReadOnly))
		fmt.Println(styYellow.Render(T.ConfigBrokenHint))
		if isTTY() {
			pressEnter()
		}
	}
	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}
	switch arg {
	case "", "menu":
		if !isTTY() {
			cmdList(st)
			return
		}
		mainMenu(st)
	case "list", "ls":
		cmdList(st)
	case "add":
		if !isTTY() {
			fmt.Println(styRed.Render(T.ErrAddTTY))
			os.Exit(1)
		}
		addServerFlow(st)
	case "version", "--version", "-v":
		fmt.Println("unissh " + version)
	case "help", "--help", "-h":
		usage()
	default:
		quickConnect(st, arg)
	}
}
