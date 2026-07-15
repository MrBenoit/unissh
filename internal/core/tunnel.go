package core

// Чистая логика проброса портов: валидация правила, флаги для ssh, канонический
// ключ и человекочитаемая подпись. Ничего про процессы и TUI — только строки.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Виды проброса портов (совпадают с флагами ssh -L / -R / -D).
const (
	TunnelLocal   = "local"   // -L: локальный порт → хост:порт со стороны сервера
	TunnelRemote  = "remote"  // -R: порт на сервере → хост:порт с нашей стороны
	TunnelDynamic = "dynamic" // -D: SOCKS5-прокси на локальном порту
)

// Ошибки валидации правила. Вызывающий переводит их в строки интерфейса —
// core ничего не знает про язык.
var (
	ErrTunnelKind = errors.New("unknown tunnel kind")
	ErrTunnelPort = errors.New("port out of range")
	ErrTunnelHost = errors.New("remote host is required")
)

// ValidPort сообщает, укладывается ли номер порта в допустимый диапазон.
func ValidPort(p int) bool { return p >= 1 && p <= 65535 }

// ValidateTunnel проверяет правило целиком: вид, порты и (для -L/-R) хост
// назначения. Для -D удалённые поля не используются и не проверяются.
func ValidateTunnel(kind string, localPort int, remoteHost string, remotePort int) error {
	switch kind {
	case TunnelLocal, TunnelRemote:
		if !ValidPort(localPort) || !ValidPort(remotePort) {
			return ErrTunnelPort
		}
		if strings.TrimSpace(remoteHost) == "" {
			return ErrTunnelHost
		}
		if strings.ContainsAny(remoteHost, " \t:") {
			return ErrTunnelHost
		}
	case TunnelDynamic:
		if !ValidPort(localPort) {
			return ErrTunnelPort
		}
	default:
		return ErrTunnelKind
	}
	return nil
}

// TunnelFlags возвращает аргументы ssh, включающие проброс. Правило считается
// уже провалидированным: на неизвестный вид отдаём nil, чтобы не собрать
// команду наполовину.
//
//	-L localPort:remoteHost:remotePort   слушаем у себя, ходим со стороны сервера
//	-R remotePort:remoteHost:localPort   слушаем на сервере, ходим с нашей стороны
//	-D localPort                         SOCKS5 у себя
func TunnelFlags(kind string, localPort int, remoteHost string, remotePort int) []string {
	switch kind {
	case TunnelLocal:
		return []string{"-L", fmt.Sprintf("%d:%s:%d", localPort, remoteHost, remotePort)}
	case TunnelRemote:
		return []string{"-R", fmt.Sprintf("%d:%s:%d", remotePort, remoteHost, localPort)}
	case TunnelDynamic:
		return []string{"-D", strconv.Itoa(localPort)}
	}
	return nil
}

// TunnelSpec — канонический ключ правила: по нему туннели различаются в
// менеджере процессов и проверяются на дубликаты. Формат машинный и стабильный.
func TunnelSpec(kind string, localPort int, remoteHost string, remotePort int) string {
	switch kind {
	case TunnelLocal:
		return fmt.Sprintf("L:%d:%s:%d", localPort, remoteHost, remotePort)
	case TunnelRemote:
		return fmt.Sprintf("R:%d:%s:%d", remotePort, remoteHost, localPort)
	case TunnelDynamic:
		return fmt.Sprintf("D:%d", localPort)
	}
	return "?:" + kind
}

// TunnelLabel — подпись правила для интерфейса. Буквы L/R/D намеренно совпадают
// с флагами ssh, а «srv:» означает порт на стороне сервера; строка одинакова
// на обоих языках и не требует перевода.
func TunnelLabel(kind string, localPort int, remoteHost string, remotePort int) string {
	switch kind {
	case TunnelLocal:
		return fmt.Sprintf("L  :%d → %s:%d", localPort, remoteHost, remotePort)
	case TunnelRemote:
		return fmt.Sprintf("R  srv:%d → %s:%d", remotePort, remoteHost, localPort)
	case TunnelDynamic:
		return fmt.Sprintf("D  :%d  SOCKS5", localPort)
	}
	return kind
}

// TunnelListensLocally сообщает, занимает ли правило порт на нашей машине.
// Для -R порт слушает сервер, поэтому локального конфликта быть не может.
func TunnelListensLocally(kind string) bool {
	return kind == TunnelLocal || kind == TunnelDynamic
}
