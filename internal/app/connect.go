package app

// Интерактивное подключение: обычный ssh-клиент с нашим known_hosts.

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// sshCommonArgs — опции, общие для ssh и scp: наш known_hosts (TOFU), ключ и
// бастион. Порт сюда НЕ входит: у ssh это «-p», у scp — «-P».
func sshCommonArgs(s *Server) []string {
	args := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=" + knownHostsPath(),
	}
	if s.Key != "" {
		args = append(args, "-i", expandPath(s.Key), "-o", "IdentitiesOnly=yes")
	}
	if s.ProxyJump != "" {
		args = append(args, "-J", s.ProxyJump) // подключение через бастион
	}
	return args
}

// sshArgs — аргументы системного ssh; цель (user@host) всегда последняя.
func sshArgs(s *Server) []string {
	args := []string{"-p", strconv.Itoa(s.Port)}
	args = append(args, sshCommonArgs(s)...)
	args = append(args, "-o", "ServerAliveInterval=30", "-o", "ServerAliveCountMax=3")
	return append(args, s.User+"@"+s.Host)
}

// serverPassword возвращает пароль сервера из Keychain, если вход парольный
// (у сервера нет ключа или auth=password). Иначе — пустая строка.
func serverPassword(s *Server) string {
	if s.Auth == "password" || s.Key == "" {
		return secretGet(s.ID, s.User)
	}
	return ""
}

// sshConnErrCode — код выхода ssh, означающий, что подключиться не удалось.
const sshConnErrCode = 255

// actionConnect подключается к серверу (новое окно или inline, с авто-вводом
// пароля). Факт подключения фиксируется только ПОСЛЕ успешной попытки —
// иначе «последний вход» проставлялся бы даже при недоступном сервере.
func actionConnect(st *Store, s *Server) {
	password := serverPassword(s) // для авто-ввода, если вход по паролю

	// Режим нового окна: запускаем во внешнем терминале (с авто-вводом пароля,
	// если он есть) и сразу возвращаемся в меню. При неудаче — откат к inline.
	if connectInNewWindow(&st.Settings) {
		cmd, cleanup := sshWindowCommand(s, password)
		if err := openConnectionWindowCmd(s, cmd); err == nil {
			recordConnection(st, s)
			showToast(styGreen.Render(T.OpenedNewWindow) + styGrey.Render("  "+s.Addr()))
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
		if code != sshConnErrCode {
			recordConnection(st, s)
		}
		sessionEnded(code)
		return
	}

	// обычный ssh (ключ, либо нет пароля/expect). Если пароль есть, но авто-ввод
	// недоступен — кладём в буфер (с автоочисткой) и подсказываем.
	if password != "" {
		if clipCopyEphemeral(password) {
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
	if code != sshConnErrCode {
		recordConnection(st, s)
	}
	sessionEnded(code)
}

// sessionEnded показывает исход интерактивной сессии. Сама сессия шла в обычном
// терминале (ssh забирает tty целиком), а сюда мы возвращаемся уже к экранам.
func sessionEnded(code int) {
	sty := styGreen
	if code != 0 {
		sty = styYellow
	}
	showToast(sty.Render(fmt.Sprintf(T.SessionEnded, code)))
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
	if clipCopy(cmd) {
		showToast(styGreen.Render(T.CopiedCmd) + "\n" + styBold.Render(cmd))
	} else {
		showToast(styBold.Render(cmd))
	}
}
