package parser

import (
	"image/color"
	"time"
)

// CommitFile represents a single file change within a commit.
type CommitFile struct {
	Path   string
	Action string // "A" (add), "M" (modify), "D" (delete)
	Color  color.Color
}

// Commit represents a single commit with its affected files.
type Commit struct {
	Timestamp time.Time
	Username  string
	Files     []CommitFile
}
