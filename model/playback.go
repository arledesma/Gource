package model

import (
	"time"

	"github.com/arledesma/gource-tui/parser"
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
	AllCommits    []parser.Commit // all commits ever received (for seeking)
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
// Does nothing until at least one commit has set the initial time.
func (p *PlaybackState) AdvanceTime(dt float64) {
	if p.Paused || p.Finished || p.TotalCommits == 0 {
		return
	}
	simSeconds := dt * p.DaysPerSecond * 86400
	p.CurrTime = p.CurrTime.Add(time.Duration(simSeconds * float64(time.Second)))
}

// EnqueueCommit adds a commit to the buffer.
func (p *PlaybackState) EnqueueCommit(c parser.Commit) {
	p.TotalCommits++
	p.AllCommits = append(p.AllCommits, c)

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

// SeekTo jumps playback to a specific progress (0.0-1.0).
// Rebuilds the commit queue from AllCommits.
func (p *PlaybackState) SeekTo(progress float64) {
	if p.StartTime.IsZero() || p.EndTime.IsZero() {
		return
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	total := p.EndTime.Sub(p.StartTime)
	p.CurrTime = p.StartTime.Add(time.Duration(float64(total) * progress))
	p.Finished = false

	// Rebuild commit queue: all commits after CurrTime
	p.CommitQueue = p.CommitQueue[:0]
	p.Elapsed = 0
	for _, c := range p.AllCommits {
		if c.Timestamp.After(p.CurrTime) {
			p.CommitQueue = append(p.CommitQueue, c)
		} else {
			p.Elapsed++
		}
	}
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
