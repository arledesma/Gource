package model

import (
	"math"
)

// Vec2 is a 2D vector for positions and forces.
type Vec2 struct {
	X, Y float64
}

func (v Vec2) Add(o Vec2) Vec2    { return Vec2{v.X + o.X, v.Y + o.Y} }
func (v Vec2) Sub(o Vec2) Vec2    { return Vec2{v.X - o.X, v.Y - o.Y} }
func (v Vec2) Scale(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }
func (v Vec2) Len() float64        { return math.Hypot(v.X, v.Y) }

func (v Vec2) Normalize() Vec2 {
	l := v.Len()
	if l < 0.0001 {
		return Vec2{0, 0}
	}
	return Vec2{v.X / l, v.Y / l}
}

// PhysicsBody holds position, velocity, acceleration for an entity.
type PhysicsBody struct {
	Pos Vec2
	Vel Vec2
	Acc Vec2
}

// Integrate applies velocity and acceleration over dt, with damping.
func (b *PhysicsBody) Integrate(dt, damping float64) {
	b.Vel = b.Vel.Add(b.Acc.Scale(dt))
	b.Vel = b.Vel.Scale(damping)
	b.Pos = b.Pos.Add(b.Vel.Scale(dt))
	b.Acc = Vec2{}
}

// ApplyForce adds a force to the accumulator.
func (b *PhysicsBody) ApplyForce(f Vec2) {
	b.Acc = b.Acc.Add(f)
}

const (
	springStiffness = 0.6   // attraction to parent
	repulsion       = 12000 // repulsion between siblings
	centerGravity   = 0.015 // pull toward origin
	damping         = 0.82  // velocity damping per frame
	maxSpeed        = 300.0 // clamp velocity
)

// UpdateLayout runs one step of the force-directed layout.
func UpdateLayout(root *DirNode, dt float64) {
	if root == nil {
		return
	}

	// Collect all dir nodes
	var allDirs []*DirNode
	collectDirs(root, &allDirs)

	// Apply forces
	for _, node := range allDirs {
		if node == root {
			continue
		}

		// Spring attraction toward parent
		if node.Parent != nil {
			delta := node.Parent.Body.Pos.Sub(node.Body.Pos)
			dist := delta.Len()
			// More files in directory = more spacing needed
			fileCount := float64(len(node.Files))
			childCount := float64(len(node.Parent.Children))
			targetDist := 100.0 + childCount*25.0 + fileCount*5.0
			force := delta.Normalize().Scale((dist - targetDist) * springStiffness)
			node.Body.ApplyForce(force)
		}

		// Repulsion from all other directories
		for _, other := range allDirs {
			if other == node || other == root {
				continue
			}
			delta := node.Body.Pos.Sub(other.Body.Pos)
			dist := delta.Len()
			if dist < 1.0 {
				dist = 1.0
			}
			if dist < 500 {
				force := delta.Normalize().Scale(repulsion / (dist * dist))
				node.Body.ApplyForce(force)
			}
		}

		// Gravity toward center
		node.Body.ApplyForce(node.Body.Pos.Scale(-centerGravity))
	}

	// Integrate
	for _, node := range allDirs {
		if node == root {
			continue
		}
		node.Body.Integrate(dt, damping)

		// Clamp speed
		speed := node.Body.Vel.Len()
		if speed > maxSpeed {
			node.Body.Vel = node.Body.Vel.Normalize().Scale(maxSpeed)
		}
	}

	// Position files in orbits around their parent directory
	for _, node := range allDirs {
		positionFiles(node)
	}
}

// positionFiles arranges files in concentric circles around a directory node,
// matching Gource's concentric ring layout.
func positionFiles(node *DirNode) {
	names := node.SortedFileNames()
	if len(names) == 0 {
		return
	}

	fileRadius := 12.0
	ringRadius := 30.0
	maxPerRing := int(math.Max(1, math.Pi*ringRadius/fileRadius))

	i := 0
	for _, name := range names {
		f := node.Files[name]

		ring := i / maxPerRing
		posInRing := i % maxPerRing
		currentRingRadius := ringRadius + float64(ring)*fileRadius*2.5

		countInRing := maxPerRing
		remaining := len(names) - ring*maxPerRing
		if remaining < maxPerRing {
			countInRing = remaining
		}

		angle := 2.0 * math.Pi * float64(posInRing) / float64(countInRing)

		f.ScreenX = node.Body.Pos.X + math.Cos(angle)*currentRingRadius
		f.ScreenY = node.Body.Pos.Y + math.Sin(angle)*currentRingRadius
		i++
	}
}

func collectDirs(node *DirNode, out *[]*DirNode) {
	*out = append(*out, node)
	for _, child := range node.Children {
		collectDirs(child, out)
	}
}
