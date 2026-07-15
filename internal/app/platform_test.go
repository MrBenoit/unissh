package app

import (
	"errors"
	"strings"
	"testing"
)

// fakeEnv подменяет швы lookPath/getenv/goos на время теста.
// present — множество «установленных» бинарников.
func fakeEnv(t *testing.T, os string, env map[string]string, present ...string) {
	t.Helper()
	origLook, origEnv, origOS := lookPath, getenv, goos
	t.Cleanup(func() { lookPath, getenv, goos = origLook, origEnv, origOS })

	set := map[string]bool{}
	for _, b := range present {
		set[b] = true
	}
	goos = os
	lookPath = func(name string) (string, error) {
		if set[name] {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
	getenv = func(k string) string { return env[k] }
}

// --- буфер обмена ---

// TestClipToolMacOS: на macOS всегда pbcopy/pbpaste.
func TestClipToolMacOS(t *testing.T) {
	fakeEnv(t, "darwin", nil, "pbcopy", "pbpaste")
	tool := detectClipTool()
	if tool == nil || tool.name != "pbcopy" {
		t.Fatalf("tool = %+v", tool)
	}
	if tool.paste != "pbpaste" || !strings.Contains(tool.clear, "pbcopy") {
		t.Errorf("команды: %+v", tool)
	}
}

// TestClipToolWaylandPreferred: под Wayland выбираем wl-clipboard, даже если
// xclip установлен — через Xwayland он не видит буфер нативных приложений.
func TestClipToolWaylandPreferred(t *testing.T) {
	fakeEnv(t, "linux", map[string]string{"WAYLAND_DISPLAY": "wayland-0"},
		"wl-copy", "wl-paste", "xclip")
	tool := detectClipTool()
	if tool == nil || tool.name != "wl-copy" {
		t.Fatalf("под Wayland ожидался wl-copy, получено %+v", tool)
	}
	// без --no-newline wl-paste дописывает \n и хеш при автоочистке не сойдётся
	if !strings.Contains(tool.paste, "--no-newline") {
		t.Errorf("wl-paste без --no-newline: %q", tool.paste)
	}
	if tool.clear != "wl-copy --clear" {
		t.Errorf("очистка Wayland-буфера: %q", tool.clear)
	}
}

// TestClipToolX11: без WAYLAND_DISPLAY — xclip, при его отсутствии xsel.
func TestClipToolX11(t *testing.T) {
	fakeEnv(t, "linux", nil, "xclip", "xsel")
	if tool := detectClipTool(); tool == nil || tool.name != "xclip" {
		t.Fatalf("ожидался xclip: %+v", tool)
	}
	fakeEnv(t, "linux", nil, "xsel")
	tool := detectClipTool()
	if tool == nil || tool.name != "xsel" {
		t.Fatalf("ожидался xsel: %+v", tool)
	}
	if !strings.Contains(tool.clear, "--clear") {
		t.Errorf("xsel умеет чистить сам: %q", tool.clear)
	}
}

// TestClipToolWaylandFallback: wl-copy есть, а WAYLAND_DISPLAY нет (tmux, ssh) —
// всё равно пробуем его, это лучше, чем ничего.
func TestClipToolWaylandFallback(t *testing.T) {
	fakeEnv(t, "linux", nil, "wl-copy", "wl-paste")
	if tool := detectClipTool(); tool == nil || tool.name != "wl-copy" {
		t.Fatalf("ожидался wl-copy: %+v", tool)
	}
}

// TestClipToolNone: ничего не установлено — nil, а не паника.
func TestClipToolNone(t *testing.T) {
	fakeEnv(t, "linux", nil)
	if tool := detectClipTool(); tool != nil {
		t.Fatalf("ожидался nil: %+v", tool)
	}
}

// TestHashCmd: sha256sum на Linux, shasum на macOS, пусто — если нет обоих.
func TestHashCmd(t *testing.T) {
	fakeEnv(t, "linux", nil, "sha256sum")
	if got := hashCmd(); got != "sha256sum" {
		t.Errorf("linux: %q", got)
	}
	fakeEnv(t, "darwin", nil, "shasum")
	if got := hashCmd(); got != "shasum -a 256" {
		t.Errorf("macos: %q", got)
	}
	fakeEnv(t, "linux", nil)
	if got := hashCmd(); got != "" {
		t.Errorf("без утилит автоочистку не завести: %q", got)
	}
}

// --- новое окно терминала ---

// TestLinuxTerminalArgv: команда уходит как argv (`sh -c <cmd>`), без вложенного
// шелл-квотинга, а порядок предпочтений соблюдается.
func TestLinuxTerminalArgv(t *testing.T) {
	fakeEnv(t, "linux", nil, "xterm", "kitty", "gnome-terminal")
	argv := linuxTerminalArgv("ssh root@h")
	if argv[0] != "kitty" {
		t.Fatalf("kitty должен выигрывать у gnome-terminal и xterm: %v", argv)
	}
	if argv[len(argv)-3] != "sh" || argv[len(argv)-2] != "-c" || argv[len(argv)-1] != "ssh root@h" {
		t.Fatalf("команда должна идти как `sh -c <cmd>`: %v", argv)
	}

	// у gnome-terminal свой разделитель
	fakeEnv(t, "linux", nil, "gnome-terminal")
	if argv := linuxTerminalArgv("x"); argv[1] != "--" {
		t.Fatalf("gnome-terminal ожидает `--`: %v", argv)
	}
	// у xfce4-terminal нужен -x, а не -e: -e принимает одну строку, а не argv
	fakeEnv(t, "linux", nil, "xfce4-terminal")
	if argv := linuxTerminalArgv("x"); argv[1] != "-x" {
		t.Fatalf("xfce4-terminal ожидает `-x`: %v", argv)
	}
}

// TestLinuxTerminalEnvWins: $TERMINAL важнее нашего списка.
func TestLinuxTerminalEnvWins(t *testing.T) {
	fakeEnv(t, "linux", map[string]string{"TERMINAL": "st"}, "st", "kitty")
	if argv := linuxTerminalArgv("x"); argv[0] != "st" {
		t.Fatalf("$TERMINAL должен побеждать: %v", argv)
	}
	// …но только если он реально установлен
	fakeEnv(t, "linux", map[string]string{"TERMINAL": "нетакого"}, "kitty")
	if argv := linuxTerminalArgv("x"); argv[0] != "kitty" {
		t.Fatalf("несуществующий $TERMINAL игнорируем: %v", argv)
	}
}

// TestCanOpenNewWindow: на macOS всегда да; на Linux — только если есть терминал.
func TestCanOpenNewWindow(t *testing.T) {
	fakeEnv(t, "darwin", nil)
	if !canOpenNewWindow() {
		t.Error("на macOS Terminal.app есть всегда")
	}
	fakeEnv(t, "linux", nil, "xterm")
	if !canOpenNewWindow() {
		t.Error("xterm найден — окно открыть можно")
	}
	fakeEnv(t, "linux", nil)
	if canOpenNewWindow() {
		t.Error("терминала нет — открывать нечем")
	}
}

// TestOpenLinuxWindowNoTerminal: без терминала возвращаем понятную ошибку,
// а не пытаемся запустить несуществующий бинарник.
func TestOpenLinuxWindowNoTerminal(t *testing.T) {
	setLang("ru")
	fakeEnv(t, "linux", nil)
	err := openLinuxWindow("ssh h")
	if err == nil || !strings.Contains(err.Error(), "TERMINAL") {
		t.Fatalf("ожидалась подсказка про $TERMINAL: %v", err)
	}
}

// --- хранилище секретов ---

// TestSecretBackend: Keychain на macOS, secret-tool на Linux, иначе файл.
func TestSecretBackend(t *testing.T) {
	fakeEnv(t, "darwin", nil, "security")
	if got := detectSecretBackend(); got != backendKeychain {
		t.Errorf("macOS: %v", got)
	}
	fakeEnv(t, "linux", nil, "secret-tool")
	if got := detectSecretBackend(); got != backendSecretTool {
		t.Errorf("linux с libsecret: %v", got)
	}
	// headless-сервер: ключницы нет, остаётся файл 0600
	fakeEnv(t, "linux", nil)
	if got := detectSecretBackend(); got != backendFile {
		t.Errorf("linux без ключницы: %v", got)
	}
	// на macOS без `security` (теоретически) — тоже файл
	fakeEnv(t, "darwin", nil)
	if got := detectSecretBackend(); got != backendKeychain {
		if got != backendFile {
			t.Errorf("macOS без security: %v", got)
		}
	}
}
