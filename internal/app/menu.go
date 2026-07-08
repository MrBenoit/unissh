package app

// Главное меню: TUI (tui.go) выбирает пункт → здесь выполняется действие.

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

func mainMenu(st *Store) {
	for {
		res := runMainTUI(st)
		if res == nil { // выход
			fmt.Println()
			fmt.Println(styViolet.Render(T.Goodbye))
			return
		}
		switch res.kind {
		case kindServer:
			// поп-ап подтверждения (защита от случайного клика; можно отключить в настройках)
			if !confirmConnectEnabled(gSettings) || confirmConnect(res.server) {
				actionConnect(res.server)
			}
		case kindAction:
			switch res.action {
			case "add":
				addServerFlow(st)
			case "manage":
				manageServerPicker(st)
			case "manage-direct":
				serverManageMenu(st, res.server)
			case "tools":
				toolsMenu(st)
			case "info":
				actionInfo(res.server)
			case "edit":
				actionEdit(st, res.server)
			case "delete":
				actionDelete(st, res.server)
			case "reboot":
				actionReboot(res.server)
			case "bootstrap":
				rebootstrapFlow(st, res.server)
			}
		}
	}
}

// manageServerPicker — выбор сервера для управления (пункт «Управление серверами»).
func manageServerPicker(st *Store) {
	var opts []huh.Option[string]
	for _, s := range st.Servers {
		opts = append(opts, huh.NewOption(fmt.Sprintf("%s  (%s)", s.Name, s.Addr()), s.ID))
	}
	opts = append(opts, huh.NewOption(T.Back, "back"))
	target, ok := askSelect(T.AskWhichServer, opts)
	if ok && target != "back" {
		if s := st.ByID(target); s != nil {
			serverManageMenu(st, s)
		}
	}
}
