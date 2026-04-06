# gource-tui

[![CI](https://github.com/arledesma/gource-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/arledesma/gource-tui/actions/workflows/ci.yml)

https://github.com/user-attachments/assets/f711acd7-1f85-4a86-92aa-d132ca4cd7c5

A terminal-based source control visualization tool inspired by [Gource](https://gource.io). Renders git repository history as an animated force-directed graph with bloom effects, using [sixel graphics](https://en.wikipedia.org/wiki/Sixel) directly in your terminal.

## Features

**Rendering**
- Sixel pixel graphics — full-color rendered frames, not character approximations
- Force-directed graph layout with spring physics and N-body repulsion
- Bloom post-processing (gaussian blur + additive blending)
- Bezier curve edges with depth-based dimming and activity glow
- Background star field for visual depth
- File type shapes: circles (source), squares (config), triangles (docs), diamonds (images)

**Animation**
- File heat visualization — recently modified files glow bright, fade over time
- Contributor entities with Harmonica spring physics and motion trails
- Action beams from users to files during commits
- Particle effects — green sparkles on creation, red bursts on deletion
- Directory pulse animation when receiving commits
- Commit message captions floating near users

**Interaction**
- Auto-fit camera with manual zoom (z/x/scroll), pan (arrows/drag), reset (Home)
- Click progress bar to seek to any point in history
- Click directory nodes to collapse/expand subtrees
- Keyboard seek with `[`/`]` (5% jumps)
- Screenshot export (press `s`)
- Minimap overview when zoomed in

**Overlays**
- File extension color legend (toggle `l`)
- Active user list with action counts (toggle `u`)
- Help overlay (toggle `?`)
- Commit activity heatmap in the progress bar
- Elapsed time display (day X/Y)

**Configuration**
- 4 color themes: dark, light, solarized, monokai
- 20+ CLI flags for speed, filters, dates, visibility, performance
- Automatic terminal pixel size detection (CSI 16t)
- Adapts to terminal resize
- Loop mode, auto-skip idle periods, regex filters

## Requirements

- **Go 1.24+**
- **Sixel-capable terminal:**
  - [Windows Terminal](https://github.com/microsoft/terminal) 1.22+
  - [WezTerm](https://wezfurlong.org/wezterm/)
  - [foot](https://codeberg.org/dnkl/foot)
  - [mlterm](http://mlterm.sourceforge.net/)
  - xterm (with `--enable-sixel-graphics`)
- **git** on PATH

## Install

```sh
go install github.com/arledesma/gource-tui@latest
```

Or build from source:

```sh
git clone git@github.com:arledesma/gource-tui.git
cd gource-tui
go build -o gource-tui .
```

## Usage

```sh
gource-tui                                              # current directory
gource-tui /path/to/repo                                # specific repo
gource-tui /path/to/custom.log                          # Gource custom log
gource-tui --speed 2.0 --theme solarized .              # fast + themed
gource-tui --user-filter "Alice" .                      # filter by user
gource-tui --file-filter "\.(go|rs|py)$" --no-bloom .   # filter + fast
gource-tui --start-date 2024-01-01 --stop-date 2024-06-30 .
gource-tui --scale 0.75 --fps 20 .                      # lower res + fps
```

## Controls

| Key | Action |
|-|-|
| `Space` | Pause / resume |
| `+` / `-` | Speed up / slow down |
| `[` / `]` | Seek back / forward 5% |
| `z` / `x` | Zoom in / out |
| `Scroll` | Zoom in / out |
| `Drag` | Pan camera |
| `Arrows` | Pan camera |
| `Home` | Reset camera to auto-fit |
| `Click bar` | Seek to position in timeline |
| `Click node` | Collapse / expand directory |
| `s` | Save screenshot as PNG |
| `l` | Toggle file extension legend |
| `u` | Toggle active user list |
| `?` | Toggle help overlay |
| `q` | Quit |

## CLI Flags

```
  -s, --speed float          Days of history per second (default 0.5)
      --fps int              Target frame rate (default 30)
      --auto-skip float      Skip idle periods longer than N days (default 3)
      --file-idle-time float Seconds before idle files fade (default 60)
      --user-idle-time float Seconds before idle users disappear (default 10)
      --loop                 Loop playback when finished
      --no-bloom             Disable bloom (faster)
      --theme string         dark, light, solarized, monokai (default "dark")
      --background string    Background hex color (e.g. #1a1a2e)
      --scale float          Render resolution scale (default 1.0)
      --start-date string    Start date (YYYY-MM-DD)
      --stop-date string     Stop date (YYYY-MM-DD)
      --user-filter string   Regex to filter users
      --file-filter string   Regex to filter file paths
      --hide-filenames       Hide file name labels
      --hide-dirnames        Hide directory name labels
      --hide-usernames       Hide user name labels
      --hide-date            Hide the date overlay
      --hide-progress        Hide the progress bar
      --cell-size string     Override cell pixel size (e.g. 10x20)
      --debug                Show frame timing breakdown
```

## Terminal Sizing

Cell pixel size is detected at startup via CSI 16t (run as a subprocess to
avoid interfering with the TUI). Works on Windows Terminal, WezTerm, foot,
and most modern terminals.

```sh
# Verify detection
go run ./cmd/terminfo

# Manual override if needed
gource-tui --cell-size 10x20 .
```

Adapts to terminal resize automatically.

## Custom Log Format

Reads [Gource's custom log format](https://github.com/acaudwell/Gource/wiki/Custom-Log-Format):

```
timestamp|username|A/M/D|filepath|color
```

Generate from an existing repo:

```sh
gource --output-custom-log project.log /path/to/repo
gource-tui project.log
```

## Performance

With `--debug`, the status bar shows: `total_ms (r:render_ms s:sixel_ms output_KB) fps`

- `--no-bloom` — biggest CPU savings
- `--scale 0.5` — half resolution
- `--fps 15` — lower frame rate
- `--hide-filenames --hide-usernames` — skip text rendering
- Click directory nodes to collapse subtrees
- Smaller terminal = fewer pixels

## Architecture

```
gource-tui/
  main.go                 CLI + subprocess dispatch
  config/
    settings.go           Configuration
    colors.go             Extension + user color palettes
    theme.go              Color themes
  parser/
    parser.go             Interface + auto-detect
    git.go                Git log parser (goroutine streaming)
    custom.go             Gource custom format parser
    commit.go             Data types
  model/
    app.go                Bubble Tea model + input handling
    view.go               Sixel output + sizing
    render.go             gg canvas rendering + bloom + overlays
    tree.go               Directory tree + physics body
    file.go               File entity (heat, lifecycle)
    user.go               User entity (springs, trails)
    action.go             User-to-file action
    playback.go           Time cursor + seeking
    layout.go             Force-directed layout engine
    particle.go           Particle system
    caption.go            Commit message captions
    termsize.go           Terminal pixel detection
  cmd/
    snapshot/             PNG snapshot tool
    terminfo/             Terminal diagnostics
  .github/workflows/
    ci.yml                Build + vet + test (Linux/Mac/Windows) + release
```

## Built With

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Harmonica](https://github.com/charmbracelet/harmonica) — spring physics
- [gg](https://github.com/fogleman/gg) — 2D graphics
- [go-sixel](https://github.com/mattn/go-sixel) — sixel encoding
- [imaging](https://github.com/disintegration/imaging) — bloom blur
- [cobra](https://github.com/spf13/cobra) — CLI
- [Go fonts](https://pkg.go.dev/golang.org/x/image/font/gofont) — embedded TTF

## Credits

Inspired by [Gource](https://gource.io) by [Andrew Caudwell](https://github.com/acaudwell).

## License

[GPLv3](COPYING)
