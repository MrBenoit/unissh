package core

import "testing"

// TestRemotePathArg: тильда остаётся живой (её раскрывает удалённый шелл),
// а всё опасное — закавычивается.
func TestRemotePathArg(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "."},
		{"  ", "."},
		{"~", "~/"},
		{"~/", "~/"},
		{"~/dumps", "~/dumps"},
		{"~/my dumps", `~/'my dumps'`},
		{"/var/log/syslog", "/var/log/syslog"},
		{"/tmp/a b", `'/tmp/a b'`},
		{"/tmp/x;rm -rf /", `'/tmp/x;rm -rf /'`},
		{"/tmp/$HOME", `'/tmp/$HOME'`},
		{"relative/path", "relative/path"},
	}
	for _, c := range cases {
		if got := RemotePathArg(c.in); got != c.want {
			t.Errorf("RemotePathArg(%q) = %q, ожидалось %q", c.in, got, c.want)
		}
	}
}

// TestRemotePathArgQuotesTildeTail: «~/» не должно попасть внутрь кавычек —
// иначе файл уедет в каталог с буквальным именем «~».
func TestRemotePathArgQuotesTildeTail(t *testing.T) {
	got := RemotePathArg("~/a b")
	if got[:2] != "~/" {
		t.Fatalf("тильда закавычена: %q", got)
	}
	if got == "~/a b" {
		t.Fatalf("хвост с пробелом обязан быть закавычен: %q", got)
	}
}

// TestLocalPathArg: двоеточие в первом сегменте scp принимает за «host:path».
func TestLocalPathArg(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "."},
		{"/abs/path", "/abs/path"},
		{"./rel", "./rel"},
		{"../up", "../up"},
		{"plain", "plain"},
		{"dir/file", "dir/file"},
		{"weird:name", "./weird:name"},
		{"weird:name/inner", "./weird:name/inner"},
		{"dir/weird:name", "dir/weird:name"}, // двоеточие не в первом сегменте — не путает scp
	}
	for _, c := range cases {
		if got := LocalPathArg(c.in); got != c.want {
			t.Errorf("LocalPathArg(%q) = %q, ожидалось %q", c.in, got, c.want)
		}
	}
}
