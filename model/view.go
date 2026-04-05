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

	var pixW, pixH int

	if m.Settings.CellSize != "" {
		// Explicit override — highest priority
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
	} else if m.Settings.DetectedPixW > 0 && m.Settings.DetectedPixH > 0 {
		// Detected window pixel size — use directly
		pixW = m.Settings.DetectedPixW
		pixH = m.Settings.DetectedPixH
	} else if m.Settings.DetectedCellW > 0 && m.Settings.DetectedCellH > 0 {
		// Detected cell size — compute from cells
		pixW = m.Width * m.Settings.DetectedCellW
		pixH = (m.Height - 1) * m.Settings.DetectedCellH
	} else {
		// Fallback: one sixel band per row (guaranteed to fit)
		pixW = m.Width * 8
		pixH = (m.Height - 2) * 6
	}

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
