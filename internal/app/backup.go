package app

// Восстановление конфига из снапшота (undo для ошибочного изменения/удаления).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
		showPage(T.BackupTitle, []string{styYellow.Render(T.BackupNothing)})
		return
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files))) // новые сверху

	var opts []selOption[string]
	for _, name := range files {
		opts = append(opts, opt(backupLabel(dir, name), name))
	}
	opts = append(opts, opt(T.Back, ""))

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
		showError(T.BackupTitle, err)
		return
	}
	if err := os.WriteFile(st.path, data, 0o600); err != nil {
		showError(T.BackupTitle, err)
		return
	}
	// перечитываем конфиг в память — иначе восстановление не видно и затрётся при
	// следующем сохранении (стор держится по указателю)
	var cfg Config
	if json.Unmarshal(data, &cfg) == nil {
		st.Config = cfg
		st.readOnly = false // конфиг снова валиден — запись разрешена
		setLang(st.Settings.Language)
		applyTheme(st.Settings.Theme)
	}
	maybeRefreshSSHConfig(st)
	showToast(styGreen.Render(T.BackupRestored))
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
