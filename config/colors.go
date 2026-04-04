package config

import (
	"hash/fnv"
	"image/color"

	"charm.land/lipgloss/v2"
)

// ExtensionColors maps file extensions to display colors.
var ExtensionColors = map[string]color.Color{
	// Source code
	".go":   lipgloss.Color("#00ADD8"),
	".rs":   lipgloss.Color("#DEA584"),
	".py":   lipgloss.Color("#3572A5"),
	".js":   lipgloss.Color("#F7DF1E"),
	".ts":   lipgloss.Color("#3178C6"),
	".jsx":  lipgloss.Color("#61DAFB"),
	".tsx":  lipgloss.Color("#61DAFB"),
	".rb":   lipgloss.Color("#CC342D"),
	".java": lipgloss.Color("#B07219"),
	".c":    lipgloss.Color("#555555"),
	".cpp":  lipgloss.Color("#F34B7D"),
	".h":    lipgloss.Color("#75507B"),
	".hpp":  lipgloss.Color("#F34B7D"),
	".cs":   lipgloss.Color("#68217A"),
	".php":  lipgloss.Color("#4F5D95"),
	".lua":  lipgloss.Color("#000080"),
	".sh":   lipgloss.Color("#89E051"),
	".bash": lipgloss.Color("#89E051"),

	// Web
	".html": lipgloss.Color("#E34C26"),
	".css":  lipgloss.Color("#563D7C"),
	".scss": lipgloss.Color("#C6538C"),
	".vue":  lipgloss.Color("#41B883"),
	".svlt": lipgloss.Color("#FF3E00"),

	// Data / Config
	".json": lipgloss.Color("#A0A0A0"),
	".yaml": lipgloss.Color("#CB171E"),
	".yml":  lipgloss.Color("#CB171E"),
	".toml": lipgloss.Color("#9C4121"),
	".xml":  lipgloss.Color("#0060AC"),
	".sql":  lipgloss.Color("#E38C00"),
	".csv":  lipgloss.Color("#237346"),

	// Docs
	".md":   lipgloss.Color("#083FA1"),
	".txt":  lipgloss.Color("#888888"),
	".rst":  lipgloss.Color("#141414"),
	".adoc": lipgloss.Color("#E40046"),

	// Build / DevOps
	".mk":         lipgloss.Color("#427819"),
	".cmake":      lipgloss.Color("#DA3434"),
	".dockerfile": lipgloss.Color("#384D54"),
	".tf":         lipgloss.Color("#5C4EE5"),
}

// DefaultFileColor is used when the extension is not in the map.
var DefaultFileColor = lipgloss.Color("#AAAAAA")

// ColorForExtension returns the color for a file extension.
func ColorForExtension(ext string) color.Color {
	if c, ok := ExtensionColors[ext]; ok {
		return c
	}
	return DefaultFileColor
}

// UserColors is a palette for assigning consistent colors to users.
var UserColors = []color.Color{
	lipgloss.Color("#E06C75"),
	lipgloss.Color("#98C379"),
	lipgloss.Color("#E5C07B"),
	lipgloss.Color("#61AFEF"),
	lipgloss.Color("#C678DD"),
	lipgloss.Color("#56B6C2"),
	lipgloss.Color("#BE5046"),
	lipgloss.Color("#D19A66"),
	lipgloss.Color("#7EC8E3"),
	lipgloss.Color("#C3E88D"),
	lipgloss.Color("#F78C6C"),
	lipgloss.Color("#89DDFF"),
	lipgloss.Color("#FFCB6B"),
	lipgloss.Color("#F07178"),
	lipgloss.Color("#82AAFF"),
	lipgloss.Color("#B2CCD6"),
}

// ColorForUser returns a consistent color for a username via hashing.
func ColorForUser(name string) color.Color {
	h := fnv.New32a()
	h.Write([]byte(name))
	return UserColors[h.Sum32()%uint32(len(UserColors))]
}
