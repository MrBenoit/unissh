package core

import (
	"errors"
	"strings"
	"testing"
)

// TestValidPort: границы диапазона портов.
func TestValidPort(t *testing.T) {
	for _, c := range []struct {
		p    int
		want bool
	}{{0, false}, {1, true}, {22, true}, {65535, true}, {65536, false}, {-1, false}} {
		if got := ValidPort(c.p); got != c.want {
			t.Errorf("ValidPort(%d) = %v", c.p, got)
		}
	}
}

// TestValidateTunnel: каждый вид проверяется по своим полям, для -D удалённые
// поля не важны, неизвестный вид — отдельная ошибка.
func TestValidateTunnel(t *testing.T) {
	cases := []struct {
		name       string
		kind       string
		lp         int
		rh         string
		rp         int
		wantErr    error
		wantNilErr bool
	}{
		{"local ok", TunnelLocal, 5432, "localhost", 5432, nil, true},
		{"remote ok", TunnelRemote, 80, "127.0.0.1", 8080, nil, true},
		{"dynamic ok", TunnelDynamic, 1080, "", 0, nil, true},
		{"dynamic игнорирует хост", TunnelDynamic, 1080, "", 0, nil, true},
		{"local без хоста", TunnelLocal, 5432, "  ", 5432, ErrTunnelHost, false},
		{"local хост с пробелом", TunnelLocal, 5432, "a b", 5432, ErrTunnelHost, false},
		{"local хост с двоеточием", TunnelLocal, 5432, "a:1", 5432, ErrTunnelHost, false},
		{"local плохой локальный порт", TunnelLocal, 0, "h", 5432, ErrTunnelPort, false},
		{"local плохой удалённый порт", TunnelLocal, 5432, "h", 99999, ErrTunnelPort, false},
		{"remote плохой порт", TunnelRemote, 70000, "h", 80, ErrTunnelPort, false},
		{"dynamic плохой порт", TunnelDynamic, 0, "", 0, ErrTunnelPort, false},
		{"неизвестный вид", "socks", 1080, "", 0, ErrTunnelKind, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateTunnel(c.kind, c.lp, c.rh, c.rp)
			if c.wantNilErr {
				if err != nil {
					t.Fatalf("ожидался nil, получено %v", err)
				}
				return
			}
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("ожидалась %v, получено %v", c.wantErr, err)
			}
		})
	}
}

// TestTunnelFlags: порядок полей в -L и -R разный — это главный источник путаницы.
func TestTunnelFlags(t *testing.T) {
	if got := TunnelFlags(TunnelLocal, 5432, "db.internal", 5433); strings.Join(got, " ") != "-L 5432:db.internal:5433" {
		t.Errorf("-L: %v", got)
	}
	// у -R первым идёт порт НА СЕРВЕРЕ (remotePort), последним — наш локальный
	if got := TunnelFlags(TunnelRemote, 3000, "localhost", 8080); strings.Join(got, " ") != "-R 8080:localhost:3000" {
		t.Errorf("-R: %v", got)
	}
	if got := TunnelFlags(TunnelDynamic, 1080, "", 0); strings.Join(got, " ") != "-D 1080" {
		t.Errorf("-D: %v", got)
	}
	if got := TunnelFlags("nope", 1, "h", 2); got != nil {
		t.Errorf("неизвестный вид должен давать nil, а не половину команды: %v", got)
	}
}

// TestTunnelSpec: ключ уникален по виду и портам, у -D хост не участвует.
func TestTunnelSpec(t *testing.T) {
	if got := TunnelSpec(TunnelLocal, 5432, "h", 5433); got != "L:5432:h:5433" {
		t.Errorf("local: %q", got)
	}
	if got := TunnelSpec(TunnelRemote, 3000, "h", 8080); got != "R:8080:h:3000" {
		t.Errorf("remote: %q", got)
	}
	if got := TunnelSpec(TunnelDynamic, 1080, "h", 9); got != "D:1080" {
		t.Errorf("dynamic: %q", got)
	}
	if got := TunnelSpec("x", 1, "h", 2); got != "?:x" {
		t.Errorf("unknown: %q", got)
	}
	// -L и -R с одинаковыми числами не должны схлопнуться в один ключ
	if TunnelSpec(TunnelLocal, 1, "h", 1) == TunnelSpec(TunnelRemote, 1, "h", 1) {
		t.Error("ключи -L и -R совпали")
	}
}

// TestTunnelLabel: подпись содержит порты и не зависит от языка.
func TestTunnelLabel(t *testing.T) {
	for _, c := range []struct{ got, want string }{
		{TunnelLabel(TunnelLocal, 5432, "db", 5433), "L  :5432 → db:5433"},
		{TunnelLabel(TunnelRemote, 3000, "localhost", 8080), "R  srv:8080 → localhost:3000"},
		{TunnelLabel(TunnelDynamic, 1080, "", 0), "D  :1080  SOCKS5"},
		{TunnelLabel("weird", 1, "", 0), "weird"},
	} {
		if c.got != c.want {
			t.Errorf("подпись = %q, ожидалось %q", c.got, c.want)
		}
	}
}

// TestTunnelListensLocally: только -L и -D занимают порт на нашей машине.
func TestTunnelListensLocally(t *testing.T) {
	if !TunnelListensLocally(TunnelLocal) || !TunnelListensLocally(TunnelDynamic) {
		t.Error("-L и -D слушают локально")
	}
	if TunnelListensLocally(TunnelRemote) || TunnelListensLocally("x") {
		t.Error("-R слушает на сервере, а не у нас")
	}
}
