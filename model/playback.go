package model

import (
	"time"

	"github.com/acaudwell/gource-tui/parser"
)

// PlaybackState manages simulation time and commit queue.
type PlaybackState struct {
	CurrTime      time.Time
	StartTime     time.Time
	EndTime       time.Time
	DaysPerSecond float64
	Paused        bool
	Finished      bool
	CommitQueue   []parser.Commit
	TotalCommits  int
	Elapsed       int // commits processed so far
}

// NewPlayback creates a playback state with given speed.
func NewPlayback(daysPerSecond float64) *PlaybackState {
	return &PlaybackState{
		DaysPerSecond: daysPerSecond,
	}
}

// AdvanceTime moves the simulation clock forward by dt real seconds.
func (p *PlaybackState) AdvanceTime(dt float64) {
	if p.Paused || p.Finished {
		return
	}
	simSeconds := dt * p.DaysPerSecond * 86400
	p.CurrTime = p.CurrTime.Add(time.Duration(simSeconds * float64(time.Second)))
}

// EnqueueCommit adds a commit to the buffer.
func (p *PlaybackState) EnqueueCommit(c parser.Commit) {
	p.TotalCommits++

	if p.StartTime.IsZero() || c.Timestamp.Before(p.StartTime) {
		p.StartTime = c.Timestamp
	}
	if c.Timestamp.After(p.EndTime) {
		p.EndTime = c.Timestamp
	}

	// Set initial time from first commit
	if p.CurrTime.IsZero() {
		p.CurrTime = c.Timestamp
	}

	p.CommitQueue = append(p.CommitQueue, c)
}

// DueCommits returns all commits with timestamp <= currTime and removes them from the queue.
func (p *PlaybackState) DueCommits() []parser.Commit {
	var due []parser.Commit
	remaining := p.CommitQueue[:0]

	for _, c := range p.CommitQueue {
		if !c.Timestamp.After(p.CurrTime) {
			due = append(due, c)
			p.Elapsed++
		} else {
			remaining = append(remaining, c)
		}
	}
	p.CommitQueue = remaining
	return due
}

// Progress returns playback progress as 0.0 to 1.0.
func (p *PlaybackState) Progress() float64 {
	if p.StartTime.IsZero() || p.EndTime.IsZero() || p.StartTime.Equal(p.EndTime) {
		return 0
	}
	total := p.EndTime.Sub(p.StartTime).Seconds()
	elapsed := p.CurrTime.Sub(p.StartTime).Seconds()
	if elapsed < 0 {
		return 0
	}
	if elapsed > total {
		return 1.0
	}
	return elapsed / total
}

// SpeedUp doubles playback speed.
func (p *PlaybackState) SpeedUp() {
	p.DaysPerSecond *= 2
	if p.DaysPerSecond > 100 {
		p.DaysPerSecond = 100
	}
}

// SlowDown halves playback speed.
func (p *PlaybackState) SlowDown() {
	p.DaysPerSecond /= 2
	if p.DaysPerSecond < 0.01 {
		p.DaysPerSecond = 0.01
	}
}
