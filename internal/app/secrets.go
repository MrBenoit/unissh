package app

// Пароли: macOS Keychain через `security`; на других ОС — файл с правами 600.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func useKeychain() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := exec.LookPath("security")
	return err == nil
}

func secretAccount(serverID, username string) string {
	return serverID + "/" + username
}

func secretsFilePath() string {
	return filepath.Join(configDir(), "secrets.json")
}

func secretSet(serverID, username, secret string) error {
	if useKeychain() {
		cmd := exec.Command("security", "add-generic-password", "-U",
			"-s", appName, "-a", secretAccount(serverID, username), "-w", secret)
		return cmd.Run()
	}
	data := secretsFileLoad()
	data[secretAccount(serverID, username)] = secret
	return secretsFileSave(data)
}

func secretGet(serverID, username string) string {
	if useKeychain() {
		out, err := exec.Command("security", "find-generic-password",
			"-s", appName, "-a", secretAccount(serverID, username), "-w").Output()
		if err != nil {
			return ""
		}
		return strings.TrimRight(string(out), "\n")
	}
	return secretsFileLoad()[secretAccount(serverID, username)]
}

func secretDelete(serverID, username string) {
	if useKeychain() {
		_ = exec.Command("security", "delete-generic-password",
			"-s", appName, "-a", secretAccount(serverID, username)).Run()
		return
	}
	data := secretsFileLoad()
	delete(data, secretAccount(serverID, username))
	_ = secretsFileSave(data)
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
