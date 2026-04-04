package model

import (
	"image/color"
	"time"

	"github.com/acaudwell/gource-tui/config"
)

// User represents a contributor in the visualization.
type User struct {
	Name        string
	Color       color.Color
	ActionCount int
	LastAction  time.Time
	Active      bool
	Body        PhysicsBody // position for rendering
	TargetFile  string      // path of file user is moving toward
}

// NewUser creates a user entity.
func NewUser(name string, now time.Time) *User {
	return &User{
		Name:       name,
		Color:      config.ColorForUser(name),
		LastAction: now,
		Active:     true,
	}
}

// Touch records a new action for this user.
func (u *User) Touch(now time.Time) {
	u.ActionCount++
	u.LastAction = now
	u.Active = true
}

// Update checks if the user should be marked inactive.
func (u *User) Update(simNow time.Time, idleTimeout time.Duration) {
	if u.Active && simNow.Sub(u.LastAction) > idleTimeout {
		u.Active = false
	}
}
