package model

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
)

// TermPixelSize holds detected terminal dimensions.
type TermPixelSize struct {
	PixW, PixH   int
	CellW, CellH int
}

// DetectTermPixelSize runs the current executable with --detect-term
// to query the terminal in a clean subprocess. This prevents any
// console state corruption from affecting Bubble Tea.
func DetectTermPixelSize() TermPixelSize {
	var result TermPixelSize

	self, err := os.Executable()
	if err != nil {
		return result
	}

	cmd := exec.Command(self, "--detect-term")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return result
	}

	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) == 4 {
		result.PixW, _ = strconv.Atoi(parts[0])
		result.PixH, _ = strconv.Atoi(parts[1])
		result.CellW, _ = strconv.Atoi(parts[2])
		result.CellH, _ = strconv.Atoi(parts[3])
	}

	return result
}

// RunDetectSubprocess queries the terminal and prints results to stdout.
// Called when the binary is invoked with --detect-term.
func RunDetectSubprocess() {
	fd := os.Stdin.Fd()
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Println("0 0 0 0")
		return
	}
	defer term.Restore(fd, oldState)

	incoming := make(chan byte, 1024)
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
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

	drainChannel(incoming, 20*time.Millisecond)

	var pixW, pixH, cellW, cellH int

	os.Stderr.WriteString("\x1b[14t")
	if w, h := parseResponse(incoming, "\x1b[4;", 300*time.Millisecond); w > 0 && h > 0 {
		pixW = w
		pixH = h
	}

	os.Stderr.WriteString("\x1b[16t")
	if w, h := parseResponse(incoming, "\x1b[6;", 300*time.Millisecond); w > 0 && h > 0 {
		cellW = w
		cellH = h
	}

	fmt.Printf("%d %d %d %d\n", pixW, pixH, cellW, cellH)
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
