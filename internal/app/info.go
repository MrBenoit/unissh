package app

// Панель информации о сервере: ОС, ресурсы, топ процессов, docker.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const infoCmd = `echo "===OS==="; (. /etc/os-release 2>/dev/null && echo "$PRETTY_NAME") || uname -s; ` +
	`echo "===HOST==="; hostname; ` +
	`echo "===KERNEL==="; uname -r; ` +
	`echo "===UPTIME==="; uptime -p 2>/dev/null || uptime; ` +
	`echo "===LOAD==="; cat /proc/loadavg 2>/dev/null; ` +
	`echo "===CPU==="; nproc 2>/dev/null; ` +
	`echo "===CPUSTAT==="; head -1 /proc/stat 2>/dev/null; sleep 0.3; head -1 /proc/stat 2>/dev/null; ` +
	`echo "===MEM==="; free -b 2>/dev/null | sed -n 2p; ` +
	`echo "===DISK==="; df -P -k / | tail -1; ` +
	`echo "===IP==="; hostname -I 2>/dev/null || hostname -i 2>/dev/null; ` +
	`echo "===IPS==="; ` +
	`(ip -o addr show scope global 2>/dev/null | awk '{print $2"\t"$4}') || ` +
	`(ifconfig 2>/dev/null | awk '/inet /{print "-\t"$2}'); ` +
	`echo "===DOCKER==="; command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}\t{{.Status}}\t{{.Image}}' 2>/dev/null; ` +
	`echo "===END==="`

// cpuPercentFromStat вычисляет загрузку CPU по двум замерам строки /proc/stat cpu.
func cpuPercentFromStat(section string) (float64, bool) {
	lines := strings.Split(strings.TrimSpace(section), "\n")
	if len(lines) < 2 {
		return 0, false
	}
	a := parseCPUStatLine(lines[0])
	b := parseCPUStatLine(lines[len(lines)-1])
	if len(a) < 5 || len(b) < 5 {
		return 0, false
	}
	n := min(min(len(a), len(b)), 8) // user..steal, без guest/guest_nice
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

func parseCPUStatLine(l string) []int64 {
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

var sectionRe = regexp.MustCompile(`^===([A-Z]+)===$`)

func parseSections(raw string) map[string]string {
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

func kv(label, value string) string {
	return styViolet.Bold(true).Width(9).Align(lipgloss.Right).Render(label) + "  " + value
}

func actionInfo(s *Server) {
	// Соединение (и возможный запрос пароля) открываем ДО спиннера: иначе huh-форма
	// пароля рисовалась бы поверх работающего спиннера — два рендерера на одном tty.
	r, err := openRemote(s, true)
	if err != nil {
		fmt.Println(styRed.Render(err.Error()))
		pressEnter()
		return
	}
	defer r.Close()

	var raw string
	err = withSpinner(fmt.Sprintf(T.Gathering, s.Name), func() error {
		_, out, err := r.Run(infoCmd, "", nil, 30*time.Second)
		raw = out
		return err
	})
	if err != nil {
		fmt.Println(styRed.Render(err.Error()))
		pressEnter()
		return
	}

	sec := parseSections(sanitizeRemote(raw))
	var rows []string
	rows = append(rows,
		kv(T.LblOS, orDash(sec["OS"])),
		kv(T.LblHost, orDash(sec["HOST"])),
		kv(T.LblKernel, orDash(sec["KERNEL"])),
		kv(T.LblUptime, orDash(sec["UPTIME"])),
	)
	// все IP-адреса интерфейсов (первый — с меткой, остальные — выровнены)
	for i, line := range ipAddrLines(sec) {
		label := ""
		if i == 0 {
			label = T.LblIPs
		}
		rows = append(rows, kv(label, line))
	}

	ncpu, _ := strconv.Atoi(strings.TrimSpace(sec["CPU"]))
	if ncpu < 1 {
		ncpu = 1
	}
	if frac, ok := cpuPercentFromStat(sec["CPUSTAT"]); ok {
		rows = append(rows, kv(T.LblCPU,
			meter(frac, 18)+styDim.Render(fmt.Sprintf(T.CPUsFmt, ncpu))))
	}
	if load := strings.Fields(sec["LOAD"]); len(load) >= 3 {
		la1, _ := strconv.ParseFloat(load[0], 64)
		frac := la1 / float64(ncpu)
		sty := styGreen
		if frac >= 1 {
			sty = styRed
		} else if frac >= 0.7 {
			sty = styYellow
		}
		rows = append(rows, kv(T.LblLoad,
			sty.Render(strings.Join(load[:3], " "))+styDim.Render(fmt.Sprintf(T.CPUsFmt, ncpu))))
	}
	if mem := strings.Fields(sec["MEM"]); len(mem) >= 3 {
		total, _ := strconv.ParseFloat(mem[1], 64)
		used, _ := strconv.ParseFloat(mem[2], 64)
		if total > 0 {
			rows = append(rows, kv(T.LblMemory,
				meter(used/total, 18)+styGrey.Render("  "+humanBytes(used)+" / "+humanBytes(total))))
		}
	}
	if disk := strings.Fields(sec["DISK"]); len(disk) >= 4 {
		total, _ := strconv.ParseFloat(disk[1], 64)
		used, _ := strconv.ParseFloat(disk[2], 64)
		if total > 0 {
			rows = append(rows, kv(T.LblDiskRoot,
				meter(used/total, 18)+styGrey.Render("  "+humanBytes(used*1024)+" / "+humanBytes(total*1024))))
		}
	}

	fmt.Println()
	fmt.Println(panelViolet.Render(
		styBold.Render(s.Name) + "\n\n" + strings.Join(rows, "\n")))

	if docker := sec["DOCKER"]; docker != "" {
		var lines []string
		for _, l := range strings.Split(docker, "\n") {
			parts := strings.Split(l, "\t")
			if len(parts) == 3 {
				lines = append(lines, styBold.Render(parts[0])+"  "+
					styGreen.Render(parts[1])+"  "+styDim.Render(parts[2]))
			}
		}
		if len(lines) > 0 {
			fmt.Println(panelCyan.Render(
				styCyan.Bold(true).Render("Docker") + "\n" + strings.Join(lines, "\n")))
		}
	}

	pressEnter()
}

// ipAddrLines формирует строки со всеми адресами интерфейсов из секции IPS
// (формат "iface\taddr/mask"); при её отсутствии — из hostname -I (секция IP).
func ipAddrLines(sec map[string]string) []string {
	var out []string
	if ips := strings.TrimSpace(sec["IPS"]); ips != "" {
		for _, line := range strings.Split(ips, "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
			if len(parts) != 2 || parts[1] == "" {
				continue
			}
			iface, addr := parts[0], parts[1]
			addrSty := styGreen // IPv4
			if strings.Contains(addr, ":") {
				addrSty = styCyan // IPv6
			}
			row := addrSty.Render(addr)
			if iface != "" && iface != "-" {
				row += styDim.Render("  (" + iface + ")")
			}
			out = append(out, row)
		}
	}
	if len(out) == 0 {
		// запасной вариант: hostname -I (адреса через пробел)
		for _, a := range strings.Fields(sec["IP"]) {
			sty := styGreen
			if strings.Contains(a, ":") {
				sty = styCyan
			}
			out = append(out, sty.Render(a))
		}
	}
	if len(out) == 0 {
		out = append(out, "—")
	}
	return out
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return strings.TrimSpace(s)
}
