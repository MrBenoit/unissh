package app

// Интерактивное подключение: обычный ssh-клиент с нашим known_hosts.

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func sshArgs(s *Server) []string {
	args := []string{
		"-p", strconv.Itoa(s.Port),
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=" + knownHostsPath(),
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
	}
	if s.Key != "" {
		args = append(args, "-i", expandPath(s.Key), "-o", "IdentitiesOnly=yes")
	}
	if s.ProxyJump != "" {
		args = append(args, "-J", s.ProxyJump) // подключение через бастион
	}
	args = append(args, s.User+"@"+s.Host)
	return args
}

// confirmConnect показывает экран-подтверждение с кнопками [Подключиться]/[Отмена].
// Возвращает true, если пользователь подтвердил.
func confirmConnect(s *Server) bool {
	header := styGrey.Render(s.Addr())
	idx := runSelect(fmt.Sprintf(T.ConfirmConnectTitle, s.Name), header,
		[]string{T.ConnectBtn, T.CancelBtn}, 0)
	return idx == 0
}

// actionConnect подключается к серверу (новое окно или inline, с авто-вводом
// пароля) и фиксирует факт подключения в журнале.
func actionConnect(s *Server) {
	recordConnection(s) // обновляем «последний вход» и журнал

	// пароль (если аутентификация по паролю) — для авто-ввода
	password := ""
	if s.Auth == "password" || s.Key == "" {
		password = secretGet(s.ID, s.User)
	}

	// Режим нового окна: запускаем во внешнем терминале (с авто-вводом пароля,
	// если он есть) и сразу возвращаемся в меню. При неудаче — откат к inline.
	if connectInNewWindow(gSettings) {
		cmd, cleanup := sshWindowCommand(s, password)
		if err := openConnectionWindowCmd(s, cmd); err == nil {
			fmt.Println(styGreen.Render(T.OpenedNewWindow) + styGrey.Render("  "+s.Addr()))
			time.Sleep(900 * time.Millisecond)
			return
		} else {
			cleanup() // окно не открылось — убираем временный скрипт с паролем
			fmt.Println(styYellow.Render(fmt.Sprintf(T.ErrOpenWindow, err)))
		}
	}

	fmt.Println()
	fmt.Println(styCyan.Render(T.Connecting) + styBold.Render(s.Name) +
		styGrey.Render("  "+s.Addr()))
	fmt.Println()

	// Авто-ввод пароля через expect (если пароль есть и expect доступен) —
	// пользователь сразу получает готовую сессию, без ручной вставки.
	if password != "" && hasExpect() {
		code, err := connectInlineAuto(s, password)
		if err != nil {
			fmt.Println(styRed.Render("ssh: " + err.Error()))
			time.Sleep(1200 * time.Millisecond)
			return
		}
		sessionEnded(code)
		return
	}

	// обычный ssh (ключ, либо нет пароля/expect). Если пароль есть, но авто-ввод
	// недоступен — копируем в буфер и подсказываем, чтобы было что вставить.
	if password != "" {
		if pbcopy(password) {
			fmt.Println(styYellow.Render(T.PwCopiedClipboard))
		}
	}
	cmd := exec.Command("ssh", sshArgs(s)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		fmt.Println(styRed.Render("ssh: " + err.Error()))
		time.Sleep(1200 * time.Millisecond)
		return
	}
	sessionEnded(code)
}

func sessionEnded(code int) {
	sty := styGreen
	if code != 0 {
		sty = styYellow
	}
	fmt.Println(sty.Render(fmt.Sprintf(T.SessionEnded, code)))
	time.Sleep(600 * time.Millisecond)
}

func actionCopySSHCmd(s *Server) {
	parts := []string{"ssh", "-p", strconv.Itoa(s.Port)}
	if s.Key != "" {
		parts = append(parts, "-i", s.Key)
	}
	parts = append(parts, s.User+"@"+s.Host)
	cmd := ""
	for i, p := range parts {
		if i > 0 {
			cmd += " "
		}
		cmd += p
	}
	if pbcopy(cmd) {
		fmt.Println(styGreen.Render(T.CopiedCmd) + styBold.Render(cmd))
	} else {
		fmt.Println(styBold.Render(cmd))
	}
	time.Sleep(1200 * time.Millisecond)
}
