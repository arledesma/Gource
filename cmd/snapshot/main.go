package main

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"time"

	"github.com/acaudwell/gource-tui/config"
	"github.com/acaudwell/gource-tui/model"
	"github.com/acaudwell/gource-tui/parser"
)

// Standalone tool to render a single frame as PNG for testing.
func main() {
	path := "."
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	p, err := parser.New(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := config.DefaultSettings()
	cfg.Path = path
	cfg.DaysPerSecond = 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := p.Stream(ctx)

	m := &model.Model{
		Settings: cfg,
		Root:     model.NewDirNode("", ""),
		Files:    make(map[string]*model.File),
		Users:    make(map[string]*model.User),
		Playback: model.NewPlayback(cfg.DaysPerSecond),
		Width:    120,
		Height:   40,
	}

	// Read all commits
	count := 0
	for c := range ch {
		m.Playback.EnqueueCommit(c)
		count++
	}
	fmt.Printf("Read %d commits\n", count)

	// Advance through ~20% of history
	dt := 1.0 / 30.0
	for m.Playback.Progress() < 0.2 {
		m.Playback.AdvanceTime(dt)
		for _, commit := range m.Playback.DueCommits() {
			processCommitDirect(m, commit, cfg)
		}
		model.UpdateLayout(m.Root, dt)
	}

	fmt.Printf("At %.0f%% progress, %d files, %d users\n",
		m.Playback.Progress()*100, len(m.Files), len(m.Users))

	// Render using the same pipeline as the TUI (includes camera auto-fit)
	width := 960
	height := 640
	sixelStr := m.RenderFrame(width, height)

	// The RenderFrame returns sixel — we need the image directly for PNG.
	// Use RenderImage instead if available, or just save the sixel test.
	// For now, write a PNG by re-rendering through the model.
	_ = sixelStr

	// Render to PNG via the model's rendering (use gg directly with camera)
	img := m.RenderToPNG(width, height)
	f, err := os.Create("snapshot.png")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	png.Encode(f, img)
	fmt.Println("Saved snapshot.png")
}

func processCommitDirect(m *model.Model, c parser.Commit, cfg config.Settings) {
	user, exists := m.Users[c.Username]
	if !exists {
		user = model.NewUser(c.Username, c.Timestamp)
		m.Users[c.Username] = user
	}
	user.Touch(c.Timestamp)

	for _, cf := range c.Files {
		f, exists := m.Files[cf.Path]
		if !exists {
			f = m.Root.InsertFile(cf.Path, c.Timestamp)
			m.Files[cf.Path] = f
		}

		switch cf.Action {
		case "D":
			f.MarkRemoved(c.Timestamp, time.Duration(cfg.FileIdleTime)*time.Second)
		default:
			f.Touch(c.Timestamp, f.Color)
		}
	}
}
