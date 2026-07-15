package app

// Открытие SSH-подключения в новом окне терминала.
//
//	macOS — Warp (launch configuration), iTerm2 и Terminal.app (AppleScript);
//	        для неизвестного терминала запасной вариант — Terminal.app, он есть всегда.
//	Linux — $TERMINAL, затем известные эмуляторы из linuxTerminals по порядку.
//
// На Linux гарантированного терминала нет. Поэтому canOpenNewWindow() заранее
// сообщает, есть ли вообще чем открывать: без этой проверки каждое подключение
// сначала пыталось бы открыть окно, падало и печатало предупреждение перед
// откатом в inline.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// sshCommandString собирает готовую shell-команду `ssh …` с экранированием
// каждого аргумента (для запуска во внешнем терминале).
func sshCommandString(s *Server) string {
	parts := append([]string{"ssh"}, sshArgs(s)...)
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = shellQuote(p)
	}
	return strings.Join(quoted, " ")
}

func appleScriptEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func runOsascript(script string) error {
	return exec.Command("osascript", "-e", script).Run()
}

// openTerminalApp — новое окно Terminal.app с командой (гарантированно есть на macOS).
func openTerminalApp(cmd string) error {
	script := fmt.Sprintf(`tell application "Terminal"
	activate
	do script "%s"
end tell`, appleScriptEscape(cmd))
	return runOsascript(script)
}

// openITerm — новое окно iTerm2 с командой.
func openITerm(cmd string) error {
	script := fmt.Sprintf(`tell application "iTerm"
	activate
	create window with default profile command "%s"
end tell`, appleScriptEscape(cmd))
	return runOsascript(script)
}

// openWarp — новое окно Warp через launch configuration + URI-схему warp://launch.
// Warp сканирует ~/.warp/launch_configurations/*.yaml; открываем по имени файла.
func openWarp(title, cmd string) error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".warp", "launch_configurations")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	const cfgName = "unissh"
	// exec — literal block scalar (|): не нужно экранировать кавычки в команде.
	// cwd — абсолютный путь (Warp не разворачивает «~»).
	yaml := "---\n" +
		"name: " + cfgName + "\n" +
		"windows:\n" +
		"  - tabs:\n" +
		"      - title: " + yamlSingle(title) + "\n" +
		"        layout:\n" +
		"          cwd: " + yamlSingle(home) + "\n" +
		"          commands:\n" +
		"            - exec: |\n" +
		"                " + cmd + "\n"
	if err := os.WriteFile(filepath.Join(dir, cfgName+".yaml"), []byte(yaml), 0o600); err != nil {
		return err
	}
	return exec.Command("open", "warp://launch/"+cfgName).Run()
}

func yamlSingle(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// linuxTerminals — эмуляторы, которым можно передать команду как argv
// (без вложенного шелл-квотинга). Порядок: сначала то, что человек скорее
// всего поставил осознанно, потом то, что приходит с окружением рабочего стола.
//
// Терминалы, у которых `-e` принимает ОДНУ строку, а не argv (xfce4-terminal
// без -x, tilix), сюда не входят: их квотинг зависит от версии, и тихо
// сломанная команда хуже отсутствия окна.
var linuxTerminals = []struct {
	bin  string
	args []string // флаги перед `sh -c <cmd>`
}{
	{"kitty", nil},
	{"foot", nil},
	{"ghostty", []string{"-e"}},
	{"wezterm", []string{"start", "--"}},
	{"alacritty", []string{"-e"}},
	{"gnome-terminal", []string{"--"}},
	{"konsole", []string{"-e"}},
	{"xfce4-terminal", []string{"-x"}}, // -x, в отличие от -e, принимает argv
	{"x-terminal-emulator", []string{"-e"}},
	{"xterm", []string{"-e"}},
}

// linuxTerminalArgv собирает argv для запуска cmd в новом окне: имя бинарника,
// его флаги и `sh -c <cmd>`. Возвращает nil, если ни один терминал не найден.
// $TERMINAL уважается первым — пользователь имеет право настоять.
func linuxTerminalArgv(cmd string) []string {
	if t := getenv("TERMINAL"); t != "" && hasBin(t) {
		return []string{t, "-e", "sh", "-c", cmd}
	}
	for _, t := range linuxTerminals {
		if hasBin(t.bin) {
			return append(append([]string{t.bin}, t.args...), "sh", "-c", cmd)
		}
	}
	return nil
}

// canOpenNewWindow сообщает, есть ли в системе чем открыть новое окно.
// На macOS Terminal.app есть всегда; на Linux — как повезёт.
func canOpenNewWindow() bool {
	if goos == "darwin" {
		return true
	}
	return linuxTerminalArgv("true") != nil
}

// openLinuxWindow открывает команду в новом окне эмулятора терминала.
func openLinuxWindow(cmd string) error {
	argv := linuxTerminalArgv(cmd)
	if argv == nil {
		return errors.New(T.ErrNoTerminal)
	}
	// #nosec G204 — бинарник из белого списка или явно заданный пользователем $TERMINAL
	return exec.Command(argv[0], argv[1:]...).Start()
}

// openConnectionWindowCmd открывает переданную команду в новом окне подходящего
// терминала. Возвращает ошибку, если ни один способ не сработал.
func openConnectionWindowCmd(s *Server, cmd string) error {
	if goos != "darwin" {
		return openLinuxWindow(cmd)
	}
	switch getenv("TERM_PROGRAM") {
	case "iTerm.app":
		if err := openITerm(cmd); err == nil {
			return nil
		}
	case "WarpTerminal":
		if err := openWarp(s.Name, cmd); err == nil {
			return nil
		}
	case "Apple_Terminal":
		return openTerminalApp(cmd)
	}
	// запасной вариант — Terminal.app есть всегда
	return openTerminalApp(cmd)
}
