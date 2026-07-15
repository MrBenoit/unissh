package app

import (
	"net"
	"strconv"
	"testing"
	"time"
)

func TestStatusTimeout(t *testing.T) {
	if got := statusTimeout(&Store{Config: Config{Settings: Settings{StatusTimeout: 2}}}); got != 2*time.Second {
		t.Fatalf("statusTimeout(2) = %v", got)
	}
	// 0 или отрицательное → безопасный дефолт 1.5с (иначе DialTimeout(0) висит)
	if got := statusTimeout(&Store{Config: Config{Settings: Settings{StatusTimeout: 0}}}); got != 1500*time.Millisecond {
		t.Fatalf("statusTimeout(0) = %v", got)
	}
}

// listenPort открывает локальный TCP-порт и возвращает listener и его номер.
func listenPort(t *testing.T) (net.Listener, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	return ln, port
}

func TestProbeOnline(t *testing.T) {
	ln, port := listenPort(t)
	defer ln.Close()
	r := probe("127.0.0.1", port, time.Second)
	if !r.Online {
		t.Fatal("слушающий порт должен быть online")
	}
	if r.Ping <= 0 {
		t.Fatal("пинг должен быть положительным")
	}
}

func TestProbeOffline(t *testing.T) {
	// открываем и сразу закрываем порт → подключение будет отклонено
	ln, port := listenPort(t)
	ln.Close()
	r := probe("127.0.0.1", port, 500*time.Millisecond)
	if r.Online {
		t.Fatal("закрытый порт не должен быть online")
	}
}

func TestProbeAll(t *testing.T) {
	ln, port := listenPort(t)
	defer ln.Close()
	servers := []*Server{{ID: "up", Host: "127.0.0.1", Port: port}}
	res := probeAll(servers, time.Second)
	if !res["up"].Online {
		t.Fatal("probeAll должен вернуть online для слушающего порта")
	}
}
