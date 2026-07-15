package app

// Инструменты: экспорт в ~/.ssh/config, публичный ключ, настройки
// (включая язык интерфейса и шаги настройки по умолчанию).

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func sshConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh", "unissh_config")
}

// writeAtomic пишет файл через временный + rename (перезапись не оставляет
// файл в полусостоянии при сбое).
func writeAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".unissh.tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// writeExportFile формирует ~/.ssh/unissh_config из списка серверов (атомарно).
func writeExportFile(st *Store) error {
	home, _ := os.UserHomeDir()
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(T.ExportHeader + "\n\n")
	for _, s := range st.Servers {
		fmt.Fprintf(&sb, "Host %s\n    HostName %s\n    Port %d\n    User %s\n",
			sshConfValue(s.Name), sshConfValue(s.Host), s.Port, sshConfValue(s.User))
		if s.Key != "" {
			fmt.Fprintf(&sb, "    IdentityFile %s\n    IdentitiesOnly yes\n", sshConfValue(s.Key))
		}
		if s.ProxyJump != "" {
			fmt.Fprintf(&sb, "    ProxyJump %s\n", sshConfValue(s.ProxyJump))
		}
		sb.WriteString("\n")
	}
	return writeAtomic(sshConfigPath(), []byte(sb.String()), 0o600)
}

// exportSSHConfig пишет Host-блоки в ~/.ssh/unissh_config и подключает его
// через Include в начале ~/.ssh/config.
func exportSSHConfig(st *Store, announce bool) {
	if err := writeExportFile(st); err != nil {
		showError(T.ToolExport, err)
		return
	}

	home, _ := os.UserHomeDir()
	mainCfg := filepath.Join(home, ".ssh", "config")
	includeLine := "Include " + sshConfigPath()
	existing := ""
	if data, err := os.ReadFile(mainCfg); err == nil {
		existing = string(data)
	}
	if !strings.Contains(existing, includeLine) {
		// бэкап существующего конфига перед правкой — на случай сбоя записи
		if existing != "" {
			_ = writeAtomic(mainCfg+".unissh-backup", []byte(existing), 0o600)
		}
		// Include должен стоять до первого Host-блока — вставляем в начало (атомарно)
		if err := writeAtomic(mainCfg, []byte(includeLine+"\n"+existing), 0o600); err != nil {
			showError(T.ToolExport, err)
			return
		}
	}
	if announce {
		showPage(T.ToolExport, []string{
			styGreen.Render(fmt.Sprintf(T.ExportedTo, sshConfigPath())),
			"",
			styDim.Render(T.ExportHint),
		})
	}
}

// maybeRefreshSSHConfig обновляет только файл экспорта, если пользователь уже
// включал его. Строку Include в основном ~/.ssh/config НЕ трогает — если её
// удалили вручную, повторно не навязываем.
func maybeRefreshSSHConfig(st *Store) {
	if fileExists(sshConfigPath()) {
		if err := writeExportFile(st); err != nil {
			fmt.Println(styRed.Render(err.Error()))
		}
	}
}

func showPubkey(st *Store) {
	pub, err := ensureKey(st.Settings.SSHKey, true)
	if err != nil {
		showError(T.TitlePubkey, err)
		return
	}
	lines := wrapText(styCyan.Render(pub), 76)
	if clipCopy(pub) {
		lines = append(lines, "", styDim.Render(T.PubkeyCopied))
	}
	showPage(T.TitlePubkey, lines)
}

func settingsMenu(st *Store) {
	for {
		stepsLabel := fmt.Sprintf("%s (%d/%d)", T.SetBootstrapSteps,
			len(userDefaultSteps(st)), len(allBootstrapSteps))
		connLabel := T.ConnectModeInline
		if connectInNewWindow(&st.Settings) {
			connLabel = T.ConnectModeWindow
		}
		themeLabel := st.Settings.Theme
		if themeLabel == "" {
			themeLabel = "blue"
		}
		field, ok := askSelectDesc(T.TitleSettings, []selOption[string]{
			opt(T.SetLanguage+langName(st.Settings.Language), "language"),
			opt(T.SetTheme+themeLabel, "theme"),
			opt(T.SetConnectMode+connLabel, "connect_mode"),
			opt(T.SetDefaultUser+st.Settings.DefaultUser, "default_user"),
			opt(T.SetSSHKey+st.Settings.SSHKey, "ssh_key"),
			opt(fmt.Sprintf(T.SetTimeout, st.Settings.StatusTimeout), "status_timeout"),
			opt(stepsLabel, "bootstrap_steps"),
			opt(T.Back, "back"),
		}, []string{
			T.DescLanguage, T.DescTheme, T.DescConnectMode,
			T.DescDefaultUser, T.DescSSHKey, T.DescTimeout,
			T.DescBootstrapSteps, "",
		})
		if !ok || field == "back" {
			return
		}
		switch field {
		case "language":
			lang, ok := askSelect(T.AskLanguage, []selOption[string]{
				opt("Русский", "ru"),
				opt("English", "en"),
			})
			if ok {
				st.Settings.Language = lang
				setLang(lang)
			}
		case "theme":
			var topts []selOption[string]
			for _, name := range themeNames {
				topts = append(topts, opt(name, name))
			}
			if theme, ok := askSelect(T.AskTheme, topts); ok {
				st.Settings.Theme = theme
				applyTheme(theme) // мгновенно применяем
			}
		case "default_user":
			if v, ok := askInput(T.AskDefaultUser, st.Settings.DefaultUser,
				validateUsername); ok && v != "" {
				st.Settings.DefaultUser = v
			}
		case "ssh_key":
			if v, ok := askInput(T.AskSSHKeyPath, st.Settings.SSHKey, nil); ok && v != "" {
				st.Settings.SSHKey = v
			}
		case "status_timeout":
			if v, ok := askInput(T.AskTimeout,
				strconv.FormatFloat(st.Settings.StatusTimeout, 'f', 1, 64),
				func(s string) error {
					f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
					if err != nil {
						return fmt.Errorf("%s", T.ErrNumber)
					}
					if f <= 0 {
						return fmt.Errorf("%s", T.ErrTimeoutRange)
					}
					return nil
				}); ok && v != "" {
				st.Settings.StatusTimeout, _ = strconv.ParseFloat(v, 64)
			}
		case "connect_mode":
			mode, ok := askSelect(T.AskConnectMode, []selOption[string]{
				opt(T.ConnectModeWindow, "window"),
				opt(T.ConnectModeInline, "inline"),
			})
			if ok {
				if mode == "inline" {
					st.Settings.ConnectMode = "inline"
				} else {
					st.Settings.ConnectMode = ""
				}
			}
		case "bootstrap_steps":
			picked, ok := askMulti(T.SetBootstrapSteps,
				bootstrapStepOptions(userDefaultSteps(st)))
			if ok {
				// пустой список означал бы «ничего» — но обязательные шаги (юзер+ключ)
				// всё равно выполняются; сохраняем ровно выбор пользователя.
				// (nil vs пустой не различаем — при полном снятии вернётся дефолт)
				st.Settings.BootstrapDefaults = picked
			}
		}
		st.MustSave()
	}
}
