package model

import (
	"image/color"
	"math"
	"math/rand"
)

// Particle is a short-lived visual effect (creation sparkle, deletion burst).
type Particle struct {
	Pos      Vec2
	Vel      Vec2
	Color    color.Color
	Life     float64 // remaining life 1.0 → 0.0
	MaxLife  float64
	Size     float64
}

// ParticleSystem manages active particles.
type ParticleSystem struct {
	Particles []*Particle
}

// Emit spawns n particles at position with given color.
func (ps *ParticleSystem) Emit(pos Vec2, c color.Color, n int, speed, life float64) {
	for range n {
		angle := rand.Float64() * 2 * math.Pi
		spd := speed * (0.5 + rand.Float64())
		ps.Particles = append(ps.Particles, &Particle{
			Pos:     pos,
			Vel:     Vec2{math.Cos(angle) * spd, math.Sin(angle) * spd},
			Color:   c,
			Life:    life,
			MaxLife: life,
			Size:    2.0 + rand.Float64()*2.0,
		})
	}
}

// Update advances all particles by dt. Removes dead particles.
func (ps *ParticleSystem) Update(dt float64) {
	alive := ps.Particles[:0]
	for _, p := range ps.Particles {
		p.Life -= dt
		if p.Life <= 0 {
			continue
		}
		p.Pos = p.Pos.Add(p.Vel.Scale(dt))
		p.Vel = p.Vel.Scale(0.96) // drag
		alive = append(alive, p)
	}
	ps.Particles = alive
}
