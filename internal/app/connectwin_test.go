package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWarpTOMLString: команда от sshCommandString содержит одинарные кавычки
// (shellQuote), поэтому TOML basic string обязан экранировать только двойные
// кавычки и бэкслеши, а одинарные оставлять как есть.
func TestWarpTOMLString(t *testing.T) {
	// одинарные кавычки проходят насквозь
	if got := warpTOMLString(`ssh -i '/tmp/my key' root@h`); got != `"ssh -i '/tmp/my key' root@h"` {
		t.Errorf("одинарные кавычки не должны экранироваться: %s", got)
	}
	// двойные кавычки и бэкслеш — экранируются
	if got := warpTOMLString(`a"b\c`); got != `"a\"b\\c"` {
		t.Errorf("двойные/бэкслеш: %s", got)
	}
	// перевод строки не должен разорвать TOML-строку
	if got := warpTOMLString("a\nb"); got != `"a\nb"` {
		t.Errorf("перевод строки: %s", got)
	}
}

// TestOpenWarpWritesTabConfig — главная проверка перехода «окно → вкладка»:
// пишется Tab Config (TOML в tab_configs), а НЕ launch config (YAML в
// launch_configurations, который открывал окно).
func TestOpenWarpWritesTabConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// open в тестовом окружении отсутствует — конфиг всё равно должен записаться
	_ = openWarp("web-1", `ssh -p 22 -i '/tmp/k' deploy@1.2.3.4`)

	toml := filepath.Join(home, ".warp", "tab_configs", "unissh.toml")
	data, err := os.ReadFile(toml)
	if err != nil {
		t.Fatalf("tab config не записан: %v", err)
	}
	s := string(data)

	// не должно быть признаков старого launch config (он открывал окно)
	if strings.Contains(s, "windows:") {
		t.Error("в конфиге остался блок windows: — это снова откроет окно, а не вкладку")
	}
	if _, err := os.Stat(filepath.Join(home, ".warp", "launch_configurations")); err == nil {
		t.Error("не должны больше создавать launch_configurations")
	}

	for _, want := range []string{"[[panes]]", `type = "terminal"`, `title = "web-1"`, "deploy@1.2.3.4"} {
		if !strings.Contains(s, want) {
			t.Errorf("в TOML нет %q:\n%s", want, s)
		}
	}
	// каталог с сокет… нет, с конфигом — 0700, там команды подключения
	fi, _ := os.Stat(filepath.Join(home, ".warp", "tab_configs"))
	if fi.Mode().Perm() != 0o700 {
		t.Errorf("права каталога = %v, ожидались 700", fi.Mode().Perm())
	}
}

// TestOpenWarpQuotesCommand: реальная команда с одинарными кавычками должна
// попасть в TOML как валидная строка, не поломав файл.
func TestOpenWarpQuotesCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := `expect -f '/tmp/unissh-abc.exp'` // путь парольного входа
	_ = openWarp("db", cmd)

	data, _ := os.ReadFile(filepath.Join(home, ".warp", "tab_configs", "unissh.toml"))
	if !strings.Contains(string(data), `commands = ["expect -f '/tmp/unissh-abc.exp'"]`) {
		t.Errorf("команда с кавычками записана неверно:\n%s", data)
	}
}
