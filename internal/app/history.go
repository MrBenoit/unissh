package app

// Журнал подключений и относительное время «последнего входа».

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// historyPath — путь к журналу подключений (по строке JSON на запись).
func historyPath() string {
	return filepath.Join(configDir(), "history.jsonl")
}

// recordConnection фиксирует подключение: обновляет last_connected сервера
// (для сортировки и колонки «последний вход») и дописывает запись в журнал.
func recordConnection(s *Server) {
	now := time.Now()
	s.LastConnected = now.Format(time.RFC3339)
	if gStore != nil {
		gStore.MustSave()
	}
	line, err := json.Marshal(map[string]string{
		"server": s.Name,
		"host":   s.Host,
		"at":     now.Format(time.RFC3339),
	})
	if err != nil {
		return
	}
	f, err := os.OpenFile(historyPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(line, '\n'))
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
