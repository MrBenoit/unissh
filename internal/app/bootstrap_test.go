package app

import (
	"strings"
	"testing"
	"time"
)

// bootWithServer поднимает in-process SSH-сервер и подключает к нему bootstrap
// через шов dialSSH. handler получает полную команду и возвращает вывод/код.
func bootWithServer(t *testing.T, h execHandler) *bootstrap {
	t.Helper()
	host, port := startTestSSHServer(t, true, h)
	// dialSSH подменять не нужно: сервер настоящий, просто внутри процесса
	b := newBootstrap(host, port, "rootpw", "", BootstrapOpts{Username: "deploy"}, "ssh-ed25519 AAAA", "/tmp/key")
	if err := b.connect(nil); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() {
		if b.r != nil {
			b.r.Close()
		}
	})
	return b
}

// TestBootstrapConnect: шаг подключения работает поверх реального SSH-транспорта.
func TestBootstrapConnect(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	b := bootWithServer(t, func(cmd string) (string, int) { return "", 0 })
	if b.r == nil {
		t.Fatal("после connect должно быть открытое соединение")
	}
}

// TestBootstrapDetectDebian: /etc/os-release с ubuntu → apt + группа sudo.
func TestBootstrapDetectDebian(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	b := bootWithServer(t, func(cmd string) (string, int) {
		if strings.Contains(cmd, "os-release") {
			return "ID=ubuntu\nID_LIKE=debian\n", 0
		}
		return "", 0
	})
	if err := b.detect(nil); err != nil {
		t.Fatal(err)
	}
	if b.family != "debian" || b.pm != "apt" || b.adminGroup != "sudo" {
		t.Fatalf("detect: family=%q pm=%q group=%q", b.family, b.pm, b.adminGroup)
	}
}

// TestBootstrapDetectRHEL: rocky + наличие dnf → rhel/dnf/wheel.
func TestBootstrapDetectRHEL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	b := bootWithServer(t, func(cmd string) (string, int) {
		switch {
		case strings.Contains(cmd, "os-release"):
			return "ID=\"rocky\"\nID_LIKE=\"rhel centos fedora\"\n", 0
		case strings.Contains(cmd, "command -v dnf"):
			return "/usr/bin/dnf\n", 0
		}
		return "", 0
	})
	if err := b.detect(nil); err != nil {
		t.Fatal(err)
	}
	if b.family != "rhel" || b.pm != "dnf" || b.adminGroup != "wheel" {
		t.Fatalf("detect: family=%q pm=%q group=%q", b.family, b.pm, b.adminGroup)
	}
}

// TestBootstrapDetectUnknown: незнакомый дистрибутив → предупреждение, не ошибка.
func TestBootstrapDetectUnknown(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	b := bootWithServer(t, func(cmd string) (string, int) { return "ID=plan9\n", 0 })
	if err := b.detect(nil); err != nil {
		t.Fatal(err)
	}
	if b.family != "unknown" {
		t.Fatalf("family = %q", b.family)
	}
	if len(b.warnings) != 1 {
		t.Fatalf("ожидалось предупреждение о неизвестном дистрибутиве: %v", b.warnings)
	}
}

// TestBootstrapCreateUser: пароль генерируется, уходит в chpasswd через stdin
// (не через argv) и сразу отдаётся в onPassword — до возможного падения дальше.
func TestBootstrapCreateUser(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var seen []string
	b := bootWithServer(t, func(cmd string) (string, int) {
		seen = append(seen, cmd)
		if strings.Contains(cmd, "getent group") && strings.Contains(cmd, "echo") {
			return "sudo\n", 0
		}
		return "", 0
	})

	var gotUser, gotPW string
	b.onPassword = func(u, pw string) { gotUser, gotPW = u, pw }

	if err := b.createUser(nil); err != nil {
		t.Fatalf("createUser: %v", err)
	}
	if gotUser != "deploy" || gotPW == "" {
		t.Fatalf("onPassword не вызван корректно: %q/%q", gotUser, gotPW)
	}
	if b.result.Password != gotPW {
		t.Fatal("сгенерированный пароль должен попасть в результат")
	}
	// пароль не должен светиться в командной строке
	for _, c := range seen {
		if strings.Contains(c, gotPW) {
			t.Fatalf("пароль просочился в argv команды: %q", c)
		}
	}
	joined := strings.Join(seen, "\n")
	for _, want := range []string{"useradd", "usermod -aG", "chpasswd"} {
		if !strings.Contains(joined, want) {
			t.Errorf("не выполнена команда %q", want)
		}
	}
}

// TestBootstrapSshdSettingsHarden: харденинг добавляет директивы, новый порт — Port.
func TestBootstrapSshdSettings(t *testing.T) {
	b := newBootstrap("h", 22, "pw", "", BootstrapOpts{NewPort: 2222, Harden: true}, "pub", "key")
	got := map[string]string{}
	for _, kv := range b.sshdSettings() {
		got[kv[0]] = kv[1]
	}
	if got["Port"] != "2222" {
		t.Errorf("новый порт: %v", got)
	}
	if got["PasswordAuthentication"] != "no" {
		t.Errorf("харденинг должен отключать вход по паролю: %v", got)
	}
	// без опций — пусто (нечего менять)
	plain := newBootstrap("h", 22, "pw", "", BootstrapOpts{}, "pub", "key")
	if len(plain.sshdSettings()) != 0 {
		t.Errorf("без опций sshd не трогаем: %v", plain.sshdSettings())
	}
	// тот же порт менять не нужно
	same := newBootstrap("h", 2222, "pw", "", BootstrapOpts{NewPort: 2222}, "pub", "key")
	if len(same.sshdSettings()) != 0 {
		t.Errorf("порт не изменился — директиву не пишем: %v", same.sshdSettings())
	}
}

// TestBootstrapVerifyUserLoginFails: если вход по ключу не работает, шаг возвращает
// ошибку — именно на этом строится защита от локаута.
func TestBootstrapVerifyUserLoginFails(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())

	orig := dialSSH
	origDelay := verifyLoginRetryDelay
	verifyLoginRetryDelay = 0 // не ждём в тесте
	defer func() { dialSSH, verifyLoginRetryDelay = orig, origDelay }()
	// подключение пользователя всегда падает
	dialSSH = func(host string, port int, user, password, keyPath, proxyJump string, to time.Duration) (*Remote, error) {
		return nil, errNoAuth
	}
	b := newBootstrap("h", 22, "pw", "", BootstrapOpts{Username: "deploy"}, "pub", "key")
	if err := b.verifyUserLogin(22); err == nil {
		t.Fatal("вход по ключу не работает — verifyUserLogin обязан вернуть ошибку")
	}
}

// TestBootstrapShSanitizes: вывод сервера с ANSI не должен доезжать до терминала.
func TestBootstrapShSanitizesErrorTail(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	b := bootWithServer(t, func(cmd string) (string, int) {
		return "\x1b[31mBOOM\x1b[0m\n", 1 // ненулевой код → хвост попадёт в ошибку
	})
	_, err := b.sh("false", nil, 5*time.Second)
	if err == nil {
		t.Fatal("ненулевой код должен давать ошибку")
	}
	if strings.Contains(err.Error(), "\x1b") {
		t.Fatalf("ANSI не вычищен из текста ошибки: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "BOOM") {
		t.Fatalf("полезный текст потерян: %q", err.Error())
	}
}
