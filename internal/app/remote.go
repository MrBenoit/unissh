package app

// SSH-клиент на golang.org/x/crypto/ssh: выполнение команд, запись файлов,
// TOFU-проверка отпечатков через собственный known_hosts.

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Remote struct {
	client *ssh.Client
	jump   *ssh.Client // соединение с бастионом (ProxyJump), закрывается вместе с client
	host   string
	port   int
	user   string
}

var knownHostsMu sync.Mutex

// hostKeyChangedError — отпечаток хоста не совпал с сохранённым.
type hostKeyChangedError struct{ inner error }

func (e *hostKeyChangedError) Error() string {
	return fmt.Sprintf(T.ErrHostKeyChanged, e.inner)
}

// tofuHostKeyCallback: незнакомый хост запоминаем, изменившийся — отвергаем.
func tofuHostKeyCallback() (ssh.HostKeyCallback, error) {
	path := knownHostsPath()
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	base, err := knownhosts.New(path)
	if err != nil {
		return nil, err
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := base(hostname, remote, key)
		if err == nil {
			return nil
		}
		var ke *knownhosts.KeyError
		if errors.As(err, &ke) && len(ke.Want) == 0 {
			knownHostsMu.Lock()
			defer knownHostsMu.Unlock()
			f, ferr := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
			if ferr != nil {
				return ferr
			}
			defer func() { _ = f.Close() }()
			_, werr := fmt.Fprintln(f, knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key))
			return werr
		}
		return &hostKeyChangedError{inner: err}
	}, nil
}

// agentAuthMethod возвращает метод аутентификации через ssh-agent
// (переменная SSH_AUTH_SOCK) или nil, если агент недоступен.
func agentAuthMethod() ssh.AuthMethod {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers)
}

// keyAuthMethod загружает приватный ключ. Для ключа под passphrase пробует
// сохранённую в Keychain фразу (аккаунт "passphrase/<путь>"); если её нет —
// возвращает (nil, nil), полагаясь на ssh-agent или пароль.
func keyAuthMethod(keyPath string) (ssh.AuthMethod, error) {
	data, err := os.ReadFile(expandPath(keyPath))
	if err != nil {
		return nil, fmt.Errorf(T.ErrReadKey, keyPath, err)
	}
	if signer, perr := ssh.ParsePrivateKey(data); perr == nil {
		return ssh.PublicKeys(signer), nil
	} else {
		var pm *ssh.PassphraseMissingError
		if !errors.As(perr, &pm) {
			return nil, fmt.Errorf(T.ErrParseKey, keyPath, perr)
		}
	}
	if pass := secretGet("passphrase", keyPath); pass != "" {
		if signer, perr := ssh.ParsePrivateKeyWithPassphrase(data, []byte(pass)); perr == nil {
			return ssh.PublicKeys(signer), nil
		}
	}
	return nil, nil // зашифрованный ключ без известной фразы — пусть решает агент/пароль
}

// dialViaJump подключается к target через бастион proxyJump (для программного
// клиента). Бастион аутентифицируется тем же ключом/агентом, что и цель.
func dialViaJump(proxyJump, defaultUser, targetHost string, targetPort int,
	targetCfg *ssh.ClientConfig, keyPath string, timeout time.Duration) (target, jump *ssh.Client, err error) {
	ju, jh, jp := parseJump(proxyJump, defaultUser)
	var jauths []ssh.AuthMethod
	if keyPath != "" {
		if m, _ := keyAuthMethod(keyPath); m != nil {
			jauths = append(jauths, m)
		}
	}
	if am := agentAuthMethod(); am != nil {
		jauths = append(jauths, am)
	}
	if len(jauths) == 0 {
		return nil, nil, fmt.Errorf("бастион %s: нужен ключ или ssh-agent", proxyJump)
	}
	jcfg := &ssh.ClientConfig{User: ju, Auth: jauths, HostKeyCallback: targetCfg.HostKeyCallback, Timeout: timeout}
	jClient, err := ssh.Dial("tcp", net.JoinHostPort(jh, jp), jcfg)
	if err != nil {
		return nil, nil, fmt.Errorf("бастион %s: %w", proxyJump, err)
	}
	targetAddr := net.JoinHostPort(targetHost, strconv.Itoa(targetPort))
	conn, err := jClient.Dial("tcp", targetAddr)
	if err != nil {
		_ = jClient.Close()
		return nil, nil, err
	}
	ncc, chans, reqs, err := ssh.NewClientConn(conn, targetAddr, targetCfg)
	if err != nil {
		_ = jClient.Close()
		return nil, nil, err
	}
	return ssh.NewClient(ncc, chans, reqs), jClient, nil
}

// dialSSH подключается по ключу, ssh-agent'у и/или паролю (в этом порядке);
// при proxyJump != "" — через бастион.
// Объявлено как var — тесты подменяют транспорт (см. sshserver_test.go), что
// открывает bootstrap.go и action* для прогона без живого сервера.
var dialSSH = func(host string, port int, user, password, keyPath, proxyJump string, timeout time.Duration) (*Remote, error) {
	var auths []ssh.AuthMethod
	if keyPath != "" {
		m, err := keyAuthMethod(keyPath)
		if err != nil {
			return nil, err
		}
		if m != nil {
			auths = append(auths, m)
		}
	}
	if am := agentAuthMethod(); am != nil {
		auths = append(auths, am)
	}
	if password != "" {
		pw := password
		auths = append(auths,
			ssh.Password(pw),
			ssh.KeyboardInteractive(func(name, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = pw
				}
				return answers, nil
			}))
	}
	if len(auths) == 0 {
		return nil, errors.New(T.ErrNoAuthMethod)
	}
	cb, err := tofuHostKeyCallback()
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: cb,
		Timeout:         timeout,
	}
	var client, jump *ssh.Client
	if proxyJump != "" {
		client, jump, err = dialViaJump(proxyJump, user, host, port, cfg, keyPath, timeout)
	} else {
		client, err = ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), cfg)
	}
	if err != nil {
		var hk *hostKeyChangedError
		if errors.As(err, &hk) {
			return nil, hk
		}
		if strings.Contains(err.Error(), "unable to authenticate") ||
			strings.Contains(err.Error(), "no supported methods remain") {
			return nil, fmt.Errorf(T.ErrAuthFailed, user, host, port)
		}
		return nil, fmt.Errorf(T.ErrConnectFailed, host, port, err)
	}
	return &Remote{client: client, jump: jump, host: host, port: port, user: user}, nil
}

// openRemote подключается к сохранённому серверу (ключ или пароль из Keychain).
// Тоже var — тот же шов для тестов.
var openRemote = func(s *Server, askPassword bool) (*Remote, error) {
	key := s.Key
	password := ""
	if s.Auth == "password" || key == "" {
		key = ""
		password = secretGet(s.ID, s.User)
		if password == "" && askPassword {
			pw, ok := askPasswordInput(fmt.Sprintf(T.AskPasswordFor, s.User, s.Host))
			if !ok {
				return nil, errors.New(T.Canceled)
			}
			password = pw
		}
		if password == "" {
			return nil, errors.New(T.ErrNoPwInKeychain)
		}
	}
	return dialSSH(s.Host, s.Port, s.User, password, key, s.ProxyJump, 15*time.Second)
}

// lineWriter копит вывод и отдаёт целые строки в колбэк (для спиннеров).
type lineWriter struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	pending []byte
	onLine  func(string)
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(p)
	if w.onLine != nil {
		w.pending = append(w.pending, p...)
		for {
			i := bytes.IndexByte(w.pending, '\n')
			if i < 0 {
				break
			}
			line := strings.TrimSpace(string(w.pending[:i]))
			w.pending = w.pending[i+1:]
			if line != "" {
				w.onLine(line)
			}
		}
	}
	return len(p), nil
}

func (w *lineWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// Run выполняет команду; возвращает (код, объединённый вывод, ошибка транспорта).
func (r *Remote) Run(cmd, stdin string, onLine func(string), timeout time.Duration) (int, string, error) {
	sess, err := r.client.NewSession()
	if err != nil {
		return -1, "", fmt.Errorf(T.ErrConnLost, err)
	}
	defer func() { _ = sess.Close() }()
	if stdin != "" {
		sess.Stdin = strings.NewReader(stdin)
	}
	lw := &lineWriter{onLine: onLine}
	sess.Stdout = lw
	sess.Stderr = lw
	full := "LANG=C LC_ALL=C sh -c " + shellQuote(cmd)
	if err := sess.Start(full); err != nil {
		return -1, "", err
	}
	errCh := make(chan error, 1)
	go func() { errCh <- sess.Wait() }()
	var timer <-chan time.Time
	if timeout > 0 {
		t := time.NewTimer(timeout)
		defer t.Stop()
		timer = t.C
	}
	select {
	case err := <-errCh:
		out := lw.String()
		if err == nil {
			return 0, out, nil
		}
		var ee *ssh.ExitError
		if errors.As(err, &ee) {
			return ee.ExitStatus(), out, nil
		}
		return -1, out, err
	case <-timer:
		_ = sess.Close()
		return -1, lw.String(), fmt.Errorf(T.ErrCmdTimeout, timeout)
	}
}

// Sh — как Run, но ненулевой код превращает в ошибку с хвостом вывода.
func (r *Remote) Sh(cmd, stdin string, onLine func(string), timeout time.Duration) (string, error) {
	code, out, err := r.Run(cmd, stdin, onLine, timeout)
	if err != nil {
		return out, err
	}
	if code != 0 {
		// хвост вывода сервера попадёт в текст ошибки и будет напечатан — санитайзим
		lines := strings.Split(strings.TrimSpace(sanitizeRemote(out)), "\n")
		if len(lines) > 6 {
			lines = lines[len(lines)-6:]
		}
		return out, fmt.Errorf(T.ErrCmdExitCode, code, strings.Join(lines, "\n"))
	}
	return out, nil
}

// WriteFile пишет файл на сервере через stdin (без зависимости от sftp).
func (r *Remote) WriteFile(path, content, mode string) error {
	cmd := fmt.Sprintf("cat > %s && chmod %s %s", shellQuote(path), mode, shellQuote(path))
	_, err := r.Sh(cmd, content, nil, 30*time.Second)
	return err
}

// Close закрывает соединение (и бастион, если он был).
func (r *Remote) Close() {
	if r.client != nil {
		_ = r.client.Close()
	}
	if r.jump != nil {
		_ = r.jump.Close()
	}
}
