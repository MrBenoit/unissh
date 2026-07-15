package app

// Групповые действия над отмеченными серверами (мультивыбор «space» в списке).
// Все три — разрушительные или дорогие, поэтому каждое показывает список целей
// и спрашивает подтверждение один раз, а не по серверу.

import (
	"fmt"
	"strings"
)

// batchNames — перечень имён целей для подтверждающих вопросов.
func batchNames(targets []*Server) string {
	names := make([]string, len(targets))
	for i, s := range targets {
		names[i] = s.Name
	}
	return strings.Join(names, ", ")
}

// batchRun выполняет одну команду на всех отмеченных серверах параллельно.
func batchRun(st *Store, targets []*Server) {
	if len(targets) == 0 {
		return
	}
	command, ok := pickCommand(st)
	if !ok || command == "" {
		return
	}
	runOnTargets(targets, command)
}

// batchMove переносит отмеченные серверы в выбранную папку (можно создать новую).
func batchMove(st *Store, targets []*Server) {
	if len(targets) == 0 {
		return
	}
	folder, ok := pickFolder(st, fmt.Sprintf(T.BatchMoveTitleFmt, len(targets)))
	if !ok {
		return
	}
	for _, s := range targets {
		s.Folder = folder
	}
	st.MustSave()
	showToast(styGreen.Render(fmt.Sprintf(T.BatchMovedFmt, len(targets), folderLabel(folder))))
}

// batchDelete удаляет отмеченные серверы: пароли из Keychain, отпечатки хостов
// и записи конфига. Подтверждение спрашивается один раз на всю группу, но текст
// перечисляет имена — стереть половину парка «вслепую» не получится.
func batchDelete(st *Store, targets []*Server) {
	if len(targets) == 0 {
		return
	}
	yes, ok := askConfirm(fmt.Sprintf(T.BatchDeleteAskFmt, len(targets), batchNames(targets)), false)
	if !ok || !yes {
		return
	}
	for _, s := range targets {
		secretDelete(s.ID, s.User)
		secretDelete(s.ID, "root")
		forgetHostKey(s.Host, s.Port)
		st.Remove(s.ID)
	}
	st.MustSave()
	maybeRefreshSSHConfig(st)
	showToast(styGreen.Render(fmt.Sprintf(T.BatchDeletedFmt, len(targets))))
}
