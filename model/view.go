package model

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Cell pixel size — detected at startup or using safe defaults.
var (
	detectedCellW int
	detectedCellH int
	detectOnce    sync.Once
)

// detectCellSize queries the terminal for actual cell pixel dimensions
// using CSI 16 t. Falls back to conservative defaults if unavailable.
func detectCellSize() (int, int) {
	detectOnce.Do(func() {
		detectedCellW, detectedCellH = queryCellSize()
		if detectedCellW == 0 || detectedCellH == 0 {
			// Conservative fallback — most terminals are between 7-10 x 14-20
			detectedCellW = 8
			detectedCellH = 18
		}
	})
	return detectedCellW, detectedCellH
}

// queryCellSize sends CSI 16 t and parses the response CSI 6 ; height ; width t
func queryCellSize() (int, int) {
	// Save terminal state
	fd := os.Stdin.Fd()
	_ = fd

	// Send query to stdout (terminal reads it and responds on stdin)
	// CSI 16 t = report cell size in pixels
	fmt.Fprint(os.Stdout, "\x1b[16t")
	os.Stdout.Sync()

	// Read response with short timeout
	// Response format: ESC [ 6 ; Ph ; Pw t
	buf := make([]byte, 64)
	done := make(chan int, 1)
	go func() {
		n, _ := os.Stdin.Read(buf)
		done <- n
	}()

	select {
	case n := <-done:
		resp := string(buf[:n])
		// Parse ESC [ 6 ; height ; width t
		if idx := strings.Index(resp, "\x1b[6;"); idx >= 0 {
			resp = resp[idx+4:]
			if end := strings.IndexByte(resp, 't'); end >= 0 {
				parts := strings.Split(resp[:end], ";")
				if len(parts) == 2 {
					h, _ := strconv.Atoi(parts[0])
					w, _ := strconv.Atoi(parts[1])
					if w > 0 && h > 0 {
						return w, h
					}
				}
			}
		}
	case <-time.After(100 * time.Millisecond):
		// Terminal doesn't support this query
	}

	return 0, 0
}

// View renders the current frame as sixel graphics.
func (m *Model) View() tea.View {
	v := tea.NewView("Loading...")
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	if m.Width == 0 || m.Height == 0 {
		return v
	}

	// Resolution scaling: manual --scale flag, with adaptive fallback
	renderScale := m.Settings.RenderScale
	if renderScale <= 0 {
		renderScale = 1.0
	}
	// Adaptive: reduce further if frames are slow
	if m.TotalFrameMs > 80 {
		renderScale *= 0.75
	}

	cellW, cellH := detectCellSize()

	// Allow manual override via --cell-size WxH
	if m.Settings.CellSize != "" {
		parts := strings.SplitN(m.Settings.CellSize, "x", 2)
		if len(parts) == 2 {
			if w, err := strconv.Atoi(parts[0]); err == nil && w > 0 {
				cellW = w
			}
			if h, err := strconv.Atoi(parts[1]); err == nil && h > 0 {
				cellH = h
			}
		}
	}

	// Convert terminal cell dimensions to pixel dimensions for sixel.
	// Subtract 1 row to prevent overflow/wrapping.
	pixW := int(float64(m.Width*cellW) * renderScale)
	pixH := int(float64((m.Height-1)*cellH) * renderScale)

	v.Content = m.RenderFrame(pixW, pixH)
	return v
}
