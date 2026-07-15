package app

import (
	"strings"
	"testing"
)

// TestSettingsMenuFlow демонстрирует шов run*→var: реальный мастер settingsMenu
// (tools.go) прогоняется без TTY через подменённые runSelect/runInput.
func TestSettingsMenuFlow(t *testing.T) {
	setLang("ru")
	applyTheme("blue")
	t.Setenv("HOME", t.TempDir())
	st := loadStore()

	origSelect, origInput := runSelect, runInput
	defer func() { runSelect, runInput = origSelect, origInput }()

	// Ищем пункт по подписи, а не по индексу — тест не ломается при смене
	// порядка настроек. Первый вызов выбирает «Пользователь по умолчанию»,
	// второй возвращает -1 (выход из меню).
	calls := 0
	runSelect = func(title, header string, labels, descs []string, initial int) int {
		calls++
		if calls > 1 {
			return -1
		}
		for i, l := range labels {
			if strings.HasPrefix(l, T.SetDefaultUser) {
				return i
			}
		}
		t.Fatalf("не нашли пункт %q среди %v", T.SetDefaultUser, labels)
		return -1
	}
	runInput = func(title, def string, password bool, validate func(string) error) (string, bool) {
		return "deployer", true
	}

	settingsMenu(st)

	if st.Settings.DefaultUser != "deployer" {
		t.Fatalf("DefaultUser = %q, ожидалось deployer", st.Settings.DefaultUser)
	}
}
