package model

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
)

// TermPixelSize holds detected terminal dimensions.
type TermPixelSize struct {
	PixW, PixH   int // total window pixel size (from CSI 14t)
	CellW, CellH int // cell pixel size (from CSI 16t)
}

// DetectTermPixelSize queries the terminal for pixel dimensions.
// Must be called BEFORE tea.Program.Run() takes over stdin.
// Opens a separate handle to the terminal for reading, so stdin
// remains clean for Bubble Tea.
func DetectTermPixelSize() TermPixelSize {
	var result TermPixelSize

	// Open a separate read handle to the terminal.
	// This avoids competing with Bubble Tea for stdin.
	var ttyPath string
	if runtime.GOOS == "windows" {
		ttyPath = "CONIN$"
	} else {
		ttyPath = "/dev/tty"
	}

	tty, err := os.OpenFile(ttyPath, os.O_RDWR, 0)
	if err != nil {
		return result
	}
	defer tty.Close()

	fd := tty.Fd()
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return result
	}
	defer term.Restore(fd, oldState)

	// Reader goroutine on our private handle — will exit when tty is closed
	incoming := make(chan byte, 1024)
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := tty.Read(buf)
			if n > 0 {
				select {
				case incoming <- buf[0]:
				default:
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// Drain any buffered input
	drainChannel(incoming, 20*time.Millisecond)

	// Write queries to our tty handle (goes to the same terminal)
	// Query window pixel size: CSI 14 t → CSI 4 ; height ; width t
	tty.WriteString("\x1b[14t")
	if w, h := parseResponse(incoming, "\x1b[4;", 300*time.Millisecond); w > 0 && h > 0 {
		result.PixW = w
		result.PixH = h
	}

	// Query cell pixel size: CSI 16 t → CSI 6 ; height ; width t
	tty.WriteString("\x1b[16t")
	if w, h := parseResponse(incoming, "\x1b[6;", 300*time.Millisecond); w > 0 && h > 0 {
		result.CellW = w
		result.CellH = h
	}

	// tty.Close() in defer will cause the reader goroutine to exit
	return result
}

func parseResponse(ch <-chan byte, prefix string, timeout time.Duration) (width, height int) {
	deadline := time.After(timeout)
	var buf []byte

	for {
		select {
		case b := <-ch:
			buf = append(buf, b)
			s := string(buf)

			idx := strings.Index(s, prefix)
			if idx < 0 {
				continue
			}
			after := s[idx+len(prefix):]
			tIdx := strings.IndexByte(after, 't')
			if tIdx < 0 {
				continue
			}

			parts := strings.Split(after[:tIdx], ";")
			if len(parts) != 2 {
				return 0, 0
			}
			h, err1 := strconv.Atoi(parts[0])
			w, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
				return 0, 0
			}
			return w, h

		case <-deadline:
			return 0, 0
		}
	}
}

func drainChannel(ch <-chan byte, duration time.Duration) {
	deadline := time.After(duration)
	for {
		select {
		case <-ch:
		case <-deadline:
			return
		}
	}
}
