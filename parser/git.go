package parser

import (
	"bufio"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GitParser parses git log output using the same format Gource uses.
type GitParser struct {
	Dir       string
	StartDate string // --since flag (YYYY-MM-DD)
	StopDate  string // --until flag (YYYY-MM-DD)
}

func (p *GitParser) Stream(ctx context.Context) <-chan Commit {
	ch := make(chan Commit, 256)

	go func() {
		defer close(ch)

		args := []string{
			"log",
			"--reverse",
			"--raw",
			"--encoding=UTF-8",
			"--no-renames",
			"--no-show-signature",
			"--pretty=format:user:%aN%n%ct",
		}

		if p.StartDate != "" {
			args = append(args, "--since", p.StartDate)
		}
		if p.StopDate != "" {
			args = append(args, "--until", p.StopDate)
		}

		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = p.Dir

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return
		}

		if err := cmd.Start(); err != nil {
			return
		}

		scanner := bufio.NewScanner(stdout)
		var current *Commit

		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "user:") {
				// Flush previous commit
				if current != nil && len(current.Files) > 0 {
					select {
					case ch <- *current:
					case <-ctx.Done():
						return
					}
				}

				username := line[5:]

				// Next line is the timestamp
				if !scanner.Scan() {
					break
				}
				ts, err := strconv.ParseInt(scanner.Text(), 10, 64)
				if err != nil || ts == 0 {
					current = nil
					continue
				}

				current = &Commit{
					Timestamp: time.Unix(ts, 0),
					Username:  username,
				}
				continue
			}

			if current == nil {
				continue
			}

			// Parse raw diff line: ":100644 100644 abc def M\tpath"
			tab := strings.IndexByte(line, '\t')
			if tab < 1 || tab >= len(line)-1 {
				continue
			}

			status := string(line[tab-1])
			file := line[tab+1:]

			// Strip surrounding quotes
			if len(file) > 2 && file[0] == '"' && file[len(file)-1] == '"' {
				file = file[1 : len(file)-1]
			}

			if file == "" {
				continue
			}

			current.Files = append(current.Files, CommitFile{
				Path:   file,
				Action: status,
			})
		}

		// Flush last commit
		if current != nil && len(current.Files) > 0 {
			select {
			case ch <- *current:
			case <-ctx.Done():
			}
		}

		cmd.Wait()
	}()

	return ch
}
