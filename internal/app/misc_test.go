package app

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// --- sshimport ---

func TestServerExists(t *testing.T) {
	st := &Store{Config: Config{Servers: []*Server{{Host: "h", Port: 22, User: "root"}}}}
	if !serverExists(st, "h", 22, "root") {
		t.Error("существующий сервер должен находиться")
	}
	if serverExists(st, "h", 2222, "root") || serverExists(st, "h", 22, "deploy") {
		t.Error("отличающийся порт/логин — другой сервер")
	}
}

func TestUniqueName(t *testing.T) {
	st := &Store{Config: Config{Servers: []*Server{{ID: "1", Name: "web"}, {ID: "2", Name: "web-2"}}}}
	if got := uniqueName(st, "web"); got != "web-3" {
		t.Errorf("uniqueName = %q, want web-3", got)
	}
	if got := uniqueName(st, "fresh"); got != "fresh" {
		t.Errorf("свободное имя должно остаться: %q", got)
	}
	if got := uniqueName(st, ""); got != "server" {
		t.Errorf("пустая база → server, got %q", got)
	}
}

// --- secrets (файловый бэкенд) ---

func TestSecretAccount(t *testing.T) {
	if secretAccount("abc", "root") != "abc/root" {
		t.Error("secretAccount формат")
	}
}

func TestSecretsFileRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	in := map[string]string{"id/root": "s3cr3t", "id/deploy": "pw"}
	if err := secretsFileSave(in); err != nil {
		t.Fatal(err)
	}
	out := secretsFileLoad()
	if out["id/root"] != "s3cr3t" || out["id/deploy"] != "pw" {
		t.Fatalf("round-trip секретов: %v", out)
	}
	// файл должен иметь права 600
	fi, err := os.Stat(secretsFilePath())
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("права файла секретов = %v, want 600", fi.Mode().Perm())
	}
}

// --- history ---

func TestRecordConnection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	s := &Server{Name: "web-1", Host: "10.0.0.1"}
	recordConnection(nil, s) // без стора: проверяем только журнал и метку времени
	if s.LastConnected == "" {
		t.Fatal("LastConnected должен проставиться")
	}
	if _, err := time.Parse(time.RFC3339, s.LastConnected); err != nil {
		t.Fatalf("LastConnected не RFC3339: %v", err)
	}
	raw, err := os.ReadFile(historyPath())
	if err != nil {
		t.Fatalf("журнал не создан: %v", err)
	}
	var rec map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &rec); err != nil {
		t.Fatalf("запись журнала не JSON: %v", err)
	}
	if rec["server"] != "web-1" || rec["host"] != "10.0.0.1" {
		t.Errorf("запись журнала неверна: %v", rec)
	}
}
