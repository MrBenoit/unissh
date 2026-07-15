package app

// Главное меню: панельный TUI (tui2.go) выбирает пункт → здесь выполняется действие.
// Диалоги (ввод, выбор) запускаются именно тут: второй tea.Program поверх
// работающего запустить нельзя, поэтому панель выходит и возвращает menuItem.

import "fmt"

func mainMenu(st *Store) {
	// Туннели — фоновые процессы, привязанные к жизни unissh: без этого они
	// пережили бы приложение и остались висеть без единого способа их найти.
	defer func() {
		if n := tunnels.StopAll(); n > 0 {
			fmt.Println(styGrey.Render(fmt.Sprintf(T.TunnelsStoppedFmt, n)))
		}
	}()
	for {
		res := runPanelsTUI(st)
		if res == nil { // выход
			fmt.Println()
			fmt.Println(styViolet.Render(T.Goodbye))
			return
		}
		dispatchMenu(st, res)
	}
}

// dispatchMenu выполняет действие, выбранное на главном экране.
func dispatchMenu(st *Store, res *menuItem) {
	switch res.action {
	case "add":
		addServerFlow(st)
	case "health":
		actionHealth(st)
	case "multi":
		actionRunMulti(st)
	case "import":
		importFromSSHConfig(st)
	case "export":
		exportSSHConfig(st, true)
	case "backups":
		restoreBackupMenu(st)
	case "snippets":
		manageSnippets(st)
	case "journal":
		actionJournal()
	case "pubkey":
		showPubkey(st)
	case "settings":
		settingsMenu(st)
	case "connect":
		// подключение — осознанный выбор кнопки/клавиши «c», попап не нужен
		actionConnect(st, res.server)
	case "info":
		actionInfo(res.server)
	case "run":
		actionRunSingle(st, res.server)
	case "password":
		actionShowPasswords(res.server)
	case "install_key":
		actionInstallKey(st, res.server)
	case "copy":
		actionCopySSHCmd(res.server)
	case "forget":
		actionForgetHostKey(res.server)
	case "reboot":
		actionReboot(res.server)
	case "edit":
		actionEdit(st, res.server)
	case "delete":
		actionDelete(st, res.server)
	case "bootstrap":
		rebootstrapFlow(st, res.server)
	case "tunnels":
		actionTunnels(st, res.server)
	case "transfer":
		actionTransfer(res.server)
	case "batch_run":
		batchRun(st, res.servers)
	case "batch_move":
		batchMove(st, res.servers)
	case "batch_delete":
		batchDelete(st, res.servers)
	}
}
