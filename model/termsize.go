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

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(uintptr(fd))
	if err != nil {
		return result
	}
	defer term.Restore(uintptr(fd), oldState)

	// Query window pixel size: CSI 14 t → response CSI 4 ; height ; width t
	if w, h := queryCSI("\x1b[14t", "\x1b[4;"); w > 0 && h > 0 {
		result.PixW = w
		result.PixH = h
	}

	// Query cell pixel size: CSI 16 t → response CSI 6 ; height ; width t
	if w, h := queryCSI("\x1b[16t", "\x1b[6;"); w > 0 && h > 0 {
		result.CellW = w
		result.CellH = h
	}

	return result
}

func queryCSI(query, responsePrefix string) (width, height int) {
	// Flush any pending input
	os.Stdin.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	discard := make([]byte, 256)
	os.Stdin.Read(discard)

	// Send query
	os.Stdout.WriteString(query)
	os.Stdout.Sync()

	// Read response with timeout
	os.Stdin.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, 256)
	total := 0
	for total < len(buf) {
		n, err := os.Stdin.Read(buf[total:])
		total += n
		if err != nil {
			break
		}
		// Look for 't' terminator
		if strings.ContainsRune(string(buf[:total]), 't') {
			break
		}
	}
	// Clear deadline
	os.Stdin.SetReadDeadline(time.Time{})

	resp := string(buf[:total])

	// Parse: prefix HEIGHT ; WIDTH t
	idx := strings.Index(resp, responsePrefix)
	if idx < 0 {
		return 0, 0
	}
	resp = resp[idx+len(responsePrefix):]
	end := strings.IndexByte(resp, 't')
	if end < 0 {
		return 0, 0
	}
	parts := strings.Split(resp[:end], ";")
	if len(parts) != 2 {
		return 0, 0
	}
	h, err1 := strconv.Atoi(parts[0])
	w, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0
	}
	return w, h
}
