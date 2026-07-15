package app

// Выбор папки для сервера — используется мастерами добавления и правки.
// Создание/переименование/удаление папок живут прямо в панельном экране
// (tui2_prompt.go), поэтому диалогов там больше нет.

import (
	"errors"
	"strings"
)

// folderNewSentinel — служебное значение пункта «＋ Новая папка…» в списке выбора.
const folderNewSentinel = "\x00new-folder"

// validateFolderName проверяет имя новой папки: непустое и ещё не занятое.
func validateFolderName(st *Store) func(string) error {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return errors.New(T.ErrNameEmpty)
		}
		if st.HasFolder(s) {
			return errors.New(T.ErrFolderExists)
		}
		return nil
	}
}

// askNewFolderName спрашивает имя новой папки и создаёт её. Возвращает имя и ok.
func askNewFolderName(st *Store) (string, bool) {
	name, ok := askInput(T.AskFolderName, "", validateFolderName(st))
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return "", false
	}
	st.AddFolder(name) // no-op, если уже существует
	return name, true
}

// pickFolder показывает выбор папки (корень + существующие + «новая»).
// Возвращает имя выбранной папки ("" — корень) и ok.
func pickFolder(st *Store, title string) (string, bool) {
	opts := []selOption[string]{opt(T.FolderRoot, "")}
	for _, f := range st.Folders {
		opts = append(opts, opt(f, f))
	}
	opts = append(opts, opt(T.FolderNewOption, folderNewSentinel))

	target, ok := askSelect(title, opts)
	if !ok {
		return "", false
	}
	if target == folderNewSentinel {
		return askNewFolderName(st)
	}
	return target, true
}
