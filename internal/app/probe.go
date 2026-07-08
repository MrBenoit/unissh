package app

// Быстрая проверка доступности серверов (TCP-подключение к SSH-порту).

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

type probeResult struct {
	Online bool
	Ping   time.Duration
}

// formatPing форматирует пинг: дробные миллисекунды для быстрых (<10 мс),
// целые — для остальных. Иначе суб-миллисекундные пинги обрезались бы в «0».
func formatPing(d time.Duration) string {
	ms := float64(d.Microseconds()) / 1000.0
	if ms < 10 {
		return strconv.FormatFloat(ms, 'f', 1, 64)
	}
	return strconv.FormatFloat(ms, 'f', 0, 64)
}

func probe(host string, port int, timeout time.Duration) probeResult {
	t0 := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, fmt.Sprint(port)), timeout)
	if err != nil {
		return probeResult{Online: false}
	}
	conn.Close()
	return probeResult{Online: true, Ping: time.Since(t0)}
}

func probeAll(servers []*Server, timeout time.Duration) map[string]probeResult {
	res := make(map[string]probeResult, len(servers))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, s := range servers {
		wg.Add(1)
		go func(s *Server) {
			defer wg.Done()
			r := probe(s.Host, s.Port, timeout)
			mu.Lock()
			res[s.ID] = r
			mu.Unlock()
		}(s)
	}
	wg.Wait()
	return res
}

func statusTimeout(st *Store) time.Duration {
	t := st.Settings.StatusTimeout
	if t <= 0 { // защита от 0 в конфиге: иначе DialTimeout(0) = «без таймаута» (~75с)
		t = 1.5
	}
	return time.Duration(t * float64(time.Second))
}
