package main

import "github.com/MrBenoit/unissh/internal/app"

// version подставляется GoReleaser через ldflags (-X main.version=...).
var version = "dev"

func main() {
	app.Run(version)
}
