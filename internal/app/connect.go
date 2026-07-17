package app

// Интерактивное подключение: обычный ssh-клиент с нашим known_hosts.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// controlDir — каталог для master-сокетов мультиплексирования.
func controlDir() string { return filepath.Join(configDir(), "cm") }

// usesPassword сообщает, что вход на сервер парольный (ключа нет или явно
// выбран пароль). Дешёвая проверка по конфигу — без похода в ключницу.
func usesPassword(s *Server) bool { return s.Auth == "password" || s.Key == "" }

// sshMuxArgs включает мультиплексирование: первое подключение поднимает
// master-сокет, все следующие переиспользуют его — без TCP, обмена ключами и
// аутентификации, то есть мгновенно. Работает и для scp.
//
// Только для входа ПО КЛЮЧУ. При парольном входе ssh запускается через expect,
// который ждёт приглашения «password:»; на переиспользованном соединении пароля
// не спросят, и expect провисит до своего таймаута, не отдавая сессию.
//
// nil (без ускорения), если каталог не создался или путь не влезает в сокет:
// это всего лишь оптимизация, ломать из-за неё подключение нельзя.
func sshMuxArgs(s *Server) []string {
	if usesPassword(s) {
		return nil
	}
	// %C — хеш (локальный хост, хост, порт, пользователь), 40 hex-символов.
	// Именно хеш, а не имя хоста: путь к unix-сокету ограничен по длине.
	path := filepath.Join(controlDir(), "%C")
	if len(path)-len("%C")+muxHashLen > maxUnixSocketPath {
		return nil // длинный $HOME: сокет не создать, ssh бы упал совсем
	}
	if err := os.MkdirAll(controlDir(), 0o700); err != nil {
		return nil
	}
	return []string{
		"-o", "ControlMaster=auto",
		"-o", "ControlPath=" + path,
		"-o", "ControlPersist=10m",
	}
}

const (
	// muxHashLen — во что разворачивается %C: SHA1 в hex.
	muxHashLen = 40
	// maxUnixSocketPath — лимит длины пути unix-сокета: 104 на macOS, 108 на
	// Linux. Берём меньший, с запасом. Превышение = ssh не подключится вообще.
	maxUnixSocketPath = 100
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
	args = append(args, sshMuxArgs(s)...)
	args = append(args, "-o", "ServerAliveInterval=30", "-o", "ServerAliveCountMax=3")
	return append(args, s.User+"@"+s.Host)
}

// serverPassword возвращает пароль сервера из ключницы, если вход парольный.
// Иначе — пустая строка.
func serverPassword(s *Server) string {
	if usesPassword(s) {
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
