package app

import (
	"os"
	"testing"
	"time"
)

// TestParseSSHConfig проверяет разбор ~/.ssh/config, пропуск wildcard-хостов
// и применение дефолтов из блока «Host *».
func TestParseSSHConfig(t *testing.T) {
	content := `
Host *
    User admin
    IdentityFile ~/.ssh/id_ed25519

Host prod
    HostName 10.0.0.1
    Port 2222

Host db
    HostName 10.0.0.2
    User dbuser
    ProxyJump prod

Host *.internal
    User nobody
`
	f, _ := os.CreateTemp("", "sshcfg-*")
	f.WriteString(content)
	f.Close()
	defer os.Remove(f.Name())

	hosts, err := parseSSHConfig(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 2 { // prod и db; «Host *» и «*.internal» пропущены
		t.Fatalf("ожидалось 2 хоста, получено %d: %+v", len(hosts), hosts)
	}
	// prod наследует User/Key от Host *
	if hosts[0].Alias != "prod" || hosts[0].User != "admin" || hosts[0].Key != "~/.ssh/id_ed25519" || hosts[0].Port != 2222 {
		t.Fatalf("prod разобран неверно (дефолты Host * не применены): %+v", hosts[0])
	}
	// db имеет свой User, наследует Key, задаёт ProxyJump
	if hosts[1].User != "dbuser" || hosts[1].Key != "~/.ssh/id_ed25519" || hosts[1].ProxyJump != "prod" {
		t.Fatalf("db разобран неверно: %+v", hosts[1])
	}
}

// TestSanitizeRemote проверяет вырезание ANSI/управляющих последовательностей
// из вывода сервера (защита от терминальной инъекции с чужой машины).
func TestSanitizeRemote(t *testing.T) {
	cases := map[string]string{
		"\x1b[31mred\x1b[0m": "red",
		"a\x1b]0;title\x07b": "ab", // OSC
		"line1\nline2":       "line1\nline2",
		"tab\there":          "tab\there",
		"bell\x07nul\x00ok":  "bellnulok",
		"\x1b[2J\x1b[Hclear": "clear",
	}
	for in, want := range cases {
		if got := sanitizeRemote(in); got != want {
			t.Errorf("sanitizeRemote(%q) = %q, ожидалось %q", in, got, want)
		}
	}
}

// TestFormatPing проверяет, что суб-миллисекундный пинг не превращается в «0».
func TestFormatPing(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{540 * time.Microsecond, "0.5"},
		{3200 * time.Microsecond, "3.2"},
		{45 * time.Millisecond, "45"},
		{120 * time.Millisecond, "120"},
	}
	for _, c := range cases {
		if got := formatPing(c.d); got != c.want {
			t.Errorf("formatPing(%v) = %q, ожидалось %q", c.d, got, c.want)
		}
	}
}

// TestRelativeTime проверяет границы «сколько назад».
func TestRelativeTime(t *testing.T) {
	now := time.Now()
	if got := relativeTime(now.Add(-30 * time.Second).Format(time.RFC3339)); got != T.RelNow {
		t.Errorf("30с назад = %q, ожидалось %q", got, T.RelNow)
	}
	if got := relativeTime(now.Add(-3 * time.Hour).Format(time.RFC3339)); got == "" {
		t.Error("3ч назад дало пусто")
	}
	if got := relativeTime(""); got != "" {
		t.Errorf("пустая метка = %q, ожидалось пусто", got)
	}
	if got := relativeTime("не дата"); got != "" {
		t.Errorf("мусор = %q, ожидалось пусто", got)
	}
}

// TestCPUPercentFromStat проверяет расчёт загрузки CPU по двум замерам /proc/stat.
func TestCPUPercentFromStat(t *testing.T) {
	section := "cpu  1000 200 500 8000 100 0 50 0 0 0\ncpu  1100 210 550 8300 110 0 55 0 0 0"
	frac, ok := cpuPercentFromStat(section)
	if !ok || frac < 0.34 || frac > 0.35 {
		t.Fatalf("ожидалось ~0.347, получено %.4f (ok=%v)", frac, ok)
	}
	if _, ok := cpuPercentFromStat(""); ok {
		t.Error("пустая секция должна давать ok=false")
	}
}
