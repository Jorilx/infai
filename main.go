package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dipankardas011/infai/db"
	"github.com/dipankardas011/infai/scanner"
	"github.com/dipankardas011/infai/tui"
)

func main() {
	database, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if theme, err := database.GetSetting("theme"); err == nil && theme != "" {
		tui.SetTheme(theme)
	}

	serverBin, _ := database.GetSetting("server_bin")
	if serverBin == "" {
		if path, err := exec.LookPath("llama-server"); err == nil {
			serverBin = path
		}
	}

	scanDirs, err := database.ListScanDirs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "list scan dirs: %v\n", err)
		os.Exit(1)
	}

	entries, err := scanner.Scan(scanDirs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan: %v\n", err)
		os.Exit(1)
	}
	for i := range entries {
		if err := database.UpsertModel(&entries[i]); err != nil {
			fmt.Fprintf(os.Stderr, "upsert model: %v\n", err)
			os.Exit(1)
		}
	}

	app := tui.NewApp(database, serverBin, scanDirs, entries, 80, 24)
	p := tea.NewProgram(&app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}
