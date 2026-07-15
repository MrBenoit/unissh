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

// spawnLine собирает `spawn <prog> "arg" "arg" …` — аргументы экранируются для Tcl.
func spawnLine(prog string, args []string) string {
	var b strings.Builder
	b.WriteString("spawn " + prog)
	for _, a := range args {
		b.WriteString(" ")
		b.WriteString(tclQuote(a))
	}
	return b.String()
}

// expectWait — общий хвост: дожидаемся программы и пробрасываем её код выхода.
const expectWait = "catch {wait} result\nexit [lindex $result 3]\n"

// expectPasswordBlock — блок ожидания приглашения. Пароль отправляется только по
// совпадению «password:»/«passphrase»: при неверном пароле повторное приглашение
// уходит пользователю, и мы не зацикливаемся. onEOF — что делать, если программа
// умерла, не спросив пароля.
func expectPasswordBlock(onEOF string) string {
	return "expect {\n" +
		"  -nocase \"password:\" { send -- \"$pw\\r\" }\n" +
		"  -nocase \"passphrase\" { send -- \"$pw\\r\" }\n" +
		"  timeout {}\n" + // вход по ключу: приглашения не будет
		"  eof { " + onEOF + " }\n" +
		"}\n"
}

// expectScript формирует Tcl-скрипт интерактивной сессии (interact отдаёт
// терминал пользователю). Если embedPassword=="" — пароль читается из
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
	b.WriteString(spawnLine("ssh", sshArgs(s)) + "\n")
	b.WriteString(expectPasswordBlock("exit"))
	b.WriteString("interact\n")
	b.WriteString(expectWait)
	return b.String()
}

// expectAutoScript собирает Tcl-скрипт для НЕинтерактивной программы (ssh -N,
// scp): пароль берётся из $UNISSH_SSHPASS, вводится при первом приглашении, а
// дальше скрипт просто ждёт завершения программы и отдаёт её код выхода.
// Пароль не пишется на диск и не попадает в argv.
//
// Отличие от expectScript: там в конце interact — сессию отдают пользователю.
// Здесь отдавать нечего, поэтому expect eof без таймаута (передача файла может
// идти часами, а `ssh -N` не завершается вовсе).
func expectAutoScript(prog string, args []string) string {
	var b strings.Builder
	b.WriteString("set timeout 30\n")
	b.WriteString("set pw $env(UNISSH_SSHPASS)\n")
	b.WriteString(spawnLine(prog, args) + "\n")
	b.WriteString(expectPasswordBlock("exit 1")) // умер до приглашения — это сбой
	b.WriteString("set timeout -1\n")
	b.WriteString("expect eof\n")
	b.WriteString(expectWait)
	return b.String()
}
func writeTempScript(content string) (string, error) {
	f, err := os.CreateTemp("", "unissh-*.exp")
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
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
	defer func() { _ = os.Remove(path) }()

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
			return "expect -f " + shellQuote(path), func() { _ = os.Remove(path) }
		}
	}
	return sshCommandString(s), func() {}
}
