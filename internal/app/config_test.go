package app

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestServerAddr(t *testing.T) {
	s := &Server{User: "root", Host: "10.0.0.1", Port: 2222}
	if got := s.Addr(); got != "root@10.0.0.1:2222" {
		t.Fatalf("Addr = %q", got)
	}
}

// TestSettingsFlags: режим «новое окно» — дефолт, но только если окно есть чем
// открыть. Фиксируем goos, иначе тест зависел бы от того, стоит ли на машине
// эмулятор терминала (на CI-раннере Linux его нет).
func TestSettingsFlags(t *testing.T) {
	defer func(o string) { goos = o }(goos)

	goos = "darwin" // на macOS Terminal.app есть всегда
	if connectInNewWindow(nil) != true {
		t.Fatal("connectInNewWindow(nil) должно быть true")
	}
	if connectInNewWindow(&Settings{ConnectMode: "inline"}) != false {
		t.Fatal("connectInNewWindow(inline) должно быть false")
	}

	// Linux без единого терминала: даже при желании открывать нечем
	goos = "linux"
	defer func(o func(string) (string, error)) { lookPath = o }(lookPath)
	lookPath = func(string) (string, error) { return "", errNoAuth }
	if connectInNewWindow(nil) != false {
		t.Fatal("без эмулятора терминала подключаемся inline, а не падаем в ошибку")
	}
}

func TestDefaultSettings(t *testing.T) {
	d := defaultSettings()
	if d.DefaultUser == "" || d.SSHKey == "" || d.StatusTimeout <= 0 || d.Language == "" {
		t.Fatalf("некорректные дефолты: %+v", d)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := expandPath("~/x/y"); got != filepath.Join(home, "x/y") {
		t.Fatalf("expandPath(~) = %q", got)
	}
	if got := expandPath("/abs/path"); got != "/abs/path" {
		t.Fatalf("expandPath(/abs) = %q", got)
	}
	if got := expandPath("rel"); got != "rel" {
		t.Fatalf("expandPath(rel) = %q", got)
	}
}

func TestNewIDFormat(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}$`)
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		id := newID()
		if !re.MatchString(id) {
			t.Fatalf("newID формат: %q", id)
		}
		if seen[id] {
			t.Fatalf("newID не уникален: %q", id)
		}
		seen[id] = true
	}
}

func TestStoreFindByIDNameTakenRemove(t *testing.T) {
	st := &Store{Config: Config{Servers: []*Server{
		{ID: "1", Name: "web-1", Host: "10.0.0.1"},
		{ID: "2", Name: "web-2", Host: "10.0.0.2"},
		{ID: "3", Name: "database", Host: "10.0.0.3"},
	}}}

	// точное имя
	if s := st.Find("web-1"); s == nil || s.ID != "1" {
		t.Fatal("Find точное имя")
	}
	// уникальный префикс
	if s := st.Find("data"); s == nil || s.ID != "3" {
		t.Fatal("Find префикс")
	}
	// уникальная подстрока по host
	if s := st.Find("0.0.2"); s == nil || s.ID != "2" {
		t.Fatal("Find подстрока по host")
	}
	// неоднозначно → nil
	if s := st.Find("web"); s != nil {
		t.Fatal("Find неоднозначный должен быть nil")
	}

	if st.ByID("2") == nil || st.ByID("nope") != nil {
		t.Fatal("ByID")
	}
	if !st.NameTaken("web-1", "") || st.NameTaken("web-1", "1") {
		t.Fatal("NameTaken должен учитывать excludeID")
	}
	if st.NameTaken("brand-new", "") {
		t.Fatal("NameTaken для свободного имени")
	}

	st.Remove("2")
	if st.ByID("2") != nil || len(st.Servers) != 2 {
		t.Fatal("Remove не удалил сервер")
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	st := loadStore()
	st.Servers = append(st.Servers, &Server{ID: "a", Name: "srv", Host: "1.2.3.4", Port: 22, User: "root"})
	if err := st.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := loadStore()
	if len(got.Servers) != 1 || got.Servers[0].Name != "srv" {
		t.Fatalf("после загрузки: %+v", got.Servers)
	}
	if got.Version != 1 {
		t.Fatalf("version = %d", got.Version)
	}
}

func TestSaveCreatesBackup(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	st := loadStore()
	st.Servers = append(st.Servers, &Server{ID: "a", Name: "one"})
	if err := st.Save(); err != nil { // первый save — бэкапить нечего
		t.Fatal(err)
	}
	st.Servers[0].Name = "two"
	if err := st.Save(); err != nil { // второй save — снимок прежнего файла
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(backupsDir())
	if len(entries) == 0 {
		t.Fatal("ожидался хотя бы один бэкап после второго Save")
	}
}

// TestCorruptConfigIsReadOnly: битый servers.json не должен затираться пустым.
func TestCorruptConfigIsReadOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(configDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(configDir(), "servers.json")
	broken := []byte(`{"servers": [ ЭТО НЕ JSON `)
	if err := os.WriteFile(cfgPath, broken, 0o600); err != nil {
		t.Fatal(err)
	}

	st := loadStore()
	if !st.readOnly {
		t.Fatal("битый конфиг должен переводить стор в read-only")
	}
	if len(st.Servers) != 0 {
		t.Fatal("при ошибке разбора список серверов пуст")
	}
	// снапшот .broken сохранён
	if _, err := os.Stat(cfgPath + ".broken"); err != nil {
		t.Fatalf("ожидался %s.broken: %v", cfgPath, err)
	}

	// запись запрещена, файл на диске не тронут
	if err := st.Save(); err == nil {
		t.Fatal("Save в read-only должен вернуть ошибку")
	}
	got, _ := os.ReadFile(cfgPath)
	if string(got) != string(broken) {
		t.Fatalf("исходный конфиг изменён: %q", got)
	}

	// восстановление снимает флаг — запись снова работает
	st.readOnly = false
	st.Servers = append(st.Servers, &Server{ID: "a", Name: "srv"})
	if err := st.Save(); err != nil {
		t.Fatalf("после снятия read-only Save должен работать: %v", err)
	}
	if got := loadStore(); len(got.Servers) != 1 || got.readOnly {
		t.Fatalf("после восстановления: servers=%d readOnly=%v", len(got.Servers), got.readOnly)
	}
}

func TestSaveWithoutPathFails(t *testing.T) {
	st := &Store{}
	if err := st.Save(); err == nil {
		t.Fatal("Save без пути должен возвращать ошибку, а не писать .tmp в CWD")
	}
}

func TestPruneBackups(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 25; i++ {
		name := filepath.Join(dir, "servers-"+string(rune('a'+i))+".json")
		if err := os.WriteFile(name, []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pruneBackups(dir, 20)
	entries, _ := os.ReadDir(dir)
	if len(entries) != 20 {
		t.Fatalf("ожидалось 20 бэкапов после prune, получили %d", len(entries))
	}
}
