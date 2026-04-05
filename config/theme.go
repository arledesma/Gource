package config

import "image/color"

// Theme defines the color palette for the visualization.
type Theme struct {
	Name       string
	Background color.RGBA
	EdgeColor  [3]float64 // RGB 0-1
	DirNode    [3]float64
	DirLabel   [4]float64 // RGBA 0-1
	UserLabel  [4]float64
	StatusBg   color.RGBA
	StatusFg   color.RGBA
	TextDim    color.RGBA
}

var Themes = map[string]Theme{
	"dark": {
		Name:       "dark",
		Background: color.RGBA{R: 13, G: 13, B: 20, A: 255},
		EdgeColor:  [3]float64{0.3, 0.4, 0.5},
		DirNode:    [3]float64{0.3, 0.5, 0.8},
		DirLabel:   [4]float64{0.7, 0.8, 1.0, 0.85},
		UserLabel:  [4]float64{1.0, 1.0, 1.0, 0.9},
		StatusBg:   color.RGBA{R: 0, G: 0, B: 0, A: 178},
		StatusFg:   color.RGBA{R: 230, G: 230, B: 230, A: 255},
		TextDim:    color.RGBA{R: 153, G: 166, B: 179, A: 230},
	},
	"light": {
		Name:       "light",
		Background: color.RGBA{R: 240, G: 240, B: 245, A: 255},
		EdgeColor:  [3]float64{0.6, 0.6, 0.7},
		DirNode:    [3]float64{0.2, 0.4, 0.7},
		DirLabel:   [4]float64{0.1, 0.15, 0.3, 0.9},
		UserLabel:  [4]float64{0.1, 0.1, 0.1, 0.9},
		StatusBg:   color.RGBA{R: 30, G: 30, B: 40, A: 220},
		StatusFg:   color.RGBA{R: 230, G: 230, B: 230, A: 255},
		TextDim:    color.RGBA{R: 80, G: 80, B: 100, A: 200},
	},
	"solarized": {
		Name:       "solarized",
		Background: color.RGBA{R: 0, G: 43, B: 54, A: 255},
		EdgeColor:  [3]float64{0.33, 0.43, 0.45},
		DirNode:    [3]float64{0.15, 0.55, 0.82},
		DirLabel:   [4]float64{0.58, 0.63, 0.50, 0.9},
		UserLabel:  [4]float64{0.93, 0.91, 0.84, 0.9},
		StatusBg:   color.RGBA{R: 7, G: 54, B: 66, A: 230},
		StatusFg:   color.RGBA{R: 238, G: 232, B: 213, A: 255},
		TextDim:    color.RGBA{R: 101, G: 123, B: 131, A: 200},
	},
	"monokai": {
		Name:       "monokai",
		Background: color.RGBA{R: 39, G: 40, B: 34, A: 255},
		EdgeColor:  [3]float64{0.45, 0.44, 0.40},
		DirNode:    [3]float64{0.40, 0.85, 0.94},
		DirLabel:   [4]float64{0.90, 0.86, 0.45, 0.9},
		UserLabel:  [4]float64{0.97, 0.97, 0.95, 0.9},
		StatusBg:   color.RGBA{R: 30, G: 30, B: 26, A: 230},
		StatusFg:   color.RGBA{R: 248, G: 248, B: 242, A: 255},
		TextDim:    color.RGBA{R: 117, G: 113, B: 94, A: 200},
	},
}

// DefaultTheme is "dark".
var DefaultTheme = Themes["dark"]

// GetTheme returns a theme by name, falling back to dark.
func GetTheme(name string) Theme {
	if t, ok := Themes[name]; ok {
		return t
	}
	return DefaultTheme
}
