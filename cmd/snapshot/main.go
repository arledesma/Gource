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
	"github.com/fogleman/gg"
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
	cfg.DaysPerSecond = 10 // fast for snapshot

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
	target := 0.2
	dt := 1.0 / 30.0
	for m.Playback.Progress() < target {
		m.Playback.AdvanceTime(dt)
		for _, commit := range m.Playback.DueCommits() {
			processCommitDirect(m, commit, cfg)
		}
		model.UpdateLayout(m.Root, dt)
	}

	fmt.Printf("At %.0f%% progress, %d files, %d users\n",
		m.Playback.Progress()*100, len(m.Files), len(m.Users))

	// Render to image
	width := 960
	height := 640
	frame := m.RenderFrame(width, height)
	_ = frame // sixel string

	// Also render directly to PNG for inspection
	dc := gg.NewContext(width, height)
	dc.SetRGB(0.05, 0.05, 0.08)
	dc.Clear()

	// Use the rendered image (re-render without sixel)
	img := renderToPNG(m, width, height)
	f, err := os.Create("snapshot.png")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	png.Encode(f, img.Image())
	fmt.Println("Saved snapshot.png")
}

func renderToPNG(m *model.Model, width, height int) *gg.Context {
	dc := gg.NewContext(width, height)
	dc.SetRGB(0.05, 0.05, 0.08)
	dc.Clear()

	cx := float64(width) / 2
	cy := float64(height) / 2
	ox := cx - m.Root.Body.Pos.X
	oy := cy - m.Root.Body.Pos.Y

	// Draw edges
	drawEdgesRecursive(dc, m.Root, ox, oy)

	// Draw file nodes
	drawFilesRecursive(dc, m.Root, ox, oy)

	// Draw dir nodes
	drawDirsRecursive(dc, m.Root, ox, oy)

	// Draw labels
	drawLabelsRecursive(dc, m.Root, ox, oy)

	return dc
}

func drawEdgesRecursive(dc *gg.Context, node *model.DirNode, ox, oy float64) {
	for _, child := range node.Children {
		dc.SetRGBA(0.3, 0.4, 0.5, 0.3)
		dc.SetLineWidth(1.5)
		dc.DrawLine(
			node.Body.Pos.X+ox, node.Body.Pos.Y+oy,
			child.Body.Pos.X+ox, child.Body.Pos.Y+oy,
		)
		dc.Stroke()
		drawEdgesRecursive(dc, child, ox, oy)
	}

	for _, name := range node.SortedFileNames() {
		f := node.Files[name]
		dc.SetRGBA(0.4, 0.4, 0.4, 0.1)
		dc.SetLineWidth(0.5)
		dc.DrawLine(
			node.Body.Pos.X+ox, node.Body.Pos.Y+oy,
			f.ScreenX+ox, f.ScreenY+oy,
		)
		dc.Stroke()
	}
}

func drawFilesRecursive(dc *gg.Context, node *model.DirNode, ox, oy float64) {
	for _, name := range node.SortedFileNames() {
		f := node.Files[name]
		r, g, b, _ := f.Color.RGBA()
		rf := float64(r) / 0xFFFF
		gf := float64(g) / 0xFFFF
		bf := float64(b) / 0xFFFF

		alpha := 0.3 + f.Heat*0.7
		dc.SetRGBA(rf, gf, bf, alpha)
		dc.DrawCircle(f.ScreenX+ox, f.ScreenY+oy, 5)
		dc.Fill()
	}

	for _, child := range node.Children {
		drawFilesRecursive(dc, child, ox, oy)
	}
}

func drawDirsRecursive(dc *gg.Context, node *model.DirNode, ox, oy float64) {
	if node.Name != "" {
		dc.SetRGBA(0.3, 0.5, 0.8, 0.6)
		dc.DrawCircle(node.Body.Pos.X+ox, node.Body.Pos.Y+oy, 8)
		dc.Fill()
	}
	for _, child := range node.Children {
		drawDirsRecursive(dc, child, ox, oy)
	}
}

func drawLabelsRecursive(dc *gg.Context, node *model.DirNode, ox, oy float64) {
	if node.Name != "" {
		dc.SetRGBA(0.7, 0.8, 1.0, 0.8)
		dc.DrawStringAnchored(node.Name, node.Body.Pos.X+ox, node.Body.Pos.Y+oy-12, 0.5, 0.5)
	}
	for _, child := range node.Children {
		drawLabelsRecursive(dc, child, ox, oy)
	}
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
