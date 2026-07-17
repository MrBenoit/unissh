package app

import (
	"os"
	"strings"
	"testing"
)

// shortHome подменяет $HOME на КОРОТКИЙ временный каталог.
// t.TempDir() на macOS даёт /var/folders/zz/…/001 — почти 90 байт, и путь к
// master-сокету перестаёт влезать в лимит unix-сокета: мультиплексирование
// честно выключается, и проверять становится нечего. В жизни $HOME короткий.
func shortHome(t *testing.T) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "unissh-t")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	t.Setenv("HOME", dir)
}

// keySrv — сервер со входом по ключу (мультиплексирование ему положено).
func keySrv() *Server {
	return &Server{ID: "k1", Name: "web", Host: "1.2.3.4", Port: 22,
		User: "deploy", Key: "/tmp/k", Auth: "key"}
}

// pwSrv — сервер с парольным входом (мультиплексирование ему ЗАПРЕЩЕНО).
func pwSrv() *Server {
	return &Server{ID: "p1", Name: "old", Host: "5.6.7.8", Port: 22,
		User: "root", Auth: "password"}
}

// TestUsesPassword: определение парольного входа. Ключа нет — значит пароль,
// даже если auth не проставлен (так бывает у импортированных из ~/.ssh/config).
func TestUsesPassword(t *testing.T) {
	if usesPassword(keySrv()) {
		t.Error("вход по ключу — не парольный")
	}
	if !usesPassword(pwSrv()) {
		t.Error("auth=password — парольный")
	}
	if !usesPassword(&Server{Auth: "key", Key: ""}) {
		t.Error("без ключа вход может быть только по паролю")
	}
}

// TestMuxOnForKeyAuth: у ключевого сервера появляются опции мультиплексирования.
func TestMuxOnForKeyAuth(t *testing.T) {
	shortHome(t)
	joined := strings.Join(sshArgs(keySrv()), " ")
	for _, want := range []string{"ControlMaster=auto", "ControlPath=", "ControlPersist=10m"} {
		if !strings.Contains(joined, want) {
			t.Errorf("нет %q в %q", want, joined)
		}
	}
	// цель обязана остаться последней, иначе ssh примет её за команду
	args := sshArgs(keySrv())
	if args[len(args)-1] != "deploy@1.2.3.4" {
		t.Errorf("цель не последняя: %v", args)
	}
}

// TestMuxOffForPasswordAuth — главная защита. При парольном входе ssh идёт через
// expect, который ждёт «password:». На переиспользованном соединении пароля не
// спросят, и expect провисит до таймаута, не отдав сессию пользователю.
func TestMuxOffForPasswordAuth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	joined := strings.Join(sshArgs(pwSrv()), " ")
	if strings.Contains(joined, "ControlMaster") || strings.Contains(joined, "ControlPath") {
		t.Fatalf("парольному серверу мультиплексирование противопоказано: %q", joined)
	}
	// и в expect-скрипте его тоже быть не должно
	if sc := expectScript(pwSrv(), "", false); strings.Contains(sc, "ControlMaster") {
		t.Errorf("ControlMaster просочился в expect-скрипт:\n%s", sc)
	}
}

// TestMuxSocketDir: каталог сокетов создаётся с правами 0700 — через него идёт
// доступ к живым SSH-сессиям, чужим там делать нечего.
func TestMuxSocketDir(t *testing.T) {
	shortHome(t)
	_ = sshArgs(keySrv())

	fi, err := os.Stat(controlDir())
	if err != nil {
		t.Fatalf("каталог сокетов не создан: %v", err)
	}
	if fi.Mode().Perm() != 0o700 {
		t.Errorf("права каталога сокетов = %v, ожидались 700", fi.Mode().Perm())
	}
}

// TestMuxPathIsHashNotHostname: длинное имя хоста не должно попадать в путь
// сокета — там стоит %C (хеш), иначе путь бы распухал от каждого хоста.
func TestMuxPathIsHashNotHostname(t *testing.T) {
	shortHome(t)
	long := keySrv()
	long.Host = "very-long-hostname.production.eu-central-1.example-company.com"

	joined := strings.Join(sshArgs(long), " ")
	if strings.Contains(joined, "cm/very-long-hostname") {
		t.Errorf("имя хоста просочилось в путь сокета: %q", joined)
	}
	if !strings.Contains(joined, "%C") {
		t.Errorf("ожидался хеш %%C в ControlPath: %q", joined)
	}
}

// TestMuxOffWhenPathTooLong — защита от реальной поломки: если путь к сокету не
// влезает в лимит unix-сокета, ssh не «работает медленнее», а НЕ ПОДКЛЮЧАЕТСЯ
// вовсе. Лучше молча обойтись без ускорения.
func TestMuxOffWhenPathTooLong(t *testing.T) {
	deep := t.TempDir() + strings.Repeat("/deeply-nested-home-directory", 4)
	t.Setenv("HOME", deep)

	joined := strings.Join(sshArgs(keySrv()), " ")
	if strings.Contains(joined, "ControlMaster") {
		t.Fatalf("при слишком длинном пути мультиплексирование надо выключать, а не ломать ssh: %q", joined)
	}
	// подключение при этом обязано остаться рабочим
	if !strings.Contains(joined, "deploy@1.2.3.4") {
		t.Errorf("цель потерялась: %q", joined)
	}
}

// TestMuxFitsTypicalHome: на обычных домашних каталогах путь обязан влезать —
// иначе защита от длинного пути вырубила бы ускорение вообще всем.
// Считаем арифметику, а не лезем в файловую систему: тест не должен ничего
// создавать в настоящем $HOME.
func TestMuxFitsTypicalHome(t *testing.T) {
	for _, home := range []string{
		"/Users/germanzaharov",             // macOS
		"/home/deploy",                     // Linux
		"/Users/some.long.corporate.login", // длинный, но реальный
	} {
		path := home + "/.config/unissh/cm/"
		if got := len(path) + muxHashLen; got > maxUnixSocketPath {
			t.Errorf("$HOME=%s → путь сокета %d байт, лимит %d — ускорение выключится зря",
				home, got, maxUnixSocketPath)
		}
	}
}

// TestTunnelKeepsOwnConnection: туннель НЕ должен ехать по общему master-сокету.
// Иначе смерть master по ControlPersist унесёт проброс, а наш kill по группе
// процессов до него не достучится.
func TestTunnelKeepsOwnConnection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	args := tunnelSSHArgs(keySrv(), tunLocal)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "ControlPath=none") {
		t.Fatalf("туннелю нужно своё соединение (ControlPath=none): %q", joined)
	}
	// У ssh при повторе опции выигрывает ПЕРВОЕ значение — значит none обязан
	// стоять раньше, чем ControlPath из sshArgs.
	iNone := strings.Index(joined, "ControlPath=none")
	iMux := strings.Index(joined, "ControlPath="+controlDir())
	if iMux >= 0 && iNone > iMux {
		t.Errorf("ControlPath=none должен идти ПЕРВЫМ, иначе выиграет мультиплексирование: %q", joined)
	}
}

// TestScpReusesMaster: scp после подключения должен идти по готовому сокету.
func TestScpReusesMaster(t *testing.T) {
	shortHome(t)
	joined := strings.Join(scpArgs(keySrv()), " ")
	if !strings.Contains(joined, "ControlMaster=auto") {
		t.Errorf("scp тоже выигрывает от мультиплексирования: %q", joined)
	}
	// у scp порт — ЗАГЛАВНАЯ -P, мультиплексирование это не должно сломать
	if !strings.Contains(joined, "-P 22") {
		t.Errorf("потерялся порт: %q", joined)
	}
}
