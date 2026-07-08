package app

// Открытие SSH-подключения в новом окне терминала (macOS).
// Поддержка Warp (launch configuration), iTerm2 и Terminal.app (AppleScript);
// для неизвестного терминала — запасной вариант через Terminal.app.

import (
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
	if err := os.MkdirAll(dir, 0o755); err != nil {
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
	if err := os.WriteFile(filepath.Join(dir, cfgName+".yaml"), []byte(yaml), 0o644); err != nil {
		return err
	}
	return exec.Command("open", "warp://launch/"+cfgName).Run()
}

func yamlSingle(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// openConnectionWindowCmd открывает переданную команду в новом окне подходящего
// терминала. Возвращает ошибку, если ни один способ не сработал.
func openConnectionWindowCmd(s *Server, cmd string) error {
	switch os.Getenv("TERM_PROGRAM") {
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
