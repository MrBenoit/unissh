package app

// Локализация: все строки интерфейса. Язык хранится в настройках
// (settings.language: "ru" | "en") и переключается на лету.

type L struct {
	// общее
	Yes, No, Back, Done, PressEnter, Cancelled string

	// баннер
	Tagline string

	// config
	ErrReadConfig, BackupSaved, ErrSaveConfig string

	// sshkey
	AskGenKey, ErrKeyMissing, KeyCreated string

	// remote
	ErrHostKeyChanged, ErrReadKey, ErrParseKey, ErrNoAuthMethod string
	ErrAuthFailed, ErrConnectFailed, ErrConnLost, ErrCmdTimeout string
	ErrCmdExitCode, AskPasswordFor, ErrNoPwInKeychain           string

	// таблица
	ColName, ColUser, ColAddr, ColPort, ColTags, ColPing, ColNotes, MsFmt string

	// подключение
	PwCopiedClipboard, Connecting, SessionEnded, CopiedCmd string

	// инфо
	Gathering, LblOS, LblHost, LblKernel, LblUptime, LblIP      string
	LblLoad, LblMemory, LblDiskRoot, LblDisk, CPUsFmt, TopProcs string

	// монитор
	ErrUnexpectedResp, MonError, MonQuitHint, MonConnecting string
	LblCPU, LblRAM, LblNet, LiveSuffix, MonitorStopped      string

	// команды
	AskCommand, RunningOn, RunningOnMany, NoServersYet string
	StatusOK, StatusCode, NoOutput, AskPickServers     string

	// bootstrap
	SetupTitle, StepConnectRoot, StepDetect, StepUpdate, StepPackages string
	StepCreateUser, StepInstallKey, StepVerifyKey, StepFirewall       string
	StepSSHD, StepFail2ban, StepAutoUpd, StepSwap, StepTimezone       string
	StepHostname, WarnUnknownDistro, ErrCritical, ErrNoPkgMgr         string
	ErrNotInGroup, ErrKeyLogin, ErrFirewallUnknown, ErrSshdTest       string
	ErrSshdRestartRevert, ErrSshdVerifyRevert, CheckingAfterSshd      string
	ErrAutoUpdUnknown, ErrBadTz, ErrBadHostname                       string

	// валидация
	ErrNameEmpty, ErrNameChars, ErrNameTaken, ErrHostFmt string
	ErrPortRange, ErrUserFmt, ErrTzFmt, ErrHostnameFmt   string
	ErrNumber                                            string

	// мастер настройки
	WizardTitle, OptUpdate, OptPackages, OptFirewall, OptFail2ban string
	OptHarden, OptRootKey, OptAutoUpd, OptPort, OptSwap           string
	OptTz, OptHostname, AskNewUsername, AskNewPort, AskTimezone   string
	ManualEntry, AskTzManual, AskHostname                         string

	// добавление / результат
	SetupAborted, ServerNotSaved, SumServer, SumLogin, SumPassword   string
	SumKey, SumKeychain, SumKeychainVal, ConfiguredOK, WarnNoPwLogin string
	UserPwCopied, AskConnectNow, TitleNewServer, AskName, AskHost    string
	AskPort, AskTags, AskHowAdd, OptWithBootstrap, OptPlainAdd       string
	AskRootAuth, OptRootPassword, OptRootPasswordKeychain            string
	OptRootKeyFile, AskRootPassword, AskRootKeyPath, ErrNoKeySetup   string
	CheckingRootAccess, RootConfirmed, AskUser, AskAuth, OptAuthKey  string
	OptAuthPassword, AskKeyPath, WarnKeychainFail, ServerAdded       string
	NeedRootAccess, RootPwFromKeychain                               string

	// действия с сервером
	NoPasswordsStored, TitlePasswords, PwUserCopied, InstallingKey string
	KeyInstalled, AskReboot, AskSudoPassword, RebootSent           string
	FingerprintReset, TitleEdit, FldName, FldHost, FldPort         string
	FldUser, FldKey, FldTags, FldNotes, KeyNoneLabel, AskNewName   string
	FldProxy, AskProxy                                             string
	AskNewHost, AskNewPortEdit, AskNewUser, AskKeyOrEmpty          string
	AskTagsEdit, AskNotes, AskDelete, DeletedMsg                   string
	MenuConnect, MenuInfo, MenuMonitor, MenuRun, MenuPasswords     string
	MenuInstallKey, MenuBootstrap, MenuCopyCmd, MenuForget         string
	MenuReboot, MenuEdit, MenuDelete, AskActions                   string

	// инструменты и настройки
	ExportHeader, ExportedTo, ExportHint, TitlePubkey, PubkeyCopied string
	SetDefaultUser, SetSSHKey, SetTimeout, SetLanguage              string
	SetBootstrapSteps, TitleSettings, AskDefaultUser, AskSSHKeyPath string
	AskTimeout, TitleTools, ToolMulti, ToolExport, ToolPubkey       string
	ToolSettings, AskLanguage, StepsSaved                           string

	// главное меню
	EmptyTitle, EmptyBody, MainPrompt, ItemAdd, ItemManage     string
	ItemTools, ItemExit, AskWhichServer, Goodbye, ProbeSpinner string

	// TUI (главный экран)
	SectionServers, SectionActions, TuiHint, TuiHintEmpty, TuiHintSelect string
	TuiHintInput, TuiHintMulti, MoreAbove, MoreBelow, BackupServersFmt   string

	// импорт / бэкапы / сниппеты / темы
	ImportReadErr, ImportNothing, ImportTitle, ImportAdded                       string
	ToolImport, ToolBackups, BackupTitle, BackupRestored                         string
	BackupNothing, BackupConfirm, SetTheme, AskTheme                             string
	SnippetManual, SnippetSaveQ, AskSnippetName                                  string
	ToolSnippets, SnippetsTitle, NoSnippetsYet, SnippetDeleted, AskDeleteSnippet string
	NoLastConnected, LastConnFmt                                                 string
	RelNow, RelMin, RelHour, RelDay                                              string

	// футер, фильтр, сортировка, помощь
	FooterServer, FooterAction, FilterPrompt, FilterNoMatch string
	SortByName, SortByPing, SortByOnline                    string
	ConfirmConnectTitle, ConnectBtn, CancelBtn              string
	SetConfirmConnect, AskConfirmConnect                    string
	HelpTitle, HelpClose, HelpNavigate, HelpConnect         string
	HelpInfo, HelpBootstrap, HelpManage, HelpEdit           string
	HelpReboot, HelpDelete, HelpFavorite, HelpSort          string
	HelpFilter, HelpHelp, HelpQuit                          string

	// CLI
	UsageTagline, UsageMenu, UsageConnect, UsageList, UsageAdd string
	UsageVersion, UsageTitle, NoServersCLI, ErrNotFound        string
	HaveList, ErrAddTTY                                        string

	// добавлено после ревью
	BytesUnits      []string // единицы размера: Б/КБ/МБ… для humanBytes
	AbortedPassword string   // «Пароль %s уже установлен на сервере: %s»
	ErrTimeoutRange string   // валидация таймаута (> 0)
	RootAuthHint    string   // подсказка про вход root после харденинга

	// публичные IP-адреса
	LblPublicIP    string // заголовок панели
	DetectingPubIP string // спиннер
	PubIPUnique    string // «Уникальные: %s»
	PubIPNone      string // не удалось определить
	SetShowPubIP   string // пункт настроек
	AskShowPubIP   string // вопрос-переключатель

	// все IP-адреса сервера / режим открытия SSH
	LblIPs            string // метка блока адресов в инфо-панели
	GatheringOnAdd    string // спиннер при инспекции после добавления
	SetConnectMode    string // пункт настроек
	AskConnectMode    string // вопрос
	ConnectModeWindow string // «в новом окне»
	ConnectModeInline string // «в этом же окне»
	OpenedNewWindow   string // сообщение после запуска нового окна
	ErrOpenWindow     string // ошибка открытия окна
}

var trRU = L{
	Yes: "Да", No: "Нет", Back: "← Назад", Done: "← Готово",
	PressEnter: "Enter — продолжить ", Cancelled: "отменено",

	Tagline: "  менеджер VPS · выбери сервер — и ты внутри",

	ErrReadConfig: "Не удалось прочитать конфиг: %s",
	BackupSaved:   "Копия сохранена в %s",
	ErrSaveConfig: "Не удалось сохранить конфиг: %s",

	AskGenKey:     "SSH-ключ %s не найден. Сгенерировать новый ed25519-ключ?",
	ErrKeyMissing: "ключ %s отсутствует",
	KeyCreated:    "Создан новый ключ: ",

	ErrHostKeyChanged: "отпечаток хоста изменился! Если сервер переустановлен — сбросьте отпечаток в меню сервера (%v)",
	ErrReadKey:        "не удалось прочитать ключ %s: %v",
	ErrParseKey:       "не удалось разобрать ключ %s: %v",
	ErrNoAuthMethod:   "не задан ни ключ, ни пароль",
	ErrAuthFailed:     "ошибка аутентификации %s@%s:%d — неверный пароль или ключ",
	ErrConnectFailed:  "не удалось подключиться к %s:%d: %v",
	ErrConnLost:       "соединение потеряно: %v",
	ErrCmdTimeout:     "команда не ответила за %s",
	ErrCmdExitCode:    "команда завершилась с кодом %d:\n%s",
	AskPasswordFor:    "Пароль %s@%s:",
	ErrNoPwInKeychain: "пароль не найден в Keychain",

	ColName: "Имя", ColUser: "Логин", ColAddr: "Адрес", ColPort: "Порт",
	ColTags: "Теги", ColPing: "Пинг", ColNotes: "Заметки", MsFmt: "%s мс",

	PwCopiedClipboard: "🔑 Пароль скопирован в буфер обмена — вставьте по запросу",
	Connecting:        "⇄ Подключаюсь: ",
	SessionEnded:      "\nСессия завершена (код %d).",
	CopiedCmd:         "✓ Скопировано: ",

	Gathering: "Собираю данные с %s…",
	LblOS:     "ОС", LblHost: "Хост", LblKernel: "Ядро", LblUptime: "Аптайм", LblIP: "IP",
	LblLoad: "Load", LblMemory: "Память", LblDiskRoot: "Диск /", LblDisk: "Диск",
	CPUsFmt: "  (%d CPU)", TopProcs: "Топ процессов (CPU)",

	ErrUnexpectedResp: "неожиданный ответ сервера",
	MonError:          "Ошибка: ", MonQuitHint: "  q — выход", MonConnecting: "  Подключаюсь…",
	LblCPU: "CPU", LblRAM: "RAM", LblNet: "Сеть",
	LiveSuffix: " · live · q — выход", MonitorStopped: "Монитор остановлен: ",

	AskCommand: "Команда:", RunningOn: "Выполняю на %s…",
	RunningOnMany: "Выполняю на %d серверах…", NoServersYet: "Нет серверов.",
	StatusOK: "OK", StatusCode: "код %d", NoOutput: "(нет вывода)",
	AskPickServers: "На каких серверах выполнить команду?",

	SetupTitle:      "⚙ Настройка %s",
	StepConnectRoot: "Подключение по SSH (root)",
	StepDetect:      "Определение дистрибутива",
	StepUpdate:      "Обновление системы (может занять несколько минут)",
	StepPackages:    "Установка базовых пакетов",
	StepCreateUser:  "Создание пользователя %s (sudo + пароль)",
	StepInstallKey:  "Установка SSH-ключа",
	StepVerifyKey:   "Проверка входа по ключу (%s)",
	StepFirewall:    "Настройка firewall",
	StepSSHD:        "Настройка sshd (порт/харденинг) с проверкой доступа",
	StepFail2ban:    "Настройка fail2ban",
	StepAutoUpd:     "Автообновления безопасности",
	StepSwap:        "Создание swap-файла 2 ГБ",
	StepTimezone:    "Часовой пояс: ",
	StepHostname:    "Имя хоста: ",

	WarnUnknownDistro:    "Неизвестный дистрибутив — пакеты, firewall и автообновления пропущены",
	ErrCritical:          "критический шаг «%s» не выполнен",
	ErrNoPkgMgr:          "нет поддерживаемого пакетного менеджера",
	ErrNotInGroup:        "пользователь %s не состоит в группе %s",
	ErrKeyLogin:          "вход по ключу как %s не работает: %v",
	ErrFirewallUnknown:   "firewall пропущен: неизвестный дистрибутив",
	ErrSshdTest:          "sshd -t: конфигурация некорректна, изменения откатились: %s",
	ErrSshdRestartRevert: "не удалось перезапустить sshd, изменения откатились: %v",
	ErrSshdVerifyRevert:  "после изменения sshd вход не работает — изменения откатились (%v)",
	CheckingAfterSshd:    "проверяю вход по ключу после изменений…",
	ErrAutoUpdUnknown:    "автообновления: неизвестный дистрибутив",
	ErrBadTz:             "подозрительное имя таймзоны: %s",
	ErrBadHostname:       "недопустимое имя хоста: %s",

	ErrNameEmpty:   "имя не может быть пустым",
	ErrNameChars:   "только буквы, цифры, точки, дефисы и подчёркивания",
	ErrNameTaken:   "имя уже занято",
	ErrHostFmt:     "введите IP-адрес или домен без пробелов",
	ErrPortRange:   "порт — число от 1 до 65535",
	ErrUserFmt:     "строчные латинские буквы, цифры, - и _; начинается с буквы",
	ErrTzFmt:       "формат: Region/City",
	ErrHostnameFmt: "только буквы, цифры, точки и дефисы",
	ErrNumber:      "введите число",

	WizardTitle:    "Что настроить на сервере? (пробел — выбрать, Enter — подтвердить)",
	OptUpdate:      "Обновить систему (apt/dnf upgrade)",
	OptPackages:    "Базовые пакеты (git, htop, tmux, ufw, fail2ban…)",
	OptFirewall:    "Firewall (ufw/firewalld: разрешить SSH, включить)",
	OptFail2ban:    "fail2ban (защита от перебора паролей)",
	OptHarden:      "Харденинг SSH (вход только по ключу)",
	OptRootKey:     "Добавить ключ и для root (запасной доступ)",
	OptAutoUpd:     "Автообновления безопасности",
	OptPort:        "Сменить SSH-порт",
	OptSwap:        "Создать swap 2 ГБ",
	OptTz:          "Установить часовой пояс",
	OptHostname:    "Установить имя хоста",
	AskNewUsername: "Имя нового пользователя:",
	AskNewPort:     "Новый SSH-порт:",
	AskTimezone:    "Часовой пояс:",
	ManualEntry:    "ввести вручную…",
	AskTzManual:    "Таймзона (например Europe/Moscow):",
	AskHostname:    "Имя хоста:",

	SetupAborted:   "Настройка прервана: ",
	ServerNotSaved: "Сервер не сохранён. Проверьте доступ и попробуйте снова.",
	SumServer:      "Сервер", SumLogin: "Логин", SumPassword: "Пароль",
	SumKey: "Ключ", SumKeychain: "Keychain", SumKeychainVal: "пароли сохранены (unissh)",
	ConfiguredOK:            "✓ %s настроен",
	WarnNoPwLogin:           "⚠ Вход по паролю отключён — не теряйте ключ %s",
	UserPwCopied:            "Пароль пользователя скопирован в буфер обмена",
	AskConnectNow:           "Подключиться сейчас?",
	TitleNewServer:          "➕ Новый сервер",
	AskName:                 "Имя (короткий алиас):",
	AskHost:                 "Адрес (IP или домен):",
	AskPort:                 "SSH-порт:",
	AskTags:                 "Теги через пробел (опционально):",
	AskHowAdd:               "Как добавить сервер?",
	OptWithBootstrap:        "🚀 С первоначальной настройкой (root → sudo-юзер, ключ, firewall…)",
	OptPlainAdd:             "📎 Просто добавить (сервер уже настроен)",
	AskRootAuth:             "Как подключиться под root?",
	OptRootPassword:         "Пароль root",
	OptRootPasswordKeychain: "Пароль root (из Keychain, если сохранён)",
	OptRootKeyFile:          "SSH-ключ (файл)",
	AskRootPassword:         "Пароль root:",
	AskRootKeyPath:          "Путь к приватному ключу root:",
	ErrNoKeySetup:           "Без SSH-ключа настройка невозможна: ",
	CheckingRootAccess:      "Проверяю доступ root@%s:%d…",
	RootConfirmed:           "Root-доступ подтверждён",
	AskUser:                 "Пользователь:",
	AskAuth:                 "Аутентификация:",
	OptAuthKey:              "SSH-ключ",
	OptAuthPassword:         "Пароль (сохранить в Keychain)",
	AskKeyPath:              "Путь к приватному ключу:",
	WarnKeychainFail:        "⚠ Не удалось сохранить пароль в Keychain: ",
	ServerAdded:             "✓ Сервер %s добавлен.",
	NeedRootAccess:          "Для настройки нужен root-доступ к %s.",
	RootPwFromKeychain:      "Пароль root найден в Keychain",

	NoPasswordsStored: "В Keychain нет паролей для этого сервера.",
	TitlePasswords:    "Пароли (Keychain)",
	PwUserCopied:      "Пароль %s скопирован в буфер обмена",
	InstallingKey:     "Устанавливаю ключ на %s…",
	KeyInstalled:      "✓ Ключ установлен — теперь вход без пароля.",
	AskReboot:         "Перезагрузить %s (%s)?",
	AskSudoPassword:   "Пароль %s (для sudo):",
	RebootSent:        "✓ Команда перезагрузки отправлена.",
	FingerprintReset:  "✓ Отпечаток %s сброшен.",
	TitleEdit:         "Редактирование %s:",
	FldName:           "Имя: ", FldHost: "Адрес: ", FldPort: "Порт: %d", FldUser: "Пользователь: ",
	FldKey: "Ключ: ", FldTags: "Теги: ", FldNotes: "Заметки: ", KeyNoneLabel: "— (пароль)",
	FldProxy: "Бастион (ProxyJump): ", AskProxy: "ProxyJump (user@host[:port], пусто — без бастиона):",
	AskNewName: "Новое имя:", AskNewHost: "Новый адрес:", AskNewPortEdit: "Новый порт:",
	AskNewUser: "Новый пользователь:", AskKeyOrEmpty: "Путь к ключу (пусто — вход по паролю):",
	AskTagsEdit: "Теги через пробел:", AskNotes: "Заметки:",
	AskDelete:   "Удалить %s (%s)?",
	DeletedMsg:  "✓ Сервер %s удалён (пароли из Keychain тоже).",
	MenuConnect: "🔌 Подключиться", MenuInfo: "📊 Информация о сервере",
	MenuMonitor: "📈 Live-монитор", MenuRun: "▶️  Выполнить команду",
	MenuPasswords:  "🔑 Показать пароли (Keychain)",
	MenuInstallKey: "🧩 Установить SSH-ключ (вход без пароля)",
	MenuBootstrap:  "🛠  Первоначальная настройка (bootstrap)",
	MenuCopyCmd:    "📋 Скопировать ssh-команду",
	MenuForget:     "♻️  Сбросить отпечаток хоста",
	MenuReboot:     "🔁 Перезагрузить сервер",
	MenuEdit:       "✏️  Редактировать", MenuDelete: "🗑  Удалить",
	AskActions: "%s — действия:",

	ExportHeader: "# Сгенерировано unissh — не редактируйте вручную",
	ExportedTo:   "✓ Экспортировано в %s",
	ExportHint:   "Теперь работает: ssh <имя-сервера> из любого терминала",
	TitlePubkey:  "Публичный ключ", PubkeyCopied: "Скопирован в буфер обмена",
	SetDefaultUser:    "Пользователь по умолчанию: ",
	SetSSHKey:         "SSH-ключ: ",
	SetTimeout:        "Таймаут проверки статуса: %.1f с",
	SetLanguage:       "Язык: ",
	SetBootstrapSteps: "Шаги настройки нового сервера (по умолчанию)",
	TitleSettings:     "Настройки:",
	AskDefaultUser:    "Пользователь по умолчанию:",
	AskSSHKeyPath:     "Путь к SSH-ключу:",
	AskTimeout:        "Таймаут (сек):",
	TitleTools:        "Инструменты:",
	ToolMulti:         "▶️  Выполнить команду на нескольких серверах",
	ToolExport:        "📤 Экспорт в ~/.ssh/config (ssh <имя>)",
	ToolPubkey:        "🔑 Показать/скопировать публичный ключ",
	ToolSettings:      "⚙️  Настройки",
	AskLanguage:       "Язык интерфейса:",
	StepsSaved:        "✓ Шаги по умолчанию сохранены.",

	EmptyTitle: "Серверов пока нет.",
	EmptyBody: "Добавьте первый — unissh сам обновит его, создаст sudo-пользователя\n" +
		"со сгенерированным паролем, установит ключ и включит защиту.",
	MainPrompt:     "Выберите сервер (Enter — подключиться) или действие:",
	ItemAdd:        "➕ Добавить сервер",
	ItemManage:     "🛠  Управление серверами",
	ItemTools:      "🧰 Инструменты",
	ItemExit:       "⏻  Выход",
	AskWhichServer: "Какой сервер?",
	Goodbye:        "До встречи!",
	ProbeSpinner:   "Проверяю доступность серверов…",

	SectionServers:   "СЕРВЕРЫ",
	SectionActions:   "ДЕЙСТВИЯ",
	TuiHint:          "↑↓/клик — выбор · Enter — подключиться · m — управление · q — выход",
	TuiHintEmpty:     "↑↓/клик — выбор · Enter — выбрать · q — выход",
	TuiHintSelect:    "↑↓/клик — выбор · Enter — выбрать · Esc — назад",
	TuiHintInput:     "Enter — подтвердить · Esc — отмена",
	TuiHintMulti:     "↑↓ — выбор · пробел/клик — отметить · Enter — готово · Esc — отмена",
	MoreAbove:        "  ▲ ещё %d",
	MoreBelow:        "  ▼ ещё %d",
	BackupServersFmt: "%s  · серверов: %d",

	ImportReadErr:    "Не удалось прочитать ~/.ssh/config: %v",
	ImportNothing:    "В ~/.ssh/config нет новых хостов для импорта.",
	ImportTitle:      "Импорт из ~/.ssh/config (найдено %d):",
	ImportAdded:      "✓ Импортировано серверов: %d",
	ToolImport:       "📥 Импорт из ~/.ssh/config",
	ToolBackups:      "🗂 Восстановить конфиг из бэкапа",
	BackupTitle:      "Выберите бэкап для восстановления:",
	BackupRestored:   "✓ Конфиг восстановлен из бэкапа.",
	BackupNothing:    "Бэкапов пока нет.",
	BackupConfirm:    "Восстановить конфиг из %s? Текущий будет перезаписан.",
	SetTheme:         "Тема: ",
	AskTheme:         "Цветовая тема:",
	SnippetManual:    "✎ ввести команду вручную…",
	SnippetSaveQ:     "Сохранить как сниппет для повторного использования?",
	AskSnippetName:   "Название сниппета:",
	ToolSnippets:     "✎ Сниппеты команд",
	SnippetsTitle:    "Сниппеты (выберите для удаления):",
	NoSnippetsYet:    "Сниппетов пока нет — их можно сохранять при выполнении команды.",
	SnippetDeleted:   "✓ Сниппет удалён.",
	AskDeleteSnippet: "Удалить сниппет «%s»?",
	NoLastConnected:  "ещё не подключались",
	LastConnFmt:      "заходили %s назад",
	RelNow:           "только что",
	RelMin:           "%dм",
	RelHour:          "%dч",
	RelDay:           "%dд",

	FooterServer:        "enter/c подключиться · i инфо · m меню · e правка · d удалить · / поиск · ? все клавиши",
	FooterAction:        "enter выбрать · s сортировка · / поиск · ? все клавиши · q выход",
	FilterPrompt:        "🔍 ",
	FilterNoMatch:       "нет серверов по фильтру",
	SortByName:          "сортировка: по имени",
	SortByPing:          "сортировка: по пингу",
	SortByOnline:        "сортировка: по доступности",
	ConfirmConnectTitle: "Подключиться к серверу %s?",
	ConnectBtn:          "❯ Подключиться",
	CancelBtn:           "✗ Отмена",
	SetConfirmConnect:   "Подтверждать подключение: ",
	AskConfirmConnect:   "Спрашивать подтверждение при подключении?",
	HelpTitle:           "Горячие клавиши",
	HelpClose:           "любая клавиша — закрыть",
	HelpNavigate:        "навигация",
	HelpConnect:         "подключиться",
	HelpInfo:            "информация о сервере",
	HelpBootstrap:       "первоначальная настройка",
	HelpManage:          "меню управления сервером",
	HelpEdit:            "редактировать",
	HelpReboot:          "перезагрузить",
	HelpDelete:          "удалить",
	HelpFavorite:        "избранное (★)",
	HelpSort:            "сменить сортировку",
	HelpFilter:          "поиск/фильтр",
	HelpHelp:            "эта справка",
	HelpQuit:            "выход",

	UsageTagline: "Терминальный менеджер VPS: выбери сервер — и ты внутри.",
	UsageTitle:   "Использование:",
	UsageMenu:    "интерактивное меню",
	UsageConnect: "быстрое подключение к серверу",
	UsageList:    "список серверов со статусами",
	UsageAdd:     "добавить сервер",
	UsageVersion: "показать версию",
	NoServersCLI: "Серверов нет. Запустите unissh и добавьте первый.",
	ErrNotFound:  "Сервер «%s» не найден.",
	HaveList:     "Есть: %s",
	ErrAddTTY:    "unissh add работает только в интерактивном терминале",

	BytesUnits:      []string{"Б", "КБ", "МБ", "ГБ", "ТБ", "ПБ"},
	AbortedPassword: "⚠ Пароль пользователя %s уже установлен на сервере: %s (скопирован в буфер)",
	ErrTimeoutRange: "введите число больше 0",
	RootAuthHint:    "После харденинга вход root по паролю отключён — используйте SSH-ключ.",

	LblPublicIP:    "Публичные IP (внешние API)",
	DetectingPubIP: "Определяю публичные адреса…",
	PubIPUnique:    "Уникальные: %s",
	PubIPNone:      "не удалось определить публичные адреса",
	SetShowPubIP:   "Публичные IP при подключении: ",
	AskShowPubIP:   "Показывать публичные IP при подключении?",

	LblIPs:            "IP-адреса",
	GatheringOnAdd:    "Собираю информацию о сервере…",
	SetConnectMode:    "Открывать SSH: ",
	AskConnectMode:    "Как открывать SSH-подключение?",
	ConnectModeWindow: "в новом окне терминала",
	ConnectModeInline: "в этом же окне",
	OpenedNewWindow:   "↗ Подключение открыто в новом окне терминала",
	ErrOpenWindow:     "не удалось открыть новое окно (%v) — подключаюсь здесь",
}

var trEN = L{
	Yes: "Yes", No: "No", Back: "← Back", Done: "← Done",
	PressEnter: "Enter — continue ", Cancelled: "cancelled",

	Tagline: "  VPS manager · pick a server and you're in",

	ErrReadConfig: "Failed to read config: %s",
	BackupSaved:   "Backup saved to %s",
	ErrSaveConfig: "Failed to save config: %s",

	AskGenKey:     "SSH key %s not found. Generate a new ed25519 key?",
	ErrKeyMissing: "key %s is missing",
	KeyCreated:    "New key created: ",

	ErrHostKeyChanged: "host key changed! If the server was reinstalled, reset the fingerprint in the server menu (%v)",
	ErrReadKey:        "failed to read key %s: %v",
	ErrParseKey:       "failed to parse key %s: %v",
	ErrNoAuthMethod:   "neither key nor password provided",
	ErrAuthFailed:     "authentication failed for %s@%s:%d — wrong password or key",
	ErrConnectFailed:  "failed to connect to %s:%d: %v",
	ErrConnLost:       "connection lost: %v",
	ErrCmdTimeout:     "command did not respond within %s",
	ErrCmdExitCode:    "command exited with code %d:\n%s",
	AskPasswordFor:    "Password for %s@%s:",
	ErrNoPwInKeychain: "password not found in Keychain",

	ColName: "Name", ColUser: "Login", ColAddr: "Address", ColPort: "Port",
	ColTags: "Tags", ColPing: "Ping", ColNotes: "Notes", MsFmt: "%s ms",

	PwCopiedClipboard: "🔑 Password copied to clipboard — paste when prompted",
	Connecting:        "⇄ Connecting: ",
	SessionEnded:      "\nSession ended (code %d).",
	CopiedCmd:         "✓ Copied: ",

	Gathering: "Gathering data from %s…",
	LblOS:     "OS", LblHost: "Host", LblKernel: "Kernel", LblUptime: "Uptime", LblIP: "IP",
	LblLoad: "Load", LblMemory: "Memory", LblDiskRoot: "Disk /", LblDisk: "Disk",
	CPUsFmt: "  (%d CPU)", TopProcs: "Top processes (CPU)",

	ErrUnexpectedResp: "unexpected server response",
	MonError:          "Error: ", MonQuitHint: "  q — quit", MonConnecting: "  Connecting…",
	LblCPU: "CPU", LblRAM: "RAM", LblNet: "Net",
	LiveSuffix: " · live · q — quit", MonitorStopped: "Monitor stopped: ",

	AskCommand: "Command:", RunningOn: "Running on %s…",
	RunningOnMany: "Running on %d servers…", NoServersYet: "No servers.",
	StatusOK: "OK", StatusCode: "code %d", NoOutput: "(no output)",
	AskPickServers: "Run the command on which servers?",

	SetupTitle:      "⚙ Setting up %s",
	StepConnectRoot: "Connecting via SSH (root)",
	StepDetect:      "Detecting distribution",
	StepUpdate:      "Updating system (may take a few minutes)",
	StepPackages:    "Installing base packages",
	StepCreateUser:  "Creating user %s (sudo + password)",
	StepInstallKey:  "Installing SSH key",
	StepVerifyKey:   "Verifying key login (%s)",
	StepFirewall:    "Configuring firewall",
	StepSSHD:        "Configuring sshd (port/hardening) with access check",
	StepFail2ban:    "Configuring fail2ban",
	StepAutoUpd:     "Security auto-updates",
	StepSwap:        "Creating 2 GB swap file",
	StepTimezone:    "Timezone: ",
	StepHostname:    "Hostname: ",

	WarnUnknownDistro:    "Unknown distribution — packages, firewall and auto-updates skipped",
	ErrCritical:          "critical step \"%s\" failed",
	ErrNoPkgMgr:          "no supported package manager",
	ErrNotInGroup:        "user %s is not in group %s",
	ErrKeyLogin:          "key login as %s does not work: %v",
	ErrFirewallUnknown:   "firewall skipped: unknown distribution",
	ErrSshdTest:          "sshd -t: invalid configuration, changes reverted: %s",
	ErrSshdRestartRevert: "failed to restart sshd, changes reverted: %v",
	ErrSshdVerifyRevert:  "login does not work after sshd changes — reverted (%v)",
	CheckingAfterSshd:    "verifying key login after changes…",
	ErrAutoUpdUnknown:    "auto-updates: unknown distribution",
	ErrBadTz:             "suspicious timezone name: %s",
	ErrBadHostname:       "invalid hostname: %s",

	ErrNameEmpty:   "name cannot be empty",
	ErrNameChars:   "only letters, digits, dots, dashes and underscores",
	ErrNameTaken:   "name already taken",
	ErrHostFmt:     "enter an IP address or domain without spaces",
	ErrPortRange:   "port must be a number from 1 to 65535",
	ErrUserFmt:     "lowercase latin letters, digits, - and _; starts with a letter",
	ErrTzFmt:       "format: Region/City",
	ErrHostnameFmt: "only letters, digits, dots and dashes",
	ErrNumber:      "enter a number",

	WizardTitle:    "What to set up on the server? (space — select, Enter — confirm)",
	OptUpdate:      "Update system (apt/dnf upgrade)",
	OptPackages:    "Base packages (git, htop, tmux, ufw, fail2ban…)",
	OptFirewall:    "Firewall (ufw/firewalld: allow SSH, enable)",
	OptFail2ban:    "fail2ban (brute-force protection)",
	OptHarden:      "SSH hardening (key-only login)",
	OptRootKey:     "Also add the key for root (backup access)",
	OptAutoUpd:     "Security auto-updates",
	OptPort:        "Change SSH port",
	OptSwap:        "Create 2 GB swap",
	OptTz:          "Set timezone",
	OptHostname:    "Set hostname",
	AskNewUsername: "New user name:",
	AskNewPort:     "New SSH port:",
	AskTimezone:    "Timezone:",
	ManualEntry:    "enter manually…",
	AskTzManual:    "Timezone (e.g. Europe/Moscow):",
	AskHostname:    "Hostname:",

	SetupAborted:   "Setup aborted: ",
	ServerNotSaved: "Server not saved. Check access and try again.",
	SumServer:      "Server", SumLogin: "Login", SumPassword: "Password",
	SumKey: "Key", SumKeychain: "Keychain", SumKeychainVal: "passwords saved (unissh)",
	ConfiguredOK:            "✓ %s is set up",
	WarnNoPwLogin:           "⚠ Password login disabled — don't lose the key %s",
	UserPwCopied:            "User password copied to clipboard",
	AskConnectNow:           "Connect now?",
	TitleNewServer:          "➕ New server",
	AskName:                 "Name (short alias):",
	AskHost:                 "Address (IP or domain):",
	AskPort:                 "SSH port:",
	AskTags:                 "Tags, space-separated (optional):",
	AskHowAdd:               "How to add the server?",
	OptWithBootstrap:        "🚀 With initial setup (root → sudo user, key, firewall…)",
	OptPlainAdd:             "📎 Just add (server already configured)",
	AskRootAuth:             "How to connect as root?",
	OptRootPassword:         "Root password",
	OptRootPasswordKeychain: "Root password (from Keychain if saved)",
	OptRootKeyFile:          "SSH key (file)",
	AskRootPassword:         "Root password:",
	AskRootKeyPath:          "Path to root's private key:",
	ErrNoKeySetup:           "Setup is impossible without an SSH key: ",
	CheckingRootAccess:      "Checking access root@%s:%d…",
	RootConfirmed:           "Root access confirmed",
	AskUser:                 "User:",
	AskAuth:                 "Authentication:",
	OptAuthKey:              "SSH key",
	OptAuthPassword:         "Password (save to Keychain)",
	AskKeyPath:              "Path to private key:",
	WarnKeychainFail:        "⚠ Failed to save password to Keychain: ",
	ServerAdded:             "✓ Server %s added.",
	NeedRootAccess:          "Setup requires root access to %s.",
	RootPwFromKeychain:      "Root password found in Keychain",

	NoPasswordsStored: "No passwords in Keychain for this server.",
	TitlePasswords:    "Passwords (Keychain)",
	PwUserCopied:      "Password for %s copied to clipboard",
	InstallingKey:     "Installing key on %s…",
	KeyInstalled:      "✓ Key installed — passwordless login now.",
	AskReboot:         "Reboot %s (%s)?",
	AskSudoPassword:   "Password for %s (sudo):",
	RebootSent:        "✓ Reboot command sent.",
	FingerprintReset:  "✓ Fingerprint for %s reset.",
	TitleEdit:         "Editing %s:",
	FldName:           "Name: ", FldHost: "Address: ", FldPort: "Port: %d", FldUser: "User: ",
	FldKey: "Key: ", FldTags: "Tags: ", FldNotes: "Notes: ", KeyNoneLabel: "— (password)",
	FldProxy: "Bastion (ProxyJump): ", AskProxy: "ProxyJump (user@host[:port], empty — no bastion):",
	AskNewName: "New name:", AskNewHost: "New address:", AskNewPortEdit: "New port:",
	AskNewUser: "New user:", AskKeyOrEmpty: "Key path (empty — password login):",
	AskTagsEdit: "Tags, space-separated:", AskNotes: "Notes:",
	AskDelete:   "Delete %s (%s)?",
	DeletedMsg:  "✓ Server %s deleted (Keychain passwords too).",
	MenuConnect: "🔌 Connect", MenuInfo: "📊 Server info",
	MenuMonitor: "📈 Live monitor", MenuRun: "▶️  Run command",
	MenuPasswords:  "🔑 Show passwords (Keychain)",
	MenuInstallKey: "🧩 Install SSH key (passwordless login)",
	MenuBootstrap:  "🛠  Initial setup (bootstrap)",
	MenuCopyCmd:    "📋 Copy ssh command",
	MenuForget:     "♻️  Reset host fingerprint",
	MenuReboot:     "🔁 Reboot server",
	MenuEdit:       "✏️  Edit", MenuDelete: "🗑  Delete",
	AskActions: "%s — actions:",

	ExportHeader: "# Generated by unissh — do not edit manually",
	ExportedTo:   "✓ Exported to %s",
	ExportHint:   "Now this works from any terminal: ssh <server-name>",
	TitlePubkey:  "Public key", PubkeyCopied: "Copied to clipboard",
	SetDefaultUser:    "Default user: ",
	SetSSHKey:         "SSH key: ",
	SetTimeout:        "Status check timeout: %.1f s",
	SetLanguage:       "Language: ",
	SetBootstrapSteps: "Default setup steps for new servers",
	TitleSettings:     "Settings:",
	AskDefaultUser:    "Default user:",
	AskSSHKeyPath:     "SSH key path:",
	AskTimeout:        "Timeout (sec):",
	TitleTools:        "Tools:",
	ToolMulti:         "▶️  Run command on multiple servers",
	ToolExport:        "📤 Export to ~/.ssh/config (ssh <name>)",
	ToolPubkey:        "🔑 Show/copy public key",
	ToolSettings:      "⚙️  Settings",
	AskLanguage:       "Interface language:",
	StepsSaved:        "✓ Default steps saved.",

	EmptyTitle: "No servers yet.",
	EmptyBody: "Add your first one — unissh will update it, create a sudo user\n" +
		"with a generated password, install a key and enable protection.",
	MainPrompt:     "Pick a server (Enter — connect) or an action:",
	ItemAdd:        "➕ Add server",
	ItemManage:     "🛠  Manage servers",
	ItemTools:      "🧰 Tools",
	ItemExit:       "⏻  Exit",
	AskWhichServer: "Which server?",
	Goodbye:        "See you! 👋",
	ProbeSpinner:   "Checking server availability…",

	SectionServers:   "SERVERS",
	SectionActions:   "ACTIONS",
	TuiHint:          "↑↓/click — select · Enter — connect · m — manage · q — quit",
	TuiHintEmpty:     "↑↓/click — select · Enter — choose · q — quit",
	TuiHintSelect:    "↑↓/click — select · Enter — choose · Esc — back",
	TuiHintInput:     "Enter — confirm · Esc — cancel",
	TuiHintMulti:     "↑↓ — move · space/click — toggle · Enter — done · Esc — cancel",
	MoreAbove:        "  ▲ %d more",
	MoreBelow:        "  ▼ %d more",
	BackupServersFmt: "%s  · servers: %d",

	ImportReadErr:    "Failed to read ~/.ssh/config: %v",
	ImportNothing:    "No new hosts to import from ~/.ssh/config.",
	ImportTitle:      "Import from ~/.ssh/config (found %d):",
	ImportAdded:      "✓ Servers imported: %d",
	ToolImport:       "📥 Import from ~/.ssh/config",
	ToolBackups:      "🗂  Restore config from backup",
	BackupTitle:      "Choose a backup to restore:",
	BackupRestored:   "✓ Config restored from backup.",
	BackupNothing:    "No backups yet.",
	BackupConfirm:    "Restore config from %s? Current one will be overwritten.",
	SetTheme:         "Theme: ",
	AskTheme:         "Color theme:",
	SnippetManual:    "✎ enter command manually…",
	SnippetSaveQ:     "Save as a snippet for reuse?",
	AskSnippetName:   "Snippet name:",
	ToolSnippets:     "✎ Command snippets",
	SnippetsTitle:    "Snippets (select to delete):",
	NoSnippetsYet:    "No snippets yet — save them when running a command.",
	SnippetDeleted:   "✓ Snippet deleted.",
	AskDeleteSnippet: "Delete snippet \"%s\"?",
	NoLastConnected:  "never connected",
	LastConnFmt:      "connected %s ago",
	RelNow:           "now",
	RelMin:           "%dm",
	RelHour:          "%dh",
	RelDay:           "%dd",

	FooterServer:        "enter/c connect · i info · m menu · e edit · d delete · / search · ? all keys",
	FooterAction:        "enter select · s sort · / search · ? all keys · q quit",
	FilterPrompt:        "🔍 ",
	FilterNoMatch:       "no servers match filter",
	SortByName:          "sort: by name",
	SortByPing:          "sort: by ping",
	SortByOnline:        "sort: by availability",
	ConfirmConnectTitle: "Connect to server %s?",
	ConnectBtn:          "❯ Connect",
	CancelBtn:           "✗ Cancel",
	SetConfirmConnect:   "Confirm before connecting: ",
	AskConfirmConnect:   "Ask for confirmation before connecting?",
	HelpTitle:           "Keybindings",
	HelpClose:           "any key — close",
	HelpNavigate:        "navigate",
	HelpConnect:         "connect",
	HelpInfo:            "server info",
	HelpBootstrap:       "initial setup",
	HelpManage:          "server management menu",
	HelpEdit:            "edit",
	HelpReboot:          "reboot",
	HelpDelete:          "delete",
	HelpFavorite:        "favorite (★)",
	HelpSort:            "change sort",
	HelpFilter:          "search/filter",
	HelpHelp:            "this help",
	HelpQuit:            "quit",

	UsageTagline: "Terminal VPS manager: pick a server and you're in.",
	UsageTitle:   "Usage:",
	UsageMenu:    "interactive menu",
	UsageConnect: "quick connect to a server",
	UsageList:    "server list with statuses",
	UsageAdd:     "add a server",
	UsageVersion: "show version",
	NoServersCLI: "No servers. Run unissh and add your first one.",
	ErrNotFound:  "Server \"%s\" not found.",
	HaveList:     "Available: %s",
	ErrAddTTY:    "unissh add only works in an interactive terminal",

	BytesUnits:      []string{"B", "KB", "MB", "GB", "TB", "PB"},
	AbortedPassword: "⚠ Password for user %s is already set on the server: %s (copied to clipboard)",
	ErrTimeoutRange: "enter a number greater than 0",
	RootAuthHint:    "After hardening, root password login is disabled — use the SSH key.",

	LblPublicIP:    "Public IP (external APIs)",
	DetectingPubIP: "Detecting public addresses…",
	PubIPUnique:    "Unique: %s",
	PubIPNone:      "could not detect public addresses",
	SetShowPubIP:   "Public IP on connect: ",
	AskShowPubIP:   "Show public IP on connect?",

	LblIPs:            "IP addresses",
	GatheringOnAdd:    "Gathering server information…",
	SetConnectMode:    "Open SSH: ",
	AskConnectMode:    "How to open the SSH connection?",
	ConnectModeWindow: "in a new terminal window",
	ConnectModeInline: "in the same window",
	OpenedNewWindow:   "↗ Connection opened in a new terminal window",
	ErrOpenWindow:     "could not open a new window (%v) — connecting here",
}

// T — активный язык. По умолчанию русский; переключается из настроек.
var T = &trRU

func setLang(code string) {
	if code == "en" {
		T = &trEN
	} else {
		T = &trRU
	}
}

func langName(code string) string {
	if code == "en" {
		return "English"
	}
	return "Русский"
}
