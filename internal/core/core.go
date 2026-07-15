// Package core содержит чистую, самодостаточную логику unissh без зависимостей
// от TUI, i18n, стилей и ввода-вывода терминала. Именно этот пакет покрывается
// тестами на 90%+ — сборка и релиз запрещены при меньшем покрытии (scripts/coverage.sh).
package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ShellQuote безопасно оборачивает строку для передачи в sh -c, экранируя
// одинарные кавычки приёмом '"'"'.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// ParseJump разбирает спецификацию бастиона «[user@]host[:port]».
func ParseJump(spec, defaultUser string) (user, host, port string) {
	user, host, port = defaultUser, spec, "22"
	if at := strings.LastIndex(spec, "@"); at >= 0 {
		user, host = spec[:at], spec[at+1:]
	}
	if c := strings.LastIndex(host, ":"); c >= 0 {
		port, host = host[c+1:], host[:c]
	}
	return
}

// HexToRGB разбирает «#rrggbb» (или «rrggbb») в компоненты RGB.
func HexToRGB(h string) (int, int, int) {
	var r, g, b int
	_, _ = fmt.Sscanf(strings.TrimPrefix(h, "#"), "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

// BlendHex линейно смешивает два цвета: f=0 → a, f=1 → b.
func BlendHex(a, b string, f float64) string {
	ar, ag, ab := HexToRGB(a)
	br, bg, bb := HexToRGB(b)
	mix := func(x, y int) int { return x + int(float64(y-x)*f) }
	return fmt.Sprintf("#%02x%02x%02x", mix(ar, br), mix(ag, bg), mix(ab, bb))
}

// FormatPing форматирует пинг: дробные мс для быстрых (<10 мс), целые — иначе.
func FormatPing(d time.Duration) string {
	ms := float64(d.Microseconds()) / 1000.0
	if ms < 10 {
		return strconv.FormatFloat(ms, 'f', 1, 64)
	}
	return strconv.FormatFloat(ms, 'f', 0, 64)
}

// DefaultByteUnits — единицы размера по умолчанию (если не переданы локализованные).
var DefaultByteUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

// HumanBytes переводит число байт в человекочитаемый вид с заданными единицами.
func HumanBytes(n float64, units []string) string {
	if len(units) == 0 {
		units = DefaultByteUnits
	}
	i := 0
	for n >= 1024 && i < len(units)-1 {
		n /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", n, units[i])
	}
	return fmt.Sprintf("%.1f %s", n, units[i])
}

// SanitizeRemote убирает управляющие символы (в т.ч. ANSI/OSC-последовательности)
// из вывода удалённого сервера, чтобы он не мог перекрасить/переписать локальный
// терминал. Сохраняются перевод строки и табуляция.
func SanitizeRemote(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == 0x1b { // ESC — пропускаем следующую управляющую последовательность
			i++
			if i < len(runes) && runes[i] == '[' { // CSI ...
				i++
				for i < len(runes) && (runes[i] < 0x40 || runes[i] > 0x7e) {
					i++
				}
			} else if i < len(runes) && runes[i] == ']' { // OSC ... BEL/ST
				i++
				for i < len(runes) && runes[i] != 0x07 && runes[i] != 0x1b {
					i++
				}
			}
			continue
		}
		if r == '\n' || r == '\t' || r >= 0x20 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var sectionRe = regexp.MustCompile(`^===([A-Z]+)===$`)

// ParseSections разбивает вывод на секции по маркерам вида ===NAME===.
func ParseSections(raw string) map[string]string {
	out := map[string]string{}
	current := ""
	var buf []string
	flush := func() {
		if current != "" {
			out[current] = strings.TrimSpace(strings.Join(buf, "\n"))
		}
	}
	for _, line := range strings.Split(raw, "\n") {
		if m := sectionRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			flush()
			current = m[1]
			buf = nil
			continue
		}
		buf = append(buf, line)
	}
	flush()
	return out
}

// ParseCPUStatLine разбирает строку «cpu ...» из /proc/stat в срез чисел.
func ParseCPUStatLine(l string) []int64 {
	f := strings.Fields(l)
	if len(f) < 2 || f[0] != "cpu" {
		return nil
	}
	var v []int64
	for _, x := range f[1:] {
		n, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			break
		}
		v = append(v, n)
	}
	return v
}

// CPUPercentFromStat вычисляет загрузку CPU по двум замерам строки /proc/stat cpu.
func CPUPercentFromStat(section string) (float64, bool) {
	lines := strings.Split(strings.TrimSpace(section), "\n")
	if len(lines) < 2 {
		return 0, false
	}
	a := ParseCPUStatLine(lines[0])
	b := ParseCPUStatLine(lines[len(lines)-1])
	if len(a) < 5 || len(b) < 5 {
		return 0, false
	}
	n := minInt(minInt(len(a), len(b)), 8) // user..steal, без guest/guest_nice
	var total int64
	for i := 0; i < n; i++ {
		total += b[i] - a[i]
	}
	idle := b[3] - a[3]
	if n > 4 {
		idle += b[4] - a[4] // + iowait
	}
	if total <= 0 {
		return 0, false
	}
	return float64(total-idle) / float64(total), true
}

// CPUNetDeltas вычисляет мгновенную загрузку CPU и скорости сети по двум снимкам
// кумулятивных счётчиков (statA/statB — строки /proc/stat cpu; rx/tx — байты;
// dt — интервал). ok=false, если данных недостаточно.
func CPUNetDeltas(statA, statB []int64, rxA, txA, rxB, txB int64, dt float64) (cpu, rxRate, txRate float64, ok bool) {
	if len(statA) < 4 || len(statB) < 4 {
		return 0, 0, 0, false
	}
	// n считаем по КОРОТКОМУ снимку: если один из замеров обрезан, обращение к
	// statA[4] по длине statB вышло бы за границы среза (паника).
	n := minInt(minInt(len(statA), len(statB)), 8)
	var total, idle int64
	for i := 0; i < n; i++ {
		total += statB[i] - statA[i]
	}
	idle = statB[3] - statA[3]
	if n > 4 && len(statA) > 4 && len(statB) > 4 {
		idle += statB[4] - statA[4] // + iowait
	}
	if total > 0 {
		cpu = float64(total-idle) / float64(total)
	}
	if dt > 0 {
		rxRate = clampNonNeg(float64(rxB-rxA) / dt)
		txRate = clampNonNeg(float64(txB-txA) / dt)
	}
	return cpu, rxRate, txRate, true
}

// ExpandPath раскрывает ведущий «~/» в домашний каталог пользователя.
func ExpandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

// NewID генерирует случайный 8-символьный hex-идентификатор.
func NewID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// PasswordAlphabet — символы для генерации паролей без неоднозначных (0/O, 1/l/I).
const PasswordAlphabet = "abcdefghijkmnpqrstuvwxyz" +
	"ABCDEFGHJKLMNPQRSTUVWXYZ" +
	"23456789" +
	"!@#%^*-_=+"

// GenPassword — криптостойкий пароль заданной длины со всеми классами символов.
func GenPassword(length int) string {
	for {
		b := make([]byte, length)
		for i := range b {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(PasswordAlphabet))))
			if err != nil {
				panic(err)
			}
			b[i] = PasswordAlphabet[n.Int64()]
		}
		pw := string(b)
		if strings.ContainsAny(pw, "abcdefghijkmnpqrstuvwxyz") &&
			strings.ContainsAny(pw, "ABCDEFGHJKLMNPQRSTUVWXYZ") &&
			strings.ContainsAny(pw, "23456789") &&
			strings.ContainsAny(pw, "!@#%^*-_=+") {
			return pw
		}
	}
}

// FirstLine возвращает первую строку многострочного текста.
func FirstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// SSHConfValue экранирует значение для ssh_config: значения с пробелами/кавычками
// оборачиваются в двойные кавычки.
//
// Переводы строк вырезаются: кавычки ssh_config не переносятся через строку, и
// значение с \n дописало бы в ~/.ssh/unissh_config произвольную директиву
// (например ProxyCommand) — то есть выполнение кода при следующем ssh.
func SSHConfValue(v string) string {
	v = strings.NewReplacer("\r", " ", "\n", " ").Replace(v)
	if v != "" && !strings.ContainsAny(v, " \t\"") {
		return v
	}
	return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampNonNeg(f float64) float64 {
	if f < 0 {
		return 0
	}
	return f
}
