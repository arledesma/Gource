package model

import (
	"image/color"
	"path/filepath"
	"time"

	"github.com/arledesma/gource-tui/config"
)

// FileState represents the lifecycle state of a file.
type FileState int

const (
	FileActive   FileState = iota
	FileIdle               // no recent activity
	FileRemoving           // delete requested, fading out
	FileRemoved            // ready for cleanup
)

// File represents a tracked file in the visualization.
type File struct {
	Name       string
	Path       string
	Extension  string
	Color      color.Color
	Heat       float64   // 1.0 = just touched, decays toward 0
	State      FileState
	LastAction time.Time
	RemoveAt   time.Time
	ScreenX    float64 // computed by layout engine
	ScreenY    float64
}

// NewFile creates a file entity from a path.
func NewFile(path string, now time.Time) *File {
	ext := filepath.Ext(path)
	return &File{
		Name:       filepath.Base(path),
		Path:       path,
		Extension:  ext,
		Color:      config.ColorForExtension(ext),
		Heat:       1.0,
		State:      FileActive,
		LastAction: now,
	}
}

// Touch marks the file as recently modified, resetting its heat.
func (f *File) Touch(now time.Time, c color.Color) {
	f.Heat = 1.0
	f.State = FileActive
	f.LastAction = now
	if c != nil {
		f.Color = c
	}
	f.RemoveAt = time.Time{}
}

// MarkRemoved schedules the file for removal with a fade delay.
func (f *File) MarkRemoved(now time.Time, delay time.Duration) {
	f.State = FileRemoving
	f.RemoveAt = now.Add(delay)
	f.LastAction = now
	f.Heat = 1.0
}

// Update advances the file's state by dt seconds of simulation time.
func (f *File) Update(simNow time.Time, decayRate float64) {
	f.Heat *= decayRate
	if f.Heat < 0.01 {
		f.Heat = 0
	}

	switch f.State {
	case FileActive:
		if f.Heat == 0 {
			f.State = FileIdle
		}
	case FileRemoving:
		if !f.RemoveAt.IsZero() && simNow.After(f.RemoveAt) {
			f.State = FileRemoved
		}
	}
}
