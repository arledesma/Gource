package model

import (
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"time"
)

// DirNode represents a directory in the file tree.
type DirNode struct {
	Name       string
	Path       string
	Parent     *DirNode
	Children   []*DirNode
	Files      map[string]*File
	LastActive time.Time
	Collapsed  bool
	Body       PhysicsBody // position for force-directed layout
}

// NewDirNode creates a directory node.
func NewDirNode(name, path string) *DirNode {
	return &DirNode{
		Name:  name,
		Path:  path,
		Files: make(map[string]*File),
	}
}

// InsertFile adds a file to the tree, creating intermediate directories as needed.
// Returns the File entity.
func (d *DirNode) InsertFile(path string, now time.Time) *File {
	parts := strings.Split(path, "/")

	// Navigate/create directory path
	node := d
	for i := 0; i < len(parts)-1; i++ {
		child := node.findChild(parts[i])
		if child == nil {
			dirPath := strings.Join(parts[:i+1], "/")
			child = NewDirNode(parts[i], dirPath)
			child.Parent = node
			// Seed position near parent with deterministic offset based on name
			h := fnv.New32a()
			h.Write([]byte(dirPath))
			angle := float64(h.Sum32()%360) * math.Pi / 180.0
			child.Body.Pos = Vec2{
				X: node.Body.Pos.X + math.Cos(angle)*60,
				Y: node.Body.Pos.Y + math.Sin(angle)*60,
			}
			node.Children = append(node.Children, child)
			sort.Slice(node.Children, func(a, b int) bool {
				return node.Children[a].Name < node.Children[b].Name
			})
		}
		child.LastActive = now
		node = child
	}

	// Add file to leaf directory
	fileName := parts[len(parts)-1]
	f, exists := node.Files[fileName]
	if !exists {
		f = NewFile(path, now)
		node.Files[fileName] = f
	}
	node.LastActive = now

	return f
}

// RemoveFile removes a file and prunes empty parent directories.
func (d *DirNode) RemoveFile(path string) {
	parts := strings.Split(path, "/")

	// Navigate to the directory containing the file
	node := d
	for i := 0; i < len(parts)-1; i++ {
		child := node.findChild(parts[i])
		if child == nil {
			return
		}
		node = child
	}

	fileName := parts[len(parts)-1]
	delete(node.Files, fileName)

	// Prune empty directories upward
	for node != d && len(node.Files) == 0 && len(node.Children) == 0 {
		parent := node.Parent
		if parent == nil {
			break
		}
		for i, c := range parent.Children {
			if c == node {
				parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
				break
			}
		}
		node = parent
	}
}

// TotalFiles returns the count of files in this subtree.
func (d *DirNode) TotalFiles() int {
	count := len(d.Files)
	for _, child := range d.Children {
		count += child.TotalFiles()
	}
	return count
}

// AllFiles returns all file entities in this subtree.
func (d *DirNode) AllFiles() []*File {
	var files []*File
	for _, f := range d.Files {
		files = append(files, f)
	}
	for _, child := range d.Children {
		files = append(files, child.AllFiles()...)
	}
	return files
}

// SortedFileNames returns file names sorted alphabetically.
func (d *DirNode) SortedFileNames() []string {
	names := make([]string, 0, len(d.Files))
	for name := range d.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (d *DirNode) findChild(name string) *DirNode {
	for _, c := range d.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}
