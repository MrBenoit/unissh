package app

// Авто-ввод пароля при SSH-подключении через expect (штатный на macOS).
//
// Пароль передаётся expect'у через переменную окружения (для inline-режима) —
// он не попадает ни на диск, ни в argv. Для нового окна пароль встраивается во
// временный скрипт (0600), который удаляет сам себя сразу после spawn.

import (
	"os"
	"os/exec"
	"strings"
)

func hasExpect() bool {
	_, err := exec.LookPath("expect")
	return err == nil
}

// tclQuote экранирует строку для двойных кавычек Tcl.
func tclQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `[`, `\[`, `]`, `\]`, `$`, `\$`)
	return `"` + r.Replace(s) + `"`
}

// spawnLine собирает `spawn ssh "arg" "arg" …` из аргументов ssh.
func spawnLine(s *Server) string {
	var b strings.Builder
	b.WriteString("spawn ssh")
	for _, a := range sshArgs(s) {
		b.WriteString(" ")
		b.WriteString(tclQuote(a))
	}
	return b.String()
}

// expectScript формирует Tcl-скрипт. Если embedPassword=="" — пароль читается из
// $env(UNISSH_SSHPASS); иначе встраивается литералом, а скрипт удаляет сам себя.
func expectScript(s *Server, embedPassword string, selfDelete bool) string {
	var b strings.Builder
	b.WriteString("set timeout 30\n")
	if embedPassword == "" {
		b.WriteString("set pw $env(UNISSH_SSHPASS)\n")
	} else {
		b.WriteString("set pw " + tclQuote(embedPassword) + "\n")
	}
	if selfDelete {
		b.WriteString("catch {file delete [info script]}\n")
	}
	b.WriteString(spawnLine(s) + "\n")
	b.WriteString("expect {\n")
	b.WriteString("  -nocase \"password:\" { send -- \"$pw\\r\" }\n")
	b.WriteString("  -nocase \"passphrase\" { send -- \"$pw\\r\" }\n")
	b.WriteString("  timeout {}\n")
	b.WriteString("  eof { exit }\n")
	b.WriteString("}\n")
	b.WriteString("interact\n")
	b.WriteString("catch {wait} result\n")
	b.WriteString("exit [lindex $result 3]\n")
	return b.String()
}

func writeTempScript(content string) (string, error) {
	f, err := os.CreateTemp("", "unissh-*.exp")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return "", err
	}
	if _, err := f.WriteString(content); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// connectInlineAuto запускает ssh через expect в текущем окне, автоматически
// вводит пароль и передаёт управление пользователю (interact). Пароль идёт
// через env — не пишется на диск. Возвращает код выхода ssh.
func connectInlineAuto(s *Server, password string) (int, error) {
	path, err := writeTempScript(expectScript(s, "", false))
	if err != nil {
		return -1, err
	}
	defer os.Remove(path)

	cmd := exec.Command("expect", "-f", path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "UNISSH_SSHPASS="+password)
	err = cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), nil
	}
	return 0, err
}

// sshWindowCommand возвращает shell-команду для запуска в новом окне терминала
// и функцию очистки. Для серверов с паролем и наличием expect — авто-ввод через
// самоудаляющийся скрипт (0600, пароль встроен, удаляется сразу после spawn);
// cleanup нужно вызвать, только если окно НЕ удалось открыть (иначе скрипт
// удалит себя сам при запуске). Иначе — обычный ssh, cleanup — no-op.
func sshWindowCommand(s *Server, password string) (cmd string, cleanup func()) {
	if password != "" && hasExpect() {
		if path, err := writeTempScript(expectScript(s, password, true)); err == nil {
			return "expect -f " + shellQuote(path), func() { os.Remove(path) }
		}
	}
	return sshCommandString(s), func() {}
}
