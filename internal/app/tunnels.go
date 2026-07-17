package app

// Проброс портов: фоновые процессы `ssh -N` (-L / -R / -D).
//
// Правила живут в конфиге (Server.Tunnels), а запущенные процессы — в
// tunnelManager. Туннели крутятся, пока жив unissh: при выходе StopAll гасит
// их все. Процесс запускается в отдельной группе (Setpgid), чтобы Ctrl+C в
// терминале не убил туннель мимо менеджера, и гасится сигналом всей группе —
// иначе при парольном входе expect умер бы, а его ssh остался сиротой.
//
// Вывод ssh никуда не печатается: stderr копится в буфер и показывается только
// как причина падения, обязательно через sanitizeRemote.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync"
	"syscall"

	"github.com/MrBenoit/unissh/internal/core"
)

// tunnelKey — уникальный ключ запущенного туннеля: сервер + каноническое правило.
func tunnelKey(serverID string, t Tunnel) string { return serverID + "|" + t.Spec() }

// tunnelProc — живой процесс одного туннеля.
type tunnelProc struct {
	cmd       *exec.Cmd
	serverID  string
	tun       Tunnel
	stderr    *bytes.Buffer
	cleanupFn func() // удаление временного expect-скрипта
}

// tunnelManager хранит запущенные туннели и причины падения последних попыток.
type tunnelManager struct {
	mu     sync.Mutex
	procs  map[string]*tunnelProc
	failed map[string]string // ключ → текст ошибки последнего падения
}

// newTunnelManager создаёт пустой менеджер туннелей.
func newTunnelManager() *tunnelManager {
	return &tunnelManager{procs: map[string]*tunnelProc{}, failed: map[string]string{}}
}

// tunnels — единственный менеджер на процесс: туннели переживают возвраты в меню.
var tunnels = newTunnelManager()

// tunnelSSHArgs собирает аргументы ssh для туннеля: флаги проброса идут ПЕРЕД
// sshArgs, потому что там последним элементом стоит user@host.
// ExitOnForwardFailure=yes заставляет ssh упасть сразу, если порт занят или
// сервер запретил проброс, — иначе туннель «висел» бы живым, но нерабочим.
func tunnelSSHArgs(s *Server, t Tunnel) []string {
	// ControlPath=none — туннелю нужно СВОЁ соединение. На общем master-сокете
	// проброс жил бы чужой жизнью: master умрёт по ControlPersist — умрёт и
	// туннель, а наш kill по группе процессов до него не достучится.
	// Идёт ПЕРВЫМ: у ssh при повторе опции выигрывает первое значение.
	args := []string{"-N", "-T", "-o", "ExitOnForwardFailure=yes", "-o", "ControlPath=none"}
	args = append(args, t.Flags()...)
	return append(args, sshArgs(s)...)
}

// tunnelSpawn — шов для тестов: готовит (но не запускает) процесс туннеля и
// функцию очистки временных файлов. При парольном входе ssh запускается через
// expect: `ssh -N` сам не завершается, поэтому скрипт вводит пароль и ждёт eof.
var tunnelSpawn = func(s *Server, t Tunnel) (*exec.Cmd, func(), error) {
	args := tunnelSSHArgs(s, t)
	if pw := serverPassword(s); pw != "" && hasExpect() {
		path, err := writeTempScript(expectAutoScript("ssh", args))
		if err != nil {
			return nil, nil, err
		}
		cmd := exec.Command("expect", "-f", path)
		cmd.Env = append(os.Environ(), "UNISSH_SSHPASS="+pw)
		return cmd, func() { _ = os.Remove(path) }, nil
	}
	// Вход по ключу: BatchMode запрещает интерактивный запрос пароля, иначе
	// фоновый ssh молча повис бы на приглашении.
	args = append([]string{"-o", "BatchMode=yes"}, args...)
	return exec.Command("ssh", args...), func() {}, nil
}

// tunnelDetach переводит процесс в свою группу — чтобы SIGINT терминала не
// убивал туннель в обход менеджера.
func tunnelDetach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// tunnelKill гасит процесс вместе со всей его группой (expect + его ssh).
func tunnelKill(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
		_ = cmd.Process.Kill() // группы нет (или уже мертва) — бьём по процессу
	}
}

// portBusy сообщает, занят ли локальный порт другим запущенным туннелем.
// Для -R порт слушает сервер, поэтому локального конфликта быть не может.
// Вызывается под m.mu.
func (m *tunnelManager) portBusy(t Tunnel) bool {
	if !core.TunnelListensLocally(t.Kind) {
		return false
	}
	for _, p := range m.procs {
		if core.TunnelListensLocally(p.tun.Kind) && p.tun.LocalPort == t.LocalPort {
			return true
		}
	}
	return false
}

// Start поднимает туннель. Повторный запуск уже работающего — не ошибка.
func (m *tunnelManager) Start(s *Server, t Tunnel) error {
	if err := t.Validate(); err != nil {
		return err
	}
	key := tunnelKey(s.ID, t)

	m.mu.Lock()
	if _, ok := m.procs[key]; ok {
		m.mu.Unlock()
		return nil
	}
	if m.portBusy(t) {
		m.mu.Unlock()
		return fmt.Errorf(T.ErrTunnelPortBusy, t.LocalPort)
	}
	m.mu.Unlock()

	cmd, cleanup, err := tunnelSpawn(s, t)
	if err != nil {
		return err
	}
	var errBuf bytes.Buffer
	cmd.Stdout = nil // вывод ssh нам не нужен
	cmd.Stderr = &errBuf
	tunnelDetach(cmd)
	if err := cmd.Start(); err != nil {
		cleanup()
		return err
	}
	p := &tunnelProc{cmd: cmd, serverID: s.ID, tun: t, stderr: &errBuf, cleanupFn: cleanup}

	m.mu.Lock()
	// Пока мы запускали процесс без замка, туннель мог подняться в другом месте.
	if _, ok := m.procs[key]; ok {
		m.mu.Unlock()
		tunnelKill(cmd)
		_ = cmd.Wait()
		cleanup()
		return nil
	}
	delete(m.failed, key)
	m.procs[key] = p
	m.mu.Unlock()

	go m.reap(key, p)
	return nil
}

// reap ждёт завершения туннеля и убирает его из карты. Если процесс упал сам
// (порт занят, сервер запретил проброс), запоминаем причину для интерфейса.
// Читать stderr безопасно только после Wait — он и дожидается копирования.
func (m *tunnelManager) reap(key string, p *tunnelProc) {
	err := p.cmd.Wait()
	p.cleanupFn()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.procs[key] != p {
		return // туннель уже заменён новым запуском — чужую ошибку не пишем
	}
	delete(m.procs, key)
	if err != nil {
		if reason := firstLine(sanitizeRemote(p.stderr.String())); reason != "" {
			m.failed[key] = reason
		} else {
			m.failed[key] = err.Error()
		}
	}
}

// Stop гасит туннель. Остановка незапущенного — не ошибка.
func (m *tunnelManager) Stop(serverID string, t Tunnel) error {
	key := tunnelKey(serverID, t)
	m.mu.Lock()
	p := m.procs[key]
	delete(m.procs, key)  // reap увидит подмену и не тронет карту повторно
	delete(m.failed, key) // остановка вручную — не сбой
	m.mu.Unlock()
	if p == nil {
		return nil
	}
	tunnelKill(p.cmd)
	return nil
}

// Toggle поднимает туннель, если он лежит, и наоборот. Возвращает новое состояние.
func (m *tunnelManager) Toggle(s *Server, t Tunnel) (up bool, err error) {
	if m.Running(s.ID, t) {
		return false, m.Stop(s.ID, t)
	}
	if err := m.Start(s, t); err != nil {
		return false, err
	}
	return true, nil
}

// Running сообщает, работает ли туннель прямо сейчас.
func (m *tunnelManager) Running(serverID string, t Tunnel) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.procs[tunnelKey(serverID, t)]
	return ok
}

// Failure возвращает причину падения туннеля (пусто, если её не было).
func (m *tunnelManager) Failure(serverID string, t Tunnel) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.failed[tunnelKey(serverID, t)]
}

// Active считает работающие туннели сервера.
func (m *tunnelManager) Active(serverID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, p := range m.procs {
		if p.serverID == serverID {
			n++
		}
	}
	return n
}

// Total — сколько туннелей поднято во всём приложении.
func (m *tunnelManager) Total() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.procs)
}

// StopAll гасит все туннели: вызывается при выходе из приложения.
// Возвращает число остановленных.
func (m *tunnelManager) StopAll() int {
	m.mu.Lock()
	procs := make([]*tunnelProc, 0, len(m.procs))
	for k, p := range m.procs {
		procs = append(procs, p)
		delete(m.procs, k)
	}
	m.mu.Unlock()
	// детерминированный порядок: тесты и логи не должны зависеть от карты
	sort.Slice(procs, func(i, j int) bool { return procs[i].tun.Spec() < procs[j].tun.Spec() })
	for _, p := range procs {
		tunnelKill(p.cmd)
	}
	return len(procs)
}

// tunnelErrText переводит ошибку правила в строку интерфейса.
func tunnelErrText(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, core.ErrTunnelPort):
		return T.ErrPortRange
	case errors.Is(err, core.ErrTunnelHost):
		return T.ErrTunnelHost
	case errors.Is(err, core.ErrTunnelKind):
		return T.ErrTunnelKind
	}
	return err.Error()
}
