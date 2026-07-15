package app

// Мультивыбор в списке серверов: «space» отмечает сервер (на строке папки —
// всю группу), а правая-верхняя панель превращается в групповые действия.
//
// Выделение живёт только внутри одного запуска панели: после любого группового
// действия mainMenu пересоздаёт модель, и отметки сбрасываются — так удалённые
// или перемещённые серверы не остаются «выбранными» призраками.

import "fmt"

// selCount — сколько серверов отмечено.
func (m *panelsModel) selCount() int { return len(m.selected) }

// selectedServers возвращает отмеченные серверы в порядке конфига (а не экрана):
// так результат не зависит от текущей сортировки и фильтра.
func (m *panelsModel) selectedServers() []*Server {
	var out []*Server
	for _, s := range m.store.Servers {
		if m.selected[s.ID] {
			out = append(out, s)
		}
	}
	return out
}

// clearSelection снимает все отметки.
func (m *panelsModel) clearSelection() {
	if len(m.selected) > 0 {
		m.selected = map[string]bool{}
	}
}

// toggleSelect отмечает/снимает сервер под курсором. На строке папки —
// переключает всю группу: если отмечена хотя бы часть, добираем остальные,
// если отмечены все — снимаем.
func (m *panelsModel) toggleSelect() {
	if m.focus != fpServers {
		return
	}
	if m.selected == nil { // модель могли собрать в обход newPanelsModel
		m.selected = map[string]bool{}
	}
	if name, ok := m.onFolderRow(); ok {
		group := m.groupServers(name)
		all := len(group) > 0
		for _, s := range group {
			if !m.selected[s.ID] {
				all = false
				break
			}
		}
		for _, s := range group {
			if all {
				delete(m.selected, s.ID)
			} else {
				m.selected[s.ID] = true
			}
		}
		m.clampActionCur()
		return
	}
	s := m.curServer()
	if s == nil {
		return
	}
	if m.selected[s.ID] {
		delete(m.selected, s.ID)
	} else {
		m.selected[s.ID] = true
	}
	m.clampActionCur()
}

// selectMark — чекбокс перед именем сервера. Показываем его только когда
// выделение уже начато: пустые скобки у каждой строки зашумляли бы список.
func (m *panelsModel) selectMark(s *Server) string {
	if m.selCount() == 0 {
		return ""
	}
	if m.selected[s.ID] {
		return styPink.Render("[✓] ")
	}
	return styDim.Render("[ ] ")
}

// batchActions — действия над выделенной группой (правая-верхняя панель).
func batchActions() []menuItemAction {
	return []menuItemAction{
		{"batch_run", T.BatchRun},
		{"batch_move", T.BatchMove},
		{"batch_delete", T.BatchDelete},
		{"batch_clear", T.BatchClear},
	}
}

// curActions возвращает пункты правой-верхней панели: групповые, если что-то
// отмечено, иначе — действия сервера под курсором.
func (m *panelsModel) curActions() []menuItemAction {
	if m.selCount() > 0 {
		return batchActions()
	}
	if s := m.curServer(); s != nil {
		return serverActions(s)
	}
	return nil
}

// clampActionCur не даёт курсору действий уехать за конец списка, когда тот
// сменился с «действий сервера» на «групповые» (их меньше).
func (m *panelsModel) clampActionCur() {
	if n := len(m.curActions()); n > 0 && m.actionCur >= n {
		m.actionCur = n - 1
	}
}

// batchTitle — заголовок панели групповых действий со счётчиком.
func (m *panelsModel) batchTitle() string {
	return fmt.Sprintf(T.PanelBatchFmt, m.selCount())
}

// batchSummary — правая-нижняя панель в режиме выделения: перечень отмеченных.
// Живая статистика тут неуместна: выбран не один сервер, а группа.
func (m *panelsModel) batchSummary(w int) []string {
	sel := m.selectedServers()
	out := []string{styBold.Render(fmt.Sprintf(T.BatchSelectedFmt, len(sel))), ""}
	for _, s := range sel {
		out = append(out, styGreen.Render("✓ ")+styBold.Render(clip(s.Name, w-16))+
			"  "+styGrey.Render(clip(s.Host, 18)))
	}
	out = append(out, "")
	for _, l := range wrapText(T.BatchHint, w) {
		out = append(out, styDim.Render(l))
	}
	return out
}
