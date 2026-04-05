package parser

import (
	"context"
	"os"
	"path/filepath"
)

// Parser streams commits from a VCS log source.
type Parser interface {
	Stream(ctx context.Context) <-chan Commit
}

// Options configures parser behavior.
type Options struct {
	StartDate string
	StopDate  string
}

// New auto-detects the appropriate parser for the given path.
func New(path string, opts ...Options) (Parser, error) {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		gitDir := filepath.Join(path, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return &GitParser{
				Dir:       path,
				StartDate: opt.StartDate,
				StopDate:  opt.StopDate,
			}, nil
		}
	}

	return &CustomParser{File: path}, nil
}
