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
	Depth      int         // tree depth (root=0)
	EdgeHeat   float64     // glow intensity on edge to parent (0-1)
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
			child.Depth = node.Depth + 1
			// Seed position using sibling index for even angular distribution
			siblingIdx := len(node.Children)
			totalSiblings := siblingIdx + 1
			// Use golden angle for even distribution as siblings are added
			goldenAngle := math.Pi * (3.0 - math.Sqrt(5.0)) // ~137.5 degrees
			angle := float64(siblingIdx) * goldenAngle
			// Add small hash-based jitter to prevent exact overlaps
			h := fnv.New32a()
			h.Write([]byte(dirPath))
			jitter := (float64(h.Sum32()%100) - 50) / 50.0 * 0.3
			angle += jitter
			dist := 80.0 + float64(totalSiblings)*10.0
			child.Body.Pos = Vec2{
				X: node.Body.Pos.X + math.Cos(angle)*dist,
				Y: node.Body.Pos.Y + math.Sin(angle)*dist,
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

// FindDir navigates to the directory containing the given file path.
func (d *DirNode) FindDir(path string) *DirNode {
	parts := strings.Split(path, "/")
	node := d
	for i := 0; i < len(parts)-1; i++ {
		child := node.findChild(parts[i])
		if child == nil {
			return node
		}
		node = child
	}
	return node
}

// PropagateEdgeHeat lights up edges from this node up to root.
func (d *DirNode) PropagateEdgeHeat() {
	node := d
	heat := 1.0
	for node != nil {
		if node.EdgeHeat < heat {
			node.EdgeHeat = heat
		}
		heat *= 0.7
		node = node.Parent
	}
}

// DecayEdgeHeat reduces edge heat across the tree.
func (d *DirNode) DecayEdgeHeat(rate float64) {
	d.EdgeHeat *= rate
	if d.EdgeHeat < 0.01 {
		d.EdgeHeat = 0
	}
	for _, child := range d.Children {
		child.DecayEdgeHeat(rate)
	}
}

func (d *DirNode) findChild(name string) *DirNode {
	for _, c := range d.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}
