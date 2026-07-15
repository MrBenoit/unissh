package app

import (
	"os"
	"testing"
)

// TestMain фиксирует язык и тему до запуска тестов — чтобы рендер и строки были
// детерминированы независимо от окружения.
//
// И подменяет HOME на временный каталог: конфиг, known_hosts и журнал живут в
// ~/.config/unissh, и ни один тест не должен дотянуться до настоящих файлов
// пользователя. Отдельные тесты всё равно зовут t.Setenv("HOME", …) — это лишь
// общая страховка для тех, кто забыл.
func TestMain(m *testing.M) {
	setLang("en")
	applyTheme("blue")

	home, err := os.MkdirTemp("", "unissh-test-home-")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("HOME", home); err != nil {
		panic(err)
	}

	// Полноэкранные страницы не запускаем: у `go test` нет tty, и tea.Program
	// либо падал бы, либо сыпал управляющими последовательностями в вывод.
	// Сбор данных (load) при этом выполняется — именно он и интересен тестам.
	// Сама модель pageModel проверяется напрямую в page_test.go.
	runPage = func(title, status string, lines []string, load func() []string, keys []pageKey) string {
		if load != nil {
			load()
		}
		return pageBack
	}
	runToast = func([]string) {}
	runSpinner = func(_ string, fn func() error) error { return fn() }

	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}
