package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// execHandler получает полную команду и возвращает stdout и код возврата.
type execHandler func(cmd string) (string, int)

// startTestSSHServer поднимает in-process SSH-сервер на 127.0.0.1 для тестов
// транспортного слоя. acceptPassword=false заставляет сервер отвергать пароль.
func startTestSSHServer(t *testing.T, acceptPassword bool, h execHandler) (string, int) {
	t.Helper()
	// изолируемся от живого ssh-agent разработчика: иначе dialSSH пытается
	// аутентифицироваться его ключами и может зависнуть в согласовании.
	t.Setenv("SSH_AUTH_SOCK", "")
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
			if acceptPassword {
				return nil, nil
			}
			return nil, errNoAuth
		},
		PublicKeyCallback: func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
			return nil, nil // принимаем любой ключ — для тестов key-auth
		},
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveTestSSH(conn, cfg, h)
		}
	}()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	return host, port
}

var errNoAuth = &authErr{}

type authErr struct{}

func (*authErr) Error() string { return "auth rejected" }

func serveTestSSH(conn net.Conn, cfg *ssh.ServerConfig, h execHandler) {
	sconn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return
	}
	defer sconn.Close()
	go ssh.DiscardRequests(reqs)
	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			newCh.Reject(ssh.UnknownChannelType, "only session")
			continue
		}
		ch, requests, err := newCh.Accept()
		if err != nil {
			continue
		}
		go handleSession(ch, requests, h)
	}
}

func handleSession(ch ssh.Channel, requests <-chan *ssh.Request, h execHandler) {
	for req := range requests {
		switch req.Type {
		case "exec":
			var p struct{ Cmd string }
			ssh.Unmarshal(req.Payload, &p)
			if req.WantReply {
				req.Reply(true, nil)
			}
			// вычитываем stdin в фоне: иначе при WriteFile клиент пишет контент
			// в канал, а мы бы закрыли его раньше — клиент получил бы EOF (флейк под -race)
			drained := make(chan struct{})
			go func() { io.Copy(io.Discard, ch); close(drained) }()
			out, code := "", 0
			if h != nil {
				out, code = h(p.Cmd)
			}
			ch.Write([]byte(out))
			ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Code uint32 }{uint32(code)}))
			ch.CloseWrite() // сигналим EOF по нашему выводу
			<-drained       // ждём, пока клиент дошлёт stdin, и только потом закрываем
			ch.Close()
			return
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

func TestDialSSHRunSuccess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	host, port := startTestSSHServer(t, true, func(cmd string) (string, int) {
		return "got: " + cmd + "\n", 0
	})
	r, err := dialSSH(host, port, "root", "secret", "", "", 5*time.Second)
	if err != nil {
		t.Fatalf("dialSSH: %v", err)
	}
	defer r.Close()
	code, out, err := r.Run("echo hi", "", nil, 5*time.Second)
	if err != nil || code != 0 {
		t.Fatalf("Run: err=%v code=%d", err, code)
	}
	if !strings.Contains(out, "echo hi") {
		t.Fatalf("вывод не содержит команду: %q", out)
	}
}

func TestRemoteShAndWriteFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var lastCmd string
	host, port := startTestSSHServer(t, true, func(cmd string) (string, int) {
		lastCmd = cmd
		return "ok\n", 0
	})
	r, err := dialSSH(host, port, "root", "pw", "", "", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if out, err := r.Sh("whoami", "", nil, 5*time.Second); err != nil || !strings.Contains(out, "ok") {
		t.Fatalf("Sh: out=%q err=%v", out, err)
	}
	if err := r.WriteFile("/tmp/x", "content", "0644"); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !strings.Contains(lastCmd, "/tmp/x") {
		t.Fatalf("WriteFile должен упоминать путь: %q", lastCmd)
	}
}

func TestRemoteRunNonZeroExit(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	host, port := startTestSSHServer(t, true, func(cmd string) (string, int) {
		return "boom\n", 3
	})
	r, err := dialSSH(host, port, "root", "pw", "", "", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	code, _, _ := r.Run("false", "", nil, 5*time.Second)
	if code != 3 {
		t.Fatalf("код возврата = %d, want 3", code)
	}
}

func TestDialSSHAuthFail(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	host, port := startTestSSHServer(t, false, nil) // пароль отвергается
	_, err := dialSSH(host, port, "root", "wrong", "", "", 5*time.Second)
	if err == nil {
		t.Fatal("ожидалась ошибка аутентификации")
	}
}

func TestDialSSHNoAuthMethods(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// ни ключа, ни пароля, ни агента → нет методов
	t.Setenv("SSH_AUTH_SOCK", "")
	_, err := dialSSH("127.0.0.1", 1, "root", "", "", "", time.Second)
	if err == nil {
		t.Fatal("ожидалась ошибка отсутствия методов аутентификации")
	}
}

func TestOpenRemoteNoPassword(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// пароль не в Keychain, askPassword=false → ошибка
	s := &Server{ID: "x", Name: "s", Host: "127.0.0.1", Port: 22, User: "root", Auth: "password"}
	if _, err := openRemote(s, false); err == nil {
		t.Fatal("ожидалась ошибка отсутствия пароля")
	}
}
