package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/angristan/hue-tui/internal/config"
	"github.com/angristan/hue-tui/internal/tui"
)

func main() {
	// Check for demo mode
	demoMode := os.Getenv("HUE_DEMO") != ""
	for _, arg := range os.Args[1:] {
		if arg == "--demo" || arg == "-demo" {
			demoMode = true
			break
		}
	}

	if demoMode {
		fmt.Fprintln(os.Stderr, "[hue] Demo mode enabled")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create and run the application
	model := tui.NewModel(cfg, demoMode)
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running app: %v\n", err)
		os.Exit(1)
	}
}
