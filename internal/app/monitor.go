package app

// Сбор live-метрик сервера (CPU, RAM, диск, load average, сеть, аптайм, IP).
// Отрисовкой занимается панель СТАТИСТИКА главного экрана (tui2.go): она
// опрашивает сервер по тикеру и считает дельты между снимками.

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

const monCmd = `cat /proc/loadavg; echo @; free -b | sed -n 2p; echo @; ` +
	`df -P -k / | tail -1; echo @; head -1 /proc/stat; echo @; ` +
	`cat /proc/net/dev | tail -n +3; echo @; nproc; echo @; uptime -p 2>/dev/null || true; ` +
	// внешние (не loopback/link-local) IP-адреса интерфейсов; запасной вариант — hostname -I
	`echo @; A=$(ip -o addr show scope global 2>/dev/null | awk '{print $4}' | cut -d/ -f1); ` +
	`[ -n "$A" ] || A=$(hostname -I 2>/dev/null); echo "$A" | tr ' ' '\n'`

// monSample — один срез метрик сервера (счётчики /proc кумулятивные).
type monSample struct {
	load    string
	memLine string
	dfLine  string
	stat    []int64
	rx, tx  int64
	ncpu    int
	uptime  string
	ips     []string // внешние IP-адреса VPS
	at      time.Time
	err     error
}

// parseMonSample разбирает вывод monCmd в снимок метрик (секции через @).
func parseMonSample(out string, at time.Time) *monSample {
	s := &monSample{at: at}
	parts := strings.Split(out, "\n@\n")
	if len(parts) < 6 {
		s.err = errors.New(T.ErrUnexpectedResp)
		return s
	}
	s.load = strings.TrimSpace(parts[0])
	s.memLine = strings.TrimSpace(parts[1])
	s.dfLine = strings.TrimSpace(parts[2])
	// первое поле — метка "cpu", далее числа; на пустой/обрезанной секции
	// среза может не быть вовсе — иначе [1:] паникует в горутине tea.Cmd
	if statFields := strings.Fields(strings.TrimSpace(parts[3])); len(statFields) > 1 {
		for _, f := range statFields[1:] {
			v, _ := strconv.ParseInt(f, 10, 64)
			s.stat = append(s.stat, v)
		}
	}
	for _, line := range strings.Split(parts[4], "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 10 && !strings.HasPrefix(fields[0], "lo") {
			rx, _ := strconv.ParseInt(fields[1], 10, 64)
			tx, _ := strconv.ParseInt(fields[9], 10, 64)
			s.rx += rx
			s.tx += tx
		}
	}
	s.ncpu, _ = strconv.Atoi(strings.TrimSpace(parts[5]))
	if s.ncpu < 1 {
		s.ncpu = 1
	}
	if len(parts) > 6 {
		s.uptime = strings.TrimSpace(parts[6])
	}
	if len(parts) > 7 {
		seen := map[string]bool{}
		for _, line := range strings.Split(parts[7], "\n") {
			ip := strings.TrimSpace(line)
			if ip != "" && !seen[ip] {
				seen[ip] = true
				s.ips = append(s.ips, ip)
			}
		}
	}
	return s
}
