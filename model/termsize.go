package model

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
)

// TermPixelSize holds the detected terminal pixel dimensions.
type TermPixelSize struct {
	CellW, CellH int // pixel size of one cell
	PixW, PixH    int // total pixel size of the terminal
}

// DetectTermPixelSize queries the terminal for cell pixel dimensions.
// Must be called BEFORE tea.Program.Run() takes over stdin.
// Returns zero values if detection fails.
func DetectTermPixelSize() TermPixelSize {
	var result TermPixelSize

	fd := os.Stdin.Fd()
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return result
	}
	defer term.Restore(fd, oldState)

	// CSI 16t — report cell size in pixels
	// Response: CSI 6 ; cellH ; cellW t
	os.Stdout.WriteString("\x1b[16t")
	os.Stdout.Sync()

	resp := readResponse(100 * time.Millisecond)
	if cellW, cellH, ok := parseSizeResponse(resp, "\x1b[6;"); ok {
		result.CellW = cellW
		result.CellH = cellH
	}

	// CSI 14t — report terminal pixel size
	// Response: CSI 4 ; pixH ; pixW t
	os.Stdout.WriteString("\x1b[14t")
	os.Stdout.Sync()

	resp = readResponse(100 * time.Millisecond)
	if pixW, pixH, ok := parseSizeResponse(resp, "\x1b[4;"); ok {
		result.PixW = pixW
		result.PixH = pixH
	}

	return result
}

func readResponse(timeout time.Duration) string {
	buf := make([]byte, 128)
	done := make(chan int, 1)
	go func() {
		n, _ := os.Stdin.Read(buf)
		done <- n
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case n := <-done:
		return string(buf[:n])
	case <-timer.C:
		return ""
	}
}

func parseSizeResponse(resp, prefix string) (int, int, bool) {
	idx := strings.Index(resp, prefix)
	if idx < 0 {
		return 0, 0, false
	}
	resp = resp[idx+len(prefix):]
	end := strings.IndexByte(resp, 't')
	if end < 0 {
		return 0, 0, false
	}
	parts := strings.Split(resp[:end], ";")
	if len(parts) != 2 {
		return 0, 0, false
	}
	a, err1 := strconv.Atoi(parts[0])
	b, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || a <= 0 || b <= 0 {
		return 0, 0, false
	}
	// Response is height;width, return as width,height
	return b, a, true
}
