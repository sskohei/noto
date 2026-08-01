package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"noto/internal/app"
	"noto/internal/config"
	"noto/internal/index"
	"noto/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Ensure the notes directory exists before the first index sync: on a
	// fresh install nothing has created it yet (storage.Write normally
	// does so lazily, on the first note).
	if err := os.MkdirAll(cfg.NotesDir, 0o755); err != nil {
		return fmt.Errorf("create notes dir: %w", err)
	}

	idx, err := index.Open(app.IndexPath(cfg))
	if err != nil {
		return fmt.Errorf("open index: %w", err)
	}
	defer idx.Close()

	if err := idx.Sync(cfg.NotesDir); err != nil {
		return fmt.Errorf("sync index: %w", err)
	}

	_, err = tea.NewProgram(ui.New(cfg, idx), tea.WithAltScreen()).Run()
	if err != nil {
		return fmt.Errorf("run program: %w", err)
	}
	return nil
}
