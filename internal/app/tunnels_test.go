package app

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/MrBenoit/unissh/internal/core"
)

// waitFor крутится, пока cond не станет истинным (или не выйдет время).
// Процессы поднимаются и падают асинхронно — без ожидания тесты флапают.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("не дождались: %s", what)
}

// fakeTunnelSpawn подменяет запуск ssh на безобидный долгоживущий процесс.
// Возвращает счётчик вызовов, чтобы проверить, что туннель поднимался ровно раз.
func fakeTunnelSpawn(t *testing.T, mk func() *exec.Cmd) *int {
	t.Helper()
	calls := 0
	orig := tunnelSpawn
	tunnelSpawn = func(s *Server, tn Tunnel) (*exec.Cmd, func(), error) {
		calls++
		return mk(), func() {}, nil
	}
	t.Cleanup(func() { tunnelSpawn = orig })
	return &calls
}

// sleeper — процесс, который просто живёт: имитация поднятого `ssh -N`.
func sleeper() *exec.Cmd { return exec.Command("sleep", "30") }

// tunSrv — сервер для тестов туннелей.
func tunSrv() *Server {
	return &Server{ID: "srv1", Name: "web", Host: "1.2.3.4", Port: 22, User: "root", Key: "/tmp/k", Auth: "key"}
}

var tunLocal = Tunnel{Kind: core.TunnelLocal, LocalPort: 5432, RemoteHost: "localhost", RemotePort: 5432}
var tunSocks = Tunnel{Kind: core.TunnelDynamic, LocalPort: 1080}

// TestTunnelSSHArgs: флаги проброса идут ДО user@host, иначе ssh примет их за
// команду; ExitOnForwardFailure обязателен, чтобы упавший проброс не «висел».
func TestTunnelSSHArgs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	args := tunnelSSHArgs(tunSrv(), tunLocal)
	joined := strings.Join(args, " ")
	if args[len(args)-1] != "root@1.2.3.4" {
		t.Fatalf("цель должна быть последней: %v", args)
	}
	for _, want := range []string{"-N", "-T", "ExitOnForwardFailure=yes", "-L 5432:localhost:5432", "-p 22"} {
		if !strings.Contains(joined, want) {
			t.Errorf("нет %q в %q", want, joined)
		}
	}
	iL := strings.Index(joined, "-L")
	iTarget := strings.Index(joined, "root@1.2.3.4")
	if iL > iTarget {
		t.Errorf("-L оказался после цели: %q", joined)
	}
}

// TestTunnelSpawnKeyAuth: вход по ключу — прямой ssh с BatchMode, чтобы фоновый
// процесс не завис на приглашении пароля.
func TestTunnelSpawnKeyAuth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cmd, cleanup, err := tunnelSpawn(tunSrv(), tunLocal)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if !strings.HasSuffix(cmd.Path, "ssh") {
		t.Fatalf("ожидался ssh, получен %q", cmd.Path)
	}
	joined := strings.Join(cmd.Args, " ")
	if !strings.Contains(joined, "BatchMode=yes") {
		t.Errorf("нет BatchMode: %q", joined)
	}
}

// TestTunnelExpectScript: пароль берётся из окружения (не из argv и не с диска),
// а после ввода скрипт держит ssh живым через expect eof.
func TestTunnelExpectScript(t *testing.T) {
	sc := expectAutoScript("ssh", []string{"-N", "-L", "5432:localhost:5432", "root@h"})
	for _, want := range []string{
		"$env(UNISSH_SSHPASS)",
		`spawn ssh "-N" "-L" "5432:localhost:5432" "root@h"`,
		"set timeout -1",
		"expect eof",
	} {
		if !strings.Contains(sc, want) {
			t.Errorf("в скрипте нет %q:\n%s", want, sc)
		}
	}
	if strings.Contains(sc, "interact") {
		t.Error("фоновому туннелю не нужен interact — ему нечего отдавать пользователю")
	}
}

// TestTunnelStartStop: обычный жизненный цикл — поднять, увидеть живым, погасить.
func TestTunnelStartStop(t *testing.T) {
	setLang("ru")
	calls := fakeTunnelSpawn(t, sleeper)
	m := newTunnelManager()
	s := tunSrv()

	if err := m.Start(s, tunLocal); err != nil {
		t.Fatal(err)
	}
	if !m.Running(s.ID, tunLocal) || m.Active(s.ID) != 1 || m.Total() != 1 {
		t.Fatal("туннель должен считаться поднятым")
	}
	// повторный запуск — идемпотентен, второй процесс не плодим
	if err := m.Start(s, tunLocal); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 {
		t.Fatalf("процесс запущен %d раз, ожидался 1", *calls)
	}

	if err := m.Stop(s.ID, tunLocal); err != nil {
		t.Fatal(err)
	}
	if m.Running(s.ID, tunLocal) || m.Total() != 0 {
		t.Fatal("после Stop туннель не должен числиться живым")
	}
	// остановка незапущенного — не ошибка
	if err := m.Stop(s.ID, tunLocal); err != nil {
		t.Fatalf("повторный Stop: %v", err)
	}
	// ручная остановка — не сбой, причины падения быть не должно
	if fail := m.Failure(s.ID, tunLocal); fail != "" {
		t.Fatalf("Stop не должен выглядеть как падение: %q", fail)
	}
}

// TestTunnelToggle: Toggle переключает состояние и сообщает новое.
func TestTunnelToggle(t *testing.T) {
	setLang("ru")
	fakeTunnelSpawn(t, sleeper)
	m := newTunnelManager()
	s := tunSrv()
	defer m.StopAll()

	up, err := m.Toggle(s, tunSocks)
	if err != nil || !up {
		t.Fatalf("первый Toggle должен поднять туннель: up=%v err=%v", up, err)
	}
	up, err = m.Toggle(s, tunSocks)
	if err != nil || up {
		t.Fatalf("второй Toggle должен погасить: up=%v err=%v", up, err)
	}
}

// TestTunnelPortBusy: два -L на один локальный порт не уживаются, а -R не
// занимает локальный порт вообще, поэтому конфликта с ним быть не может.
func TestTunnelPortBusy(t *testing.T) {
	setLang("ru")
	fakeTunnelSpawn(t, sleeper)
	m := newTunnelManager()
	s := tunSrv()
	defer m.StopAll()

	if err := m.Start(s, tunLocal); err != nil {
		t.Fatal(err)
	}
	// другой сервер, тот же локальный порт → занято
	other := &Server{ID: "srv2", Host: "5.6.7.8", Port: 22, User: "root", Key: "/k", Auth: "key"}
	err := m.Start(other, tunLocal)
	if err == nil || !strings.Contains(err.Error(), "5432") {
		t.Fatalf("ожидался конфликт локального порта, получено: %v", err)
	}
	// -R с тем же числом слушает на сервере — конфликта нет
	rev := Tunnel{Kind: core.TunnelRemote, LocalPort: 5432, RemoteHost: "localhost", RemotePort: 5432}
	if err := m.Start(other, rev); err != nil {
		t.Fatalf("-R не должен конфликтовать с локальным портом: %v", err)
	}
}

// TestTunnelStartInvalid: битое правило до процесса не доходит.
func TestTunnelStartInvalid(t *testing.T) {
	setLang("ru")
	calls := fakeTunnelSpawn(t, sleeper)
	m := newTunnelManager()
	if err := m.Start(tunSrv(), Tunnel{Kind: "bogus"}); err == nil {
		t.Fatal("неизвестный вид проброса должен отклоняться")
	}
	if *calls != 0 {
		t.Fatal("процесс не должен запускаться для невалидного правила")
	}
}

// TestTunnelFailureRecorded: если ssh умер сам (порт занят, проброс запрещён),
// туннель исчезает из живых, а причина сохраняется — очищенная от ANSI.
func TestTunnelFailureRecorded(t *testing.T) {
	setLang("ru")
	fakeTunnelSpawn(t, func() *exec.Cmd {
		return exec.Command("sh", "-c", "printf '\\033[31mbind: Address already in use\\033[0m\\n' >&2; exit 1")
	})
	m := newTunnelManager()
	s := tunSrv()

	if err := m.Start(s, tunLocal); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "процесс упал и убран из живых", func() bool { return !m.Running(s.ID, tunLocal) })

	fail := m.Failure(s.ID, tunLocal)
	if !strings.Contains(fail, "Address already in use") {
		t.Fatalf("причина падения потеряна: %q", fail)
	}
	if strings.Contains(fail, "\x1b") {
		t.Fatalf("ANSI не вычищен из причины: %q", fail)
	}
	if m.Total() != 0 {
		t.Fatal("упавший туннель не должен числиться живым")
	}
	// успешный перезапуск сбрасывает старую причину
	fakeTunnelSpawn(t, sleeper)
	if err := m.Start(s, tunLocal); err != nil {
		t.Fatal(err)
	}
	defer m.StopAll()
	if fail := m.Failure(s.ID, tunLocal); fail != "" {
		t.Fatalf("после успешного старта причина падения должна стереться: %q", fail)
	}
}

// TestTunnelStopAll: гасит всё разом и возвращает число остановленных.
func TestTunnelStopAll(t *testing.T) {
	setLang("ru")
	fakeTunnelSpawn(t, sleeper)
	m := newTunnelManager()
	s := tunSrv()
	if err := m.Start(s, tunLocal); err != nil {
		t.Fatal(err)
	}
	if err := m.Start(s, tunSocks); err != nil {
		t.Fatal(err)
	}
	if n := m.StopAll(); n != 2 {
		t.Fatalf("StopAll вернул %d, ожидалось 2", n)
	}
	if m.Total() != 0 {
		t.Fatal("после StopAll живых туннелей быть не должно")
	}
	if n := m.StopAll(); n != 0 {
		t.Fatalf("повторный StopAll вернул %d", n)
	}
}

// TestTunnelErrText: ошибки ядра переводятся в строки интерфейса.
func TestTunnelErrText(t *testing.T) {
	setLang("ru")
	if tunnelErrText(nil) != "" {
		t.Error("nil → пустая строка")
	}
	if got := tunnelErrText(core.ErrTunnelHost); got != T.ErrTunnelHost {
		t.Errorf("host: %q", got)
	}
	if got := tunnelErrText(core.ErrTunnelKind); got != T.ErrTunnelKind {
		t.Errorf("kind: %q", got)
	}
	if got := tunnelErrText(core.ErrTunnelPort); got != T.ErrPortRange {
		t.Errorf("port: %q", got)
	}
	if got := tunnelErrText(errNoAuth); got != errNoAuth.Error() {
		t.Errorf("прочие ошибки проходят как есть: %q", got)
	}
}

// TestServerTunnelHelpers: дубликаты ловятся по канонич. ключу, удаление — тоже.
func TestServerTunnelHelpers(t *testing.T) {
	s := tunSrv()
	s.Tunnels = []Tunnel{tunLocal, tunSocks}
	if !s.HasTunnel(tunLocal) || !s.HasTunnel(tunSocks) {
		t.Fatal("оба правила должны находиться")
	}
	if s.HasTunnel(Tunnel{Kind: core.TunnelDynamic, LocalPort: 9999}) {
		t.Fatal("чужое правило не должно находиться")
	}
	if !s.RemoveTunnel(tunLocal) || len(s.Tunnels) != 1 {
		t.Fatalf("правило не удалено: %v", s.Tunnels)
	}
	if s.RemoveTunnel(tunLocal) {
		t.Fatal("повторное удаление должно вернуть false")
	}
}

// TestTunnelLines: панель статистики показывает правила и их состояние.
func TestTunnelLines(t *testing.T) {
	setLang("ru")
	s := tunSrv()
	if lines := tunnelLines(s, 40); lines != nil {
		t.Fatalf("без правил строк быть не должно: %v", lines)
	}
	s.Tunnels = []Tunnel{tunLocal, tunSocks}

	fakeTunnelSpawn(t, sleeper)
	if err := tunnels.Start(s, tunLocal); err != nil {
		t.Fatal(err)
	}
	defer tunnels.StopAll()

	lines := tunnelLines(s, 40)
	if len(lines) != 3 { // пустая строка-разделитель + два правила
		t.Fatalf("ожидалось 3 строки, получено %d: %v", len(lines), lines)
	}
	body := strings.Join(lines, "\n")
	if !strings.Contains(body, T.LblTunnels) {
		t.Error("нет метки «Туннели»")
	}
	if !strings.Contains(body, "●") || !strings.Contains(body, "○") {
		t.Errorf("должен быть один поднятый (●) и один спящий (○): %q", body)
	}
}

// TestTunnelStateFailure: у упавшего правила — крестик и причина вместо подписи.
func TestTunnelStateFailure(t *testing.T) {
	setLang("ru")
	fakeTunnelSpawn(t, func() *exec.Cmd {
		return exec.Command("sh", "-c", "echo nope >&2; exit 1")
	})
	s := tunSrv()
	if err := tunnels.Start(s, tunSocks); err != nil {
		t.Fatal(err)
	}
	defer tunnels.StopAll()
	waitFor(t, "процесс упал", func() bool { return tunnels.Failure(s.ID, tunSocks) != "" })

	dot, state := tunnelState(s.ID, tunSocks)
	if !strings.Contains(dot, "✗") {
		t.Errorf("ожидался ✗, получено %q", dot)
	}
	if !strings.Contains(state, "nope") {
		t.Errorf("причина падения не показана: %q", state)
	}
}

// TestServerActionsHaveTunnels: пункт проброса портов есть у любого сервера.
func TestServerActionsHaveTunnels(t *testing.T) {
	setLang("ru")
	found := false
	for _, a := range serverActions(tunSrv()) {
		if a.action == "tunnels" {
			found = true
		}
	}
	if !found {
		t.Fatal("в действиях сервера нет пункта «Проброс портов»")
	}
}
