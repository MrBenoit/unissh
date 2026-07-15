package app

// Хранилище конфига: ~/.config/unissh/servers.json

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MrBenoit/unissh/internal/core"
)

const appName = "unissh"

// Snippet — сохранённая команда для быстрого выполнения на сервере.
type Snippet struct {
	Name string `json:"name"`
	Cmd  string `json:"cmd"`
}

// Tunnel — правило проброса порта, привязанное к серверу. Kind — один из
// core.TunnelLocal / TunnelRemote / TunnelDynamic. Для «dynamic» заполнен
// только LocalPort (SOCKS5-прокси). Само правило хранится в конфиге, а живой
// процесс `ssh -N` держит менеджер из tunnels.go.
type Tunnel struct {
	Kind       string `json:"kind"`
	LocalPort  int    `json:"local_port"`
	RemoteHost string `json:"remote_host,omitempty"`
	RemotePort int    `json:"remote_port,omitempty"`
}

// Spec возвращает канонический ключ правила (по нему ищем запущенный процесс).
func (t Tunnel) Spec() string {
	return core.TunnelSpec(t.Kind, t.LocalPort, t.RemoteHost, t.RemotePort)
}

// Label возвращает подпись правила для интерфейса.
func (t Tunnel) Label() string {
	return core.TunnelLabel(t.Kind, t.LocalPort, t.RemoteHost, t.RemotePort)
}

// Flags возвращает аргументы ssh, поднимающие этот проброс.
func (t Tunnel) Flags() []string {
	return core.TunnelFlags(t.Kind, t.LocalPort, t.RemoteHost, t.RemotePort)
}

// Validate проверяет корректность правила.
func (t Tunnel) Validate() error {
	return core.ValidateTunnel(t.Kind, t.LocalPort, t.RemoteHost, t.RemotePort)
}

type Server struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	User          string   `json:"user"`
	Key           string   `json:"key,omitempty"`    // путь к приватному ключу; пусто = вход по паролю
	Auth          string   `json:"auth"`             // "key" | "password"
	Folder        string   `json:"folder,omitempty"` // папка-группа; пусто = корень («Без папки»)
	Notes         string   `json:"notes"`
	Bootstrapped  bool     `json:"bootstrapped"`
	Favorite      bool     `json:"favorite,omitempty"`       // закреплён наверху списка
	ProxyJump     string   `json:"proxy_jump,omitempty"`     // бастион: user@host[:port]
	LastConnected string   `json:"last_connected,omitempty"` // RFC3339 последнего подключения
	Tunnels       []Tunnel `json:"tunnels,omitempty"`        // правила проброса портов
}

// HasTunnel сообщает, есть ли у сервера правило с таким же ключом.
func (s *Server) HasTunnel(t Tunnel) bool {
	spec := t.Spec()
	for _, x := range s.Tunnels {
		if x.Spec() == spec {
			return true
		}
	}
	return false
}

// RemoveTunnel удаляет правило по ключу и сообщает, нашлось ли оно.
func (s *Server) RemoveTunnel(t Tunnel) bool {
	spec := t.Spec()
	for i, x := range s.Tunnels {
		if x.Spec() == spec {
			s.Tunnels = append(s.Tunnels[:i], s.Tunnels[i+1:]...)
			return true
		}
	}
	return false
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
	// свёрнутые папки списка серверов ("" = группа «Без папки»)
	CollapsedFolders []string `json:"collapsed_folders,omitempty"`
}

// connectInNewWindow сообщает, открывать ли ssh в новом окне терминала.
// Даже если пользователь этого хочет, на Linux без эмулятора терминала открывать
// нечем — тогда честно подключаемся в текущем окне, не пугая ошибкой.
func connectInNewWindow(s *Settings) bool {
	return (s == nil || s.ConnectMode != "inline") && canOpenNewWindow()
}

type Config struct {
	Version  int       `json:"version"`
	Settings Settings  `json:"settings"`
	Servers  []*Server `json:"servers"`
	// Папки-группы в порядке отображения. Хранятся отдельно от серверов, чтобы
	// пустая папка не исчезала сразу после создания.
	Folders []string `json:"folders,omitempty"`
}

type Store struct {
	Config
	path string
	// readOnly = конфиг на диске не распарсился. Тогда в памяти пустой список,
	// и любая запись запрещена — иначе первое же сохранение затрёт все серверы.
	// Снимается только успешным восстановлением из бэкапа.
	readOnly bool
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
		st.readOnly = true // не даём затереть непрочитанный конфиг пустым
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

// Save атомарно сохраняет конфиг (tmp + fsync + rename + fsync каталога),
// предварительно снимая снапшот прежнего файла в backups/. Без fsync «атомарно
// сохранённый» файл после потери питания мог оказаться пустым.
// В read-only режиме запись запрещена: конфиг на диске не распарсился, и его
// нельзя перетереть пустым списком.
func (st *Store) Save() error {
	if st.readOnly {
		return errors.New(T.ErrConfigReadOnly)
	}
	if st.path == "" {
		return errors.New("config path is not set")
	}
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		return err
	}
	st.backupCurrent()
	data, err := json.MarshalIndent(st.Config, "", "  ")
	if err != nil {
		return err
	}
	tmp := st.path + ".tmp"
	if err := writeFileSync(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, st.path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(st.path))
}

// writeFileSync пишет файл и сбрасывает его на диск (fsync) до закрытия.
func writeFileSync(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// syncDir фиксирует запись самой записи каталога (чтобы rename пережил сбой).
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
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
		_ = os.Remove(filepath.Join(dir, files[i]))
	}
}

func (st *Store) MustSave() {
	if err := st.Save(); err != nil {
		fmt.Println(styRed.Render(fmt.Sprintf(T.ErrSaveConfig, err.Error())))
	}
}

// --- папки-группы (плоские, один уровень) ---

// HasFolder сообщает, существует ли папка с таким именем.
func (st *Store) HasFolder(name string) bool {
	for _, f := range st.Folders {
		if f == name {
			return true
		}
	}
	return false
}

// AddFolder создаёт папку; false — имя пустое или уже занято.
func (st *Store) AddFolder(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || st.HasFolder(name) {
		return false
	}
	st.Folders = append(st.Folders, name)
	return true
}

// RenameFolder переименовывает папку, обновляя серверы и список свёрнутых.
// false — имя пустое, папки нет или новое имя уже занято.
func (st *Store) RenameFolder(old, name string) bool {
	name = strings.TrimSpace(name)
	if old == "" || name == "" || !st.HasFolder(old) {
		return false
	}
	if name != old && st.HasFolder(name) {
		return false
	}
	for i, f := range st.Folders {
		if f == old {
			st.Folders[i] = name
		}
	}
	for _, s := range st.Servers {
		if s.Folder == old {
			s.Folder = name
		}
	}
	for i, f := range st.Settings.CollapsedFolders {
		if f == old {
			st.Settings.CollapsedFolders[i] = name
		}
	}
	return true
}

// MoveFolder сдвигает папку в порядке отображения на delta позиций.
// false — папки нет или двигать некуда (уже с краю).
func (st *Store) MoveFolder(name string, delta int) bool {
	idx := -1
	for i, f := range st.Folders {
		if f == name {
			idx = i
			break
		}
	}
	dst := idx + delta
	if idx < 0 || dst < 0 || dst >= len(st.Folders) {
		return false
	}
	st.Folders[idx], st.Folders[dst] = st.Folders[dst], st.Folders[idx]
	return true
}

// RemoveFolder удаляет папку, а её серверы возвращает в корень («Без папки»).
func (st *Store) RemoveFolder(name string) {
	out := st.Folders[:0]
	for _, f := range st.Folders {
		if f != name {
			out = append(out, f)
		}
	}
	st.Folders = out
	for _, s := range st.Servers {
		if s.Folder == name {
			s.Folder = ""
		}
	}
	st.setCollapsed(name, false)
}

// FolderServers возвращает серверы папки (name == "" — серверы без папки).
func (st *Store) FolderServers(name string) []*Server {
	var out []*Server
	for _, s := range st.Servers {
		if s.Folder == name {
			out = append(out, s)
		}
	}
	return out
}

// FolderCollapsed сообщает, свёрнута ли папка.
func (st *Store) FolderCollapsed(name string) bool {
	for _, f := range st.Settings.CollapsedFolders {
		if f == name {
			return true
		}
	}
	return false
}

// setCollapsed задаёт состояние свёртки папки.
func (st *Store) setCollapsed(name string, collapsed bool) {
	cur := st.FolderCollapsed(name)
	if cur == collapsed {
		return
	}
	if collapsed {
		st.Settings.CollapsedFolders = append(st.Settings.CollapsedFolders, name)
		return
	}
	out := st.Settings.CollapsedFolders[:0]
	for _, f := range st.Settings.CollapsedFolders {
		if f != name {
			out = append(out, f)
		}
	}
	st.Settings.CollapsedFolders = out
}

// ToggleFolder переключает свёртку папки.
func (st *Store) ToggleFolder(name string) {
	st.setCollapsed(name, !st.FolderCollapsed(name))
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
