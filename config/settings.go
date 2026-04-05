package config

import "time"

// Settings holds all configuration for gource-tui.
type Settings struct {
	Path           string
	DaysPerSecond  float64
	AutoSkip       float64 // skip idle periods longer than this (days of sim time)
	FileIdleTime   float64 // seconds before idle files fade
	UserIdleTime   float64 // seconds before idle users disappear
	MaxFiles       int     // max files to display (0 = unlimited)
	StartDate      time.Time
	StopDate       time.Time
	TickRate       time.Duration
	Loop           bool
	NoBloom        bool
	HideFilenames  bool
	HideDirnames   bool
	HideUsernames  bool
	HideProgress   bool
	HideDate       bool
	UserFilter     string
	FileFilter     string
	Background     string // hex color for background
	Debug          bool
	CellSize       string  // WxH override for cell pixel dimensions (e.g. "8x18")
	Theme          string  // color theme name
	RenderScale    float64 // output resolution scale (0.5 = half res, 1.0 = full)

	// Detected at startup (not CLI flags)
	DetectedCellW int
	DetectedCellH int
	DetectedPixW  int
	DetectedPixH  int
}

// DefaultSettings returns sensible defaults.
func DefaultSettings() Settings {
	return Settings{
		Path:          ".",
		DaysPerSecond: 0.5,
		AutoSkip:      3.0,
		FileIdleTime:  60.0,
		UserIdleTime:  10.0,
		TickRate:      time.Second / 30,
		RenderScale:   1.0,
	}
}
