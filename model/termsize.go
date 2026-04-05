package model

import (
	"os"
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
func DetectTermPixelSize() TermPixelSize {
	var result TermPixelSize

	fd := uintptr(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return result
	}
	defer term.Restore(fd, oldState)

	// Start a background reader that continuously reads stdin.
	// This avoids blocking issues with SetReadDeadline on Windows.
	incoming := make(chan byte, 1024)
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				incoming <- buf[0]
			}
			if err != nil {
				return
			}
		}
	}()

	// Small delay to let any buffered input arrive
	drainChannel(incoming, 20*time.Millisecond)

	// Query window pixel size: CSI 14 t → CSI 4 ; height ; width t
	os.Stdout.WriteString("\x1b[14t")
	os.Stdout.Sync()
	if w, h := parseResponse(incoming, "\x1b[4;", 300*time.Millisecond); w > 0 && h > 0 {
		result.PixW = w
		result.PixH = h
	}

	// Query cell pixel size: CSI 16 t → CSI 6 ; height ; width t
	os.Stdout.WriteString("\x1b[16t")
	os.Stdout.Sync()
	if w, h := parseResponse(incoming, "\x1b[6;", 300*time.Millisecond); w > 0 && h > 0 {
		result.CellW = w
		result.CellH = h
	}

	return result
}

// parseResponse reads bytes from the channel until it finds a complete
// CSI response matching the prefix, or times out.
func parseResponse(ch <-chan byte, prefix string, timeout time.Duration) (width, height int) {
	deadline := time.After(timeout)
	var buf []byte

	for {
		select {
		case b := <-ch:
			buf = append(buf, b)
			s := string(buf)

			// Look for complete response: prefix + H;W + 't'
			idx := strings.Index(s, prefix)
			if idx < 0 {
				continue
			}
			after := s[idx+len(prefix):]
			tIdx := strings.IndexByte(after, 't')
			if tIdx < 0 {
				continue
			}

			// Parse "height;width"
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
