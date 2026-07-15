package app

// Пароли. Три хранилища, в порядке предпочтения:
//
//	macOS  — Keychain через CLI `security`
//	Linux  — Secret Service через `secret-tool` (GNOME Keyring, KWallet)
//	иначе  — JSON-файл с правами 0600
//
// Файл — не «запасной вариант на всякий случай», а честный последний рубеж:
// на headless-сервере демона ключницы нет, и записывать пароль больше некуда.
// Поэтому чтение всегда проверяет и ключницу, и файл: пароли могли лечь в файл
// до того, как в системе появилась ключница.
//
// Приятная деталь: `secret-tool store` читает секрет из STDIN, а `security
// add-generic-password` — только из argv. То есть на Linux пароль не светится
// в `ps` даже на мгновение, а на macOS светится (см. SECURITY.md).

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// secretBackend — где лежат пароли на этой машине.
type secretBackend int

const (
	backendFile secretBackend = iota
	backendKeychain
	backendSecretTool
)

// detectSecretBackend выбирает хранилище по ОС и наличию утилит.
func detectSecretBackend() secretBackend {
	if goos == "darwin" && hasBin("security") {
		return backendKeychain
	}
	if goos == "linux" && hasBin("secret-tool") {
		return backendSecretTool
	}
	return backendFile
}

// secretStore кеширует выбор: PATH за время работы не меняется.
var secretStore = sync.OnceValue(detectSecretBackend)

// secretAccount — ключ записи. Привязан к стабильному ID сервера, а не к имени,
// поэтому переименование сервера не отвязывает от него пароли.
func secretAccount(serverID, username string) string {
	return serverID + "/" + username
}

func secretsFilePath() string {
	return filepath.Join(configDir(), "secrets.json")
}

// secretSet сохраняет пароль. Если ключница отказала (нет демона, залочена),
// откатываемся на файл — иначе пользователь потерял бы только что
// сгенерированный пароль от свежего сервера.
func secretSet(serverID, username, secret string) error {
	acct := secretAccount(serverID, username)
	switch secretStore() {
	case backendKeychain:
		// -w <секрет> в argv: у macOS CLI `security` нет способа подать его через stdin
		cmd := exec.Command("security", "add-generic-password", "-U",
			"-s", appName, "-a", acct, "-w", secret)
		if err := cmd.Run(); err == nil {
			return nil
		}
	case backendSecretTool:
		cmd := exec.Command("secret-tool", "store", "--label="+appName+": "+acct,
			"service", appName, "account", acct)
		cmd.Stdin = strings.NewReader(secret) // секрет не попадает в argv
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return secretFileSet(acct, secret)
}

// secretGet читает пароль: сначала ключница, потом файл.
func secretGet(serverID, username string) string {
	acct := secretAccount(serverID, username)
	switch secretStore() {
	case backendKeychain:
		out, err := exec.Command("security", "find-generic-password",
			"-s", appName, "-a", acct, "-w").Output()
		if err == nil {
			return strings.TrimRight(string(out), "\n")
		}
	case backendSecretTool:
		out, err := exec.Command("secret-tool", "lookup",
			"service", appName, "account", acct).Output()
		if err == nil && len(out) > 0 {
			return strings.TrimRight(string(out), "\n")
		}
	}
	return secretsFileLoad()[acct]
}

// secretDelete убирает пароль отовсюду: и из ключницы, и из файла.
func secretDelete(serverID, username string) {
	acct := secretAccount(serverID, username)
	switch secretStore() {
	case backendKeychain:
		_ = exec.Command("security", "delete-generic-password",
			"-s", appName, "-a", acct).Run()
	case backendSecretTool:
		_ = exec.Command("secret-tool", "clear",
			"service", appName, "account", acct).Run()
	}
	data := secretsFileLoad()
	if _, ok := data[acct]; ok {
		delete(data, acct)
		_ = secretsFileSave(data)
	}
}

// secretFileSet кладёт пароль в JSON-файл 0600.
func secretFileSet(acct, secret string) error {
	data := secretsFileLoad()
	data[acct] = secret
	return secretsFileSave(data)
}

func secretsFileLoad() map[string]string {
	data := map[string]string{}
	raw, err := os.ReadFile(secretsFilePath())
	if err == nil {
		_ = json.Unmarshal(raw, &data)
	}
	return data
}

func secretsFileSave(data map[string]string) error {
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(secretsFilePath(), raw, 0o600)
}
