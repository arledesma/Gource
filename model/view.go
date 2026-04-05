package model

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// View renders the current frame as sixel graphics.
func (m *Model) View() tea.View {
	v := tea.NewView("Loading...")
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	if m.Width == 0 || m.Height == 0 {
		return v
	}

	// Determine cell pixel size. Priority:
	// 1. --cell-size flag
	// 2. Detected via CSI 16t at startup
	// 3. Fallback 8x6 (one sixel band — guaranteed to fit)
	cellW, cellH := 8, 6

	if m.Settings.DetectedCellW > 0 && m.Settings.DetectedCellH > 0 {
		cellW = m.Settings.DetectedCellW
		cellH = m.Settings.DetectedCellH
	}

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

	// Compute pixel dimensions from current cell count × cell pixel size.
	// Width/Height update on resize via WindowSizeMsg, so this adapts
	// automatically. Subtract 1 row for safety margin.
	pixW := m.Width * cellW
	pixH := (m.Height - 1) * cellH

	// Apply render scale
	scale := m.Settings.RenderScale
	if scale <= 0 {
		scale = 1.0
	}
	pixW = int(float64(pixW) * scale)
	pixH = int(float64(pixH) * scale)

	v.Content = m.RenderFrame(pixW, pixH)
	return v
}
