package core

import "testing"

// TestResolveVersion: линкер важнее вшитой информации о сборке, а «go install
// module@version» (где ldflags не применяются) всё равно должен показывать версию.
func TestResolveVersion(t *testing.T) {
	cases := []struct {
		name             string
		ldflags, buildIn string
		want             string
	}{
		// goreleaser / make build: линкер подставил версию — она и побеждает
		{"линкер задал версию", "v1.0.0", "", "v1.0.0"},
		{"линкер важнее build info", "v1.2.3", "v1.0.0", "v1.2.3"},

		// go install module@version: ldflags не применились, версия во build info
		{"go install", DevVersion, "v1.0.0", "v1.0.0"},
		{"go install, ldflags пуст", "", "v1.0.0", "v1.0.0"},

		// обычный go build: версии нет ниоткуда
		{"локальная сборка", DevVersion, develBuildInfo, DevVersion},
		{"всё пусто", "", "", DevVersion},
		{"build info пуст", DevVersion, "", DevVersion},

		// «(devel)» — это не версия, а признак её отсутствия
		{"(devel) не версия", "", develBuildInfo, DevVersion},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ResolveVersion(c.ldflags, c.buildIn); got != c.want {
				t.Errorf("ResolveVersion(%q, %q) = %q, ожидалось %q",
					c.ldflags, c.buildIn, got, c.want)
			}
		})
	}
}
