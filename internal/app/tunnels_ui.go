package app

// Диалоги проброса портов: список правил сервера, включение/выключение,
// добавление и удаление. Запускается из главного экрана (действие «tunnels»
// или клавиша «t») — как и остальные мастера, уже вне работающего tea.Program.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MrBenoit/unissh/internal/core"
)

// tunnelSettleDelay — сколько ждать после запуска, прежде чем поверить, что
// туннель поднялся. ssh с ExitOnForwardFailure падает почти сразу (занятый
// порт, запрет проброса на сервере), и без паузы мы бы отрапортовали успех.
var tunnelSettleDelay = 500 * time.Millisecond

// tunnelState возвращает индикатор и подпись состояния правила.
func tunnelState(serverID string, t Tunnel) (dot, state string) {
	switch {
	case tunnels.Running(serverID, t):
		return styGreen.Render("●"), T.TunnelUp
	case tunnels.Failure(serverID, t) != "":
		return styRed.Render("✗"), clip(tunnels.Failure(serverID, t), 60)
	}
	return styDim.Render("○"), T.TunnelDown
}

// actionTunnels — экран управления туннелями сервера. Enter на правиле
// переключает его, остальные пункты ведут в мастера.
func actionTunnels(st *Store, s *Server) {
	for {
		opts := make([]selOption[string], 0, len(s.Tunnels)+4)
		descs := make([]string, 0, len(s.Tunnels)+4)
		for i, t := range s.Tunnels {
			dot, state := tunnelState(s.ID, t)
			opts = append(opts, opt(dot+"  "+t.Label(), "t"+strconv.Itoa(i)))
			descs = append(descs, state)
		}
		opts = append(opts, opt(T.TunnelAdd, "add"))
		descs = append(descs, T.TunnelAddDesc)
		if len(s.Tunnels) > 0 {
			opts = append(opts, opt(T.TunnelDelete, "del"))
			descs = append(descs, T.TunnelDeleteDesc)
		}
		if tunnels.Active(s.ID) > 0 {
			opts = append(opts, opt(T.TunnelStopAll, "stopall"))
			descs = append(descs, T.TunnelStopAllDesc)
		}
		opts = append(opts, opt(T.Back, "back"))
		descs = append(descs, "")

		header := styGrey.Render(s.Addr()) + "\n" + styDim.Render(T.TunnelsHeader)
		if len(s.Tunnels) == 0 {
			header = styGrey.Render(s.Addr()) + "\n" + styYellow.Render(T.NoTunnelsYet)
		}
		pick, ok := askSelectFull(T.TunnelsTitle, header, opts, descs)
		if !ok || pick == "back" {
			return
		}
		switch pick {
		case "add":
			addTunnelFlow(st, s)
		case "del":
			deleteTunnelFlow(st, s)
		case "stopall":
			stopServerTunnels(s)
		default:
			i, err := strconv.Atoi(strings.TrimPrefix(pick, "t"))
			if err != nil || i < 0 || i >= len(s.Tunnels) {
				continue
			}
			toggleTunnel(s, s.Tunnels[i])
		}
	}
}

// toggleTunnel поднимает или гасит правило и печатает результат. После запуска
// выжидает tunnelSettleDelay: упавший сразу ssh не должен выглядеть живым.
func toggleTunnel(s *Server, t Tunnel) {
	if tunnels.Running(s.ID, t) {
		_ = tunnels.Stop(s.ID, t)
		showToast(styGreen.Render(T.TunnelStopped) + styGrey.Render("  "+t.Label()))
		return
	}
	if err := tunnels.Start(s, t); err != nil {
		showPage(T.TunnelsTitle, []string{styRed.Render(tunnelErrText(err))})
		return
	}
	time.Sleep(tunnelSettleDelay)
	if !tunnels.Running(s.ID, t) {
		reason := tunnels.Failure(s.ID, t)
		if reason == "" {
			reason = T.TunnelDown
		}
		showPage(T.TunnelsTitle, wrapText(styRed.Render(fmt.Sprintf(T.ErrTunnelStart, reason)), 76))
		return
	}
	showToast(styGreen.Render(T.TunnelStarted) + styGrey.Render("  "+t.Label()))
}

// stopServerTunnels гасит все туннели одного сервера.
func stopServerTunnels(s *Server) {
	n := 0
	for _, t := range s.Tunnels {
		if tunnels.Running(s.ID, t) {
			_ = tunnels.Stop(s.ID, t)
			n++
		}
	}
	showToast(styGreen.Render(fmt.Sprintf(T.TunnelsStoppedFmt, n)))
}

// validateTunnelHost — проверка хоста назначения теми же правилами, что и в ядре.
func validateTunnelHost(v string) error {
	if err := core.ValidateTunnel(core.TunnelLocal, 1, v, 1); err != nil {
		return errors.New(tunnelErrText(err))
	}
	return nil
}

// addTunnelFlow спрашивает вид проброса и порты, кладёт правило в конфиг и
// предлагает сразу поднять туннель.
func addTunnelFlow(st *Store, s *Server) {
	kind, ok := askSelectDesc(T.AskTunnelKind, []selOption[string]{
		opt(T.TunnelKindLocal, core.TunnelLocal),
		opt(T.TunnelKindRemote, core.TunnelRemote),
		opt(T.TunnelKindDynamic, core.TunnelDynamic),
	}, []string{T.TunnelDescLocal, T.TunnelDescRemote, T.TunnelDescDynamic})
	if !ok {
		return
	}

	portTitle := T.AskTunnelLocalPort
	if kind == core.TunnelDynamic {
		portTitle = T.AskTunnelSocksPort
	}
	lp, ok := askInput(portTitle, "", validatePort)
	if !ok {
		return
	}
	t := Tunnel{Kind: kind}
	t.LocalPort, _ = strconv.Atoi(strings.TrimSpace(lp))

	if kind != core.TunnelDynamic {
		host, ok := askInput(T.AskTunnelRemoteHost, "localhost", validateTunnelHost)
		if !ok {
			return
		}
		t.RemoteHost = strings.TrimSpace(host)
		rp, ok := askInput(T.AskTunnelRemotePort, strconv.Itoa(t.LocalPort), validatePort)
		if !ok {
			return
		}
		t.RemotePort, _ = strconv.Atoi(strings.TrimSpace(rp))
	}

	if err := t.Validate(); err != nil {
		showPage(T.TunnelsTitle, []string{styRed.Render(tunnelErrText(err))})
		return
	}
	if s.HasTunnel(t) {
		showToast(styYellow.Render(T.TunnelExists))
		return
	}
	s.Tunnels = append(s.Tunnels, t)
	st.MustSave()

	if yes, ok := askConfirm(T.AskTunnelStartNow, true); ok && yes {
		toggleTunnel(s, t)
	}
}

// deleteTunnelFlow удаляет правило. Работающий туннель сначала гасится —
// иначе процесс остался бы висеть без своей строки в конфиге.
func deleteTunnelFlow(st *Store, s *Server) {
	opts := make([]selOption[string], 0, len(s.Tunnels)+1)
	for i, t := range s.Tunnels {
		dot, _ := tunnelState(s.ID, t)
		opts = append(opts, opt(dot+"  "+t.Label(), strconv.Itoa(i)))
	}
	opts = append(opts, opt(T.Back, "back"))
	pick, ok := askSelect(T.TunnelDelete, opts)
	if !ok || pick == "back" {
		return
	}
	i, err := strconv.Atoi(pick)
	if err != nil || i < 0 || i >= len(s.Tunnels) {
		return
	}
	t := s.Tunnels[i]
	if yes, ok := askConfirm(fmt.Sprintf(T.AskDeleteTunnel, t.Label()), false); !ok || !yes {
		return
	}
	_ = tunnels.Stop(s.ID, t)
	s.RemoveTunnel(t)
	st.MustSave()
}

// tunnelLines — строки для панели статистики: правила сервера и их состояние.
func tunnelLines(s *Server, w int) []string {
	if len(s.Tunnels) == 0 {
		return nil
	}
	out := []string{""}
	for i, t := range s.Tunnels {
		label := ""
		if i == 0 {
			label = T.LblTunnels
		}
		dot, _ := tunnelState(s.ID, t)
		out = append(out, kv(label, dot+" "+styGrey.Render(clip(t.Label(), w-14))))
	}
	return out
}
