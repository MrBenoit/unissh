package app

// Панель информации о сервере: ОС, ресурсы, топ процессов, docker.

import (
	"fmt"
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

func kv(label, value string) string {
	return styViolet.Bold(true).Width(9).Align(lipgloss.Right).Render(label) + "  " + value
}

// actionInfo показывает полную карточку сервера отдельной полноэкранной
// страницей. Соединение (и возможный запрос пароля) открывается ДО страницы:
// иначе форма пароля рисовалась бы поверх работающего рендерера — два
// tea.Program на одном tty не уживаются.
func actionInfo(s *Server) {
	r, err := openRemote(s, true)
	if err != nil {
		showError(s.Name, err)
		return
	}
	defer r.Close()

	runPage(s.Name, fmt.Sprintf(T.Gathering, s.Name), nil, func() []string {
		_, raw, err := r.Run(infoCmd, "", nil, 30*time.Second)
		if err != nil {
			return wrapText(styRed.Render(err.Error()), 76)
		}
		return infoLines(s, sanitizeRemote(raw))
	}, nil)
}

// infoLines превращает вывод infoCmd в строки карточки сервера.
func infoLines(s *Server, out string) []string {
	sec := parseSections(out)
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
			rows = append(rows, "", styCyan.Bold(true).Render("Docker"))
			rows = append(rows, lines...)
		}
	}
	return rows
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
