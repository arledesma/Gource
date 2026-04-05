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

	// Resolution scaling: manual --scale flag, with adaptive fallback
	renderScale := m.Settings.RenderScale
	if renderScale <= 0 {
		renderScale = 1.0
	}
	if m.TotalFrameMs > 80 {
		renderScale *= 0.75
	}

	// Cell pixel size: use --cell-size override or reasonable default.
	// The CSI 16t query conflicts with Bubble Tea's input loop, so we
	// don't attempt it. Default of 8x16 is conservative (most common).
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

	// Render dimensions in pixels.
	// Subtract 2 rows: 1 for safety margin, 1 because Bubble Tea's cursor
	// positioning can add a line. The sixel output is further halved in
	// RenderFrame for encoding efficiency.
	pixW := int(float64(m.Width*cellW) * renderScale)
	pixH := int(float64((m.Height-2)*cellH) * renderScale)

	v.Content = m.RenderFrame(pixW, pixH)
	return v
}
