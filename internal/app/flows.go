package app

// Мастера: добавление сервера (с настройкой и без), повторная настройка,
// редактирование, удаление, меню управления сервером.

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

var (
	nameRe     = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	usernameRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
)

func validateName(st *Store, excludeID string) func(string) error {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return errors.New(T.ErrNameEmpty)
		}
		if !nameRe.MatchString(s) {
			return errors.New(T.ErrNameChars)
		}
		if st.NameTaken(s, excludeID) {
			return errors.New(T.ErrNameTaken)
		}
		return nil
	}
}

func validateHost(s string) error {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, " \t") {
		return errors.New(T.ErrHostFmt)
	}
	return nil
}

func validatePort(s string) error {
	p, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || p < 1 || p > 65535 {
		return errors.New(T.ErrPortRange)
	}
	return nil
}

func validateUsername(s string) error {
	if !usernameRe.MatchString(strings.TrimSpace(s)) {
		return errors.New(T.ErrUserFmt)
	}
	return nil
}

var commonTimezones = []string{
	"Europe/Moscow", "Europe/Kyiv", "Europe/Berlin", "Europe/Amsterdam",
	"Europe/London", "UTC", "Asia/Almaty", "Asia/Tbilisi", "Asia/Yerevan",
	"Asia/Dubai", "America/New_York",
}

func splitTags(s string) []string {
	var tags []string
	for _, t := range regexp.MustCompile(`[,\s]+`).Split(strings.TrimSpace(s), -1) {
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// -- шаги настройки: список, подписи, дефолты из настроек пользователя --

var allBootstrapSteps = []string{
	"update", "packages", "firewall", "fail2ban", "harden",
	"root_key", "auto_updates", "port", "swap", "tz", "hostname",
}

var defaultBootstrapSteps = []string{
	"update", "packages", "firewall", "fail2ban", "harden", "root_key", "auto_updates",
}

func bootstrapStepLabel(key string) string {
	switch key {
	case "update":
		return T.OptUpdate
	case "packages":
		return T.OptPackages
	case "firewall":
		return T.OptFirewall
	case "fail2ban":
		return T.OptFail2ban
	case "harden":
		return T.OptHarden
	case "root_key":
		return T.OptRootKey
	case "auto_updates":
		return T.OptAutoUpd
	case "port":
		return T.OptPort
	case "swap":
		return T.OptSwap
	case "tz":
		return T.OptTz
	case "hostname":
		return T.OptHostname
	}
	return key
}

func bootstrapStepOptions(preselected []string) []multiOption {
	var opts []multiOption
	for _, key := range allBootstrapSteps {
		opts = append(opts, multiOption{
			Label:    bootstrapStepLabel(key),
			Value:    key,
			Selected: slices.Contains(preselected, key),
		})
	}
	return opts
}

// userDefaultSteps возвращает сохранённые пользователем шаги (или дефолт).
func userDefaultSteps(st *Store) []string {
	if len(st.Settings.BootstrapDefaults) > 0 {
		return st.Settings.BootstrapDefaults
	}
	return defaultBootstrapSteps
}

// bootstrapOptionsWizard спрашивает параметры настройки.
// Выбранный набор шагов запоминается как новый дефолт пользователя.
func bootstrapOptionsWizard(st *Store) (BootstrapOpts, bool) {
	var opts BootstrapOpts
	picked, ok := askMulti(T.WizardTitle, bootstrapStepOptions(userDefaultSteps(st)))
	if !ok {
		return opts, false
	}
	st.Settings.BootstrapDefaults = picked
	st.MustSave()

	has := func(k string) bool { return slices.Contains(picked, k) }
	opts.Update = has("update")
	opts.Packages = has("packages")
	opts.Firewall = has("firewall")
	opts.Fail2ban = has("fail2ban")
	opts.Harden = has("harden")
	opts.RootKey = has("root_key")
	opts.AutoUpdates = has("auto_updates")
	opts.Swap = has("swap")

	username, ok := askInput(T.AskNewUsername, st.Settings.DefaultUser, validateUsername)
	if !ok || username == "" {
		return opts, false
	}
	opts.Username = username

	if has("port") {
		p, ok := askInput(T.AskNewPort, "2222", validatePort)
		if !ok {
			return opts, false
		}
		opts.NewPort, _ = strconv.Atoi(p)
	}
	if has("tz") {
		var tzOpts []huh.Option[string]
		for _, tz := range commonTimezones {
			tzOpts = append(tzOpts, huh.NewOption(tz, tz))
		}
		tzOpts = append(tzOpts, huh.NewOption(T.ManualEntry, "__manual__"))
		tz, ok := askSelect(T.AskTimezone, tzOpts)
		if !ok {
			return opts, false
		}
		if tz == "__manual__" {
			tz, ok = askInput(T.AskTzManual, "", func(s string) error {
				if !tzRe.MatchString(strings.TrimSpace(s)) {
					return errors.New(T.ErrTzFmt)
				}
				return nil
			})
			if !ok || tz == "" {
				return opts, false
			}
		}
		opts.Timezone = tz
	}
	if has("hostname") {
		hn, ok := askInput(T.AskHostname, "", func(s string) error {
			if !hostnameRe.MatchString(strings.TrimSpace(s)) {
				return errors.New(T.ErrHostnameFmt)
			}
			return nil
		})
		if !ok || hn == "" {
			return opts, false
		}
		opts.Hostname = hn
	}
	return opts, true
}

func runBootstrapAndSave(st *Store, server *Server, host string, port int,
	rootPW, rootKey string, opts BootstrapOpts, keyPath, pubkey string,
	name string, tags []string) {

	boot := newBootstrap(host, port, rootPW, rootKey, opts, pubkey, keyPath)
	// Для уже существующего сервера сохраняем новый пароль в Keychain сразу после
	// его установки на сервере — до потенциального отказа следующих шагов.
	if server != nil {
		sid := server.ID
		boot.onPassword = func(user, pw string) { _ = secretSet(sid, user, pw) }
	}
	result, err := boot.Run()
	if err != nil {
		fmt.Println()
		fmt.Println(styRed.Render(T.SetupAborted + err.Error()))
		// Пароль уже мог быть применён на сервере (chpasswd) — показываем его,
		// чтобы он не потерялся, даже если настройка прервалась.
		if result.Password != "" {
			fmt.Println(styYellow.Render(fmt.Sprintf(T.AbortedPassword, opts.Username, result.Password)))
			pbcopy(result.Password)
		}
		if server == nil {
			fmt.Println(styYellow.Render(T.ServerNotSaved))
		}
		pressEnter()
		return
	}

	isNew := server == nil
	if isNew {
		server = &Server{ID: newID(), Name: name, Tags: tags, Notes: ""}
		st.Servers = append(st.Servers, server)
	}
	server.Host = host
	server.Port = result.Port
	server.User = opts.Username
	server.Key = keyPath
	server.Auth = "key"
	server.Bootstrapped = true
	st.MustSave()

	if result.Password != "" {
		if err := secretSet(server.ID, opts.Username, result.Password); err != nil {
			fmt.Println(styYellow.Render(T.WarnKeychainFail + err.Error()))
		}
	}
	if rootPW != "" {
		_ = secretSet(server.ID, "root", rootPW)
	}
	maybeRefreshSSHConfig(st)

	// итоговая карточка
	var rows []string
	rows = append(rows,
		kv(T.SumServer, fmt.Sprintf("%s:%d", host, result.Port)),
		kv(T.SumLogin, opts.Username),
		kv(T.SumPassword, styYellow.Bold(true).Render(orDash(result.Password))),
		kv(T.SumKey, keyPath),
		kv(T.SumKeychain, T.SumKeychainVal),
	)
	fmt.Println()
	fmt.Println(panelGreen.Render(
		styGreen.Bold(true).Render(fmt.Sprintf(T.ConfiguredOK, server.Name)) + "\n\n" +
			strings.Join(rows, "\n")))
	for _, w := range result.Warnings {
		fmt.Println("  " + styYellow.Render("⚠ "+w))
	}
	if opts.Harden {
		fmt.Println("  " + styYellow.Render(fmt.Sprintf(T.WarnNoPwLogin, keyPath)))
	}
	if result.Password != "" && pbcopy(result.Password) {
		fmt.Println("  " + styDim.Render(T.UserPwCopied))
	}
	fmt.Println()
	// показываем полную информацию о сервере (в т.ч. все IP-адреса)
	actionInfo(server)
	if yes, ok := askConfirm(T.AskConnectNow, true); ok && yes {
		actionConnect(server)
	}
}

func addServerFlow(st *Store) {
	fmt.Println()
	fmt.Println(styTitle.Render(T.TitleNewServer))

	name, ok := askInput(T.AskName, "", validateName(st, ""))
	if !ok || name == "" {
		return
	}
	host, ok := askInput(T.AskHost, "", validateHost)
	if !ok || host == "" {
		return
	}
	portS, ok := askInput(T.AskPort, "22", validatePort)
	if !ok {
		return
	}
	port, _ := strconv.Atoi(portS)
	tagsS, ok := askInput(T.AskTags, "", nil)
	if !ok {
		return
	}
	tags := splitTags(tagsS)

	mode, ok := askSelect(T.AskHowAdd, []huh.Option[string]{
		huh.NewOption(T.OptWithBootstrap, "bootstrap"),
		huh.NewOption(T.OptPlainAdd, "plain"),
	})
	if !ok {
		return
	}

	keyPath := st.Settings.SSHKey

	if mode == "bootstrap" {
		rootAuth, ok := askSelect(T.AskRootAuth, []huh.Option[string]{
			huh.NewOption(T.OptRootPassword, "password"),
			huh.NewOption(T.OptRootKeyFile, "key"),
		})
		if !ok {
			return
		}
		rootPW, rootKey := "", ""
		if rootAuth == "password" {
			rootPW, ok = askPasswordInput(T.AskRootPassword)
			if !ok || rootPW == "" {
				return
			}
		} else {
			rootKey, ok = askInput(T.AskRootKeyPath, keyPath, nil)
			if !ok || rootKey == "" {
				return
			}
		}

		pubkey, err := ensureKey(keyPath, true)
		if err != nil {
			fmt.Println(styRed.Render(T.ErrNoKeySetup + err.Error()))
			pressEnter()
			return
		}

		err = withSpinner(fmt.Sprintf(T.CheckingRootAccess, host, port), func() error {
			r, err := dialSSH(host, port, "root", rootPW, rootKey, "", 15*time.Second)
			if err != nil {
				return err
			}
			r.Close()
			return nil
		})
		if err != nil {
			fmt.Println(styRed.Render(err.Error()))
			pressEnter()
			return
		}
		fmt.Println("  " + styGreen.Render("✓ ") + T.RootConfirmed)

		opts, ok := bootstrapOptionsWizard(st)
		if !ok {
			return
		}
		runBootstrapAndSave(st, nil, host, port, rootPW, rootKey, opts, keyPath, pubkey, name, tags)
		return
	}

	// -- просто добавить --
	user, ok := askInput(T.AskUser, st.Settings.DefaultUser, nil)
	if !ok || user == "" {
		return
	}
	auth, ok := askSelect(T.AskAuth, []huh.Option[string]{
		huh.NewOption(T.OptAuthKey, "key"),
		huh.NewOption(T.OptAuthPassword, "password"),
	})
	if !ok {
		return
	}
	key := ""
	if auth == "key" {
		key, ok = askInput(T.AskKeyPath, keyPath, nil)
		if !ok || key == "" {
			return
		}
	}
	server := &Server{
		ID: newID(), Name: name, Host: host, Port: port,
		User: user, Key: key, Auth: auth, Tags: tags,
	}
	if auth == "password" {
		pw, ok := askPasswordInput(fmt.Sprintf(T.AskPasswordFor, user, host))
		if !ok {
			return
		}
		if pw != "" {
			if err := secretSet(server.ID, user, pw); err != nil {
				fmt.Println(styYellow.Render(T.WarnKeychainFail + err.Error()))
			}
		}
	}
	st.Servers = append(st.Servers, server)
	st.MustSave()
	maybeRefreshSSHConfig(st)
	fmt.Println(styGreen.Render(fmt.Sprintf(T.ServerAdded, name)))
	time.Sleep(600 * time.Millisecond)
	// подключаемся и показываем полную информацию о сервере (в т.ч. все IP-адреса)
	actionInfo(server)
}

func rebootstrapFlow(st *Store, s *Server) {
	fmt.Println(styDim.Render(fmt.Sprintf(T.NeedRootAccess, s.Host)))
	// Ключ первым: после стандартного харденинга парольный вход root отключён,
	// но unissh кладёт ключ и для root (опция root_key), поэтому вход по ключу — рабочий путь.
	if s.Bootstrapped {
		fmt.Println(styDim.Render(T.RootAuthHint))
	}
	rootAuth, ok := askSelect(T.AskRootAuth, []huh.Option[string]{
		huh.NewOption(T.OptRootKeyFile, "key"),
		huh.NewOption(T.OptRootPasswordKeychain, "password"),
	})
	if !ok {
		return
	}
	rootPW, rootKey := "", ""
	if rootAuth == "password" {
		rootPW = secretGet(s.ID, "root")
		if rootPW != "" {
			fmt.Println("  " + styGreen.Render("✓ ") + T.RootPwFromKeychain)
		} else {
			rootPW, ok = askPasswordInput(T.AskRootPassword)
			if !ok || rootPW == "" {
				return
			}
		}
	} else {
		rootKey, ok = askInput(T.AskRootKeyPath, st.Settings.SSHKey, nil)
		if !ok || rootKey == "" {
			return
		}
	}
	keyPath := st.Settings.SSHKey
	pubkey, err := ensureKey(keyPath, true)
	if err != nil {
		fmt.Println(styRed.Render(err.Error()))
		pressEnter()
		return
	}
	opts, ok := bootstrapOptionsWizard(st)
	if !ok {
		return
	}
	runBootstrapAndSave(st, s, s.Host, s.Port, rootPW, rootKey, opts, keyPath, pubkey, s.Name, s.Tags)
}

func actionShowPasswords(s *Server) {
	pw := secretGet(s.ID, s.User)
	rootPW := secretGet(s.ID, "root")
	if pw == "" && rootPW == "" {
		fmt.Println(styYellow.Render(T.NoPasswordsStored))
		pressEnter()
		return
	}
	var rows []string
	if pw != "" {
		rows = append(rows, kv(s.User, styYellow.Bold(true).Render(pw)))
	}
	if rootPW != "" {
		rows = append(rows, kv("root", styYellow.Bold(true).Render(rootPW)))
	}
	fmt.Println(panelYellow.Render(styBold.Render(T.TitlePasswords) + "\n\n" +
		strings.Join(rows, "\n")))
	if pw != "" && pbcopy(pw) {
		fmt.Println(styDim.Render(fmt.Sprintf(T.PwUserCopied, s.User)))
	}
	pressEnter()
}

func actionInstallKey(st *Store, s *Server) {
	pubkey, err := ensureKey(st.Settings.SSHKey, true)
	if err != nil {
		fmt.Println(styRed.Render(err.Error()))
		pressEnter()
		return
	}
	// соединение (и запрос пароля) — до спиннера
	r, err := openRemote(s, true)
	if err != nil {
		fmt.Println(styRed.Render(err.Error()))
		pressEnter()
		return
	}
	defer r.Close()
	err = withSpinner(fmt.Sprintf(T.InstallingKey, s.Name), func() error {
		// ключ передаём через stdin ($KEY) — комментарий .pub не ломает shell
		_, err := r.Sh(`HOME_DIR=$HOME; `+authKeyInstallScript, pubkey, nil, 30*time.Second)
		return err
	})
	if err != nil {
		fmt.Println(styRed.Render(err.Error()))
		pressEnter()
		return
	}
	s.Key = st.Settings.SSHKey
	s.Auth = "key"
	st.MustSave()
	maybeRefreshSSHConfig(st)
	fmt.Println(styGreen.Render(T.KeyInstalled))
	time.Sleep(time.Second)
}

func actionReboot(s *Server) {
	yes, ok := askConfirm(fmt.Sprintf(T.AskReboot, s.Name, s.Host), false)
	if !ok || !yes {
		return
	}
	r, err := openRemote(s, true)
	if err != nil {
		fmt.Println(styRed.Render(err.Error()))
		pressEnter()
		return
	}
	defer r.Close()
	if s.User == "root" {
		r.Run("reboot", "", nil, 10*time.Second)
	} else {
		pw := secretGet(s.ID, s.User)
		if pw == "" {
			var ok bool
			pw, ok = askPasswordInput(fmt.Sprintf(T.AskSudoPassword, s.User))
			if !ok || pw == "" {
				return
			}
		}
		r.Run("sudo -S -p '' reboot", pw+"\n", nil, 10*time.Second)
	}
	fmt.Println(styGreen.Render(T.RebootSent))
	time.Sleep(1200 * time.Millisecond)
}

// forgetHostKey удаляет сохранённый отпечаток хоста во всех формах записи.
func forgetHostKey(host string, port int) {
	hosts := map[string]bool{
		host:                               true,
		fmt.Sprintf("[%s]:%d", host, port): true,
	}
	for h := range hosts {
		exec.Command("ssh-keygen", "-R", h, "-f", knownHostsPath()).Run()
	}
}

func actionForgetHostKey(s *Server) {
	forgetHostKey(s.Host, s.Port)
	fmt.Println(styGreen.Render(fmt.Sprintf(T.FingerprintReset, s.Host)))
	time.Sleep(time.Second)
}

func actionEdit(st *Store, s *Server) {
	for {
		keyLabel := s.Key
		if keyLabel == "" {
			keyLabel = T.KeyNoneLabel
		}
		field, ok := askSelect(fmt.Sprintf(T.TitleEdit, s.Name), []huh.Option[string]{
			huh.NewOption(T.FldName+s.Name, "name"),
			huh.NewOption(T.FldHost+s.Host, "host"),
			huh.NewOption(fmt.Sprintf(T.FldPort, s.Port), "port"),
			huh.NewOption(T.FldUser+s.User, "user"),
			huh.NewOption(T.FldKey+keyLabel, "key"),
			huh.NewOption(T.FldProxy+orDash(s.ProxyJump), "proxy"),
			huh.NewOption(T.FldTags+orDash(strings.Join(s.Tags, " ")), "tags"),
			huh.NewOption(T.FldNotes+orDash(s.Notes), "notes"),
			huh.NewOption(T.Done, "done"),
		})
		if !ok || field == "done" {
			break
		}
		switch field {
		case "name":
			if v, ok := askInput(T.AskNewName, s.Name, validateName(st, s.ID)); ok && v != "" {
				s.Name = v
			}
		case "host":
			if v, ok := askInput(T.AskNewHost, s.Host, validateHost); ok && v != "" {
				s.Host = v
			}
		case "port":
			if v, ok := askInput(T.AskNewPortEdit, strconv.Itoa(s.Port), validatePort); ok && v != "" {
				s.Port, _ = strconv.Atoi(v)
			}
		case "user":
			if v, ok := askInput(T.AskNewUser, s.User, nil); ok && v != "" && v != s.User {
				// переносим пароль в Keychain под новым именем, старую запись удаляем
				if pw := secretGet(s.ID, s.User); pw != "" {
					_ = secretSet(s.ID, v, pw)
					secretDelete(s.ID, s.User)
				}
				s.User = v
			}
		case "key":
			if v, ok := askInput(T.AskKeyOrEmpty, s.Key, nil); ok {
				s.Key = strings.TrimSpace(v)
				if s.Key == "" {
					s.Auth = "password"
				} else {
					s.Auth = "key"
				}
			}
		case "tags":
			if v, ok := askInput(T.AskTagsEdit, strings.Join(s.Tags, " "), nil); ok {
				s.Tags = splitTags(v)
			}
		case "proxy":
			if v, ok := askInput(T.AskProxy, s.ProxyJump, nil); ok {
				s.ProxyJump = strings.TrimSpace(v)
			}
		case "notes":
			if v, ok := askInput(T.AskNotes, s.Notes, nil); ok {
				s.Notes = strings.TrimSpace(v)
			}
		}
		st.MustSave()
	}
	maybeRefreshSSHConfig(st)
}

func actionDelete(st *Store, s *Server) bool {
	yes, ok := askConfirm(fmt.Sprintf(T.AskDelete, s.Name, s.Host), false)
	if !ok || !yes {
		return false
	}
	secretDelete(s.ID, s.User)
	secretDelete(s.ID, "root")
	// чистим отпечаток хоста — иначе при повторном добавлении переустановленного
	// VPS подключение упадёт на несовпадении host key, а сбросить его будет негде
	forgetHostKey(s.Host, s.Port)
	st.Remove(s.ID)
	st.MustSave()
	maybeRefreshSSHConfig(st)
	fmt.Println(styGreen.Render(fmt.Sprintf(T.DeletedMsg, s.Name)))
	time.Sleep(time.Second)
	return true
}

func serverManageMenu(st *Store, s *Server) {
	for {
		statuses := map[string]probeResult{s.ID: probe(s.Host, s.Port, statusTimeout(st))}
		header := serversTable([]*Server{s}, statuses)

		opts := []huh.Option[string]{
			huh.NewOption(T.MenuConnect, "connect"),
			huh.NewOption(T.MenuInfo, "info"),
			huh.NewOption(T.MenuMonitor, "monitor"),
			huh.NewOption(T.MenuRun, "run"),
			huh.NewOption(T.MenuPasswords, "password"),
		}
		if s.Auth == "password" {
			opts = append(opts, huh.NewOption(T.MenuInstallKey, "install_key"))
		}
		opts = append(opts,
			huh.NewOption(T.MenuBootstrap, "rebootstrap"),
			huh.NewOption(T.MenuCopyCmd, "copy"),
			huh.NewOption(T.MenuForget, "forget"),
			huh.NewOption(T.MenuReboot, "reboot"),
			huh.NewOption(T.MenuEdit, "edit"),
			huh.NewOption(T.MenuDelete, "delete"),
			huh.NewOption(T.Back, "back"),
		)
		act, ok := askSelectHeader(fmt.Sprintf(T.AskActions, s.Name), header, opts)
		if !ok || act == "back" {
			return
		}
		switch act {
		case "connect":
			actionConnect(s)
		case "info":
			actionInfo(s)
		case "monitor":
			actionMonitor(s)
		case "run":
			actionRunSingle(s)
		case "password":
			actionShowPasswords(s)
		case "install_key":
			actionInstallKey(st, s)
		case "rebootstrap":
			rebootstrapFlow(st, s)
		case "copy":
			actionCopySSHCmd(s)
		case "forget":
			actionForgetHostKey(s)
		case "reboot":
			actionReboot(s)
		case "edit":
			actionEdit(st, s)
		case "delete":
			if actionDelete(st, s) {
				return
			}
		}
	}
}
