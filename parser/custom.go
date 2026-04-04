package parser

import (
	"bufio"
	"context"
	"encoding/hex"
	"image/color"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var customRegex = regexp.MustCompile(`^(?:\xEF\xBB\xBF)?([^|]+)\|([^|]*)\|([ADM]?)\|([^|]+)(?:\|#?([a-fA-F0-9]{6}(?:[a-fA-F0-9]{2})?))?`)

// CustomParser parses the Gource custom log format:
// timestamp|username|A/M/D|filepath|color
type CustomParser struct {
	File string
}

func (p *CustomParser) Stream(ctx context.Context) <-chan Commit {
	ch := make(chan Commit, 256)

	go func() {
		defer close(ch)

		f, err := os.Open(p.File)
		if err != nil {
			return
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		var current *Commit

		for scanner.Scan() {
			line := scanner.Text()
			matches := customRegex.FindStringSubmatch(line)
			if matches == nil {
				continue
			}

			ts := parseTimestamp(matches[1])
			if ts.IsZero() {
				continue
			}

			username := matches[2]
			if username == "" {
				username = "Unknown"
			}

			action := matches[3]
			if action == "" {
				action = "A"
			}

			file := matches[4]

			// Group entries with same timestamp+user into one commit
			if current != nil && (current.Timestamp != ts || current.Username != username) {
				if len(current.Files) > 0 {
					select {
					case ch <- *current:
					case <-ctx.Done():
						return
					}
				}
				current = nil
			}

			if current == nil {
				current = &Commit{
					Timestamp: ts,
					Username:  username,
				}
			}

			cf := CommitFile{
				Path:   file,
				Action: action,
			}

			if len(matches) >= 6 && matches[5] != "" {
				cf.Color = parseHexColor(matches[5])
			}

			current.Files = append(current.Files, cf)
		}

		if current != nil && len(current.Files) > 0 {
			select {
			case ch <- *current:
			case <-ctx.Done():
			}
		}
	}()

	return ch
}

func parseTimestamp(s string) time.Time {
	s = strings.TrimSpace(s)
	if strings.Contains(s[1:], "-") {
		for _, layout := range []string{
			"2006-01-02",
			"2006-01-02 15:04:05",
			time.RFC3339,
		} {
			if t, err := time.Parse(layout, s); err == nil {
				return t
			}
		}
		return time.Time{}
	}

	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil || ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

func parseHexColor(s string) color.Color {
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	switch len(b) {
	case 3:
		return color.RGBA{R: b[0], G: b[1], B: b[2], A: 255}
	case 4:
		return color.RGBA{R: b[0], G: b[1], B: b[2], A: b[3]}
	}
	return nil
}
