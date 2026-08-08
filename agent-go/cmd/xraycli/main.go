package main

import (
	"os"
	"v2ray-agent/internal/app"
)

func main() {
	// If CLI arguments provided, handle them directly
	if app.HandleCLI(os.Args) {
		return
	}

	// Otherwise, enter the interactive TUI Dashboard
	app.ShowDashboard()
}
