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

	// Cell pixel size for internal rendering resolution.
	// This controls how many pixels we generate before sixel encoding.
	// The sixel output is further halved in RenderFrame.
	// Default 8x16 is conservative — override with --cell-size if needed.
	cellW := 8
	cellH := 16
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

	// Internal render dimensions (full res for quality).
	pixW := int(float64(m.Width*cellW) * renderScale)
	pixH := int(float64(m.Height*cellH) * renderScale)

	// The sixel output size (after halving in RenderFrame) must fit
	// within the terminal. We pass the terminal dimensions so
	// RenderFrame can clamp the sixel output size.
	m.termRows = m.Height
	m.termCols = m.Width

	v.Content = m.RenderFrame(pixW, pixH)
	return v
}
