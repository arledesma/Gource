package model

import (
	tea "charm.land/bubbletea/v2"
)

// Approximate pixels per terminal cell.
// Most terminals use roughly 8x16 pixel cells; sixel is rendered at pixel level.
const (
	cellPixelW = 8
	cellPixelH = 16
)

// View renders the current frame as sixel graphics.
func (m *Model) View() tea.View {
	v := tea.NewView("Loading...")
	v.AltScreen = true

	if m.Width == 0 || m.Height == 0 {
		return v
	}

	// Convert terminal cell dimensions to pixel dimensions for sixel
	pixW := m.Width * cellPixelW
	pixH := m.Height * cellPixelH

	v.Content = m.RenderFrame(pixW, pixH)
	return v
}
