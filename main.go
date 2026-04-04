package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/acaudwell/gource-tui/config"
	"github.com/acaudwell/gource-tui/model"
	"github.com/acaudwell/gource-tui/parser"
)

func main() {
	cfg := config.DefaultSettings()

	if len(os.Args) > 1 {
		cfg.Path = os.Args[1]
	}

	p, err := parser.New(cfg.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	m := model.New(cfg, p)
	prog := tea.NewProgram(m)

	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
