package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- sshkey ---

func TestEnsureKeyGenerates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "id_test")
	pub, err := ensureKey(path, false) // не интерактивно → сгенерирует без вопроса
	if err != nil {
		t.Skipf("ssh-keygen недоступен: %v", err)
	}
	if !strings.HasPrefix(pub, "ssh-ed25519") {
		t.Fatalf("публичный ключ неожиданного вида: %q", pub)
	}
	if !fileExists(path) || !fileExists(path+".pub") {
		t.Fatal("ключи не созданы")
	}
	// повторный вызов читает существующий ключ
	pub2, err := ensureKey(path, false)
	if err != nil || pub2 != pub {
		t.Fatalf("повторное чтение ключа: %v", err)
	}
}

func TestGenPassword(t *testing.T) {
	pw := genPassword(20)
	if len(pw) != 20 {
		t.Fatalf("длина пароля = %d", len(pw))
	}
	if !strings.ContainsAny(pw, "abcdefghijkmnpqrstuvwxyz") ||
		!strings.ContainsAny(pw, "23456789") {
		t.Fatalf("пароль не содержит требуемых классов: %q", pw)
	}
	if strings.ContainsAny(pw, "0O1lI") {
		t.Errorf("пароль содержит неоднозначные символы: %q", pw)
	}
}

// --- tools (экспорт ssh-config) ---

func TestSSHConfValue(t *testing.T) {
	if sshConfValue("simple") != "simple" {
		t.Error("простое значение не должно оборачиваться")
	}
	if !strings.HasPrefix(sshConfValue("has space"), `"`) {
		t.Error("значение с пробелом должно быть в кавычках")
	}
}

func TestWriteExportFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	st := &Store{Config: Config{Servers: []*Server{
		{Name: "web", Host: "10.0.0.1", Port: 22, User: "root", Key: "~/.ssh/id"},
		{Name: "db", Host: "10.0.0.2", Port: 2222, User: "deploy", ProxyJump: "bastion"},
	}}}
	if err := writeExportFile(st); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(sshConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{"Host web", "HostName 10.0.0.1", "IdentityFile", "ProxyJump bastion", "Port 2222"} {
		if !strings.Contains(s, want) {
			t.Errorf("экспорт не содержит %q", want)
		}
	}
	// maybeRefreshSSHConfig теперь обновит существующий файл без паники
	maybeRefreshSSHConfig(st)
}

func TestWriteAtomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := writeAtomic(p, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "data" {
		t.Fatalf("writeAtomic содержимое = %q", got)
	}
}

// --- ui: таблица серверов ---

func TestServersTable(t *testing.T) {
	servers := []*Server{
		{ID: "1", Name: "web", Host: "10.0.0.1", User: "root", Port: 22, Notes: strings.Repeat("x", 40)},
		{ID: "2", Name: "db", Host: "10.0.0.2", User: "deploy", Port: 22},
	}
	statuses := map[string]probeResult{
		"1": {Online: true, Ping: 5 * time.Millisecond},
		"2": {Online: false},
	}
	out := serversTable(servers, statuses)
	if !strings.Contains(out, "web") || !strings.Contains(out, "db") {
		t.Fatalf("таблица не содержит серверов:\n%s", out)
	}
}

// --- backup label ---

func TestBackupLabel(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Servers: []*Server{{Name: "a"}, {Name: "b"}}}
	raw, _ := json.Marshal(cfg)
	name := "servers-123.json"
	os.WriteFile(filepath.Join(dir, name), raw, 0o600)
	label := backupLabel(dir, name)
	if label == "" {
		t.Fatal("метка бэкапа пуста")
	}
}

// --- connect: завершение сессии ---

func TestSessionEnded(t *testing.T) {
	sessionEnded(0)   // успех
	sessionEnded(130) // прервано пользователем
	sessionEnded(1)   // ошибка
}

// --- bootstrap: чистые хелперы ---

func TestBootstrapHelpers(t *testing.T) {
	if firstLine("one\ntwo") != "one" {
		t.Error("firstLine многострочный")
	}
	if firstLine("single") != "single" {
		t.Error("firstLine однострочный")
	}
	if bootErr("err %d", 5).Error() != "err 5" {
		t.Error("bootErr форматирование")
	}
	b := newBootstrap("h", 22, "pw", "", BootstrapOpts{NewPort: 2222, Harden: true}, "pub", "keypath")
	settings := b.sshdSettings()
	found := false
	for _, kv := range settings {
		if kv[0] == "Port" && kv[1] == "2222" {
			found = true
		}
	}
	if !found {
		t.Fatalf("sshdSettings должны включать новый порт: %v", settings)
	}
}
