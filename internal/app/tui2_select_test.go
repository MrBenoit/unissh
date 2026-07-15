package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// selStore — два сервера в одной папке и один в корне: хватает и на группу,
// и на «отметить одиночку».
func selStore() *Store {
	st := demoStore()
	st.Folders = []string{"prod"}
	st.Servers[0].Folder = "prod"
	st.Servers[1].Folder = "prod"
	st.Servers = append(st.Servers, &Server{ID: "3", Name: "solo", Host: "10.0.0.3", User: "root", Port: 22, Auth: "key"})
	return st
}

// rowIndexOf находит строку сервера по имени.
func rowIndexOf(m *panelsModel, name string) int {
	for i, r := range m.rows {
		if r.kind == rowServer && r.server.Name == name {
			return i
		}
	}
	return -1
}

// rowIndexOfFolder находит строку папки.
func rowIndexOfFolder(m *panelsModel, name string) int {
	for i, r := range m.rows {
		if r.kind == rowFolder && r.folder == name {
			return i
		}
	}
	return -1
}

// TestSelectToggleServer: «space» отмечает и снимает сервер под курсором.
func TestSelectToggleServer(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	m := newPanelsModel(selStore())
	m.rowCur = rowIndexOf(m, "solo")

	m.toggleSelect()
	if m.selCount() != 1 || !m.selected["3"] {
		t.Fatalf("сервер не отмечен: %v", m.selected)
	}
	m.toggleSelect()
	if m.selCount() != 0 {
		t.Fatalf("повторный space должен снимать отметку: %v", m.selected)
	}
}

// TestSelectToggleFolder: «space» на папке отмечает всю группу, повторный —
// снимает; частично отмеченная группа сначала добирается целиком.
func TestSelectToggleFolder(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	m := newPanelsModel(selStore())

	m.rowCur = rowIndexOf(m, "web-1")
	m.toggleSelect() // отметили один сервер группы
	m.rowCur = rowIndexOfFolder(m, "prod")
	m.toggleSelect() // группа отмечена частично → добираем остальные
	if m.selCount() != 2 {
		t.Fatalf("ожидалось 2 отмеченных, получено %d", m.selCount())
	}
	m.toggleSelect() // теперь отмечена целиком → снимаем
	if m.selCount() != 0 {
		t.Fatalf("группа должна сняться целиком: %v", m.selected)
	}
}

// TestSelectSwitchesActionsPanel: при выделении правая панель показывает
// групповые действия, а курсор действий не уезжает за их конец.
func TestSelectSwitchesActionsPanel(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	m := newPanelsModel(selStore())
	m.rowCur = rowIndexOf(m, "solo")

	if len(m.curActions()) != len(serverActions(m.curServer())) {
		t.Fatal("без выделения — действия одного сервера")
	}
	m.actionCur = len(serverActions(m.curServer())) - 1 // курсор на последнем действии
	m.toggleSelect()

	acts := m.curActions()
	if len(acts) != len(batchActions()) {
		t.Fatalf("с выделением ожидались групповые действия: %v", acts)
	}
	if m.actionCur >= len(acts) {
		t.Fatalf("курсор действий (%d) вышел за конец групповых (%d)", m.actionCur, len(acts))
	}
}

// TestSelectedServersOrder: порядок — по конфигу, а не по экрану, чтобы
// сортировка списка не меняла порядок выполнения групповой команды.
func TestSelectedServersOrder(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	m := newPanelsModel(selStore())
	m.selected = map[string]bool{"3": true, "1": true}

	sel := m.selectedServers()
	if len(sel) != 2 || sel[0].ID != "1" || sel[1].ID != "3" {
		t.Fatalf("порядок отмеченных: %v", sel)
	}
}

// TestSelectBatchResult: Enter по групповому действию выходит из TUI с целями,
// а «Снять выделение» остаётся внутри TUI и просто чистит отметки.
func TestSelectBatchResult(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	m := newPanelsModel(selStore())
	m.rowCur = rowIndexOf(m, "solo")
	m.toggleSelect()
	m.focus = fpActions

	m.actionCur = 0 // batch_run
	if _, cmd := m.activate(); cmd == nil {
		t.Fatal("групповое действие должно завершать TUI")
	}
	if m.result == nil || m.result.action != "batch_run" || len(m.result.servers) != 1 {
		t.Fatalf("результат: %+v", m.result)
	}

	m2 := newPanelsModel(selStore())
	m2.rowCur = rowIndexOf(m2, "solo")
	m2.toggleSelect()
	m2.focus = fpActions
	for i, a := range batchActions() {
		if a.action == "batch_clear" {
			m2.actionCur = i
		}
	}
	if _, cmd := m2.activate(); cmd != nil {
		t.Fatal("«снять выделение» не должно закрывать TUI")
	}
	if m2.selCount() != 0 || m2.focus != fpServers {
		t.Fatalf("после «снять выделение»: sel=%d focus=%v", m2.selCount(), m2.focus)
	}
}

// TestSelectEscClears: esc снимает выделение раньше, чем закрывает приложение.
func TestSelectEscClears(t *testing.T) {
	setLang("ru")
	t.Setenv("HOME", t.TempDir())
	m := newPanelsModel(selStore())
	var mm tea.Model = m
	mm, _ = mm.Update(tea.WindowSizeMsg{Width: 110, Height: 32})
	mm, _ = mm.Update(keyMsg(" "))
	if mm.(*panelsModel).selCount() == 0 {
		t.Fatal("space должен отмечать")
	}
	mm, cmd := mm.Update(keyMsg("esc"))
	if cmd != nil {
		t.Fatal("esc с выделением не должен закрывать TUI")
	}
	if mm.(*panelsModel).selCount() != 0 {
		t.Fatal("esc должен снять выделение")
	}
}

// TestSelectRender: с выделением справа появляется панель группы и чекбоксы,
// а внизу — сводка вместо живой статистики.
func TestSelectRender(t *testing.T) {
	setLang("ru")
	applyTheme("blue")
	t.Setenv("HOME", t.TempDir())
	m := newPanelsModel(selStore())
	m.width, m.height = 120, 34
	m.rowCur = rowIndexOf(m, "solo")

	if strings.Contains(strings.Join(m.serverItems(), "\n"), "[ ]") {
		t.Fatal("пока ничего не отмечено, чекбоксов быть не должно")
	}
	m.toggleSelect()

	items := strings.Join(m.serverItems(), "\n")
	if !strings.Contains(items, "[✓]") || !strings.Contains(items, "[ ]") {
		t.Fatalf("ожидались отмеченный и пустой чекбоксы:\n%s", items)
	}
	view := m.View()
	if !strings.Contains(view, T.PanelSelected) {
		t.Error("нет панели «ВЫДЕЛЕНО»")
	}
	if !strings.Contains(view, "ГРУППА") {
		t.Error("нет заголовка групповых действий")
	}
	if !strings.Contains(m.footer(), "1") {
		t.Errorf("футер должен показывать счётчик: %q", m.footer())
	}
}

// TestBatchNames: имена целей склеиваются через запятую — их видит пользователь
// в вопросе на удаление.
func TestBatchNames(t *testing.T) {
	got := batchNames([]*Server{{Name: "a"}, {Name: "b"}})
	if got != "a, b" {
		t.Fatalf("batchNames = %q", got)
	}
}

// TestBatchEmptyNoop: пустой список целей ничего не делает и никого не спрашивает.
func TestBatchEmptyNoop(t *testing.T) {
	setLang("ru")
	st := demoStore()
	before := len(st.Servers)
	batchRun(st, nil)
	batchMove(st, nil)
	batchDelete(st, nil)
	if len(st.Servers) != before {
		t.Fatal("пустая группа не должна ничего менять")
	}
}
