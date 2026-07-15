package app

// Локализация: все строки интерфейса. Язык хранится в настройках
// (settings.language: "ru" | "en") и переключается на лету.

type L struct {
	// общее
	Yes, No, Back, Done, PressEnter, Canceled string

	// баннер

	// config
	ErrReadConfig, BackupSaved, ErrSaveConfig string
	ErrConfigReadOnly, ConfigBrokenHint       string

	// sshkey
	AskGenKey, ErrKeyMissing, KeyCreated string

	// remote
	ErrHostKeyChanged, ErrReadKey, ErrParseKey, ErrNoAuthMethod string
	ErrAuthFailed, ErrConnectFailed, ErrConnLost, ErrCmdTimeout string
	ErrCmdExitCode, AskPasswordFor, ErrNoPwInKeychain           string

	// таблица
	ColName, ColUser, ColAddr, ColPort, ColPing, ColNotes, MsFmt string

	// подключение
	PwCopiedClipboard, Connecting, SessionEnded, CopiedCmd string

	// инфо
	Gathering, LblOS, LblHost, LblKernel, LblUptime   string
	LblLoad, LblMemory, LblDiskRoot, LblDisk, CPUsFmt string

	// монитор
	ErrUnexpectedResp      string
	LblCPU, LblRAM, LblNet string

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
	AskPort, AskHowAdd, OptWithBootstrap, OptPlainAdd                string
	AskRootAuth, OptRootPassword, OptRootPasswordKeychain            string
	OptRootKeyFile, AskRootPassword, AskRootKeyPath, ErrNoKeySetup   string
	CheckingRootAccess, RootConfirmed, AskUser, AskAuth, OptAuthKey  string
	OptAuthPassword, AskKeyPath, WarnKeychainFail, ServerAdded       string
	NeedRootAccess, RootPwFromKeychain                               string

	// действия с сервером
	NoPasswordsStored, TitlePasswords, PwUserCopied, InstallingKey string
	KeyInstalled, AskReboot, AskSudoPassword, RebootSent           string
	FingerprintReset, TitleEdit, FldName, FldHost, FldPort         string
	FldUser, FldKey, FldNotes, KeyNoneLabel, AskNewName            string
	FldProxy, AskProxy                                             string
	AskNewHost, AskNewPortEdit, AskNewUser, AskKeyOrEmpty          string
	AskNotes, AskDelete, DeletedMsg                                string
	MenuConnect, MenuInfo, MenuRun, MenuPasswords                  string
	MenuInstallKey, MenuBootstrap, MenuCopyCmd, MenuForget         string
	MenuReboot, MenuEdit, MenuDelete, MenuTunnels, MenuTransfer    string

	// подключение в новом окне
	ErrNoTerminal string

	// полноэкранный просмотрщик
	PageScroll, PageBack string

	// журнал
	ToolJournal, MenuJournalDesc, JournalTitle, JournalEmpty string
	JournalColWhen, JournalColWhat, JournalColDetails        string
	JournalKindConnect, JournalKindRun, JournalKindTransfer  string
	JournalClearKey, JournalClearAsk, JournalCleared         string
	JournalSecretsHint                                       string

	// передача файлов (scp)
	TransferTitle, TransferUpload, TransferDownload             string
	TransferUploadDesc, TransferDownloadDesc                    string
	AskLocalPath, AskRemoteDest, AskRemotePath, AskLocalDest    string
	TransferUpFmt, TransferDownFmt, TransferDone                string
	TransferFailedFmt                                           string
	ErrPathEmpty, ErrPathMissing, ErrDirMissing, ErrPathNewline string

	// мультивыбор и групповые действия
	BatchRun, BatchMove, BatchDelete, BatchClear, BatchHint string
	PanelBatchFmt, PanelSelected, BatchSelectedFmt          string
	BatchFooterFmt, BatchMoveTitleFmt                       string
	BatchMovedFmt, BatchDeleteAskFmt, BatchDeletedFmt       string
	HelpSelect                                              string

	// дашборд здоровья парка
	ToolHealth, MenuHealthDesc, HealthTitle, HealthColStatus string
	HealthCollectingFmt, HealthOnlineFmt, HealthNoDataFmt    string

	// проброс портов
	TunnelsTitle, TunnelsHeader, LblTunnels, NoTunnelsYet       string
	TunnelAdd, TunnelAddDesc, TunnelDelete, TunnelDeleteDesc    string
	TunnelStopAll, TunnelStopAllDesc, TunnelUp, TunnelDown      string
	TunnelKindLocal, TunnelKindRemote, TunnelKindDynamic        string
	TunnelDescLocal, TunnelDescRemote, TunnelDescDynamic        string
	AskTunnelKind, AskTunnelLocalPort, AskTunnelRemoteHost      string
	AskTunnelRemotePort, AskTunnelSocksPort, AskTunnelStartNow  string
	AskDeleteTunnel, TunnelStarted, TunnelStopped, TunnelExists string
	TunnelsStoppedFmt, ErrTunnelPortBusy, ErrTunnelHost         string
	ErrTunnelKind, ErrTunnelStart                               string

	// инструменты и настройки
	ExportHeader, ExportedTo, ExportHint, TitlePubkey, PubkeyCopied string
	SetDefaultUser, SetSSHKey, SetTimeout, SetLanguage              string
	SetBootstrapSteps, TitleSettings, AskDefaultUser, AskSSHKeyPath string
	AskTimeout, ToolMulti, ToolExport, ToolPubkey                   string
	ToolSettings, AskLanguage                                       string

	// главное меню
	ItemAdd           string
	ItemExit, Goodbye string

	// TUI (главный экран)
	TuiHintSelect                                                      string
	TuiHintInput, TuiHintMulti, MoreAbove, MoreBelow, BackupServersFmt string

	// импорт / бэкапы / сниппеты / темы
	ImportReadErr, ImportNothing, ImportTitle, ImportAdded                       string
	ToolImport, ToolBackups, BackupTitle, BackupRestored                         string
	BackupNothing, BackupConfirm, SetTheme, AskTheme                             string
	SnippetManual, SnippetSaveQ, AskSnippetName                                  string
	ToolSnippets, SnippetsTitle, NoSnippetsYet, SnippetDeleted, AskDeleteSnippet string
	RelNow, RelMin, RelHour, RelDay                                              string

	// панельный лейаут (v2)
	PanelServers, PanelMenu, PanelActions, PanelStats, PanelDesc     string
	PanelHint, StatsLoading, StatsError, NoServersHint               string
	StatsOffline, TermTooSmall                                       string
	MenuAddDesc, MenuMultiDesc, MenuImportDesc, MenuExportDesc       string
	MenuBackupsDesc                                                  string
	MenuSnippetsDesc, MenuPubkeyDesc, MenuSettingsDesc, MenuExitDesc string

	// футер, фильтр, сортировка, помощь
	FilterPrompt                                    string
	SortByName, SortByPing, SortByOnline            string
	HelpTitle, HelpClose, HelpNavigate, HelpConnect string
	HelpTunnels                                     string
	HelpFavorite, HelpSort                          string
	HelpFilter, HelpHelp, HelpQuit                  string

	// папки-группы в списке серверов
	FolderRoot, PanelFolder, FolderServersFmt, FolderHint string
	FilterHint, MoveHint                                  string
	HelpPanels, HelpJump, HelpOpenActions, HelpBack       string
	HelpFolderNew, HelpFolderMove, HelpFolderDel          string
	HelpFolderRename, FolderRenameTitle, FolderNewTitle   string
	HelpFolderOrder                                       string
	FolderDeleteBody, PromptHint, ConfirmHint             string
	AskFolderName, ErrFolderExists                        string
	FolderNewOption, AskDeleteFolder                      string
	FldFolder, AskFolderChoose                            string

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

	// все IP-адреса сервера / режим открытия SSH
	LblIPs         string // метка блока адресов в инфо-панели
	SetConnectMode string // пункт настроек
	// описания пунктов настроек (правая панель панельного меню)
	DescLanguage, DescTheme, DescConnectMode string
	DescDefaultUser, DescSSHKey, DescTimeout string
	DescBootstrapSteps                       string
	AskConnectMode                           string // вопрос
	ConnectModeWindow                        string // «в новом окне»
	ConnectModeInline                        string // «в этом же окне»
	OpenedNewWindow                          string // сообщение после запуска нового окна
	ErrOpenWindow                            string // ошибка открытия окна
}

var trRU = L{
	Yes: "Да", No: "Нет", Back: "← Назад", Done: "← Готово",
	PressEnter: "Enter — продолжить ", Canceled: "отменено",

	ErrReadConfig:     "Не удалось прочитать конфиг: %s",
	BackupSaved:       "Копия сохранена в %s",
	ErrSaveConfig:     "Не удалось сохранить конфиг: %s",
	ErrConfigReadOnly: "Конфиг не прочитан — запись запрещена, чтобы не потерять серверы.",
	ConfigBrokenHint:  "Восстановите его: Меню → «Восстановить конфиг из бэкапа» (или почините servers.json вручную).",

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
	ColPing: "Пинг", ColNotes: "Заметки", MsFmt: "%s мс",

	PwCopiedClipboard: "🔑 Пароль скопирован в буфер обмена — вставьте по запросу",
	Connecting:        "⇄ Подключаюсь: ",
	SessionEnded:      "\nСессия завершена (код %d).",
	CopiedCmd:         "✓ Скопировано: ",

	Gathering: "Собираю данные с %s…",
	LblOS:     "ОС", LblHost: "Хост", LblKernel: "Ядро", LblUptime: "Аптайм",
	LblLoad: "Load", LblMemory: "Память", LblDiskRoot: "Диск /", LblDisk: "Диск",
	CPUsFmt: "  (%d CPU)",

	ErrUnexpectedResp: "неожиданный ответ сервера",
	LblCPU:            "CPU", LblRAM: "RAM", LblNet: "Сеть",

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
	FldKey: "Ключ: ", FldNotes: "Заметки: ", KeyNoneLabel: "— (пароль)",
	FldProxy: "Бастион (ProxyJump): ", AskProxy: "ProxyJump (user@host[:port], пусто — без бастиона):",
	AskNewName: "Новое имя:", AskNewHost: "Новый адрес:", AskNewPortEdit: "Новый порт:",
	AskNewUser: "Новый пользователь:", AskKeyOrEmpty: "Путь к ключу (пусто — вход по паролю):",
	AskNotes:    "Заметки:",
	AskDelete:   "Удалить %s (%s)?",
	DeletedMsg:  "✓ Сервер %s удалён (пароли из Keychain тоже).",
	MenuConnect: "🔌 Подключиться", MenuInfo: "📊 Информация о сервере",
	MenuRun:        "▶️ Выполнить команду",
	MenuPasswords:  "🔑 Показать пароли (Keychain)",
	MenuInstallKey: "🧩 Установить SSH-ключ (вход без пароля)",
	MenuBootstrap:  "🛠 Первоначальная настройка (bootstrap)",
	MenuCopyCmd:    "📋 Скопировать ssh-команду",
	MenuForget:     "♻️ Сбросить отпечаток хоста",
	MenuReboot:     "🔁 Перезагрузить сервер",
	MenuEdit:       "✏️ Редактировать", MenuDelete: "🗑  Удалить",
	MenuTunnels: "🔀 Проброс портов", MenuTransfer: "📦 Передача файлов",

	ErrNoTerminal: "Не найден эмулятор терминала. Задайте $TERMINAL или переключитесь на «в этом же окне» в Настройках.",

	PageScroll: "↑↓ прокрутка",
	PageBack:   "esc — назад",

	ToolJournal:     "📜 Журнал",
	MenuJournalDesc: "Что unissh делал на серверах: подключения, команды и их коды выхода, передачи файлов. Лежит в ~/.config/unissh/history.jsonl (0600).",
	JournalTitle:    "Журнал",
	JournalEmpty:    "Журнал пуст — здесь появятся подключения и команды.",
	JournalColWhen:  "Когда", JournalColWhat: "Что", JournalColDetails: "Подробности",
	JournalKindConnect: "вход", JournalKindRun: "команда", JournalKindTransfer: "файлы",
	JournalClearKey:    "очистить",
	JournalClearAsk:    "Удалить журнал целиком?",
	JournalCleared:     "✓ Журнал очищен",
	JournalSecretsHint: "В командах могли остаться секреты — файл доступен только вам (0600).",

	TransferTitle:        "Передача файлов",
	TransferUpload:       "⬆️ Отправить на сервер",
	TransferDownload:     "⬇️ Скачать с сервера",
	TransferUploadDesc:   "Скопировать локальный файл или каталог на сервер (scp -r).",
	TransferDownloadDesc: "Забрать файл или каталог с сервера к себе (scp -r).",
	AskLocalPath:         "Что отправить (путь у вас): ",
	AskRemoteDest:        "Куда на сервере: ",
	AskRemotePath:        "Что скачать (путь на сервере): ",
	AskLocalDest:         "Куда сохранить (каталог у вас): ",
	TransferUpFmt:        "Отправляю %s → %s",
	TransferDownFmt:      "Скачиваю %s ← %s",
	TransferDone:         "✓ Готово",
	TransferFailedFmt:    "scp завершился с кодом %d",
	ErrPathEmpty:         "Путь не может быть пустым",
	ErrPathMissing:       "Такого файла или каталога нет",
	ErrDirMissing:        "Нужен существующий каталог",
	ErrPathNewline:       "Путь не должен содержать перевод строки",

	BatchRun:    "▶️ Выполнить команду на всех",
	BatchMove:   "📁 Перенести в папку",
	BatchDelete: "🗑  Удалить выделенные",
	BatchClear:  "✕  Снять выделение",
	BatchHint:   "space — отметить сервер (на папке — всю группу), esc — снять выделение.",

	PanelBatchFmt:     "ГРУППА (%d)",
	PanelSelected:     "ВЫДЕЛЕНО",
	BatchSelectedFmt:  "Выбрано серверов: %d",
	BatchFooterFmt:    "выделено: %d  ·  enter — групповые действия  ·  esc — снять",
	BatchMoveTitleFmt: "Куда перенести (%d шт.)",
	BatchMovedFmt:     "✓ Перенесено серверов: %d → %s",
	BatchDeleteAskFmt: "Удалить %d сервер(ов): %s?",
	BatchDeletedFmt:   "✓ Удалено серверов: %d (пароли из Keychain тоже).",

	ToolHealth:          "❤️ Здоровье парка",
	MenuHealthDesc:      "Собрать CPU, память, диск и load со всех серверов сразу и показать одной таблицей. Самые нагруженные — наверху.",
	HealthTitle:         "Здоровье парка",
	HealthColStatus:     "Состояние",
	HealthCollectingFmt: "Опрашиваю серверы (%d)…",
	HealthOnlineFmt:     "Онлайн: %d из %d",
	HealthNoDataFmt:     "без метрик: %d",

	TunnelsTitle:  "Проброс портов",
	TunnelsHeader: "Туннели живут, пока открыт unissh. Enter — включить/выключить.",
	LblTunnels:    "Туннели",
	NoTunnelsYet:  "Правил пока нет — добавьте первое.",
	TunnelAdd:     "➕ Добавить правило", TunnelAddDesc: "Новый -L, -R или -D",
	TunnelDelete: "🗑  Удалить правило", TunnelDeleteDesc: "Убрать из конфига",
	TunnelStopAll: "⏹  Остановить все", TunnelStopAllDesc: "Погасить туннели этого сервера",
	TunnelUp: "работает", TunnelDown: "остановлен",
	TunnelKindLocal:     "L — локальный порт (-L)",
	TunnelKindRemote:    "R — обратный проброс (-R)",
	TunnelKindDynamic:   "D — SOCKS5-прокси (-D)",
	TunnelDescLocal:     "Порт у вас → хост:порт со стороны сервера. Классика: БД на 5432.",
	TunnelDescRemote:    "Порт на сервере → хост:порт с вашей машины. Показать локальный сайт.",
	TunnelDescDynamic:   "SOCKS5-прокси на вашем порту: весь трафик идёт через сервер.",
	AskTunnelKind:       "Вид проброса",
	AskTunnelLocalPort:  "Локальный порт: ",
	AskTunnelRemoteHost: "Хост назначения (со стороны сервера): ",
	AskTunnelRemotePort: "Порт назначения: ",
	AskTunnelSocksPort:  "Порт SOCKS5-прокси: ",
	AskTunnelStartNow:   "Запустить прямо сейчас?",
	AskDeleteTunnel:     "Удалить правило %s?",
	TunnelStarted:       "✓ Туннель поднят",
	TunnelStopped:       "✓ Туннель остановлен",
	TunnelExists:        "Такое правило уже есть",
	TunnelsStoppedFmt:   "Остановлено туннелей: %d",
	ErrTunnelPortBusy:   "Локальный порт %d уже занят другим туннелем",
	ErrTunnelHost:       "Нужен хост назначения без пробелов и двоеточий",
	ErrTunnelKind:       "Неизвестный вид проброса",
	ErrTunnelStart:      "Туннель не поднялся: %s",

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
	ToolMulti:         "▶️ Выполнить команду на нескольких серверах",
	ToolExport:        "📤 Экспорт в ~/.ssh/config (ssh <имя>)",
	ToolPubkey:        "🔑 Показать/скопировать публичный ключ",
	ToolSettings:      "⚙️ Настройки",
	AskLanguage:       "Язык интерфейса:",

	ItemAdd:  "➕ Добавить сервер",
	ItemExit: "⏻  Выход",
	Goodbye:  "До встречи!",

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
	RelNow:           "только что",
	RelMin:           "%dм",
	RelHour:          "%dч",
	RelDay:           "%dд",

	PanelServers:     "СЕРВЕРЫ",
	PanelMenu:        "МЕНЮ",
	PanelActions:     "ДЕЙСТВИЯ",
	PanelStats:       "СТАТИСТИКА",
	PanelDesc:        "ОПИСАНИЕ",
	PanelHint:        "Tab — панель · Enter/→ — действия · c — подключиться · / — поиск · n/m/r/d — папки · ? — справка · q — выход",
	StatsLoading:     "загружаю…",
	StatsError:       "нет данных",
	StatsOffline:     "сервер недоступен",
	TermTooSmall:     "Окно терминала слишком маленькое. Нужно минимум %d×%d, сейчас %d×%d — увеличьте окно.",
	NoServersHint:    "Серверов нет — добавьте первый в меню слева.",
	MenuAddDesc:      "Добавить новый VPS: вручную или с первоначальной настройкой (bootstrap) — обновление системы, sudo-пользователь, ключ, firewall, fail2ban.",
	MenuMultiDesc:    "Выполнить одну команду сразу на нескольких серверах (параллельно).",
	MenuImportDesc:   "Импортировать серверы из вашего ~/.ssh/config (Host-блоки) — быстро перенести весь парк.",
	MenuExportDesc:   "Выгрузить серверы в ~/.ssh/config, чтобы работал `ssh <имя>` из любого терминала.",
	MenuBackupsDesc:  "Восстановить конфиг из автоматического снапшота (создаётся перед каждым изменением).",
	MenuSnippetsDesc: "Сохранённые команды для быстрого выполнения на серверах; здесь можно удалить лишние.",
	MenuPubkeyDesc:   "Показать и скопировать публичный SSH-ключ unissh для добавления на серверы.",
	MenuSettingsDesc: "Язык, тема, режим подключения, подтверждение, шаги bootstrap по умолчанию и др.",
	MenuExitDesc:     "Выйти из unissh.",

	FilterPrompt: "🔍 ",
	SortByName:   "сортировка: по имени",
	SortByPing:   "сортировка: по пингу",
	SortByOnline: "сортировка: по доступности",
	HelpTitle:    "Горячие клавиши",
	HelpClose:    "любая клавиша — закрыть",
	HelpNavigate: "навигация",
	HelpConnect:  "подключиться",
	HelpTunnels:  "проброс портов",
	HelpSelect:   "отметить сервер (группу)",
	HelpFavorite: "избранное (★)",
	HelpSort:     "сменить сортировку",
	HelpFilter:   "поиск/фильтр",
	HelpHelp:     "эта справка",
	HelpQuit:     "выход",

	FolderRoot:        "Без папки",
	PanelFolder:       "ПАПКА",
	FolderServersFmt:  "Серверов:",
	FolderHint:        "enter/←→ — свернуть/развернуть · n — новая · r — переименовать · d — удалить · shift+↑↓ — порядок",
	FilterHint:        "печатайте для поиска · Enter — применить · Esc — сбросить",
	MoveHint:          "перемещение: ↑↓ — выбрать папку · Enter — сохранить · Esc — отмена",
	HelpPanels:        "переключить панель (серверы/меню)",
	HelpJump:          "в начало / в конец списка",
	HelpOpenActions:   "открыть действия сервера / развернуть папку",
	HelpBack:          "назад / свернуть папку",
	HelpFolderNew:     "создать папку",
	HelpFolderMove:    "взять сервер и вести по папкам (Enter — сохранить)",
	HelpFolderDel:     "удалить папку (серверы — в корень)",
	AskFolderName:     "Имя папки:",
	FolderNewTitle:    "Новая папка",
	FolderRenameTitle: "Переименовать папку",
	HelpFolderRename:  "переименовать папку",
	HelpFolderOrder:   "порядок папок",
	FolderDeleteBody:  "Серверы вернутся в «Без папки».",
	PromptHint:        "Enter — подтвердить · Esc — отмена",
	ConfirmHint:       "Enter/y — удалить · Esc/n — отмена",
	ErrFolderExists:   "папка с таким именем уже есть",
	FolderNewOption:   "＋ Новая папка…",
	AskDeleteFolder:   "Удалить папку «%s»?",
	FldFolder:         "Папка: ",
	AskFolderChoose:   "Папка сервера:",

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

	LblIPs:             "IP-адреса",
	SetConnectMode:     "Открывать SSH: ",
	DescLanguage:       "Язык интерфейса программы: русский или английский.",
	DescTheme:          "Цветовая тема оформления. Применяется сразу ко всем экранам.",
	DescConnectMode:    "Открывать SSH в новом окне терминала или прямо в текущем окне (inline).",
	DescDefaultUser:    "Имя пользователя, подставляемое по умолчанию при добавлении нового сервера.",
	DescSSHKey:         "Путь к приватному SSH-ключу, который используется для входа и установки на серверы.",
	DescTimeout:        "Сколько секунд ждать ответа при проверке статуса сервера (пинг). Больше — надёжнее на медленных каналах.",
	DescBootstrapSteps: "Какие шаги первоначальной настройки (bootstrap) отмечать по умолчанию для нового сервера.",
	AskConnectMode:     "Как открывать SSH-подключение?",
	ConnectModeWindow:  "в новом окне терминала",
	ConnectModeInline:  "в этом же окне",
	OpenedNewWindow:    "↗ Подключение открыто в новом окне терминала",
	ErrOpenWindow:      "не удалось открыть новое окно (%v) — подключаюсь здесь",
}

var trEN = L{
	Yes: "Yes", No: "No", Back: "← Back", Done: "← Done",
	PressEnter: "Enter — continue ", Canceled: "canceled",

	ErrReadConfig:     "Failed to read config: %s",
	BackupSaved:       "Backup saved to %s",
	ErrSaveConfig:     "Failed to save config: %s",
	ErrConfigReadOnly: "Config could not be parsed — writing is disabled so your servers are not lost.",
	ConfigBrokenHint:  "Restore it: Menu → \"Restore config from backup\" (or fix servers.json by hand).",

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
	ColPing: "Ping", ColNotes: "Notes", MsFmt: "%s ms",

	PwCopiedClipboard: "🔑 Password copied to clipboard — paste when prompted",
	Connecting:        "⇄ Connecting: ",
	SessionEnded:      "\nSession ended (code %d).",
	CopiedCmd:         "✓ Copied: ",

	Gathering: "Gathering data from %s…",
	LblOS:     "OS", LblHost: "Host", LblKernel: "Kernel", LblUptime: "Uptime",
	LblLoad: "Load", LblMemory: "Memory", LblDiskRoot: "Disk /", LblDisk: "Disk",
	CPUsFmt: "  (%d CPU)",

	ErrUnexpectedResp: "unexpected server response",
	LblCPU:            "CPU", LblRAM: "RAM", LblNet: "Net",

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
	FldKey: "Key: ", FldNotes: "Notes: ", KeyNoneLabel: "— (password)",
	FldProxy: "Bastion (ProxyJump): ", AskProxy: "ProxyJump (user@host[:port], empty — no bastion):",
	AskNewName: "New name:", AskNewHost: "New address:", AskNewPortEdit: "New port:",
	AskNewUser: "New user:", AskKeyOrEmpty: "Key path (empty — password login):",
	AskNotes:    "Notes:",
	AskDelete:   "Delete %s (%s)?",
	DeletedMsg:  "✓ Server %s deleted (Keychain passwords too).",
	MenuConnect: "🔌 Connect", MenuInfo: "📊 Server info",
	MenuRun:        "▶️  Run command",
	MenuPasswords:  "🔑 Show passwords (Keychain)",
	MenuInstallKey: "🧩 Install SSH key (passwordless login)",
	MenuBootstrap:  "🛠  Initial setup (bootstrap)",
	MenuCopyCmd:    "📋 Copy ssh command",
	MenuForget:     "♻️  Reset host fingerprint",
	MenuReboot:     "🔁 Reboot server",
	MenuEdit:       "✏️  Edit", MenuDelete: "🗑  Delete",
	MenuTunnels: "🔀 Port forwarding", MenuTransfer: "📦 File transfer",

	ErrNoTerminal: "No terminal emulator found. Set $TERMINAL or switch to “same window” in Settings.",

	PageScroll: "↑↓ scroll",
	PageBack:   "esc — back",

	ToolJournal:     "📜 Journal",
	MenuJournalDesc: "What unissh did on your servers: connections, commands and their exit codes, file transfers. Stored in ~/.config/unissh/history.jsonl (0600).",
	JournalTitle:    "Journal",
	JournalEmpty:    "The journal is empty — connections and commands will show up here.",
	JournalColWhen:  "When", JournalColWhat: "What", JournalColDetails: "Details",
	JournalKindConnect: "login", JournalKindRun: "command", JournalKindTransfer: "files",
	JournalClearKey:    "clear",
	JournalClearAsk:    "Delete the whole journal?",
	JournalCleared:     "✓ Journal cleared",
	JournalSecretsHint: "Commands may contain secrets — the file is readable only by you (0600).",

	TransferTitle:        "File transfer",
	TransferUpload:       "⬆️  Upload to the server",
	TransferDownload:     "⬇️  Download from the server",
	TransferUploadDesc:   "Copy a local file or directory to the server (scp -r).",
	TransferDownloadDesc: "Fetch a file or directory from the server (scp -r).",
	AskLocalPath:         "What to upload (local path): ",
	AskRemoteDest:        "Where on the server: ",
	AskRemotePath:        "What to download (remote path): ",
	AskLocalDest:         "Where to save (local directory): ",
	TransferUpFmt:        "Uploading %s → %s",
	TransferDownFmt:      "Downloading %s ← %s",
	TransferDone:         "✓ Done",
	TransferFailedFmt:    "scp exited with code %d",
	ErrPathEmpty:         "The path cannot be empty",
	ErrPathMissing:       "No such file or directory",
	ErrDirMissing:        "An existing directory is required",
	ErrPathNewline:       "The path must not contain a newline",

	BatchRun:    "▶️  Run a command on all",
	BatchMove:   "📁 Move to a folder",
	BatchDelete: "🗑  Delete selected",
	BatchClear:  "✕  Clear selection",
	BatchHint:   "space marks a server (on a folder — the whole group), esc clears the selection.",

	PanelBatchFmt:     "BATCH (%d)",
	PanelSelected:     "SELECTED",
	BatchSelectedFmt:  "Servers selected: %d",
	BatchFooterFmt:    "selected: %d  ·  enter — batch actions  ·  esc — clear",
	BatchMoveTitleFmt: "Move where (%d servers)",
	BatchMovedFmt:     "✓ Servers moved: %d → %s",
	BatchDeleteAskFmt: "Delete %d server(s): %s?",
	BatchDeletedFmt:   "✓ Servers deleted: %d (Keychain passwords too).",

	ToolHealth:          "❤️ Fleet health",
	MenuHealthDesc:      "Collect CPU, memory, disk and load from every server at once and show them in one table. The busiest ones come first.",
	HealthTitle:         "Fleet health",
	HealthColStatus:     "Status",
	HealthCollectingFmt: "Polling servers (%d)…",
	HealthOnlineFmt:     "Online: %d of %d",
	HealthNoDataFmt:     "no metrics: %d",

	TunnelsTitle:  "Port forwarding",
	TunnelsHeader: "Tunnels live while unissh is open. Enter toggles one on/off.",
	LblTunnels:    "Tunnels",
	NoTunnelsYet:  "No rules yet — add the first one.",
	TunnelAdd:     "➕ Add rule", TunnelAddDesc: "A new -L, -R or -D",
	TunnelDelete: "🗑  Delete rule", TunnelDeleteDesc: "Remove it from the config",
	TunnelStopAll: "⏹  Stop all", TunnelStopAllDesc: "Bring down this server's tunnels",
	TunnelUp: "up", TunnelDown: "stopped",
	TunnelKindLocal:     "L — local port (-L)",
	TunnelKindRemote:    "R — reverse forward (-R)",
	TunnelKindDynamic:   "D — SOCKS5 proxy (-D)",
	TunnelDescLocal:     "A port here → host:port as seen from the server. Classic: a DB on 5432.",
	TunnelDescRemote:    "A port on the server → host:port on your machine. Expose a local site.",
	TunnelDescDynamic:   "SOCKS5 proxy on your port: all traffic goes through the server.",
	AskTunnelKind:       "Forwarding type",
	AskTunnelLocalPort:  "Local port: ",
	AskTunnelRemoteHost: "Destination host (as seen from the server): ",
	AskTunnelRemotePort: "Destination port: ",
	AskTunnelSocksPort:  "SOCKS5 proxy port: ",
	AskTunnelStartNow:   "Start it right away?",
	AskDeleteTunnel:     "Delete rule %s?",
	TunnelStarted:       "✓ Tunnel is up",
	TunnelStopped:       "✓ Tunnel stopped",
	TunnelExists:        "That rule already exists",
	TunnelsStoppedFmt:   "Tunnels stopped: %d",
	ErrTunnelPortBusy:   "Local port %d is already taken by another tunnel",
	ErrTunnelHost:       "Need a destination host without spaces or colons",
	ErrTunnelKind:       "Unknown forwarding type",
	ErrTunnelStart:      "Tunnel failed to start: %s",

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
	ToolMulti:         "▶️ Run command on multiple servers",
	ToolExport:        "📤 Export to ~/.ssh/config (ssh <name>)",
	ToolPubkey:        "🔑 Show/copy public key",
	ToolSettings:      "⚙️ Settings",
	AskLanguage:       "Interface language:",

	ItemAdd:  "➕ Add server",
	ItemExit: "⏻  Exit",
	Goodbye:  "See you! 👋",

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
	RelNow:           "now",
	RelMin:           "%dm",
	RelHour:          "%dh",
	RelDay:           "%dd",

	PanelServers:     "SERVERS",
	PanelMenu:        "MENU",
	PanelActions:     "ACTIONS",
	PanelStats:       "STATS",
	PanelDesc:        "DESCRIPTION",
	PanelHint:        "Tab — panel · Enter/→ — actions · c — connect · / — search · n/m/r/d — folders · ? — help · q — quit",
	StatsLoading:     "loading…",
	StatsError:       "no data",
	StatsOffline:     "server unreachable",
	TermTooSmall:     "Terminal window is too small. Minimum is %d×%d, current %d×%d — please enlarge it.",
	NoServersHint:    "No servers — add your first one from the menu on the left.",
	MenuAddDesc:      "Add a new VPS: manually or with automatic first-time setup (bootstrap) — system update, sudo user, key, firewall, fail2ban.",
	MenuMultiDesc:    "Run one command on several servers at once (in parallel).",
	MenuImportDesc:   "Import servers from your ~/.ssh/config (Host blocks) — bring your whole fleet in quickly.",
	MenuExportDesc:   "Export servers to ~/.ssh/config so `ssh <name>` works from any terminal.",
	MenuBackupsDesc:  "Restore config from an automatic snapshot (taken before every change).",
	MenuSnippetsDesc: "Saved commands for quick execution on servers; delete unneeded ones here.",
	MenuPubkeyDesc:   "Show and copy the unissh public SSH key to add to your servers.",
	MenuSettingsDesc: "Language, theme, connect mode, confirmation, default bootstrap steps and more.",
	MenuExitDesc:     "Quit unissh.",

	FilterPrompt: "🔍 ",
	SortByName:   "sort: by name",
	SortByPing:   "sort: by ping",
	SortByOnline: "sort: by availability",
	HelpTitle:    "Keybindings",
	HelpClose:    "any key — close",
	HelpNavigate: "navigate",
	HelpConnect:  "connect",
	HelpTunnels:  "port forwarding",
	HelpSelect:   "mark server (or group)",
	HelpFavorite: "favorite (★)",
	HelpSort:     "change sort",
	HelpFilter:   "search/filter",
	HelpHelp:     "this help",
	HelpQuit:     "quit",

	FolderRoot:        "No folder",
	PanelFolder:       "FOLDER",
	FolderServersFmt:  "Servers:",
	FolderHint:        "enter/←→ — collapse/expand · n — new · r — rename · d — delete · shift+↑↓ — reorder",
	FilterHint:        "type to search · Enter — apply · Esc — reset",
	MoveHint:          "moving: ↑↓ — pick folder · Enter — save · Esc — cancel",
	HelpPanels:        "switch panel (servers/menu)",
	HelpJump:          "jump to start / end of list",
	HelpOpenActions:   "open server actions / expand folder",
	HelpBack:          "back / collapse folder",
	HelpFolderNew:     "create folder",
	HelpFolderMove:    "grab a server and walk it across folders (Enter — save)",
	HelpFolderDel:     "delete folder (servers go to root)",
	AskFolderName:     "Folder name:",
	FolderNewTitle:    "New folder",
	FolderRenameTitle: "Rename folder",
	HelpFolderRename:  "rename folder",
	HelpFolderOrder:   "reorder folders",
	FolderDeleteBody:  "Its servers move back to \"No folder\".",
	PromptHint:        "Enter — confirm · Esc — cancel",
	ConfirmHint:       "Enter/y — delete · Esc/n — cancel",
	ErrFolderExists:   "a folder with this name already exists",
	FolderNewOption:   "＋ New folder…",
	AskDeleteFolder:   "Delete folder \"%s\"?",
	FldFolder:         "Folder: ",
	AskFolderChoose:   "Server folder:",

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

	LblIPs:             "IP addresses",
	SetConnectMode:     "Open SSH: ",
	DescLanguage:       "Interface language: Russian or English.",
	DescTheme:          "Color theme. Applies instantly to every screen.",
	DescConnectMode:    "Open SSH in a new terminal window or inline in the current one.",
	DescDefaultUser:    "Username pre-filled by default when adding a new server.",
	DescSSHKey:         "Path to the private SSH key used to log in and to install onto servers.",
	DescTimeout:        "How many seconds to wait for a status check (ping). Higher is more reliable on slow links.",
	DescBootstrapSteps: "Which first-time setup (bootstrap) steps are checked by default for a new server.",
	AskConnectMode:     "How to open the SSH connection?",
	ConnectModeWindow:  "in a new terminal window",
	ConnectModeInline:  "in the same window",
	OpenedNewWindow:    "↗ Connection opened in a new terminal window",
	ErrOpenWindow:      "could not open a new window (%v) — connecting here",
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
