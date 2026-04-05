package model

import (
	"image/color"
	"time"

	"github.com/arledesma/gource-tui/config"
)

const maxTrailLen = 12

// User represents a contributor in the visualization.
type User struct {
	Name           string
	Color          color.Color
	ActionCount    int
	LastAction     time.Time // simulation time of last action
	LastActionReal time.Time // wall clock time of last action
	Active         bool
	Body           PhysicsBody // position for rendering
	TargetFile     string      // path of file user is moving toward
	Trail          []Vec2      // recent positions for motion trail
}

// NewUser creates a user entity.
func NewUser(name string, simNow time.Time) *User {
	return &User{
		Name:           name,
		Color:          config.ColorForUser(name),
		LastAction:     simNow,
		LastActionReal: time.Now(),
		Active:         true,
	}
}

// Touch records a new action for this user.
func (u *User) Touch(simNow time.Time) {
	u.ActionCount++
	u.LastAction = simNow
	u.LastActionReal = time.Now()
	u.Active = true
}

// Update checks if the user should be marked inactive.
// Uses real (wall clock) time so users stay visible regardless of playback speed.
func (u *User) Update(idleSeconds float64) {
	if u.Active {
		elapsed := time.Since(u.LastActionReal).Seconds()
		if elapsed > idleSeconds {
			u.Active = false
		}
	}
}
