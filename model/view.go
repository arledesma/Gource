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

	// Determine cell pixel size. Priority:
	// 1. --cell-size flag (explicit override)
	// 2. Detected at startup via CSI 16t
	// 3. Fallback to 8x16
	cellW, cellH := 8, 16
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

	// Internal render dimensions
	pixW := int(float64(m.Width*cellW) * renderScale)
	pixH := int(float64(m.Height*cellH) * renderScale)

	// Pass actual terminal pixel height for sixel clamping in RenderFrame.
	// If we detected it, use it. Otherwise estimate from cells.
	if m.Settings.DetectedPixH > 0 {
		m.termPixH = m.Settings.DetectedPixH
	} else {
		m.termPixH = m.Height * cellH
	}
	m.termRows = m.Height
	m.termCols = m.Width

	v.Content = m.RenderFrame(pixW, pixH)
	return v
}
