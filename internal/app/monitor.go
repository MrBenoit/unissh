package app

// Live-монитор: CPU, RAM, диск, load average, сеть — обновление каждые 2 секунды.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const monCmd = `cat /proc/loadavg; echo @; free -b | sed -n 2p; echo @; ` +
	`df -P -k / | tail -1; echo @; head -1 /proc/stat; echo @; ` +
	`cat /proc/net/dev | tail -n +3; echo @; nproc; echo @; uptime -p 2>/dev/null || true`

type monSample struct {
	load    string
	memLine string
	dfLine  string
	stat    []int64
	rx, tx  int64
	ncpu    int
	uptime  string
	at      time.Time
	err     error
}

type monTickMsg struct{}

type monModel struct {
	server *Server
	remote *Remote

	prev    *monSample
	cpuFrac float64
	rxRate  float64
	txRate  float64
	sample  *monSample
	errText string
}

func fetchSample(r *Remote) tea.Cmd {
	return func() tea.Msg {
		code, out, err := r.Run(monCmd, "", nil, 10*time.Second)
		s := &monSample{at: time.Now()}
		if err != nil {
			s.err = err
			return s
		}
		_ = code
		parts := strings.Split(out, "\n@\n")
		if len(parts) < 6 {
			s.err = errors.New(T.ErrUnexpectedResp)
			return s
		}
		s.load = strings.TrimSpace(parts[0])
		s.memLine = strings.TrimSpace(parts[1])
		s.dfLine = strings.TrimSpace(parts[2])
		for _, f := range strings.Fields(strings.TrimSpace(parts[3]))[1:] {
			v, _ := strconv.ParseInt(f, 10, 64)
			s.stat = append(s.stat, v)
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
		return s
	}
}

func (m *monModel) Init() tea.Cmd {
	return fetchSample(m.remote)
}

func (m *monModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case monTickMsg:
		return m, fetchSample(m.remote)
	case *monSample:
		if msg.err != nil {
			m.errText = msg.err.Error()
		} else {
			m.errText = ""
			if m.prev != nil && len(m.prev.stat) >= 4 && len(msg.stat) >= 4 {
				// /proc/stat cpu: user nice system idle iowait irq softirq steal guest guest_nice.
				// Суммируем только поля 0..7 (user..steal): guest/guest_nice ядро уже учитывает
				// в user/nice, иначе CPU% завышается на KVM-хостах.
				n := len(msg.stat)
				if n > 8 {
					n = 8
				}
				var total, idle int64
				for i := 0; i < n; i++ {
					if i < len(m.prev.stat) {
						total += msg.stat[i] - m.prev.stat[i]
					}
				}
				idle = msg.stat[3] - m.prev.stat[3]
				if n > 4 {
					idle += msg.stat[4] - m.prev.stat[4]
				}
				if total > 0 {
					m.cpuFrac = float64(total-idle) / float64(total)
				}
				dt := msg.at.Sub(m.prev.at).Seconds()
				if dt > 0 {
					m.rxRate = float64(msg.rx-m.prev.rx) / dt
					m.txRate = float64(msg.tx-m.prev.tx) / dt
					if m.rxRate < 0 {
						m.rxRate = 0
					}
					if m.txRate < 0 {
						m.txRate = 0
					}
				}
			}
			m.sample = msg
			m.prev = msg
		}
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return monTickMsg{} })
	}
	return m, nil
}

func (m *monModel) View() string {
	if m.errText != "" {
		return panelRed.Render(styRed.Render(T.MonError+m.errText)) + "\n" +
			styDim.Render(T.MonQuitHint) + "\n"
	}
	if m.sample == nil {
		return styDim.Render(T.MonConnecting) + "\n"
	}
	s := m.sample
	var rows []string
	rows = append(rows, kv(T.LblCPU, meter(m.cpuFrac, 18)))
	if mem := strings.Fields(s.memLine); len(mem) >= 3 {
		total, _ := strconv.ParseFloat(mem[1], 64)
		used, _ := strconv.ParseFloat(mem[2], 64)
		if total > 0 {
			rows = append(rows, kv(T.LblRAM,
				meter(used/total, 18)+styGrey.Render("  "+humanBytes(used)+" / "+humanBytes(total))))
		}
	}
	if disk := strings.Fields(s.dfLine); len(disk) >= 4 {
		total, _ := strconv.ParseFloat(disk[1], 64)
		used, _ := strconv.ParseFloat(disk[2], 64)
		if total > 0 {
			rows = append(rows, kv(T.LblDisk,
				meter(used/total, 18)+styGrey.Render("  "+humanBytes(used*1024)+" / "+humanBytes(total*1024))))
		}
	}
	if load := strings.Fields(s.load); len(load) >= 3 {
		rows = append(rows, kv(T.LblLoad,
			styGrey.Render(strings.Join(load[:3], " ")+fmt.Sprintf(T.CPUsFmt, s.ncpu))))
	}
	rows = append(rows, kv(T.LblNet,
		styCyan.Render("↓ "+humanBytes(m.rxRate)+"/s")+"  "+
			styPink.Render("↑ "+humanBytes(m.txRate)+"/s")))
	if s.uptime != "" {
		rows = append(rows, kv(T.LblUptime, styGrey.Render(s.uptime)))
	}
	title := styBold.Render(m.server.Name) + styDim.Render(T.LiveSuffix)
	return panelViolet.Render(title+"\n\n"+strings.Join(rows, "\n")) + "\n"
}

func actionMonitor(s *Server) {
	r, err := openRemote(s, true)
	if err != nil {
		fmt.Println(styRed.Render(err.Error()))
		pressEnter()
		return
	}
	defer r.Close()
	p := tea.NewProgram(&monModel{server: s, remote: r})
	if _, err := p.Run(); err != nil {
		fmt.Println(styRed.Render(T.MonitorStopped + err.Error()))
	}
}
