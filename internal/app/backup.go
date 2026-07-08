package app

// Восстановление конфига из снапшота (undo для ошибочного изменения/удаления).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
)

// restoreBackupMenu показывает список снапшотов и восстанавливает выбранный.
func restoreBackupMenu(st *Store) {
	dir := backupsDir()
	entries, _ := os.ReadDir(dir)
	var files []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "servers-") && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		fmt.Println(styYellow.Render(T.BackupNothing))
		pressEnter()
		return
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files))) // новые сверху

	var opts []huh.Option[string]
	for _, name := range files {
		opts = append(opts, huh.NewOption(backupLabel(dir, name), name))
	}
	opts = append(opts, huh.NewOption(T.Back, ""))

	pick, ok := askSelect(T.BackupTitle, opts)
	if !ok || pick == "" {
		return
	}
	yes, ok := askConfirm(fmt.Sprintf(T.BackupConfirm, pick), false)
	if !ok || !yes {
		return
	}
	data, err := os.ReadFile(filepath.Join(dir, pick))
	if err != nil {
		fmt.Println(styRed.Render(err.Error()))
		pressEnter()
		return
	}
	if err := os.WriteFile(st.path, data, 0o600); err != nil {
		fmt.Println(styRed.Render(err.Error()))
		pressEnter()
		return
	}
	// перечитываем конфиг в память — иначе восстановление не видно и затрётся при
	// следующем сохранении (gStore == st держится по указателю)
	var cfg Config
	if json.Unmarshal(data, &cfg) == nil {
		st.Config = cfg
		setLang(st.Settings.Language)
		applyTheme(st.Settings.Theme)
	}
	maybeRefreshSSHConfig(st)
	fmt.Println(styGreen.Render(T.BackupRestored))
	pressEnter()
}

// backupLabel формирует читаемую подпись снапшота: сколько серверов и когда.
func backupLabel(dir, name string) string {
	full := filepath.Join(dir, name)
	count := 0
	if data, err := os.ReadFile(full); err == nil {
		var cfg Config
		if json.Unmarshal(data, &cfg) == nil {
			count = len(cfg.Servers)
		}
	}
	when := ""
	if fi, err := os.Stat(full); err == nil {
		when = fi.ModTime().Format("2006-01-02 15:04")
	}
	return fmt.Sprintf(T.BackupServersFmt, when, count)
}
