package model

import (
	"context"
	"image/color"
	"math"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/acaudwell/gource-tui/config"
	"github.com/acaudwell/gource-tui/parser"
)

// ActivityEntry represents a line in the activity log.
type ActivityEntry struct {
	Username string
	Action   string
	FilePath string
	Time     time.Time
}

// Model is the top-level Bubble Tea model.
type Model struct {
	Settings config.Settings
	Root     *DirNode
	Files    map[string]*File
	Users    map[string]*User
	Actions  []*Action
	Playback *PlaybackState
	Activity []ActivityEntry

	commitCh  <-chan parser.Commit
	cancelPar context.CancelFunc
	chanDone  bool

	Width  int
	Height int
}

type tickMsg time.Time

type commitBatchMsg []parser.Commit

// New creates a new app model.
func New(cfg config.Settings, p parser.Parser) *Model {
	ctx, cancel := context.WithCancel(context.Background())
	ch := p.Stream(ctx)

	return &Model{
		Settings:  cfg,
		Root:      NewDirNode("", ""),
		Files:     make(map[string]*File),
		Users:     make(map[string]*User),
		Playback:  NewPlayback(cfg.DaysPerSecond),
		commitCh:  ch,
		cancelPar: cancel,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.tickCmd(),
		m.drainCommits(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case commitBatchMsg:
		for _, c := range msg {
			m.Playback.EnqueueCommit(c)
		}
		if len(msg) > 0 {
			return m, m.drainCommits()
		}
		m.chanDone = true
		return m, nil

	case tickMsg:
		m.tick()
		return m, m.tickCmd()
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.cancelPar()
		return m, tea.Quit
	case " ":
		m.Playback.Paused = !m.Playback.Paused
	case "+", "=":
		m.Playback.SpeedUp()
	case "-":
		m.Playback.SlowDown()
	}
	return m, nil
}

func (m *Model) tick() {
	dt := m.Settings.TickRate.Seconds()

	m.Playback.AdvanceTime(dt)

	for _, commit := range m.Playback.DueCommits() {
		m.processCommit(commit)
	}

	if m.chanDone && len(m.Playback.CommitQueue) == 0 && m.Playback.Elapsed > 0 {
		m.Playback.Finished = true
	}

	// Heat halves roughly every 2 sim-seconds
	simDt := dt * m.Playback.DaysPerSecond * 86400
	decayRate := math.Pow(0.5, simDt/2.0)

	for path, f := range m.Files {
		f.Update(m.Playback.CurrTime, decayRate)
		if f.State == FileRemoved {
			m.Root.RemoveFile(path)
			delete(m.Files, path)
		}
	}

	idleTimeout := time.Duration(m.Settings.UserIdleTime * float64(time.Second))
	for _, u := range m.Users {
		u.Update(m.Playback.CurrTime, idleTimeout)
	}

	remaining := m.Actions[:0]
	for _, a := range m.Actions {
		if !a.Update(dt) {
			remaining = append(remaining, a)
		}
	}
	m.Actions = remaining

	// Update user positions — move toward their target file
	for _, u := range m.Users {
		if !u.Active || u.TargetFile == "" {
			continue
		}
		if f, ok := m.Files[u.TargetFile]; ok {
			target := Vec2{f.ScreenX - 30, f.ScreenY}
			delta := target.Sub(u.Body.Pos)
			dist := delta.Len()
			if dist > 2.0 {
				u.Body.ApplyForce(delta.Normalize().Scale(dist * 2.0))
			}
			u.Body.Integrate(dt, 0.8)
		}
	}

	// Run force-directed layout
	UpdateLayout(m.Root, dt)
}

func (m *Model) processCommit(c parser.Commit) {
	user, exists := m.Users[c.Username]
	if !exists {
		user = NewUser(c.Username, c.Timestamp)
		m.Users[c.Username] = user
	}
	user.Touch(c.Timestamp)

	// Set target to last file in commit
	if len(c.Files) > 0 {
		user.TargetFile = c.Files[len(c.Files)-1].Path
	}

	for _, cf := range c.Files {
		path := cf.Path

		f, exists := m.Files[path]
		if !exists {
			f = m.Root.InsertFile(path, c.Timestamp)
			m.Files[path] = f
		}

		var fileColor color.Color = f.Color
		if cf.Color != nil {
			fileColor = cf.Color
		}

		switch cf.Action {
		case "D":
			f.MarkRemoved(c.Timestamp, time.Duration(m.Settings.FileIdleTime)*time.Second)
		default:
			f.Touch(c.Timestamp, fileColor)
		}

		m.Actions = append(m.Actions, &Action{
			Username: c.Username,
			FilePath: path,
			Type:     cf.Action,
		})

		m.Activity = append(m.Activity, ActivityEntry{
			Username: c.Username,
			Action:   cf.Action,
			FilePath: path,
			Time:     c.Timestamp,
		})
	}

	if len(m.Activity) > 1000 {
		m.Activity = m.Activity[len(m.Activity)-500:]
	}
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(m.Settings.TickRate, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) drainCommits() tea.Cmd {
	return func() tea.Msg {
		var batch []parser.Commit
		for {
			select {
			case c, ok := <-m.commitCh:
				if !ok {
					return commitBatchMsg(batch)
				}
				batch = append(batch, c)
				if len(batch) >= 100 {
					return commitBatchMsg(batch)
				}
			default:
				if len(batch) > 0 {
					return commitBatchMsg(batch)
				}
				c, ok := <-m.commitCh
				if !ok {
					return commitBatchMsg(batch)
				}
				batch = append(batch, c)
				return commitBatchMsg(batch)
			}
		}
	}
}
