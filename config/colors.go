package config

import (
	"encoding/hex"
	"hash/fnv"
	"image/color"
)

func hexColor(s string) color.RGBA {
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	b, _ := hex.DecodeString(s)
	if len(b) >= 3 {
		return color.RGBA{R: b[0], G: b[1], B: b[2], A: 255}
	}
	return color.RGBA{R: 170, G: 170, B: 170, A: 255}
}

// ExtensionColors maps file extensions to display colors.
var ExtensionColors = map[string]color.Color{
	// Source code
	".go":   hexColor("#00ADD8"),
	".rs":   hexColor("#DEA584"),
	".py":   hexColor("#3572A5"),
	".js":   hexColor("#F7DF1E"),
	".ts":   hexColor("#3178C6"),
	".jsx":  hexColor("#61DAFB"),
	".tsx":  hexColor("#61DAFB"),
	".rb":   hexColor("#CC342D"),
	".java": hexColor("#B07219"),
	".c":    hexColor("#555555"),
	".cpp":  hexColor("#F34B7D"),
	".h":    hexColor("#75507B"),
	".hpp":  hexColor("#F34B7D"),
	".cs":   hexColor("#68217A"),
	".php":  hexColor("#4F5D95"),
	".lua":  hexColor("#000080"),
	".sh":   hexColor("#89E051"),
	".bash": hexColor("#89E051"),

	// Web
	".html": hexColor("#E34C26"),
	".css":  hexColor("#563D7C"),
	".scss": hexColor("#C6538C"),
	".vue":  hexColor("#41B883"),
	".svlt": hexColor("#FF3E00"),

	// Data / Config
	".json": hexColor("#A0A0A0"),
	".yaml": hexColor("#CB171E"),
	".yml":  hexColor("#CB171E"),
	".toml": hexColor("#9C4121"),
	".xml":  hexColor("#0060AC"),
	".sql":  hexColor("#E38C00"),
	".csv":  hexColor("#237346"),

	// Docs
	".md":   hexColor("#083FA1"),
	".txt":  hexColor("#888888"),
	".rst":  hexColor("#141414"),
	".adoc": hexColor("#E40046"),

	// Build / DevOps
	".mk":         hexColor("#427819"),
	".cmake":      hexColor("#DA3434"),
	".dockerfile": hexColor("#384D54"),
	".tf":         hexColor("#5C4EE5"),
}

// DefaultFileColor is used when the extension is not in the map.
var DefaultFileColor color.Color = hexColor("#AAAAAA")

// ColorForExtension returns the color for a file extension.
func ColorForExtension(ext string) color.Color {
	if c, ok := ExtensionColors[ext]; ok {
		return c
	}
	return DefaultFileColor
}

// UserColors is a palette for assigning consistent colors to users.
var UserColors = []color.Color{
	hexColor("#E06C75"),
	hexColor("#98C379"),
	hexColor("#E5C07B"),
	hexColor("#61AFEF"),
	hexColor("#C678DD"),
	hexColor("#56B6C2"),
	hexColor("#BE5046"),
	hexColor("#D19A66"),
	hexColor("#7EC8E3"),
	hexColor("#C3E88D"),
	hexColor("#F78C6C"),
	hexColor("#89DDFF"),
	hexColor("#FFCB6B"),
	hexColor("#F07178"),
	hexColor("#82AAFF"),
	hexColor("#B2CCD6"),
}

// ColorForUser returns a consistent color for a username via hashing.
func ColorForUser(name string) color.Color {
	h := fnv.New32a()
	h.Write([]byte(name))
	return UserColors[h.Sum32()%uint32(len(UserColors))]
}
