package app

// Модель строк левой-верхней панели: плоские папки-группы и серверы внутри них.
// Строка — либо заголовок папки, либо сервер. Фильтр, сортировка и избранное
// применяются внутри каждой группы.

import (
	"sort"
	"strings"
)

type rowKind int

const (
	rowServer rowKind = iota
	rowFolder
)

// listRow — строка списка: заголовок папки или сервер.
type listRow struct {
	kind   rowKind
	folder string  // имя папки; "" — группа «Без папки»
	server *Server // заполнен только для rowServer
}

// matchesFilter сообщает, подходит ли сервер под текущий фильтр
// (по имени, хосту или логину, без учёта регистра).
func (m *panelsModel) matchesFilter(s *Server) bool {
	q := strings.ToLower(strings.TrimSpace(m.filter))
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s.Name+" "+s.Host+" "+s.User), q)
}

// pingKey — ключ сортировки по пингу: онлайн с меньшим пингом раньше, офлайн в конце.
func (m *panelsModel) pingKey(s *Server) int64 {
	if pr, ok := m.statuses[s.ID]; ok && pr.Online {
		return pr.Ping.Microseconds()
	}
	return 1 << 50
}

// onlineKey — ключ сортировки по доступности: онлайн (0) раньше офлайн (1).
func (m *panelsModel) onlineKey(s *Server) int {
	if pr, ok := m.statuses[s.ID]; ok && pr.Online {
		return 0
	}
	return 1
}

// filterSort фильтрует и сортирует список по текущему режиму;
// избранные всегда поднимаются наверх своей группы.
func (m *panelsModel) filterSort(list []*Server) []*Server {
	var out []*Server
	for _, s := range list {
		if m.matchesFilter(s) {
			out = append(out, s)
		}
	}
	switch m.store.Settings.SortMode {
	case "name":
		sort.SliceStable(out, func(i, j int) bool {
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		})
	case "ping":
		sort.SliceStable(out, func(i, j int) bool { return m.pingKey(out[i]) < m.pingKey(out[j]) })
	case "online":
		sort.SliceStable(out, func(i, j int) bool { return m.onlineKey(out[i]) < m.onlineKey(out[j]) })
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Favorite && !out[j].Favorite })
	return out
}

// folderOf возвращает папку сервера, приводя неизвестную папку к корню —
// чтобы сервер не исчез из списка, если его папку удалили в обход RemoveFolder.
func (m *panelsModel) folderOf(s *Server) string {
	if s.Folder != "" && m.store.HasFolder(s.Folder) {
		return s.Folder
	}
	return ""
}

// groupServers возвращает отфильтрованные и отсортированные серверы группы.
func (m *panelsModel) groupServers(folder string) []*Server {
	var list []*Server
	for _, s := range m.store.Servers {
		if m.folderOf(s) == folder {
			list = append(list, s)
		}
	}
	return m.filterSort(list)
}

// expanded сообщает, развёрнута ли папка. Во время поиска разворачиваем всё —
// иначе совпадения прятались бы внутри свёрнутых групп.
func (m *panelsModel) expanded(folder string) bool {
	if strings.TrimSpace(m.filter) != "" {
		return true
	}
	return !m.store.FolderCollapsed(folder)
}

// rebuildRows пересобирает видимые строки и удерживает курсор в границах.
func (m *panelsModel) rebuildRows() {
	var rows []listRow

	if len(m.store.Folders) == 0 {
		// папок нет — плоский список серверов
		for _, s := range m.filterSort(m.store.Servers) {
			rows = append(rows, listRow{kind: rowServer, server: s})
		}
	} else {
		for _, f := range m.store.Folders {
			children := m.groupServers(f)
			if len(children) == 0 && strings.TrimSpace(m.filter) != "" {
				continue // при активном поиске пустые группы не показываем
			}
			rows = append(rows, listRow{kind: rowFolder, folder: f})
			if m.expanded(f) {
				for _, s := range children {
					rows = append(rows, listRow{kind: rowServer, folder: f, server: s})
				}
			}
		}
		if root := m.groupServers(""); len(root) > 0 {
			rows = append(rows, listRow{kind: rowFolder, folder: ""}) // «Без папки»
			if m.expanded("") {
				for _, s := range root {
					rows = append(rows, listRow{kind: rowServer, server: s})
				}
			}
		}
	}

	m.rows = rows
	if m.rowCur >= len(rows) {
		m.rowCur = len(rows) - 1
	}
	if m.rowCur < 0 {
		m.rowCur = 0
	}
}

// curRow возвращает строку под курсором (nil, если строк нет).
func (m *panelsModel) curRow() *listRow {
	if m.rowCur >= 0 && m.rowCur < len(m.rows) {
		return &m.rows[m.rowCur]
	}
	return nil
}

// curServer возвращает выделенный сервер (nil, если курсор стоит на папке).
func (m *panelsModel) curServer() *Server {
	if r := m.curRow(); r != nil && r.kind == rowServer {
		return r.server
	}
	return nil
}

// onFolderRow сообщает, стоит ли курсор на заголовке папки, и возвращает её имя.
func (m *panelsModel) onFolderRow() (string, bool) {
	if r := m.curRow(); r != nil && r.kind == rowFolder {
		return r.folder, true
	}
	return "", false
}

// folderLabel — подпись группы («Без папки» для корня).
func folderLabel(name string) string {
	if name == "" {
		return T.FolderRoot
	}
	return name
}

// --- режим перемещения сервера между папками ---

// moveTargets — папки в порядке отображения; корень («Без папки») последним.
func (m *panelsModel) moveTargets() []string {
	t := make([]string, 0, len(m.store.Folders)+1)
	t = append(t, m.store.Folders...)
	return append(t, "")
}

// focusServer ставит курсор на строку указанного сервера (если он виден).
func (m *panelsModel) focusServer(s *Server) {
	for i, r := range m.rows {
		if r.kind == rowServer && r.server == s {
			m.rowCur = i
			return
		}
	}
}

// startMove «захватывает» выделенный сервер: дальше он перемещается по папкам
// вместе с курсором. Ничего не делает, если папок нет или курсор не на сервере.
func (m *panelsModel) startMove() {
	s := m.curServer()
	if s == nil || len(m.moveTargets()) < 2 {
		return
	}
	m.movingFrom = m.folderOf(s) // нормализуем «призрачную» папку
	s.Folder = m.movingFrom
	m.moving = s
	m.rebuildRows()
	m.focusServer(s)
}

// shiftMove переносит схваченный сервер в следующую/предыдущую папку по кругу.
func (m *panelsModel) shiftMove(delta int) {
	if m.moving == nil {
		return
	}
	targets := m.moveTargets()
	cur := 0
	for i, f := range targets {
		if f == m.moving.Folder {
			cur = i
			break
		}
	}
	dst := targets[(cur+delta+len(targets))%len(targets)]
	m.moving.Folder = dst
	m.store.setCollapsed(dst, false) // целевую папку разворачиваем, иначе сервер не виден
	m.rebuildRows()
	m.focusServer(m.moving)
}

// commitMove фиксирует перемещение и сохраняет конфиг.
func (m *panelsModel) commitMove() {
	if m.moving == nil {
		return
	}
	s := m.moving
	m.moving = nil
	m.store.MustSave()
	m.rebuildRows()
	m.focusServer(s)
}

// cancelMove возвращает сервер в исходную папку (конфиг не сохраняем).
func (m *panelsModel) cancelMove() {
	if m.moving == nil {
		return
	}
	s := m.moving
	s.Folder = m.movingFrom
	m.moving = nil
	m.rebuildRows()
	m.focusServer(s)
}
