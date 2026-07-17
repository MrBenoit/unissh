package app

// Передача файлов по SCP: отправка на сервер и скачивание с него.
//
// Используем системный scp, а не отдельную SFTP-библиотеку: он уже уважает наши
// known_hosts, ключи и ProxyJump, умеет рекурсию и сам рисует прогресс.
// Внимание: у scp порт задаётся ЗАГЛАВНОЙ -P (у ssh — строчной), поэтому свои
// аргументы, а не sshArgs.
//
// Пароль (если вход парольный) вводится через expect и не попадает ни на диск,
// ни в argv. Удалённый путь проходит через core.RemotePathArg: пробел или «;»
// в нём не должны стать командой на сервере.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MrBenoit/unissh/internal/core"
)

// scpArgs — опции scp для сервера. Порт задаётся ЗАГЛАВНОЙ -P: строчная -p у
// scp означает «сохранить времена файла». Остальное общее с ssh.
func scpArgs(s *Server) []string {
	args := []string{"-r", "-P", strconv.Itoa(s.Port)} // -r безвреден и на обычном файле
	args = append(args, sshCommonArgs(s)...)
	// переиспользуем master-сокет, если он уже поднят интерактивным подключением
	return append(args, sshMuxArgs(s)...)
}

// scpRemote собирает «user@host:путь» с экранированием удалённого пути.
func scpRemote(s *Server, path string) string {
	return s.User + "@" + s.Host + ":" + core.RemotePathArg(path)
}

// uploadArgs — аргументы scp для отправки локального файла на сервер.
func uploadArgs(s *Server, local, remote string) []string {
	return append(scpArgs(s), core.LocalPathArg(expandPath(local)), scpRemote(s, remote))
}

// downloadArgs — аргументы scp для скачивания файла с сервера.
func downloadArgs(s *Server, remote, local string) []string {
	return append(scpArgs(s), scpRemote(s, remote), core.LocalPathArg(expandPath(local)))
}

// scpRun запускает scp, подставляя пароль через expect, если он есть. Прогресс
// scp пишет прямо в терминал — своей крутилки поверх него не рисуем.
// Возвращает код выхода scp. Шов для тестов.
var scpRun = func(s *Server, args []string) (int, error) {
	password := serverPassword(s)
	var cmd *exec.Cmd
	if password != "" && hasExpect() {
		path, err := writeTempScript(expectAutoScript("scp", args))
		if err != nil {
			return -1, err
		}
		defer func() { _ = os.Remove(path) }()
		cmd = exec.Command("expect", "-f", path)
		cmd.Env = append(os.Environ(), "UNISSH_SSHPASS="+password)
	} else {
		cmd = exec.Command("scp", args...)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	return 0, err
}

// validateLocalExists — путь должен существовать: отправлять нечего иначе.
func validateLocalExists(p string) error {
	p = strings.TrimSpace(p)
	if p == "" {
		return errors.New(T.ErrPathEmpty)
	}
	if _, err := os.Stat(expandPath(p)); err != nil {
		return errors.New(T.ErrPathMissing)
	}
	return nil
}

// validateLocalDir — каталог назначения для скачивания должен существовать.
func validateLocalDir(p string) error {
	p = strings.TrimSpace(p)
	if p == "" {
		return errors.New(T.ErrPathEmpty)
	}
	fi, err := os.Stat(expandPath(p))
	if err != nil || !fi.IsDir() {
		return errors.New(T.ErrDirMissing)
	}
	return nil
}

// validateRemotePath — непустой путь без переводов строк (их не должно быть в
// argv scp) и без подстановок шелла, которые мы не можем показать пользователю.
func validateRemotePath(p string) error {
	p = strings.TrimSpace(p)
	if p == "" {
		return errors.New(T.ErrPathEmpty)
	}
	if strings.ContainsAny(p, "\n\r") {
		return errors.New(T.ErrPathNewline)
	}
	return nil
}

// defaultDownloadDir — куда класть скачанное по умолчанию: ~/Downloads, если он
// есть, иначе текущий каталог.
func defaultDownloadDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		d := filepath.Join(home, "Downloads")
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			return d
		}
	}
	return "."
}

// actionTransfer — меню передачи файлов для сервера.
func actionTransfer(s *Server) {
	for {
		pick, ok := askSelectFull(T.TransferTitle, styGrey.Render(s.Addr()),
			[]selOption[string]{
				opt(T.TransferUpload, "up"),
				opt(T.TransferDownload, "down"),
				opt(T.Back, "back"),
			},
			[]string{T.TransferUploadDesc, T.TransferDownloadDesc, ""})
		if !ok || pick == "back" {
			return
		}
		if pick == "up" {
			uploadFlow(s)
		} else {
			downloadFlow(s)
		}
	}
}

// uploadFlow спрашивает локальный и удалённый пути и копирует файл на сервер.
func uploadFlow(s *Server) {
	local, ok := askInput(T.AskLocalPath, "", validateLocalExists)
	if !ok {
		return
	}
	remote, ok := askInput(T.AskRemoteDest, "~/", validateRemotePath)
	if !ok {
		return
	}
	local, remote = strings.TrimSpace(local), strings.TrimSpace(remote)
	if runTransfer(s, uploadArgs(s, local, remote),
		fmt.Sprintf(T.TransferUpFmt, filepath.Base(local), s.Name)) {
		journalRecordTransfer(s, "up", local, remote)
	}
}

// downloadFlow спрашивает удалённый и локальный пути и копирует файл с сервера.
func downloadFlow(s *Server) {
	remote, ok := askInput(T.AskRemotePath, "", validateRemotePath)
	if !ok {
		return
	}
	local, ok := askInput(T.AskLocalDest, defaultDownloadDir(), validateLocalDir)
	if !ok {
		return
	}
	remote, local = strings.TrimSpace(remote), strings.TrimSpace(local)
	if runTransfer(s, downloadArgs(s, remote, local),
		fmt.Sprintf(T.TransferDownFmt, path.Base(remote), s.Name)) {
		journalRecordTransfer(s, "down", remote, local)
	}
}

// runTransfer выполняет scp и показывает исход. Сам scp — единственное место,
// которое намеренно рисует в обычном терминале: его прогресс-бар нужен человеку
// и требует настоящего tty. Исход возвращается уже полноэкранной страницей.
// Возвращает true, если передача прошла успешно (только тогда её пишут в журнал).
func runTransfer(s *Server, args []string, title string) bool {
	fmt.Println()
	fmt.Println(styViolet.Render(title))
	code, err := scpRun(s, args)
	switch {
	case err != nil:
		showError(T.TransferTitle, err)
	case code != 0:
		showPage(T.TransferTitle, []string{styRed.Render(fmt.Sprintf(T.TransferFailedFmt, code))})
	default:
		showToast(styGreen.Render(T.TransferDone))
		return true
	}
	return false
}
