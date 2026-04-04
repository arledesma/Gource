package model

import (
	"bytes"
	"fmt"
	"image"
	imgcolor "image/color"
	"math"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/mattn/go-sixel"
)

const (
	fileNodeRadius = 5.0
	dirNodeRadius  = 8.0
	userRadius     = 7.0
	edgeAlpha      = 0.3
	bloomSigma     = 8.0
	bloomIntensity = 0.6
)

// RenderFrame renders the current state to a sixel-encoded string.
func (m *Model) RenderFrame(width, height int) string {
	if width < 10 || height < 10 {
		return ""
	}

	dc := gg.NewContext(width, height)

	// Background
	dc.SetRGB(0.05, 0.05, 0.08)
	dc.Clear()

	// Camera: center the view on the root node, offset to center of canvas
	cx := float64(width) / 2
	cy := float64(height) / 2
	ox := cx - m.Root.Body.Pos.X
	oy := cy - m.Root.Body.Pos.Y

	// Draw directory edges (lines from parent to child)
	m.drawEdges(dc, m.Root, ox, oy)

	// Draw directory-to-file edges
	m.drawFileEdges(dc, m.Root, ox, oy)

	// Draw file nodes
	m.drawFiles(dc, m.Root, ox, oy)

	// Draw directory nodes
	m.drawDirNodes(dc, m.Root, ox, oy)

	// Draw action beams (user → file)
	m.drawBeams(dc, ox, oy)

	// Draw users
	m.drawUsers(dc, ox, oy)

	// Apply bloom post-process
	img := applyBloom(dc.Image(), bloomSigma, bloomIntensity)

	// Draw text overlay on top of bloom (date, labels)
	dc2 := gg.NewContextForImage(img)
	m.drawLabels(dc2, m.Root, ox, oy)
	m.drawUserLabels(dc2, ox, oy)
	m.drawDateOverlay(dc2, width, height)
	img = dc2.Image()

	// Encode to sixel
	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	enc.Encode(img)
	return buf.String()
}

func (m *Model) drawEdges(dc *gg.Context, node *DirNode, ox, oy float64) {
	for _, child := range node.Children {
		dc.SetRGBA(0.3, 0.4, 0.5, edgeAlpha)
		dc.SetLineWidth(1.5)
		dc.DrawLine(
			node.Body.Pos.X+ox, node.Body.Pos.Y+oy,
			child.Body.Pos.X+ox, child.Body.Pos.Y+oy,
		)
		dc.Stroke()
		m.drawEdges(dc, child, ox, oy)
	}
}

func (m *Model) drawFileEdges(dc *gg.Context, node *DirNode, ox, oy float64) {
	for _, name := range node.SortedFileNames() {
		f := node.Files[name]
		if f.State == FileRemoved {
			continue
		}
		alpha := 0.08 + f.Heat*0.15
		dc.SetRGBA(0.4, 0.4, 0.4, alpha)
		dc.SetLineWidth(0.5)
		dc.DrawLine(
			node.Body.Pos.X+ox, node.Body.Pos.Y+oy,
			f.ScreenX+ox, f.ScreenY+oy,
		)
		dc.Stroke()
	}

	for _, child := range node.Children {
		m.drawFileEdges(dc, child, ox, oy)
	}
}

func (m *Model) drawFiles(dc *gg.Context, node *DirNode, ox, oy float64) {
	for _, name := range node.SortedFileNames() {
		f := node.Files[name]
		if f.State == FileRemoved {
			continue
		}

		x := f.ScreenX + ox
		y := f.ScreenY + oy

		r, g, b, _ := f.Color.RGBA()
		rf := float64(r) / 0xFFFF
		gf := float64(g) / 0xFFFF
		bf := float64(b) / 0xFFFF

		// Glow halo when hot
		if f.Heat > 0.1 {
			glowRadius := fileNodeRadius + f.Heat*12.0
			dc.SetRGBA(rf, gf, bf, f.Heat*0.3)
			dc.DrawCircle(x, y, glowRadius)
			dc.Fill()
		}

		// File node
		alpha := 0.3 + f.Heat*0.7
		if f.State == FileRemoving {
			alpha *= 0.5
			rf = 0.9
			gf = 0.2
			bf = 0.2
		}
		radius := fileNodeRadius
		if f.Heat > 0.5 {
			radius += f.Heat * 3.0
		}
		dc.SetRGBA(rf, gf, bf, alpha)
		dc.DrawCircle(x, y, radius)
		dc.Fill()
	}

	for _, child := range node.Children {
		m.drawFiles(dc, child, ox, oy)
	}
}

func (m *Model) drawDirNodes(dc *gg.Context, node *DirNode, ox, oy float64) {
	if node.Name == "" && node.Parent == nil {
		// Skip root with no name
	} else {
		x := node.Body.Pos.X + ox
		y := node.Body.Pos.Y + oy

		// Dir glow
		maxHeat := 0.0
		for _, f := range node.Files {
			if f.Heat > maxHeat {
				maxHeat = f.Heat
			}
		}
		if maxHeat > 0.1 {
			dc.SetRGBA(0.4, 0.6, 0.9, maxHeat*0.25)
			dc.DrawCircle(x, y, dirNodeRadius+maxHeat*8.0)
			dc.Fill()
		}

		alpha := 0.4 + maxHeat*0.6
		dc.SetRGBA(0.3, 0.5, 0.8, alpha)
		dc.DrawCircle(x, y, dirNodeRadius)
		dc.Fill()
	}

	for _, child := range node.Children {
		m.drawDirNodes(dc, child, ox, oy)
	}
}

func (m *Model) drawBeams(dc *gg.Context, ox, oy float64) {
	for _, a := range m.Actions {
		user, uok := m.Users[a.Username]
		file, fok := m.Files[a.FilePath]
		if !uok || !fok || !user.Active {
			continue
		}

		r, g, b, _ := user.Color.RGBA()
		rf := float64(r) / 0xFFFF
		gf := float64(g) / 0xFFFF
		bf := float64(b) / 0xFFFF

		alpha := (1.0 - a.Progress) * 0.6
		dc.SetRGBA(rf, gf, bf, alpha)
		dc.SetLineWidth(2.0 * (1.0 - a.Progress))
		dc.DrawLine(
			user.Body.Pos.X+ox, user.Body.Pos.Y+oy,
			file.ScreenX+ox, file.ScreenY+oy,
		)
		dc.Stroke()
	}
}

func (m *Model) drawUsers(dc *gg.Context, ox, oy float64) {
	for _, u := range m.Users {
		if !u.Active {
			continue
		}

		x := u.Body.Pos.X + ox
		y := u.Body.Pos.Y + oy

		r, g, b, _ := u.Color.RGBA()
		rf := float64(r) / 0xFFFF
		gf := float64(g) / 0xFFFF
		bf := float64(b) / 0xFFFF

		// User glow
		dc.SetRGBA(rf, gf, bf, 0.2)
		dc.DrawCircle(x, y, userRadius+5)
		dc.Fill()

		// User circle
		dc.SetRGBA(rf, gf, bf, 0.9)
		dc.DrawCircle(x, y, userRadius)
		dc.Fill()
	}
}

func (m *Model) drawLabels(dc *gg.Context, node *DirNode, ox, oy float64) {
	if node.Name != "" {
		x := node.Body.Pos.X + ox
		y := node.Body.Pos.Y + oy
		dc.SetRGBA(0.7, 0.8, 1.0, 0.8)
		dc.DrawStringAnchored(node.Name, x, y-dirNodeRadius-4, 0.5, 0.5)
	}

	for _, child := range node.Children {
		m.drawLabels(dc, child, ox, oy)
	}
}

func (m *Model) drawUserLabels(dc *gg.Context, ox, oy float64) {
	for _, u := range m.Users {
		if !u.Active {
			continue
		}
		x := u.Body.Pos.X + ox
		y := u.Body.Pos.Y + oy
		dc.SetRGBA(1, 1, 1, 0.9)
		dc.DrawStringAnchored(u.Name, x, y-userRadius-4, 0.5, 0.5)
	}
}

func (m *Model) drawDateOverlay(dc *gg.Context, width, height int) {
	if m.Playback.CurrTime.IsZero() {
		return
	}

	dateStr := m.Playback.CurrTime.Format("2006-01-02")

	// Background bar
	dc.SetRGBA(0, 0, 0, 0.6)
	dc.DrawRectangle(0, float64(height-24), float64(width), 24)
	dc.Fill()

	// Date text
	dc.SetRGB(0.9, 0.9, 0.9)
	dc.DrawStringAnchored(dateStr, 10, float64(height-12), 0, 0.5)

	// Progress bar
	progress := m.Playback.Progress()
	barX := 140.0
	barW := float64(width) - barX - 200
	if barW > 20 {
		dc.SetRGBA(0.3, 0.3, 0.3, 0.8)
		dc.DrawRoundedRectangle(barX, float64(height-18), barW, 12, 3)
		dc.Fill()

		dc.SetRGBA(0.3, 0.6, 1.0, 0.8)
		dc.DrawRoundedRectangle(barX, float64(height-18), barW*progress, 12, 3)
		dc.Fill()
	}

	// File/user counts and speed
	infoStr := ""
	activeUsers := 0
	for _, u := range m.Users {
		if u.Active {
			activeUsers++
		}
	}
	infoStr = fmt.Sprintf("%d files  %d users  %.1f d/s", len(m.Files), activeUsers, m.Playback.DaysPerSecond)
	if m.Playback.Paused {
		infoStr += "  PAUSED"
	}
	dc.SetRGB(0.7, 0.7, 0.7)
	dc.DrawStringAnchored(infoStr, float64(width-10), float64(height-12), 1.0, 0.5)
}

// applyBloom creates a bloom glow effect by blurring bright areas and blending back.
func applyBloom(src image.Image, sigma, intensity float64) image.Image {
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Extract bright pixels
	bright := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			luminance := (float64(r) + float64(g) + float64(b)) / (3.0 * 0xFFFF)
			if luminance > 0.3 {
				factor := math.Min(1.0, (luminance-0.3)/0.7)
				bright.Set(x, y, imgcolor.RGBA{
					R: uint8(float64(r>>8) * factor),
					G: uint8(float64(g>>8) * factor),
					B: uint8(float64(b>>8) * factor),
					A: uint8(a >> 8),
				})
			}
		}
	}

	// Blur the bright pixels
	blurred := imaging.Blur(bright, sigma)

	// Composite: src + blurred * intensity (additive blend)
	result := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sr, sg, sb, sa := src.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			br, bg, bb, _ := blurred.At(x, y).RGBA()

			rr := math.Min(float64(sr)/0xFFFF+float64(br)/0xFFFF*intensity, 1.0)
			rg := math.Min(float64(sg)/0xFFFF+float64(bg)/0xFFFF*intensity, 1.0)
			rb := math.Min(float64(sb)/0xFFFF+float64(bb)/0xFFFF*intensity, 1.0)
			ra := float64(sa) / 0xFFFF

			result.Set(x, y, imgcolor.RGBA{
				R: uint8(rr * 255),
				G: uint8(rg * 255),
				B: uint8(rb * 255),
				A: uint8(ra * 255),
			})
		}
	}

	return result
}
