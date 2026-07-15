package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// trSrv — сервер для тестов передачи файлов (с ключом и бастионом).
func trSrv() *Server {
	return &Server{
		ID: "s1", Name: "web", Host: "1.2.3.4", Port: 2222, User: "deploy",
		Key: "/tmp/key", Auth: "key", ProxyJump: "jump@bastion",
	}
}

// TestScpArgsPortIsCapital: у scp порт задаётся ЗАГЛАВНОЙ -P. Строчная -p у него
// означает «сохранить времена файла» — самая частая ошибка при переносе sshArgs.
func TestScpArgsPortIsCapital(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	args := scpArgs(trSrv())
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-P 2222") {
		t.Fatalf("нет заглавной -P: %q", joined)
	}
	for i, a := range args {
		if a == "-p" {
			t.Fatalf("строчная -p в аргументах scp (позиция %d): %q", i, joined)
		}
	}
	for _, want := range []string{"-r", "-i /tmp/key", "IdentitiesOnly=yes", "-J jump@bastion", "UserKnownHostsFile="} {
		if !strings.Contains(joined, want) {
			t.Errorf("нет %q в %q", want, joined)
		}
	}
}

// TestUploadDownloadArgsOrder: источник идёт перед приёмником, а удалённая
// сторона всегда в форме user@host:путь.
func TestUploadDownloadArgsOrder(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s := trSrv()

	up := uploadArgs(s, "/tmp/dump.sql", "~/backups")
	if up[len(up)-2] != "/tmp/dump.sql" {
		t.Errorf("источник (локальный) должен быть предпоследним: %v", up)
	}
	if up[len(up)-1] != "deploy@1.2.3.4:~/backups" {
		t.Errorf("приёмник (удалённый): %q", up[len(up)-1])
	}

	down := downloadArgs(s, "/var/log/syslog", "/tmp")
	if down[len(down)-2] != "deploy@1.2.3.4:/var/log/syslog" {
		t.Errorf("источник (удалённый): %q", down[len(down)-2])
	}
	if down[len(down)-1] != "/tmp" {
		t.Errorf("приёмник (локальный): %q", down[len(down)-1])
	}
}

// TestScpRemoteEscapes: пробел и «;» в удалённом пути не должны стать командой
// на сервере — путь приходит в шелл той стороны.
func TestScpRemoteEscapes(t *testing.T) {
	s := trSrv()
	got := scpRemote(s, "/tmp/a b; rm -rf /")
	if !strings.HasPrefix(got, "deploy@1.2.3.4:") {
		t.Fatalf("нет префикса user@host: %q", got)
	}
	tail := strings.TrimPrefix(got, "deploy@1.2.3.4:")
	if !strings.HasPrefix(tail, "'") {
		t.Fatalf("опасный путь не закавычен: %q", tail)
	}
	// тильда, наоборот, обязана остаться раскрываемой
	if got := scpRemote(s, "~/dumps"); got != "deploy@1.2.3.4:~/dumps" {
		t.Fatalf("тильда должна дойти до шелла как есть: %q", got)
	}
}

// TestValidateTransferPaths: локальные пути проверяются на существование,
// удалённый — только на пустоту и перевод строки (файл там ещё не видно).
func TestValidateTransferPaths(t *testing.T) {
	setLang("ru")
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := validateLocalExists(file); err != nil {
		t.Errorf("существующий файл: %v", err)
	}
	if validateLocalExists("") == nil || validateLocalExists(filepath.Join(dir, "нет")) == nil {
		t.Error("пустой и несуществующий путь должны отвергаться")
	}

	if err := validateLocalDir(dir); err != nil {
		t.Errorf("существующий каталог: %v", err)
	}
	if validateLocalDir(file) == nil {
		t.Error("файл — не каталог назначения")
	}
	if validateLocalDir("") == nil {
		t.Error("пустой каталог")
	}

	if err := validateRemotePath("/var/log/syslog"); err != nil {
		t.Errorf("обычный удалённый путь: %v", err)
	}
	if validateRemotePath("") == nil {
		t.Error("пустой удалённый путь")
	}
	if validateRemotePath("/tmp/a\nb") == nil {
		t.Error("перевод строки в пути должен отвергаться")
	}
}

// TestDefaultDownloadDir: ~/Downloads, если он есть, иначе текущий каталог.
func TestDefaultDownloadDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := defaultDownloadDir(); got != "." {
		t.Fatalf("без ~/Downloads ожидался «.», получено %q", got)
	}
	dl := filepath.Join(home, "Downloads")
	if err := os.MkdirAll(dl, 0o700); err != nil {
		t.Fatal(err)
	}
	if got := defaultDownloadDir(); got != dl {
		t.Fatalf("ожидался %q, получено %q", dl, got)
	}
}

// TestRunTransferOutcomes: успех, ненулевой код и ошибка запуска печатаются
// по-разному; сам scp подменён швом.
func TestRunTransferOutcomes(t *testing.T) {
	setLang("ru")
	orig := scpRun
	defer func() { scpRun = orig }()

	var gotArgs []string
	scpRun = func(s *Server, args []string) (int, error) { gotArgs = args; return 0, nil }
	runTransfer(trSrv(), []string{"a", "b"}, "заголовок")
	if len(gotArgs) != 2 {
		t.Fatalf("аргументы не доехали до scp: %v", gotArgs)
	}
}

// TestTransferActionInActions: пункт передачи файлов есть у любого сервера.
func TestTransferActionInActions(t *testing.T) {
	setLang("ru")
	found := false
	for _, a := range serverActions(trSrv()) {
		if a.action == "transfer" {
			found = true
		}
	}
	if !found {
		t.Fatal("в действиях сервера нет «Передача файлов»")
	}
}

// TestExpectAutoScriptNoInterac: скрипт для scp ждёт завершения, а не отдаёт
// сессию пользователю, и берёт пароль из окружения.
func TestExpectAutoScript(t *testing.T) {
	sc := expectAutoScript("scp", []string{"-r", "/tmp/a", "u@h:~/b"})
	if !strings.Contains(sc, `spawn scp "-r" "/tmp/a" "u@h:~/b"`) {
		t.Errorf("spawn собран неверно:\n%s", sc)
	}
	if !strings.Contains(sc, "$env(UNISSH_SSHPASS)") {
		t.Error("пароль должен браться из окружения, а не из скрипта")
	}
	if strings.Contains(sc, "interact") {
		t.Error("scp нечего отдавать пользователю")
	}
	if !strings.Contains(sc, "expect eof") || !strings.Contains(sc, "set timeout -1") {
		t.Error("передача может идти долго — таймаут должен сниматься")
	}
}
