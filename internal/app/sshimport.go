package app

// Импорт серверов из ~/.ssh/config — снимает главный барьер входа: пользователь
// с готовым парком серверов не вбивает каждый вручную.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// sshConfigHost — распарсенный Host-блок из ~/.ssh/config.
type sshConfigHost struct {
	Alias     string
	HostName  string
	Port      int
	User      string
	Key       string
	ProxyJump string
}

// parseSSHConfig разбирает ~/.ssh/config и возвращает Host-блоки с конкретным
// (не-wildcard) именем. Директива Include не разворачивается — чтобы не
// затянуть собственный экспорт unissh обратно.
func parseSSHConfig(path string) ([]sshConfigHost, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var all []sshConfigHost // включая wildcard-блоки (для дефолтов Host *)
	var cur *sshConfigHost
	flush := func() {
		if cur != nil && cur.Alias != "" {
			all = append(all, *cur)
		}
		cur = nil
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, _ := strings.Cut(strings.ReplaceAll(line, "=", " "), " ")
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		switch key {
		case "host":
			flush()
			// берём только первый шаблон имени
			alias := strings.Fields(val)
			if len(alias) > 0 {
				cur = &sshConfigHost{Alias: alias[0], Port: 22}
			}
		case "hostname":
			if cur != nil {
				cur.HostName = val
			}
		case "port":
			if cur != nil {
				if p, err := strconv.Atoi(val); err == nil {
					cur.Port = p
				}
			}
		case "user":
			if cur != nil {
				cur.User = val
			}
		case "identityfile":
			if cur != nil {
				cur.Key = val
			}
		case "proxyjump":
			if cur != nil {
				cur.ProxyJump = val
			}
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, err
	}

	// дефолты из блока «Host *» применяем к конкретным хостам, где значения пусты
	var def sshConfigHost
	for _, h := range all {
		if h.Alias == "*" {
			def = h
		}
	}
	var hosts []sshConfigHost
	for _, h := range all {
		if strings.ContainsAny(h.Alias, "*?") {
			continue // wildcard-блоки не импортируем как серверы
		}
		if h.User == "" {
			h.User = def.User
		}
		if h.Key == "" {
			h.Key = def.Key
		}
		if h.ProxyJump == "" {
			h.ProxyJump = def.ProxyJump
		}
		hosts = append(hosts, h)
	}
	return hosts, nil
}

// importFromSSHConfig — пункт меню: показывает найденные хосты (кроме уже
// добавленных) чекбоксами и заносит выбранные в конфиг unissh.
func importFromSSHConfig(st *Store) {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".ssh", "config")
	hosts, err := parseSSHConfig(path)
	if err != nil {
		fmt.Println(styRed.Render(fmt.Sprintf(T.ImportReadErr, err)))
		pressEnter()
		return
	}

	// исключаем уже добавленные (по host:port+user) и пустые записи
	var opts []multiOption
	var candidates []sshConfigHost
	for _, h := range hosts {
		hostName := h.HostName
		if hostName == "" {
			hostName = h.Alias
		}
		user := h.User
		if user == "" {
			user = st.Settings.DefaultUser
		}
		if serverExists(st, hostName, h.Port, user) {
			continue
		}
		label := fmt.Sprintf("%s  (%s@%s:%d)", h.Alias, user, hostName, h.Port)
		opts = append(opts, multiOption{Label: label, Value: strconv.Itoa(len(candidates)), Selected: true})
		candidates = append(candidates, h)
	}
	if len(candidates) == 0 {
		fmt.Println(styYellow.Render(T.ImportNothing))
		pressEnter()
		return
	}

	picked, ok := askMulti(fmt.Sprintf(T.ImportTitle, len(candidates)), opts)
	if !ok || len(picked) == 0 {
		return
	}
	pickedSet := map[string]bool{}
	for _, p := range picked {
		pickedSet[p] = true
	}

	added := 0
	for i, h := range candidates {
		if !pickedSet[strconv.Itoa(i)] {
			continue
		}
		hostName := h.HostName
		if hostName == "" {
			hostName = h.Alias
		}
		user := h.User
		if user == "" {
			user = st.Settings.DefaultUser
		}
		auth := "password"
		if h.Key != "" {
			auth = "key"
		}
		st.Servers = append(st.Servers, &Server{
			ID: newID(), Name: uniqueName(st, h.Alias), Host: hostName, Port: h.Port,
			User: user, Key: h.Key, Auth: auth, ProxyJump: h.ProxyJump, Tags: []string{},
		})
		added++
	}
	st.MustSave()
	maybeRefreshSSHConfig(st)
	fmt.Println(styGreen.Render(fmt.Sprintf(T.ImportAdded, added)))
	time.Sleep(900 * time.Millisecond)
}

// serverExists сообщает, есть ли уже сервер с таким host:port и пользователем.
func serverExists(st *Store, host string, port int, user string) bool {
	for _, s := range st.Servers {
		if s.Host == host && s.Port == port && s.User == user {
			return true
		}
	}
	return false
}

// uniqueName подбирает свободное имя на основе базового (base, base-2, base-3…).
func uniqueName(st *Store, base string) string {
	if base == "" {
		base = "server"
	}
	name := base
	for i := 2; st.NameTaken(name, ""); i++ {
		name = fmt.Sprintf("%s-%d", base, i)
	}
	return name
}
