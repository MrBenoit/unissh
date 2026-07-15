package app

// Генерация и чтение локального SSH-ключа для unissh.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ensureKey гарантирует наличие пары ключей и возвращает публичный ключ.
// При отсутствии спрашивает разрешение на генерацию (если interactive).
func ensureKey(path string, interactive bool) (string, error) {
	priv := expandPath(path)
	pub := priv + ".pub"
	if fileExists(priv) && fileExists(pub) {
		data, err := os.ReadFile(pub)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	if interactive {
		ok, answered := askConfirm(fmt.Sprintf(T.AskGenKey, path), true)
		if !answered || !ok {
			return "", fmt.Errorf(T.ErrKeyMissing, path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(priv), 0o700); err != nil {
		return "", err
	}
	out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-f", priv,
		"-N", "", "-C", appName+"@mac").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ssh-keygen: %s", strings.TrimSpace(string(out)))
	}
	fmt.Println(styGreen.Render("✓ ") + T.KeyCreated + styBold.Render(path))
	data, err := os.ReadFile(pub)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
