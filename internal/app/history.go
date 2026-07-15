package app

// Журнал подключений и относительное время «последнего входа».

import (
	"fmt"
	"path/filepath"
	"time"
)

// historyPath — путь к журналу подключений (по строке JSON на запись).
func historyPath() string {
	return filepath.Join(configDir(), "history.jsonl")
}

// recordConnection фиксирует подключение: обновляет last_connected сервера
// (для сортировки и колонки «последний вход») и дописывает запись в журнал.
func recordConnection(st *Store, s *Server) {
	now := time.Now()
	s.LastConnected = now.Format(time.RFC3339)
	if st != nil {
		st.MustSave()
	}
	journalAppend(journalEntry{
		At: now.Format(time.RFC3339), Kind: journalConnect,
		Server: s.Name, Host: s.Host,
	})
}

// relativeTime возвращает компактную локализованную строку «сколько назад»
// для метки времени RFC3339 (пусто, если метка не распознана).
func relativeTime(rfc3339 string) string {
	if rfc3339 == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return T.RelNow
	case d < time.Hour:
		return fmt.Sprintf(T.RelMin, int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf(T.RelHour, int(d.Hours()))
	default:
		return fmt.Sprintf(T.RelDay, int(d.Hours()/24))
	}
}
