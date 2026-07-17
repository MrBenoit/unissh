// Command demoserver — фиктивный SSH-сервер для записи demo.gif.
//
// Зачем: записывать демо на настоящих серверах нельзя (в кадр попадают их IP,
// логины и имена), а на пустых TCP-слушателях панель статистики вечно висит на
// «loading…» — не видно шкал CPU/RAM/диска, то есть главной фичи.
// Этот сервер говорит по SSH и отвечает на monCmd правдоподобными данными
// /proc, поэтому в демо видно живые шкалы и скорость сети.
//
// Он НЕ входит в поставку unissh: это инструмент записи, его собирает
// scripts/record-demo.sh. Никакой безопасности здесь нет и быть не должно —
// сервер принимает любую аутентификацию и слушает только loopback.
//
// Запуск: demoserver '203.0.113.10:2222=42,31,24' '203.0.113.11:2222=17,55,61' …
// где числа — проценты загрузки CPU, памяти и диска.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// profile — как выглядит «сервер»: целевые проценты загрузки и растущие
// счётчики. Счётчики /proc кумулятивные, а unissh считает CPU и скорость сети
// по дельте двух замеров — поэтому от вызова к вызову они должны расти.
type profile struct {
	mu               sync.Mutex
	cpu, mem, disk   float64 // доли 0..1
	ticks            int64   // сколько раз опрашивали
	rxBytes, txBytes int64
	addr             string
}

// nextStat отдаёт очередной срез /proc/stat. Между двумя вызовами занятых
// тиков прибавляется ровно cpu*100, простоя — остальное: unissh поделит одно
// на другое и получит нужный процент.
func (p *profile) nextStat() (user, idle int64) {
	busy := int64(p.cpu * 100)
	return baseUser + p.ticks*busy, baseIdle + p.ticks*(100-busy)
}

const (
	baseUser = 1_000_000
	baseIdle = 9_000_000
	// system НЕ растёт: unissh считает CPU по дельте, и любой растущий счётчик
	// кроме user/idle сместил бы процент (было 47.7% вместо 42%).
	constSystem = 500_000
	// сколько байт «прилетает» между опросами: панель обновляется раз в 2 с,
	// поэтому 2.4 МБ дают правдоподобные ~1.2 МБ/с
	rxPerTick = 2_400_000
	txPerTick = 176_000
)

// monSample собирает ответ на monCmd: восемь секций через «@», ровно как их
// ждёт parseMonSample.
func (p *profile) monSample() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ticks++
	p.rxBytes += rxPerTick
	p.txBytes += txPerTick
	user, idle := p.nextStat()

	const memTotal = 16_777_216_000 // 16 ГБ
	memUsed := int64(float64(memTotal) * p.mem)
	const diskTotalKB = 104_857_600 // 100 ГБ в КБ (df -P -k)
	diskUsedKB := int64(float64(diskTotalKB) * p.disk)

	host, _, _ := net.SplitHostPort(p.addr)

	sections := []string{
		// /proc/loadavg — load average подбираем под загрузку CPU
		fmt.Sprintf("%.2f %.2f %.2f 2/312 12345", p.cpu*8, p.cpu*7, p.cpu*6),
		// free -b, вторая строка: parseMonSample берёт поля [1]=total, [2]=used
		fmt.Sprintf("Mem: %d %d %d 0 0 %d", memTotal, memUsed, memTotal-memUsed, memTotal-memUsed),
		// df -P -k /: поля [1]=total, [2]=used (в КБ)
		fmt.Sprintf("/dev/sda1 %d %d %d %.0f%% /", diskTotalKB, diskUsedKB,
			diskTotalKB-diskUsedKB, p.disk*100),
		// head -1 /proc/stat: user nice system idle iowait irq softirq steal.
		// Растут только user и idle: их дельты дают ровно p.cpu.
		fmt.Sprintf("cpu %d 0 %d %d 0 0 0 0", user, constSystem, idle),
		// /proc/net/dev: поля [1]=rx_bytes, [9]=tx_bytes; lo пропускается
		fmt.Sprintf("  eth0: %d 0 0 0 0 0 0 0 %d 0 0 0 0 0 0 0", p.rxBytes, p.txBytes),
		"8",                  // nproc
		"up 3 weeks, 2 days", // uptime -p
		host,                 // IP-адреса
	}
	return strings.Join(sections, "\n@\n")
}

// respond решает, что ответить на команду. Нас интересует только monCmd —
// демо больше ничего не запускает; на прочее отдаём пустой успешный вывод.
func (p *profile) respond(cmd string) string {
	if strings.Contains(cmd, "/proc/loadavg") {
		return p.monSample()
	}
	return ""
}

// parseSpec разбирает «addr:port=cpu,mem,disk».
func parseSpec(spec string) (*profile, error) {
	addr, nums, ok := strings.Cut(spec, "=")
	if !ok {
		return nil, fmt.Errorf("нужен формат addr:port=cpu,mem,disk, получено %q", spec)
	}
	parts := strings.Split(nums, ",")
	if len(parts) != 3 {
		return nil, fmt.Errorf("нужны три числа (cpu,mem,disk), получено %q", nums)
	}
	vals := make([]float64, 3)
	for i, s := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return nil, fmt.Errorf("%q: не число", s)
		}
		vals[i] = v / 100
	}
	return &profile{addr: addr, cpu: vals[0], mem: vals[1], disk: vals[2]}, nil
}

// newServerConfig — SSH-конфиг, принимающий кого угодно. Так и задумано:
// сервер эфемерный, слушает loopback и живёт только на время записи.
func newServerConfig() (*ssh.ServerConfig, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ServerConfig{
		NoClientAuth: true,
		PasswordCallback: func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
			return nil, nil
		},
		PublicKeyCallback: func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
			return nil, nil
		},
	}
	cfg.AddHostKey(signer)
	return cfg, nil
}

func (p *profile) serve(conn net.Conn, cfg *ssh.ServerConfig) {
	sconn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return
	}
	defer func() { _ = sconn.Close() }()
	go ssh.DiscardRequests(reqs)
	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(ssh.UnknownChannelType, "only session")
			continue
		}
		ch, requests, err := newCh.Accept()
		if err != nil {
			continue
		}
		go p.session(ch, requests)
	}
}

func (p *profile) session(ch ssh.Channel, requests <-chan *ssh.Request) {
	for req := range requests {
		if req.Type != "exec" {
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
			continue
		}
		var payload struct{ Cmd string }
		if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
			return
		}
		if req.WantReply {
			_ = req.Reply(true, nil)
		}
		// stdin вычитываем в фоне: клиент может дослать его после нашего вывода
		drained := make(chan struct{})
		go func() { _, _ = io.Copy(io.Discard, ch); close(drained) }()

		_, _ = ch.Write([]byte(p.respond(payload.Cmd)))
		_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Code uint32 }{0}))
		_ = ch.CloseWrite()
		<-drained
		_ = ch.Close()
		return
	}
}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		log.Fatal("укажите хотя бы один сервер: addr:port=cpu,mem,disk")
	}
	cfg, err := newServerConfig()
	if err != nil {
		log.Fatalf("ключ хоста: %v", err)
	}
	var wg sync.WaitGroup
	for _, spec := range os.Args[1:] {
		p, err := parseSpec(spec)
		if err != nil {
			log.Fatalf("%v", err)
		}
		ln, err := net.Listen("tcp", p.addr)
		if err != nil {
			log.Fatalf("слушать %s: %v", p.addr, err)
		}
		fmt.Printf("демо-сервер: %s (cpu %.0f%%, mem %.0f%%, disk %.0f%%)\n",
			p.addr, p.cpu*100, p.mem*100, p.disk*100)
		wg.Add(1)
		go func(p *profile, ln net.Listener) {
			defer wg.Done()
			for {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				go p.serve(conn, cfg)
			}
		}(p, ln)
	}
	wg.Wait()
}
