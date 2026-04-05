package model

import "image/color"

// Caption is a floating text message displayed near an entity.
type Caption struct {
	Text  string
	Pos   Vec2
	Color color.Color
	Life  float64 // remaining life (seconds)
	Alpha float64 // current alpha
}

// CaptionSystem manages active captions.
type CaptionSystem struct {
	Captions []*Caption
	MaxShow  int
}

// NewCaptionSystem creates a caption system showing at most n captions.
func NewCaptionSystem(maxShow int) *CaptionSystem {
	return &CaptionSystem{MaxShow: maxShow}
}

// Add creates a new caption.
func (cs *CaptionSystem) Add(text string, pos Vec2, c color.Color) {
	cs.Captions = append(cs.Captions, &Caption{
		Text:  text,
		Pos:   pos,
		Color: c,
		Life:  3.0,
		Alpha: 1.0,
	})

	// Limit total captions
	if len(cs.Captions) > cs.MaxShow*2 {
		cs.Captions = cs.Captions[len(cs.Captions)-cs.MaxShow:]
	}
}

// Update advances all captions.
func (cs *CaptionSystem) Update(dt float64) {
	alive := cs.Captions[:0]
	for _, c := range cs.Captions {
		c.Life -= dt
		if c.Life <= 0 {
			continue
		}
		// Fade out in the last second
		if c.Life < 1.0 {
			c.Alpha = c.Life
		}
		alive = append(alive, c)
	}
	cs.Captions = alive
}

// Visible returns the most recent N captions that are still alive.
func (cs *CaptionSystem) Visible() []*Caption {
	if len(cs.Captions) <= cs.MaxShow {
		return cs.Captions
	}
	return cs.Captions[len(cs.Captions)-cs.MaxShow:]
}
