package model

// Action represents an in-progress user→file interaction.
type Action struct {
	Username string
	FilePath string
	Type     string  // "A", "M", "D"
	Progress float64 // 0.0 to 1.0
}

// Update advances action progress. Returns true when complete.
func (a *Action) Update(dt float64) bool {
	a.Progress += dt * 2.0 // complete in ~0.5 seconds
	return a.Progress >= 1.0
}
