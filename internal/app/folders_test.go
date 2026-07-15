package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func folderStore() *Store {
	st := &Store{Config: Config{
		Folders: []string{"prod", "staging"},
		Servers: []*Server{
			{ID: "1", Name: "web-1", Host: "10.0.0.1", User: "root", Folder: "prod"},
			{ID: "2", Name: "db-1", Host: "10.0.0.2", User: "root", Folder: "prod"},
			{ID: "3", Name: "test-1", Host: "10.0.1.1", User: "root", Folder: "staging"},
			{ID: "4", Name: "sandbox", Host: "10.0.9.9", User: "deploy"},
		},
	}}
	st.path = "/dev/null" // MustSave в тестах не должен писать реальный конфиг
	return st
}

// --- операции стора ---

func TestStoreFolderOps(t *testing.T) {
	st := folderStore()
	if !st.HasFolder("prod") || st.HasFolder("nope") {
		t.Fatal("HasFolder")
	}
	if st.AddFolder("prod") {
		t.Error("дубликат папки не должен создаваться")
	}
	if st.AddFolder("  ") {
		t.Error("пустое имя не должно создаваться")
	}
	if !st.AddFolder("dev") || !st.HasFolder("dev") {
		t.Error("новая папка должна создаваться")
	}
	if got := len(st.FolderServers("prod")); got != 2 {
		t.Errorf("FolderServers(prod) = %d, want 2", got)
	}
	if got := len(st.FolderServers("")); got != 1 {
		t.Errorf("FolderServers(корень) = %d, want 1", got)
	}

	// свёртка
	if st.FolderCollapsed("prod") {
		t.Error("по умолчанию папка развёрнута")
	}
	st.ToggleFolder("prod")
	if !st.FolderCollapsed("prod") {
		t.Error("ToggleFolder должен свернуть")
	}
	st.ToggleFolder("prod")
	if st.FolderCollapsed("prod") {
		t.Error("ToggleFolder должен развернуть обратно")
	}

	// удаление папки возвращает серверы в корень
	st.RemoveFolder("prod")
	if st.HasFolder("prod") {
		t.Error("папка должна удалиться")
	}
	if got := len(st.FolderServers("")); got != 3 { // sandbox + web-1 + db-1
		t.Errorf("после удаления в корне %d серверов, want 3", got)
	}
}

// --- строки списка ---

func TestRebuildRowsGrouping(t *testing.T) {
	setLang("ru")
	m := newPanelsModel(folderStore())

	// заголовок папки + её серверы, в конце — группа «Без папки»
	var kinds []string
	for _, r := range m.rows {
		if r.kind == rowFolder {
			kinds = append(kinds, "F:"+folderLabel(r.folder))
		} else {
			kinds = append(kinds, "S:"+r.server.Name)
		}
	}
	want := []string{"F:prod", "S:web-1", "S:db-1", "F:staging", "S:test-1", "F:Без папки", "S:sandbox"}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Fatalf("строки:\n got %v\nwant %v", kinds, want)
	}
}

func TestRebuildRowsCollapse(t *testing.T) {
	setLang("ru")
	st := folderStore()
	st.setCollapsed("prod", true)
	m := newPanelsModel(st)
	for _, r := range m.rows {
		if r.kind == rowServer && r.folder == "prod" {
			t.Fatal("серверы свёрнутой папки не должны попадать в строки")
		}
	}
	// заголовок свёрнутой папки остаётся
	if _, ok := findFolderRow(m, "prod"); !ok {
		t.Fatal("заголовок свёрнутой папки должен остаться")
	}
}

func TestRebuildRowsFilterExpandsAndHidesEmpty(t *testing.T) {
	setLang("ru")
	st := folderStore()
	st.setCollapsed("prod", true) // при поиске свёртка игнорируется
	m := newPanelsModel(st)
	m.filter = "web"
	m.rebuildRows()

	if _, ok := findFolderRow(m, "staging"); ok {
		t.Error("папка без совпадений должна скрываться при поиске")
	}
	found := false
	for _, r := range m.rows {
		if r.kind == rowServer && r.server.Name == "web-1" {
			found = true
		}
	}
	if !found {
		t.Error("совпадение должно показываться даже в свёрнутой папке")
	}
}

func TestRebuildRowsFlatWithoutFolders(t *testing.T) {
	st := &Store{Config: Config{Servers: []*Server{{ID: "1", Name: "a"}, {ID: "2", Name: "b"}}}}
	m := newPanelsModel(st)
	for _, r := range m.rows {
		if r.kind == rowFolder {
			t.Fatal("без папок список должен быть плоским")
		}
	}
	if len(m.rows) != 2 {
		t.Fatalf("строк %d, want 2", len(m.rows))
	}
}

// TestFolderOfUnknownGoesToRoot: сервер со ссылкой на несуществующую папку
// не должен исчезать из списка.
func TestFolderOfUnknownGoesToRoot(t *testing.T) {
	st := folderStore()
	st.Servers[0].Folder = "ghost"
	m := newPanelsModel(st)
	if m.folderOf(st.Servers[0]) != "" {
		t.Fatal("неизвестная папка должна приводиться к корню")
	}
}

func findFolderRow(m *panelsModel, name string) (int, bool) {
	for i, r := range m.rows {
		if r.kind == rowFolder && r.folder == name {
			return i, true
		}
	}
	return 0, false
}

// --- инлайновые промпты папок (без выхода из TUI) ---

func promptModel(t *testing.T) (*panelsModel, *Store) {
	t.Helper()
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	st := loadStore()
	st.Folders = []string{"prod"}
	st.Servers = append(st.Servers, &Server{ID: "1", Name: "web-1", Folder: "prod"})
	m := newPanelsModel(st)
	m.focus = fpServers
	m.width, m.height = 100, 30
	return m, st
}

// typeInto печатает текст в активный промпт.
func typeInto(m *panelsModel, s string) {
	for _, r := range s {
		m.handlePromptKey(keyMsg(string(r)))
	}
}

func TestPromptFolderNew(t *testing.T) {
	m, st := promptModel(t)
	m.askFolderNew()
	if m.prompt == nil {
		t.Fatal("«n» должен открыть промпт")
	}
	typeInto(m, "staging")
	_ = m.renderPrompt() // не паникует
	m.handlePromptKey(keyMsg("enter"))
	if m.prompt != nil || !st.HasFolder("staging") {
		t.Fatal("Enter должен создать папку и закрыть промпт")
	}
}

func TestPromptFolderNewDuplicate(t *testing.T) {
	m, _ := promptModel(t)
	m.askFolderNew()
	typeInto(m, "prod") // уже существует
	m.handlePromptKey(keyMsg("enter"))
	if m.prompt == nil {
		t.Fatal("промпт должен остаться открытым при ошибке")
	}
	if m.prompt.err == "" {
		t.Fatal("ожидалась ошибка «папка уже есть»")
	}
	m.handlePromptKey(keyMsg("esc"))
	if m.prompt != nil {
		t.Fatal("Esc должен закрыть промпт")
	}
}

func TestPromptFolderRename(t *testing.T) {
	m, st := promptModel(t)
	m.rowCur = 0 // строка папки prod
	m.askFolderRename()
	if m.prompt == nil || m.prompt.target != "prod" {
		t.Fatal("«r» на папке должен открыть переименование")
	}
	// стираем "prod" и печатаем новое имя
	for range "prod" {
		m.handlePromptKey(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	typeInto(m, "production")
	m.handlePromptKey(keyMsg("enter"))

	if st.HasFolder("prod") || !st.HasFolder("production") {
		t.Fatalf("папка не переименована: %v", st.Folders)
	}
	if st.Servers[0].Folder != "production" {
		t.Fatalf("сервер должен переехать в новую папку, got %q", st.Servers[0].Folder)
	}
}

func TestPromptFolderDelete(t *testing.T) {
	m, st := promptModel(t)
	m.rowCur = 0
	m.askFolderDelete()
	if m.prompt == nil || m.prompt.kind != promptFolderDelete {
		t.Fatal("«d» на папке должен открыть подтверждение")
	}
	_ = m.renderPrompt()
	m.handlePromptKey(keyMsg("y"))
	if st.HasFolder("prod") {
		t.Fatal("папка должна удалиться")
	}
	if st.Servers[0].Folder != "" {
		t.Fatalf("сервер должен вернуться в корень, got %q", st.Servers[0].Folder)
	}
}

func TestPromptFolderDeleteCancel(t *testing.T) {
	m, st := promptModel(t)
	m.rowCur = 0
	m.askFolderDelete()
	m.handlePromptKey(keyMsg("n"))
	if m.prompt != nil || !st.HasFolder("prod") {
		t.Fatal("«n» должен отменить удаление")
	}
}

// «Без папки» нельзя переименовать или удалить.
func TestPromptRootFolderProtected(t *testing.T) {
	m, _ := promptModel(t)
	m.store.Servers = append(m.store.Servers, &Server{ID: "2", Name: "sandbox"})
	m.rebuildRows()
	idx, ok := findFolderRow(m, "")
	if !ok {
		t.Fatal("ожидалась группа «Без папки»")
	}
	m.rowCur = idx
	m.askFolderRename()
	m.askFolderDelete()
	if m.prompt != nil {
		t.Fatal("корневую группу нельзя переименовать/удалить")
	}
}

// --- переименование и порядок папок ---

func TestStoreRenameFolder(t *testing.T) {
	st := folderStore()
	st.setCollapsed("prod", true)
	if !st.RenameFolder("prod", "production") {
		t.Fatal("переименование должно пройти")
	}
	if st.HasFolder("prod") || !st.HasFolder("production") {
		t.Fatalf("список папок: %v", st.Folders)
	}
	if !st.FolderCollapsed("production") {
		t.Error("свёрнутость должна переехать на новое имя")
	}
	if got := len(st.FolderServers("production")); got != 2 {
		t.Errorf("серверы должны переехать: %d", got)
	}
	if st.RenameFolder("production", "staging") {
		t.Error("занятое имя должно отклоняться")
	}
	if st.RenameFolder("", "x") || st.RenameFolder("nope", "x") {
		t.Error("корень и несуществующую папку переименовать нельзя")
	}
}

func TestStoreMoveFolder(t *testing.T) {
	st := folderStore() // [prod, staging]
	if !st.MoveFolder("staging", -1) {
		t.Fatal("staging должен подняться")
	}
	if st.Folders[0] != "staging" || st.Folders[1] != "prod" {
		t.Fatalf("порядок: %v", st.Folders)
	}
	if st.MoveFolder("staging", -1) {
		t.Error("выше первой позиции двигать нельзя")
	}
	if st.MoveFolder("prod", 1) {
		t.Error("ниже последней позиции двигать нельзя")
	}
	if st.MoveFolder("nope", 1) {
		t.Error("несуществующую папку двигать нельзя")
	}
}

// --- интерактивный режим перемещения ---

// TestMoveModeCommit: «m» захватывает сервер, ↑↓ ведут его по папкам, Enter сохраняет.
func TestMoveModeCommit(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	st := loadStore()
	st.Folders = []string{"prod", "staging"}
	s := &Server{ID: "1", Name: "web-1", Folder: "prod"}
	st.Servers = append(st.Servers, s)

	m := newPanelsModel(st)
	m.focus = fpServers
	m.focusServer(s)

	m.startMove()
	if m.moving != s {
		t.Fatal("«m» должен захватить сервер под курсором")
	}
	// цели по порядку: prod, staging, «Без папки»
	m.shiftMove(1)
	if s.Folder != "staging" {
		t.Fatalf("после ↓ папка = %q, want staging", s.Folder)
	}
	m.shiftMove(1)
	if s.Folder != "" {
		t.Fatalf("после второго ↓ ожидался корень, got %q", s.Folder)
	}
	// курсор должен ехать вместе с сервером
	if m.curServer() != s {
		t.Fatal("курсор должен оставаться на перемещаемом сервере")
	}
	m.commitMove()
	if m.moving != nil || s.Folder != "" {
		t.Fatal("Enter должен зафиксировать перемещение")
	}
}

// TestMoveModeCancel: Esc возвращает сервер в исходную папку.
func TestMoveModeCancel(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	st := loadStore()
	st.Folders = []string{"prod", "staging"}
	s := &Server{ID: "1", Name: "web-1", Folder: "prod"}
	st.Servers = append(st.Servers, s)

	m := newPanelsModel(st)
	m.focus = fpServers
	m.focusServer(s)
	m.startMove()
	m.shiftMove(1)
	if s.Folder == "prod" {
		t.Fatal("сервер должен был сдвинуться")
	}
	m.cancelMove()
	if m.moving != nil {
		t.Fatal("Esc должен выйти из режима перемещения")
	}
	if s.Folder != "prod" {
		t.Fatalf("Esc должен вернуть исходную папку, got %q", s.Folder)
	}
}

// TestMoveModeNoFolders: без папок перемещать некуда — режим не включается.
func TestMoveModeNoFolders(t *testing.T) {
	st := &Store{Config: Config{Servers: []*Server{{ID: "1", Name: "a"}}}}
	st.path = "/dev/null"
	m := newPanelsModel(st)
	m.focus = fpServers
	m.startMove()
	if m.moving != nil {
		t.Fatal("без папок режим перемещения включаться не должен")
	}
}

// TestPickFolderNewOption проверяет ветку «＋ Новая папка…».
func TestPickFolderNewOption(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	st := loadStore()

	origSelect, origInput := runSelect, runInput
	defer func() { runSelect, runInput = origSelect, origInput }()

	// последний пункт — «Новая папка…»
	runSelect = func(title, header string, labels, descs []string, initial int) int {
		return len(labels) - 1
	}
	runInput = func(title, def string, password bool, validate func(string) error) (string, bool) {
		return "fresh", true
	}

	got, ok := pickFolder(st, "t")
	if !ok || got != "fresh" || !st.HasFolder("fresh") {
		t.Fatalf("pickFolder(новая) = %q ok=%v, папка создана=%v", got, ok, st.HasFolder("fresh"))
	}
}
