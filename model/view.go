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

	// Resolution scaling
	renderScale := m.Settings.RenderScale
	if renderScale <= 0 {
		renderScale = 1.0
	}
	if m.TotalFrameMs > 80 {
		renderScale *= 0.75
	}

	// Determine the sixel output pixel dimensions directly.
	// Render at this size — no oversized render + downscale.
	var pixW, pixH int

	if m.Settings.DetectedPixW > 0 && m.Settings.DetectedPixH > 0 {
		// Best case: we know exact terminal pixel size
		pixW = m.Settings.DetectedPixW
		pixH = m.Settings.DetectedPixH - 6 // one sixel band margin
	} else if m.Settings.DetectedCellW > 0 && m.Settings.DetectedCellH > 0 {
		// We know cell size but not window size
		pixW = m.Width * m.Settings.DetectedCellW
		pixH = (m.Height - 1) * m.Settings.DetectedCellH
	} else {
		// Fallback: conservative cell size
		pixW = m.Width * 8
		pixH = (m.Height - 1) * 12 // 12px per row is safe minimum
	}

	// --cell-size override replaces detection
	if m.Settings.CellSize != "" {
		cellW, cellH := 8, 16
		parts := strings.SplitN(m.Settings.CellSize, "x", 2)
		if len(parts) == 2 {
			if w, err := strconv.Atoi(parts[0]); err == nil && w > 0 {
				cellW = w
			}
			if h, err := strconv.Atoi(parts[1]); err == nil && h > 0 {
				cellH = h
			}
		}
		pixW = m.Width * cellW
		pixH = (m.Height - 1) * cellH
	}

	// Apply render scale
	pixW = int(float64(pixW) * renderScale)
	pixH = int(float64(pixH) * renderScale)

	v.Content = m.RenderFrame(pixW, pixH)
	return v
}
