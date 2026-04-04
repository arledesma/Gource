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

// New auto-detects the appropriate parser for the given path.
// If path is a directory with a .git folder, uses git log.
// If path is a file, tries custom format.
func New(path string) (Parser, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		gitDir := filepath.Join(path, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return &GitParser{Dir: path}, nil
		}
	}

	return &CustomParser{File: path}, nil
}
