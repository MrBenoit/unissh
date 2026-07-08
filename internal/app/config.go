package app

// Хранилище конфига: ~/.config/unissh/servers.json

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const appName = "unissh"

// Snippet — сохранённая команда для быстрого выполнения на сервере.
type Snippet struct {
	Name string `json:"name"`
	Cmd  string `json:"cmd"`
}

type Server struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	User          string   `json:"user"`
	Key           string   `json:"key,omitempty"` // путь к приватному ключу; пусто = вход по паролю
	Auth          string   `json:"auth"`          // "key" | "password"
	Tags          []string `json:"tags"`
	Notes         string   `json:"notes"`
	Bootstrapped  bool     `json:"bootstrapped"`
	Favorite      bool     `json:"favorite,omitempty"`       // закреплён наверху списка
	ProxyJump     string   `json:"proxy_jump,omitempty"`     // бастион: user@host[:port]
	LastConnected string   `json:"last_connected,omitempty"` // RFC3339 последнего подключения
}

// Addr возвращает адрес сервера в формате user@host:port.
func (s *Server) Addr() string {
	return fmt.Sprintf("%s@%s:%d", s.User, s.Host, s.Port)
}

type Settings struct {
	DefaultUser       string   `json:"default_user"`
	SSHKey            string   `json:"ssh_key"`
	StatusTimeout     float64  `json:"status_timeout"` // секунды
	Language          string   `json:"language"`       // "ru" | "en"
	BootstrapDefaults []string `json:"bootstrap_defaults,omitempty"`
	// "" = новое окно (по умолчанию), "inline" = в текущем окне
	ConnectMode string `json:"connect_mode,omitempty"`
	// "" (по добавлению) | "name" | "ping" | "online" — режим сортировки списка
	SortMode string `json:"sort_mode,omitempty"`
	// "" (авто → blue) | "blue" | "dark" | "light" | "dracula" | "gruvbox"
	Theme    string    `json:"theme,omitempty"`
	Snippets []Snippet `json:"snippets,omitempty"` // глобальные сниппеты команд
	// nil = спрашивать подтверждение при подключении (по умолчанию включено)
	ConfirmConnect *bool `json:"confirm_connect,omitempty"`
}

// confirmConnectEnabled сообщает, спрашивать ли подтверждение перед подключением.
func confirmConnectEnabled(s *Settings) bool {
	if s == nil || s.ConfirmConnect == nil {
		return true
	}
	return *s.ConfirmConnect
}

// connectInNewWindow сообщает, открывать ли ssh в новом окне терминала.
func connectInNewWindow(s *Settings) bool {
	return s == nil || s.ConnectMode != "inline"
}

// gSettings — указатель на живые настройки (для доступа из actionConnect,
// у которого нет *Store). Устанавливается в main() после загрузки.
var gSettings *Settings

// gStore — указатель на загруженный Store (для доступа из функций без *Store,
// например управления сниппетами). Устанавливается в main().
var gStore *Store

type Config struct {
	Version  int       `json:"version"`
	Settings Settings  `json:"settings"`
	Servers  []*Server `json:"servers"`
}

type Store struct {
	Config
	path string
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", appName)
}

func knownHostsPath() string {
	return filepath.Join(configDir(), "known_hosts")
}

func defaultSettings() Settings {
	return Settings{
		DefaultUser:   "deploy",
		SSHKey:        "~/.ssh/unissh_ed25519",
		StatusTimeout: 1.5,
		Language:      "ru",
	}
}

func loadStore() *Store {
	st := &Store{
		Config: Config{Version: 1, Settings: defaultSettings()},
		path:   filepath.Join(configDir(), "servers.json"),
	}
	data, err := os.ReadFile(st.path)
	if err != nil {
		return st
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		broken := st.path + ".broken"
		_ = os.WriteFile(broken, data, 0o600)
		fmt.Println(styRed.Render(fmt.Sprintf(T.ErrReadConfig, err.Error())))
		fmt.Println(styYellow.Render(fmt.Sprintf(T.BackupSaved, broken)))
		return st
	}
	if cfg.Settings.DefaultUser == "" {
		cfg.Settings.DefaultUser = "deploy"
	}
	if cfg.Settings.SSHKey == "" {
		cfg.Settings.SSHKey = "~/.ssh/unissh_ed25519"
	}
	if cfg.Settings.StatusTimeout <= 0 {
		cfg.Settings.StatusTimeout = 1.5
	}
	if cfg.Settings.Language == "" {
		cfg.Settings.Language = "ru"
	}
	cfg.Version = 1
	st.Config = cfg
	return st
}

// backupsDir — каталог со снапшотами конфига для восстановления (undo).
func backupsDir() string {
	return filepath.Join(configDir(), "backups")
}

// Save атомарно сохраняет конфиг (tmp + rename), предварительно снимая снапшот
// прежнего файла в backups/ (для восстановления при ошибочном изменении).
func (st *Store) Save() error {
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		return err
	}
	st.backupCurrent()
	data, err := json.MarshalIndent(st.Config, "", "  ")
	if err != nil {
		return err
	}
	tmp := st.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, st.path)
}

// backupCurrent копирует текущий servers.json в backups/ (best-effort) и держит
// не более 20 последних снапшотов.
func (st *Store) backupCurrent() {
	data, err := os.ReadFile(st.path)
	if err != nil || len(data) == 0 {
		return
	}
	dir := backupsDir()
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	// имя без Date.now(): берём mtime текущего файла как метку снапшота
	stamp := "backup"
	if fi, err := os.Stat(st.path); err == nil {
		stamp = fmt.Sprintf("%d", fi.ModTime().UnixNano())
	}
	_ = os.WriteFile(filepath.Join(dir, "servers-"+stamp+".json"), data, 0o600)
	pruneBackups(dir, 20)
}

// pruneBackups удаляет самые старые снапшоты, оставляя не более keep штук.
func pruneBackups(dir string, keep int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var files []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "servers-") && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files) // имена содержат наносекунды → лексикографически = по времени
	for i := 0; i < len(files)-keep; i++ {
		os.Remove(filepath.Join(dir, files[i]))
	}
}

func (st *Store) MustSave() {
	if err := st.Save(); err != nil {
		fmt.Println(styRed.Render(fmt.Sprintf(T.ErrSaveConfig, err.Error())))
	}
}

// Find ищет сервер: точное имя → уникальный префикс → уникальная подстрока.
func (st *Store) Find(query string) *Server {
	q := strings.ToLower(strings.TrimSpace(query))
	var pref, sub []*Server
	for _, s := range st.Servers {
		name := strings.ToLower(s.Name)
		if name == q {
			return s
		}
		if strings.HasPrefix(name, q) {
			pref = append(pref, s)
		}
		if strings.Contains(name, q) || strings.Contains(strings.ToLower(s.Host), q) {
			sub = append(sub, s)
		}
	}
	if len(pref) == 1 {
		return pref[0]
	}
	if len(sub) == 1 {
		return sub[0]
	}
	return nil
}

func (st *Store) ByID(id string) *Server {
	for _, s := range st.Servers {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func (st *Store) NameTaken(name, excludeID string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	for _, s := range st.Servers {
		if strings.ToLower(s.Name) == n && s.ID != excludeID {
			return true
		}
	}
	return false
}

func (st *Store) Remove(id string) {
	out := st.Servers[:0]
	for _, s := range st.Servers {
		if s.ID != id {
			out = append(out, s)
		}
	}
	st.Servers = out
}

func newID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}
