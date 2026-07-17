package main

import (
	"runtime/debug"

	"github.com/MrBenoit/unissh/internal/app"
	"github.com/MrBenoit/unissh/internal/core"
)

// version подставляется GoReleaser и `make build` через ldflags (-X main.version=...).
var version = "dev"

func main() {
	app.Run(core.ResolveVersion(version, moduleVersion()))
}

// moduleVersion возвращает версию модуля из вшитой информации о сборке. Её Go
// проставляет сам при `go install module@version` — там, где ldflags не
// применяются и var version остаётся «dev».
func moduleVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return bi.Main.Version
}
