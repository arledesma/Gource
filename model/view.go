package model

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"

	"github.com/acaudwell/gource-tui/config"
)

var (
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))

	actionColors = map[string]color.Color{
		"A": lipgloss.Color("#98C379"), // green
		"M": lipgloss.Color("#E5C07B"), // yellow
		"D": lipgloss.Color("#E06C75"), // red
	}
)

// View renders the full terminal UI.
func (m *Model) View() tea.View {
	v := tea.NewView("Loading...")
	v.AltScreen = true

	if m.Width == 0 || m.Height == 0 {
		return v
	}

	statusHeight := 1
	contentHeight := m.Height - statusHeight - 2

	treeWidth := m.Width * 55 / 100
	logWidth := m.Width - treeWidth - 3

	treePanel := m.renderTreePanel(treeWidth, contentHeight)
	logPanel := m.renderActivityPanel(logWidth, contentHeight)
	statusBar := m.renderStatusBar(m.Width)

	treePanelStyled := lipgloss.NewStyle().
		Width(treeWidth).
		Height(contentHeight).
		Render(treePanel)

	var sepLines []string
	for range contentHeight {
		sepLines = append(sepLines, borderStyle.Render("│"))
	}
	separator := strings.Join(sepLines, "\n")

	logPanelStyled := lipgloss.NewStyle().
		Width(logWidth).
		Height(contentHeight).
		Render(logPanel)

	content := lipgloss.JoinHorizontal(lipgloss.Top, treePanelStyled, separator, logPanelStyled)

	v.Content = lipgloss.JoinVertical(lipgloss.Left, content, statusBar)
	return v
}

func (m *Model) renderTreePanel(width, height int) string {
	if m.Root == nil || (len(m.Root.Children) == 0 && len(m.Root.Files) == 0) {
		return dimStyle.Render("Waiting for commits...")
	}

	t := m.buildTree(m.Root)
	return t.String()
}

func (m *Model) buildTree(node *DirNode) *tree.Tree {
	var t *tree.Tree
	if node.Name == "" {
		t = tree.New()
	} else {
		style := m.dirStyle(node)
		t = tree.Root(style.Render(node.Name + "/"))
	}

	t.Enumerator(tree.DefaultEnumerator)

	for _, child := range node.Children {
		subtree := m.buildTree(child)
		t.Child(subtree)
	}

	for _, name := range node.SortedFileNames() {
		f := node.Files[name]
		label := m.fileLabel(f)
		t.Child(label)
	}

	return t
}

func (m *Model) dirStyle(node *DirNode) lipgloss.Style {
	style := lipgloss.NewStyle().Bold(true)

	maxHeat := 0.0
	for _, f := range node.Files {
		if f.Heat > maxHeat {
			maxHeat = f.Heat
		}
	}
	for _, child := range node.Children {
		for _, f := range child.AllFiles() {
			if f.Heat > maxHeat {
				maxHeat = f.Heat
			}
		}
	}

	if maxHeat > 0.3 {
		style = style.Foreground(lipgloss.Color("#FFFFFF"))
	} else {
		style = style.Foreground(lipgloss.Color("#888888"))
	}

	return style
}

func (m *Model) fileLabel(f *File) string {
	nameStyle := lipgloss.NewStyle().Foreground(f.Color)

	if f.Heat > 0.5 {
		nameStyle = nameStyle.Bold(true)
	}
	if f.State == FileRemoving {
		nameStyle = nameStyle.Strikethrough(true).Foreground(lipgloss.Color("#E06C75"))
	}
	if f.State == FileIdle {
		nameStyle = nameStyle.Foreground(lipgloss.Color("#555555"))
	}

	label := nameStyle.Render(f.Name)

	if f.Heat > 0.05 {
		barLen := int(math.Ceil(f.Heat * 8))
		bar := strings.Repeat("█", barLen)
		barStyle := lipgloss.NewStyle().Foreground(heatColor(f.Heat))
		label += " " + barStyle.Render(bar)
	}

	return label
}

func heatColor(heat float64) color.Color {
	if heat > 0.8 {
		return lipgloss.Color("#FFFFFF")
	}
	if heat > 0.5 {
		return lipgloss.Color("#FFAA44")
	}
	if heat > 0.2 {
		return lipgloss.Color("#CC6622")
	}
	return lipgloss.Color("#884411")
}

func (m *Model) renderActivityPanel(width, height int) string {
	if len(m.Activity) == 0 {
		return dimStyle.Render("No activity yet...")
	}

	var lines []string
	start := len(m.Activity) - height
	if start < 0 {
		start = 0
	}

	maxUser := 12
	maxFile := width - maxUser - 6
	if maxFile < 10 {
		maxFile = 10
	}

	for _, entry := range m.Activity[start:] {
		user := entry.Username
		if len(user) > maxUser {
			user = user[:maxUser-1] + "…"
		}

		file := filepath.Base(entry.FilePath)
		if len(file) > maxFile {
			file = file[:maxFile-1] + "…"
		}

		userColor := config.ColorForUser(entry.Username)
		userStyled := lipgloss.NewStyle().
			Foreground(userColor).
			Width(maxUser).
			Render(user)

		ac, ok := actionColors[entry.Action]
		if !ok {
			ac = lipgloss.Color("#888888")
		}
		actionStyled := lipgloss.NewStyle().
			Foreground(ac).
			Render(entry.Action)

		fileStyled := dimStyle.Render(file)

		lines = append(lines, fmt.Sprintf("%s %s %s", userStyled, actionStyled, fileStyled))
	}

	return strings.Join(lines, "\n")
}

func (m *Model) renderStatusBar(width int) string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#CCCCCC")).
		Width(width)

	dateStr := ""
	if !m.Playback.CurrTime.IsZero() {
		dateStr = m.Playback.CurrTime.Format("2006-01-02 15:04")
	}

	progress := m.Playback.Progress()
	barWidth := 20
	filled := int(progress * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	activeUsers := 0
	for _, u := range m.Users {
		if u.Active {
			activeUsers++
		}
	}

	speedStr := fmt.Sprintf("%.1f d/s", m.Playback.DaysPerSecond)

	pauseStr := ""
	if m.Playback.Paused {
		pauseStr = " PAUSED"
	}
	if m.Playback.Finished {
		pauseStr = " END"
	}

	status := fmt.Sprintf(" %s  %s %3.0f%%  %d files  %d users  %s%s",
		dateStr, bar, progress*100, len(m.Files), activeUsers, speedStr, pauseStr)

	return style.Render(status)
}
