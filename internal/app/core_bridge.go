package app

// Тонкие обёртки над пакетом internal/core: вся чистая логика живёт в core
// (и покрыта тестами на 90%+), а здесь — привязка к типам/i18n приложения.

import (
	"time"

	"github.com/MrBenoit/unissh/internal/core"
)

func shellQuote(s string) string                          { return core.ShellQuote(s) }
func parseJump(spec, def string) (string, string, string) { return core.ParseJump(spec, def) }
func hexToRGB(h string) (int, int, int)                   { return core.HexToRGB(h) }
func blendHex(a, b string, f float64) string              { return core.BlendHex(a, b, f) }
func sanitizeRemote(s string) string                      { return core.SanitizeRemote(s) }
func humanBytes(n float64) string                         { return core.HumanBytes(n, T.BytesUnits) }
func formatPing(d time.Duration) string                   { return core.FormatPing(d) }
func cpuPercentFromStat(s string) (float64, bool)         { return core.CPUPercentFromStat(s) }
func parseCPUStatLine(l string) []int64                   { return core.ParseCPUStatLine(l) }
func parseSections(raw string) map[string]string          { return core.ParseSections(raw) }
func expandPath(p string) string                          { return core.ExpandPath(p) }
func newID() string                                       { return core.NewID() }
func genPassword(n int) string                            { return core.GenPassword(n) }
func sshConfValue(v string) string                        { return core.SSHConfValue(v) }
func firstLine(s string) string                           { return core.FirstLine(s) }

// computeDeltas — обёртка над core.CPUNetDeltas для снимков монитора.
func computeDeltas(prev, cur *monSample) (cpu, rxRate, txRate float64, ok bool) {
	if prev == nil {
		return 0, 0, 0, false
	}
	return core.CPUNetDeltas(prev.stat, cur.stat, prev.rx, prev.tx, cur.rx, cur.tx,
		cur.at.Sub(prev.at).Seconds())
}
