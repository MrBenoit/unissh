package app

// Первоначальная настройка сервера от root:
// обновление, пакеты, sudo-пользователь со сгенерированным паролем, SSH-ключ,
// firewall, харденинг sshd, fail2ban, автообновления, swap, таймзона, hostname.
//
// Безопасность от потери доступа:
//  1. Вход по ключу для нового пользователя ПРОВЕРЯЕТСЯ до харденинга.
//  2. После изменения sshd (порт/харденинг) вход проверяется снова;
//     при неудаче конфигурация откатывается через живую root-сессию.

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type BootstrapOpts struct {
	Username    string
	Update      bool
	Packages    bool
	Firewall    bool
	Fail2ban    bool
	Harden      bool
	RootKey     bool
	AutoUpdates bool
	Swap        bool
	NewPort     int    // 0 — не менять
	Timezone    string // "" — не менять
	Hostname    string // "" — не менять
}

type BootstrapResult struct {
	Port     int
	Password string
	Warnings []string
}

type bootstrapError struct{ msg string }

func (e *bootstrapError) Error() string { return e.msg }

func bootErr(format string, a ...any) error {
	return &bootstrapError{fmt.Sprintf(format, a...)}
}

type bootstrap struct {
	host     string
	port     int
	rootUser string
	rootPW   string
	rootKey  string

	opts        BootstrapOpts
	pubkey      string
	userKeyPath string

	// onPassword вызывается сразу после успешного chpasswd — чтобы сохранить
	// сгенерированный пароль (в Keychain) до потенциального отказа следующих шагов.
	onPassword func(user, password string)

	r          *Remote
	family     string // debian | rhel | unknown
	pm         string // apt | dnf | yum | ""
	adminGroup string

	warnings []string
	result   BootstrapResult
}

func newBootstrap(host string, port int, rootPW, rootKey string,
	opts BootstrapOpts, pubkey, userKeyPath string) *bootstrap {
	return &bootstrap{
		host: host, port: port, rootUser: "root",
		rootPW: rootPW, rootKey: rootKey,
		opts: opts, pubkey: pubkey, userKeyPath: userKeyPath,
		adminGroup: "sudo",
		result:     BootstrapResult{Port: port},
	}
}

// step выполняет шаг со спиннером; критические ошибки прерывают настройку.
func (b *bootstrap) step(title string, critical bool, fn func(onLine func(string)) error) error {
	sp := newStepSpinner(title)
	err := fn(sp.setLine)
	sp.finish(err == nil)
	if err != nil {
		fmt.Println(styRed.Render("      " + strings.ReplaceAll(err.Error(), "\n", "\n      ")))
		if critical {
			return bootErr(T.ErrCritical, title)
		}
		b.warnings = append(b.warnings, title+": "+firstLine(err.Error()))
	}
	return nil
}

func (b *bootstrap) sh(cmd string, onLine func(string), timeout time.Duration) (string, error) {
	return b.r.Sh(cmd, "", onLine, timeout)
}

// -- шаги --

func (b *bootstrap) connect(_ func(string)) error {
	r, err := dialSSH(b.host, b.port, b.rootUser, b.rootPW, b.rootKey, "", 15*time.Second)
	if err != nil {
		return err
	}
	b.r = r
	return nil
}

var osReleaseIDRe = regexp.MustCompile(`(?m)^(?:ID|ID_LIKE)=["']?([^"'\n]+)`)

func (b *bootstrap) detect(_ func(string)) error {
	out, _ := b.sh("cat /etc/os-release 2>/dev/null || true", nil, 15*time.Second)
	var ids []string
	for _, m := range osReleaseIDRe.FindAllStringSubmatch(out, -1) {
		ids = append(ids, strings.ToLower(m[1]))
	}
	joined := strings.Join(ids, " ")
	switch {
	case strings.Contains(joined, "debian") || strings.Contains(joined, "ubuntu"):
		b.family, b.pm, b.adminGroup = "debian", "apt", "sudo"
	case strings.Contains(joined, "rhel") || strings.Contains(joined, "centos") ||
		strings.Contains(joined, "fedora") || strings.Contains(joined, "rocky") ||
		strings.Contains(joined, "alma"):
		b.family, b.adminGroup = "rhel", "wheel"
		if out, _ := b.sh("command -v dnf || true", nil, 15*time.Second); strings.TrimSpace(out) != "" {
			b.pm = "dnf"
		} else {
			b.pm = "yum"
		}
	default:
		b.family = "unknown"
		b.warnings = append(b.warnings, T.WarnUnknownDistro)
	}
	return nil
}

func (b *bootstrap) updateSystem(onLine func(string)) error {
	switch b.pm {
	case "apt":
		_, err := b.sh("export DEBIAN_FRONTEND=noninteractive; "+
			"apt-get -y update && "+
			"apt-get -y -o Dpkg::Options::=--force-confdef "+
			"-o Dpkg::Options::=--force-confold full-upgrade && "+
			"apt-get -y autoremove", onLine, 30*time.Minute)
		return err
	case "dnf", "yum":
		_, err := b.sh(b.pm+" -y update", onLine, 30*time.Minute)
		return err
	}
	return bootErr("%s", T.ErrNoPkgMgr)
}

func (b *bootstrap) installPackages(onLine func(string)) error {
	switch b.pm {
	case "apt":
		_, err := b.sh("export DEBIAN_FRONTEND=noninteractive; "+
			"apt-get install -y sudo curl wget git htop vim tmux "+
			"ufw fail2ban unattended-upgrades ca-certificates", onLine, 15*time.Minute)
		return err
	case "dnf", "yum":
		_, _ = b.sh(b.pm+" -y install epel-release || true", onLine, 5*time.Minute)
		_, err := b.sh(b.pm+" -y install sudo curl wget git htop vim tmux "+
			"firewalld fail2ban", onLine, 15*time.Minute)
		return err
	}
	return bootErr("%s", T.ErrNoPkgMgr)
}

func (b *bootstrap) createUser(_ func(string)) error {
	u := b.opts.Username
	pw := genPassword(20)
	b.result.Password = pw

	group, _ := b.sh(fmt.Sprintf(
		"getent group %s >/dev/null 2>&1 && echo %s || "+
			"(getent group sudo >/dev/null 2>&1 && echo sudo || echo wheel)",
		b.adminGroup, b.adminGroup), nil, 15*time.Second)
	if g := strings.TrimSpace(group); g != "" {
		b.adminGroup = g
	}
	pmInstall := "apt-get"
	if b.pm != "" && b.pm != "apt" {
		pmInstall = b.pm
	}
	_, _ = b.sh("command -v sudo >/dev/null 2>&1 || "+pmInstall+" install -y sudo", nil, 5*time.Minute)
	_, _ = b.sh(fmt.Sprintf("getent group %s >/dev/null 2>&1 || groupadd %s",
		b.adminGroup, b.adminGroup), nil, 15*time.Second)
	if _, err := b.sh(fmt.Sprintf("id -u %s >/dev/null 2>&1 || useradd -m -s /bin/bash %s", u, u),
		nil, 30*time.Second); err != nil {
		return err
	}
	if _, err := b.sh(fmt.Sprintf("usermod -aG %s %s", b.adminGroup, u), nil, 30*time.Second); err != nil {
		return err
	}
	// пароль через stdin — без подстановки в командную строку
	if _, err := b.r.Sh("chpasswd", u+":"+pw+"\n", nil, 30*time.Second); err != nil {
		return err
	}
	// Пароль уже применён на сервере — фиксируем его сразу, чтобы он не потерялся,
	// если упадёт следующий критический шаг (иначе пароль был бы заменён на
	// неизвестный нигде). onPassword сохраняет его в Keychain.
	if b.onPassword != nil {
		b.onPassword(u, pw)
	}
	return nil
}

// authKeyInstallScript добавляет ключ из stdin (переменная $KEY) в
// authorized_keys заданной домашней директории, без дублей.
// Ключ передаётся через stdin, а не в командной строке, — .pub-комментарий
// может содержать любые символы (включая кавычки) и не должен ломать shell.
const authKeyInstallScript = `KEY=$(cat); ` +
	`mkdir -p "$HOME_DIR/.ssh" && chmod 700 "$HOME_DIR/.ssh"; ` +
	`touch "$HOME_DIR/.ssh/authorized_keys"; ` +
	`grep -qxF "$KEY" "$HOME_DIR/.ssh/authorized_keys" || printf '%s\n' "$KEY" >> "$HOME_DIR/.ssh/authorized_keys"; ` +
	`chmod 600 "$HOME_DIR/.ssh/authorized_keys"`

func (b *bootstrap) installKeys(_ func(string)) error {
	u := b.opts.Username
	// домашнюю директорию берём из getent; имя пользователя валидируется usernameRe
	userScript := fmt.Sprintf(`HOME_DIR=$(getent passwd %s | cut -d: -f6); `, u) +
		authKeyInstallScript +
		fmt.Sprintf(`; chown -R %s:%s "$HOME_DIR/.ssh"`, u, u)
	if _, err := b.r.Sh(userScript, b.pubkey, nil, 30*time.Second); err != nil {
		return err
	}
	if b.opts.RootKey {
		rootScript := `HOME_DIR=/root; ` + authKeyInstallScript
		if _, err := b.r.Sh(rootScript, b.pubkey, nil, 30*time.Second); err != nil {
			return err
		}
	}
	return nil
}

// verifyLoginRetryDelay — пауза между попытками входа. var, чтобы тесты не ждали.
var verifyLoginRetryDelay = 2 * time.Second

func (b *bootstrap) verifyUserLogin(port int) error {
	u := b.opts.Username
	var lastErr error
	for i := 0; i < 6; i++ {
		r, err := dialSSH(b.host, port, u, "", b.userKeyPath, "", 10*time.Second)
		if err == nil {
			out, err2 := r.Sh("id -nG", "", nil, 15*time.Second)
			r.Close()
			if err2 != nil {
				lastErr = err2
			} else {
				for _, g := range strings.Fields(out) {
					if g == b.adminGroup {
						return nil
					}
				}
				return bootErr(T.ErrNotInGroup, u, b.adminGroup)
			}
		} else {
			lastErr = err
		}
		time.Sleep(verifyLoginRetryDelay)
	}
	return bootErr(T.ErrKeyLogin, u, lastErr)
}

func (b *bootstrap) firewall(onLine func(string)) error {
	switch b.family {
	case "debian":
		ports := map[int]bool{b.port: true}
		if b.opts.NewPort > 0 {
			ports[b.opts.NewPort] = true
		}
		var allows []string
		for p := range ports {
			allows = append(allows, fmt.Sprintf("ufw allow %d/tcp", p))
		}
		_, err := b.sh("command -v ufw >/dev/null 2>&1 && ("+
			strings.Join(allows, "; ")+"; ufw --force enable) "+
			"|| echo 'ufw не установлен'", onLine, 2*time.Minute)
		return err
	case "rhel":
		if _, err := b.sh("systemctl enable --now firewalld", onLine, 2*time.Minute); err != nil {
			return err
		}
		_, _ = b.sh("firewall-cmd --permanent --add-service=ssh", nil, 30*time.Second)
		// открываем и текущий порт (если нестандартный), и новый — до смены порта
		ports := map[int]bool{b.port: true}
		if b.opts.NewPort > 0 {
			ports[b.opts.NewPort] = true
		}
		for p := range ports {
			_, _ = b.sh(fmt.Sprintf("firewall-cmd --permanent --add-port=%d/tcp", p),
				nil, 30*time.Second)
		}
		_, err := b.sh("firewall-cmd --reload", nil, 30*time.Second)
		return err
	}
	return bootErr("%s", T.ErrFirewallUnknown)
}

var hardenSettings = [][2]string{
	{"PasswordAuthentication", "no"},
	{"KbdInteractiveAuthentication", "no"},
	{"ChallengeResponseAuthentication", "no"},
	{"PubkeyAuthentication", "yes"},
	{"PermitRootLogin", "prohibit-password"},
	{"PermitEmptyPasswords", "no"},
	{"MaxAuthTries", "5"},
	{"LoginGraceTime", "30"},
	{"ClientAliveInterval", "120"},
	{"ClientAliveCountMax", "3"},
	{"X11Forwarding", "no"},
}

func (b *bootstrap) sshdSettings() [][2]string {
	var settings [][2]string
	if b.opts.NewPort > 0 && b.opts.NewPort != b.port {
		settings = append(settings, [2]string{"Port", fmt.Sprint(b.opts.NewPort)})
	}
	if b.opts.Harden {
		settings = append(settings, hardenSettings...)
	}
	return settings
}

const restartSSHD = "systemctl restart sshd 2>/dev/null || systemctl restart ssh " +
	"|| service sshd restart || service ssh restart"

func (b *bootstrap) configureSSHD(onLine func(string)) error {
	settings := b.sshdSettings()
	if len(settings) == 0 {
		return nil
	}
	newPort := b.opts.NewPort

	out, _ := b.sh("grep -Eq '^[[:space:]]*Include[[:space:]]+/etc/ssh/sshd_config.d' "+
		"/etc/ssh/sshd_config && echo yes || echo no", nil, 15*time.Second)
	hasInclude := strings.TrimSpace(out) == "yes"

	const dropIn = "/etc/ssh/sshd_config.d/00-unissh.conf"
	// Бэкап предыдущего состояния кладём ВНЕ sshd_config.d (иначе .conf-бэкап
	// сам попал бы под Include *.conf). revert восстанавливает именно его.
	const dropInBak = "/etc/ssh/00-unissh.conf.unissh-prev"

	apply := func() error {
		if hasInclude {
			// Сохраняем прежнее содержимое drop-in (или помечаем его отсутствие),
			// чтобы повторный bootstrap не затирал рабочую конфигурацию без отката.
			if _, err := b.sh(fmt.Sprintf(
				"if [ -f %s ]; then cp %s %s; else rm -f %s; fi",
				dropIn, dropIn, dropInBak, dropInBak), nil, 15*time.Second); err != nil {
				return err
			}
			// 00- лексически раньше 50-cloud-init.conf: в sshd выигрывает ПЕРВОЕ значение
			var sb strings.Builder
			sb.WriteString("# managed by unissh\n")
			for _, kv := range settings {
				fmt.Fprintf(&sb, "%s %s\n", kv[0], kv[1])
			}
			if _, err := b.sh("mkdir -p /etc/ssh/sshd_config.d", nil, 15*time.Second); err != nil {
				return err
			}
			return b.r.WriteFile(dropIn, sb.String(), "644")
		}
		// backup только если ещё не делали (cp -n на coreutils>=9.2 даёт код 1 —
		// используем явную проверку существования)
		if _, err := b.sh("test -e /etc/ssh/sshd_config.unissh-backup || "+
			"cp /etc/ssh/sshd_config /etc/ssh/sshd_config.unissh-backup",
			nil, 15*time.Second); err != nil {
			return err
		}
		for _, kv := range settings {
			// Удаляем прежние глобальные директивы только ДО первого Match-блока
			// (внутри Match отступ не требуется — трогать их нельзя), вставляем свои в начало.
			if _, err := b.sh(fmt.Sprintf(
				"sed -i -e '/^[Mm]atch /,$!{/^%s[[:space:]]/d}' /etc/ssh/sshd_config",
				kv[0]), nil, 15*time.Second); err != nil {
				return err
			}
			if _, err := b.sh(fmt.Sprintf("sed -i -e '1i %s %s' /etc/ssh/sshd_config",
				kv[0], kv[1]), nil, 15*time.Second); err != nil {
				return err
			}
		}
		return nil
	}
	revert := func() {
		if hasInclude {
			// восстанавливаем прежнее содержимое drop-in, а не удаляем его вслепую
			_, _ = b.sh(fmt.Sprintf("if [ -f %s ]; then mv -f %s %s; else rm -f %s; fi",
				dropInBak, dropInBak, dropIn, dropIn), nil, 15*time.Second)
		} else {
			_, _ = b.sh("cp /etc/ssh/sshd_config.unissh-backup /etc/ssh/sshd_config", nil, 15*time.Second)
		}
		_, _ = b.sh(restartSSHD, nil, time.Minute)
	}

	if err := apply(); err != nil {
		revert()
		return err
	}
	if code, out, _ := b.r.Run("sshd -t 2>&1 || /usr/sbin/sshd -t 2>&1", "", nil, 30*time.Second); code != 0 {
		revert()
		return bootErr(T.ErrSshdTest, firstLine(out))
	}

	if newPort > 0 {
		// Гарантируем, что новый порт открыт в файрволе, даже если шаг «firewall»
		// не выбирался (иначе смена порта отрежет доступ). Идемпотентно, ошибки игнорируем.
		_, _ = b.sh(fmt.Sprintf(
			"command -v ufw >/dev/null 2>&1 && ufw allow %d/tcp || true; "+
				"command -v firewall-cmd >/dev/null 2>&1 && "+
				"firewall-cmd --permanent --add-port=%d/tcp >/dev/null 2>&1 && "+
				"firewall-cmd --reload >/dev/null 2>&1 || true",
			newPort, newPort), nil, time.Minute)
		// Ubuntu 22.10+ слушает порт через ssh.socket — переключаемся на службу
		_, _ = b.sh("if systemctl -q is-enabled ssh.socket 2>/dev/null || "+
			"systemctl -q is-active ssh.socket 2>/dev/null; then "+
			"systemctl disable --now ssh.socket; systemctl enable ssh.service; fi",
			nil, time.Minute)
		if b.family == "rhel" {
			_, _ = b.sh(fmt.Sprintf("command -v semanage >/dev/null 2>&1 && "+
				"semanage port -a -t ssh_port_t -p tcp %d || true", newPort),
				nil, time.Minute)
		}
	}

	if _, err := b.sh(restartSSHD, nil, time.Minute); err != nil {
		revert()
		return bootErr(T.ErrSshdRestartRevert, err)
	}

	checkPort := b.port
	if newPort > 0 {
		checkPort = newPort
	}
	onLine(T.CheckingAfterSshd)
	if err := b.verifyUserLogin(checkPort); err != nil {
		revert()
		return bootErr(T.ErrSshdVerifyRevert, err)
	}
	if newPort > 0 {
		b.result.Port = newPort
	}
	return nil
}

func (b *bootstrap) fail2ban(_ func(string)) error {
	jail := fmt.Sprintf("[sshd]\nenabled = true\nport = %d\nmaxretry = 5\nbantime = 1h\nfindtime = 10m\n",
		b.result.Port)
	if _, err := b.sh("mkdir -p /etc/fail2ban", nil, 15*time.Second); err != nil {
		return err
	}
	if err := b.r.WriteFile("/etc/fail2ban/jail.local", jail, "644"); err != nil {
		return err
	}
	_, err := b.sh("systemctl enable --now fail2ban && systemctl restart fail2ban",
		nil, 2*time.Minute)
	return err
}

func (b *bootstrap) autoUpdates(onLine func(string)) error {
	switch b.family {
	case "debian":
		content := "APT::Periodic::Update-Package-Lists \"1\";\n" +
			"APT::Periodic::Unattended-Upgrade \"1\";\n"
		return b.r.WriteFile("/etc/apt/apt.conf.d/20auto-upgrades", content, "644")
	case "rhel":
		_, err := b.sh(b.pm+" -y install dnf-automatic && "+
			"systemctl enable --now dnf-automatic.timer", onLine, 5*time.Minute)
		return err
	}
	return bootErr("%s", T.ErrAutoUpdUnknown)
}

func (b *bootstrap) swap(onLine func(string)) error {
	_, err := b.sh(
		"if swapon --show 2>/dev/null | grep -q .; then echo 'swap уже есть'; else "+
			"(fallocate -l 2G /swapfile || dd if=/dev/zero of=/swapfile bs=1M count=2048) "+
			"&& chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile "+
			"&& (grep -q '^/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab) "+
			"&& sysctl vm.swappiness=10 "+
			"&& (grep -q 'vm.swappiness' /etc/sysctl.conf || echo 'vm.swappiness=10' >> /etc/sysctl.conf); fi",
		onLine, 5*time.Minute)
	return err
}

var tzRe = regexp.MustCompile(`^[A-Za-z0-9_+\-/]+$`)
var hostnameRe = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)

func (b *bootstrap) timezone(_ func(string)) error {
	if !tzRe.MatchString(b.opts.Timezone) {
		return bootErr(T.ErrBadTz, b.opts.Timezone)
	}
	_, err := b.sh("timedatectl set-timezone "+b.opts.Timezone, nil, 30*time.Second)
	return err
}

func (b *bootstrap) hostname(_ func(string)) error {
	if !hostnameRe.MatchString(b.opts.Hostname) {
		return bootErr(T.ErrBadHostname, b.opts.Hostname)
	}
	_, err := b.sh("hostnamectl set-hostname "+b.opts.Hostname, nil, 30*time.Second)
	return err
}

// Run выполняет все шаги по порядку.
func (b *bootstrap) Run() (BootstrapResult, error) {
	o := b.opts
	fmt.Println()
	fmt.Println(styTitle.Render(fmt.Sprintf(T.SetupTitle, b.host)))
	fmt.Println()
	defer func() {
		if b.r != nil {
			b.r.Close()
		}
	}()

	type stage struct {
		title    string
		critical bool
		enabled  bool
		fn       func(func(string)) error
	}
	stages := []stage{
		{T.StepConnectRoot, true, true, b.connect},
		{T.StepDetect, true, true, b.detect},
		{T.StepUpdate, false, o.Update, b.updateSystem},
		{T.StepPackages, false, o.Packages, b.installPackages},
		{fmt.Sprintf(T.StepCreateUser, o.Username), true, true, b.createUser},
		{T.StepInstallKey, true, true, b.installKeys},
		{fmt.Sprintf(T.StepVerifyKey, o.Username), true, true,
			func(_ func(string)) error { return b.verifyUserLogin(b.port) }},
		{T.StepFirewall, false, o.Firewall, b.firewall},
		{T.StepSSHD, false, len(b.sshdSettings()) > 0, b.configureSSHD},
		{T.StepFail2ban, false, o.Fail2ban, b.fail2ban},
		{T.StepAutoUpd, false, o.AutoUpdates, b.autoUpdates},
		{T.StepSwap, false, o.Swap, b.swap},
		{T.StepTimezone + o.Timezone, false, o.Timezone != "", b.timezone},
		{T.StepHostname + o.Hostname, false, o.Hostname != "", b.hostname},
	}
	for _, st := range stages {
		if !st.enabled {
			continue
		}
		if err := b.step(st.title, st.critical, st.fn); err != nil {
			return b.result, err
		}
	}
	b.result.Warnings = b.warnings
	return b.result, nil
}
