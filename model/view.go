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

	// Cell pixel size. Default 8x14 is deliberately conservative —
	// it's better to be slightly small (sharp) than slightly too big (wrapping).
	// Users can tune up with --cell-size if they want larger output.
	cellW := 8
	cellH := 14

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

	// Render directly at output size. Subtract 2 rows for margin.
	scale := m.Settings.RenderScale
	if scale <= 0 {
		scale = 1.0
	}
	pixW := int(float64(m.Width*cellW) * scale)
	pixH := int(float64((m.Height-2)*cellH) * scale)

	v.Content = m.RenderFrame(pixW, pixH)
	return v
}
