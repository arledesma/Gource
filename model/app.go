package model

import (
	"context"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"os"
	"regexp"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/harmonica"

	"github.com/arledesma/gource-tui/config"
	"github.com/arledesma/gource-tui/parser"
)

// ActivityEntry represents a line in the activity log.
type ActivityEntry struct {
	Username string
	Action   string
	FilePath string
	Time     time.Time
}

// UserSpring holds Harmonica springs for smooth user movement.
type UserSpring struct {
	SpringX  harmonica.Spring
	SpringY  harmonica.Spring
	VelX     float64
	VelY     float64
	TargetX  float64
	TargetY  float64
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
	Particles ParticleSystem
	Captions  *CaptionSystem

	// User animation springs (keyed by username)
	UserSprings map[string]*UserSpring

	commitCh  <-chan parser.Commit
	cancelPar context.CancelFunc
	chanDone  bool

	Width        int
	Height       int
	CameraZoom   float64 // 0 = auto-fit, >0 = manual zoom
	CameraOffset Vec2    // manual pan offset in pixels
	dragging     bool
	lastMouseX   int
	lastMouseY   int
	termRows     int // terminal rows for sixel clamping
	termCols     int
	ShowLegend   bool
	ShowHelp     bool
	LastFrameMs  float64 // render time in ms (image generation)
	SixelEncMs   float64 // sixel encoding time in ms
	SixelBytes   int     // sixel output size in bytes
	TotalFrameMs float64 // full pipeline: render + encode
	FrameCount   int64

	userFilterRe *regexp.Regexp
	fileFilterRe *regexp.Regexp
}

type tickMsg time.Time

type commitBatchMsg []parser.Commit

// New creates a new app model.
func New(cfg config.Settings, p parser.Parser) *Model {
	ctx, cancel := context.WithCancel(context.Background())
	ch := p.Stream(ctx)

	m := &Model{
		Settings:    cfg,
		Root:        NewDirNode("", ""),
		Files:       make(map[string]*File),
		Users:       make(map[string]*User),
		Playback:    NewPlayback(cfg.DaysPerSecond),
		Captions:    NewCaptionSystem(5),
		UserSprings: make(map[string]*UserSpring),
		commitCh:    ch,
		cancelPar:   cancel,
	}

	if cfg.UserFilter != "" {
		m.userFilterRe, _ = regexp.Compile(cfg.UserFilter)
	}
	if cfg.FileFilter != "" {
		m.fileFilterRe, _ = regexp.Compile(cfg.FileFilter)
	}

	return m
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

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)

	case tea.MouseMotionMsg:
		return m.handleMouseMotion(msg)

	case tea.MouseReleaseMsg:
		m.dragging = false
		return m, nil

	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg)

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
	panStep := 30.0
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
	case "z":
		if m.CameraZoom == 0 {
			m.CameraZoom = 1.0
		}
		m.CameraZoom *= 1.3
		if m.CameraZoom > 10 {
			m.CameraZoom = 10
		}
	case "x":
		if m.CameraZoom == 0 {
			m.CameraZoom = 1.0
		}
		m.CameraZoom /= 1.3
		if m.CameraZoom < 0.1 {
			m.CameraZoom = 0.1
		}
	case "up":
		m.CameraOffset.Y += panStep
	case "down":
		m.CameraOffset.Y -= panStep
	case "left":
		m.CameraOffset.X += panStep
	case "right":
		m.CameraOffset.X -= panStep
	case "home":
		m.CameraZoom = 0
		m.CameraOffset = Vec2{}
	case "l":
		m.ShowLegend = !m.ShowLegend
	case "?":
		m.ShowHelp = !m.ShowHelp
	case "[": // seek back 5%
		m.seekToProgress(m.Playback.Progress() - 0.05)
	case "]": // seek forward 5%
		m.seekToProgress(m.Playback.Progress() + 0.05)
	case "s": // screenshot
		m.saveScreenshot()
	}
	return m, nil
}

func (m *Model) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	clickX := msg.X
	clickY := msg.Y

	// Check if click is in the bottom 2 rows (status bar / progress bar)
	if clickY >= m.Height-2 {
		barStartFrac := 160.0 / float64(m.Width*8)
		barEndFrac := 1.0 - 280.0/float64(m.Width*8)
		frac := float64(clickX) / float64(m.Width)

		if frac >= barStartFrac && frac <= barEndFrac {
			progress := (frac - barStartFrac) / (barEndFrac - barStartFrac)
			m.seekToProgress(progress)
			return m, nil
		}
	}

	// Start drag for panning
	if msg.Button == tea.MouseLeft {
		m.dragging = true
		m.lastMouseX = clickX
		m.lastMouseY = clickY
	}

	return m, nil
}

func (m *Model) handleMouseMotion(msg tea.MouseMotionMsg) (tea.Model, tea.Cmd) {
	// Only pan while left button is held
	if msg.Button != tea.MouseLeft {
		m.dragging = false
		return m, nil
	}

	if !m.dragging {
		// Start drag from current position
		m.dragging = true
		m.lastMouseX = msg.X
		m.lastMouseY = msg.Y
		return m, nil
	}

	dx := msg.X - m.lastMouseX
	dy := msg.Y - m.lastMouseY
	m.lastMouseX = msg.X
	m.lastMouseY = msg.Y

	// Convert cell delta to pixel delta
	cellW, cellH := 8, 16
	m.CameraOffset.X += float64(dx * cellW)
	m.CameraOffset.Y += float64(dy * cellH)

	return m, nil
}

func (m *Model) seekToProgress(progress float64) {
	m.Playback.SeekTo(progress)
	m.resetVisualization()

	// Replay all commits up to the seek point
	for _, c := range m.Playback.AllCommits {
		if c.Timestamp.After(m.Playback.CurrTime) {
			break
		}
		m.processCommit(c)
	}

	// Run a few layout iterations so the graph isn't all clumped at origin
	for range 30 {
		UpdateLayout(m.Root, 1.0/30.0)
	}
}

func (m *Model) saveScreenshot() {
	cellW, cellH := 8, 16
	pixW := m.Width * cellW
	pixH := (m.Height - 1) * cellH
	img := m.renderImage(pixW, pixH)

	ts := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("gource-tui-%s.png", ts)

	f, err := os.Create(filename)
	if err != nil {
		return
	}
	defer f.Close()
	png.Encode(f, img)
}

func (m *Model) resetVisualization() {
	m.Root = NewDirNode("", "")
	m.Files = make(map[string]*File)
	m.Users = make(map[string]*User)
	m.Actions = nil
	m.Activity = nil
	m.Particles.Particles = nil
	m.Captions = NewCaptionSystem(5)
	m.UserSprings = make(map[string]*UserSpring)
}

func (m *Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	mouse := msg.Mouse()
	switch mouse.Button {
	case tea.MouseWheelUp:
		if m.CameraZoom == 0 {
			m.CameraZoom = 1.0
		}
		m.CameraZoom *= 1.15
		if m.CameraZoom > 10 {
			m.CameraZoom = 10
		}
	case tea.MouseWheelDown:
		if m.CameraZoom == 0 {
			m.CameraZoom = 1.0
		}
		m.CameraZoom /= 1.15
		if m.CameraZoom < 0.1 {
			m.CameraZoom = 0.1
		}
	}
	return m, nil
}

func (m *Model) tick() {
	dt := m.Settings.TickRate.Seconds()

	m.Playback.AdvanceTime(dt)

	// Auto-skip: if no commits are due and next commit is far away, jump ahead
	if m.Settings.AutoSkip > 0 && !m.Playback.Paused && len(m.Playback.CommitQueue) > 0 {
		next := m.Playback.CommitQueue[0].Timestamp
		gap := next.Sub(m.Playback.CurrTime).Seconds()
		skipThreshold := m.Settings.AutoSkip * 86400 * m.Playback.DaysPerSecond
		if gap > skipThreshold {
			// Jump to just before the next commit
			m.Playback.CurrTime = next.Add(-time.Second)
		}
	}

	for _, commit := range m.Playback.DueCommits() {
		m.processCommit(commit)
	}

	if m.chanDone && len(m.Playback.CommitQueue) == 0 && m.Playback.Elapsed > 0 {
		if m.Settings.Loop {
			// Reset for loop
			m.Playback.CurrTime = m.Playback.StartTime
			m.Playback.Elapsed = 0
			m.Playback.Finished = false
		} else {
			m.Playback.Finished = true
		}
	}

	// Heat decay
	simDt := dt * m.Playback.DaysPerSecond * 86400
	decayRate := math.Pow(0.5, simDt/2.0)

	for path, f := range m.Files {
		f.Update(m.Playback.CurrTime, decayRate)
		if f.State == FileRemoved {
			m.Root.RemoveFile(path)
			delete(m.Files, path)
		}
	}

	// Decay edge heat
	m.Root.DecayEdgeHeat(decayRate)

	for _, u := range m.Users {
		u.Update(m.Settings.UserIdleTime)
	}

	// Update actions
	remaining := m.Actions[:0]
	for _, a := range m.Actions {
		if !a.Update(dt) {
			remaining = append(remaining, a)
		}
	}
	m.Actions = remaining

	// Update user positions via Harmonica springs
	for name, u := range m.Users {
		if !u.Active || u.TargetFile == "" {
			continue
		}
		f, ok := m.Files[u.TargetFile]
		if !ok {
			continue
		}

		spring, exists := m.UserSprings[name]
		if !exists {
			spring = &UserSpring{
				SpringX: harmonica.NewSpring(harmonica.FPS(30), 6.0, 0.6),
				SpringY: harmonica.NewSpring(harmonica.FPS(30), 6.0, 0.6),
			}
			spring.TargetX = f.ScreenX - 25
			spring.TargetY = f.ScreenY
			m.UserSprings[name] = spring
		}

		spring.TargetX = f.ScreenX - 25
		spring.TargetY = f.ScreenY

		u.Body.Pos.X, spring.VelX = spring.SpringX.Update(u.Body.Pos.X, spring.VelX, spring.TargetX)
		u.Body.Pos.Y, spring.VelY = spring.SpringY.Update(u.Body.Pos.Y, spring.VelY, spring.TargetY)
	}

	// Update particles and captions
	m.Particles.Update(dt)
	m.Captions.Update(dt)

	// Run force-directed layout
	UpdateLayout(m.Root, dt)
}

func (m *Model) processCommit(c parser.Commit) {
	// Filter by user regex
	if m.userFilterRe != nil && !m.userFilterRe.MatchString(c.Username) {
		return
	}

	user, exists := m.Users[c.Username]
	if !exists {
		user = NewUser(c.Username, c.Timestamp)
		m.Users[c.Username] = user
	}
	user.Touch(c.Timestamp)

	if len(c.Files) > 0 {
		targetPath := c.Files[len(c.Files)-1].Path
		user.TargetFile = targetPath

		// Initialize new user position near their first target file
		if user.ActionCount == 1 {
			if f, ok := m.Files[targetPath]; ok {
				user.Body.Pos = Vec2{f.ScreenX - 25, f.ScreenY}
			} else {
				// File not yet positioned — place near root
				user.Body.Pos = m.Root.Body.Pos
			}
		}
	}

	// Add commit message caption near user
	if c.Message != "" {
		msg := c.Message
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}
		m.Captions.Add(msg, user.Body.Pos, user.Color)
	}

	for _, cf := range c.Files {
		// Filter by file regex
		if m.fileFilterRe != nil && !m.fileFilterRe.MatchString(cf.Path) {
			continue
		}
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
			// Red burst particles on delete
			m.Particles.Emit(Vec2{f.ScreenX, f.ScreenY}, color.RGBA{R: 230, G: 60, B: 60, A: 255}, 8, 40, 0.6)
		default:
			f.Touch(c.Timestamp, fileColor)
			// Green sparkle on create
			if cf.Action == "A" {
				m.Particles.Emit(Vec2{f.ScreenX, f.ScreenY}, color.RGBA{R: 100, G: 220, B: 100, A: 255}, 6, 30, 0.5)
			}
		}

		// Propagate edge glow
		dir := m.Root.FindDir(path)
		dir.PropagateEdgeHeat()

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
