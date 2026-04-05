package model

import (
	"bytes"
	"fmt"
	"image"
	imgcolor "image/color"
	"math"
	"sync"

	"github.com/acaudwell/gource-tui/config"
	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/mattn/go-sixel"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

const (
	fileNodeRadius = 5.0
	dirNodeRadius  = 8.0
	userRadius     = 7.0
	edgeAlpha      = 0.3
	bloomSigma     = 6.0
	bloomIntensity = 0.5
	bloomScale     = 0.5 // render bloom at half resolution
)

// Cached font faces — loaded once.
var (
	fontOnce     sync.Once
	fontRegular  font.Face
	fontBold     font.Face
	fontSmall    font.Face
	fontStatus   font.Face
)

func loadFonts() {
	fontOnce.Do(func() {
		ttReg, _ := opentype.Parse(goregular.TTF)
		ttBold, _ := opentype.Parse(gobold.TTF)

		fontRegular, _ = opentype.NewFace(ttReg, &opentype.FaceOptions{Size: 11, DPI: 72})
		fontBold, _ = opentype.NewFace(ttBold, &opentype.FaceOptions{Size: 12, DPI: 72})
		fontSmall, _ = opentype.NewFace(ttReg, &opentype.FaceOptions{Size: 9, DPI: 72})
		fontStatus, _ = opentype.NewFace(ttReg, &opentype.FaceOptions{Size: 13, DPI: 72})
	})
}

// camera holds the computed camera transform for a frame.
type camera struct {
	ox, oy float64 // pixel offset
	scale  float64 // zoom scale
}

// computeCamera calculates auto-fit or manual camera transform.
func (m *Model) computeCamera(width, height float64) camera {
	// Manual zoom mode
	if m.CameraZoom > 0 {
		return camera{
			ox:    width/2 + m.CameraOffset.X,
			oy:    height/2 + m.CameraOffset.Y,
			scale: m.CameraZoom,
		}
	}

	// Auto-fit: compute bounding box of all entities
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	count := 0

	var expandBounds func(node *DirNode)
	expandBounds = func(node *DirNode) {
		if node.Name != "" || node.Parent == nil {
			px, py := node.Body.Pos.X, node.Body.Pos.Y
			if px < minX { minX = px }
			if py < minY { minY = py }
			if px > maxX { maxX = px }
			if py > maxY { maxY = py }
			count++
		}
		for _, name := range node.SortedFileNames() {
			f := node.Files[name]
			if f.State == FileRemoved {
				continue
			}
			if f.ScreenX < minX { minX = f.ScreenX }
			if f.ScreenY < minY { minY = f.ScreenY }
			if f.ScreenX > maxX { maxX = f.ScreenX }
			if f.ScreenY > maxY { maxY = f.ScreenY }
			count++
		}
		for _, child := range node.Children {
			expandBounds(child)
		}
	}
	expandBounds(m.Root)

	if count == 0 {
		return camera{ox: width / 2, oy: height / 2, scale: 1.0}
	}

	// Add padding
	pad := 60.0
	minX -= pad
	minY -= pad
	maxX += pad
	maxY += pad

	boundsW := maxX - minX
	boundsH := maxY - minY
	if boundsW < 1 { boundsW = 1 }
	if boundsH < 1 { boundsH = 1 }

	// Scale to fit, leave room for status bar
	usableH := height - 30
	scaleX := width / boundsW
	scaleY := usableH / boundsH
	scale := math.Min(scaleX, scaleY) * 0.90

	// Clamp scale
	if scale > 3.0 { scale = 3.0 }
	if scale < 0.05 { scale = 0.05 }

	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2

	return camera{
		ox:    width/2 - centerX*scale + m.CameraOffset.X,
		oy:    usableH/2 - centerY*scale + m.CameraOffset.Y,
		scale: scale,
	}
}

// worldToScreen converts world coordinates to screen coordinates.
func (c camera) worldToScreen(wx, wy float64) (float64, float64) {
	return wx*c.scale + c.ox, wy*c.scale + c.oy
}

// renderImage renders the current state to an image.Image.
func (m *Model) renderImage(width, height int) image.Image {
	loadFonts()

	dc := gg.NewContext(width, height)
	// Background color
	if m.Settings.Background != "" {
		bgColor := config.ColorForExtension("") // fallback
		if c := parseHexToRGB(m.Settings.Background); c != nil {
			bgColor = c
		}
		r, g, b, _ := bgColor.RGBA()
		dc.SetRGB(float64(r)/0xFFFF, float64(g)/0xFFFF, float64(b)/0xFFFF)
	} else {
		dc.SetRGB(0.05, 0.05, 0.08)
	}
	dc.Clear()

	cam := m.computeCamera(float64(width), float64(height))

	// Draw layers back-to-front
	m.drawEdges(dc, m.Root, cam)
	m.drawFileEdges(dc, m.Root, cam)
	m.drawFiles(dc, m.Root, cam)
	m.drawDirNodes(dc, m.Root, cam)
	m.drawBeams(dc, cam)
	m.drawUsers(dc, cam)
	m.drawParticles(dc, cam)

	// Bloom
	if !m.Settings.NoBloom {
		img := applyBloom(dc.Image(), bloomSigma, bloomIntensity)
		dc = gg.NewContextForImage(img)
	}

	// Text overlay on top of bloom
	dc.SetFontFace(fontBold)
	m.drawLabels(dc, m.Root, cam)
	dc.SetFontFace(fontSmall)
	m.drawFileLabels(dc, m.Root, cam)
	dc.SetFontFace(fontRegular)
	m.drawUserLabels(dc, cam)
	dc.SetFontFace(fontStatus)
	m.drawDateOverlay(dc, width, height)
	if m.ShowLegend {
		m.drawLegend(dc, width)
	}
	if m.ShowHelp {
		m.drawHelp(dc, width, height)
	}

	return dc.Image()
}

// RenderFrame renders the current state to a sixel-encoded string.
func (m *Model) RenderFrame(width, height int) string {
	if width < 10 || height < 10 {
		return ""
	}

	img := m.renderImage(width, height)

	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	enc.Encode(img)
	return buf.String()
}

// RenderToPNG renders the current state to an image (for PNG export/testing).
func (m *Model) RenderToPNG(width, height int) image.Image {
	return m.renderImage(width, height)
}

func (m *Model) drawEdges(dc *gg.Context, node *DirNode, cam camera) {
	for _, child := range node.Children {
		dc.SetRGBA(0.3, 0.4, 0.5, edgeAlpha)
		dc.SetLineWidth(math.Max(0.5, 1.5*cam.scale))
		x1, y1 := cam.worldToScreen(node.Body.Pos.X, node.Body.Pos.Y)
		x2, y2 := cam.worldToScreen(child.Body.Pos.X, child.Body.Pos.Y)

		// Quadratic bezier with control point offset perpendicular to the edge
		mx := (x1 + x2) / 2
		my := (y1 + y2) / 2
		dx := x2 - x1
		dy := y2 - y1
		dist := math.Hypot(dx, dy)
		if dist > 0 {
			// Perpendicular offset proportional to distance
			offset := dist * 0.1
			cx := mx + (-dy/dist)*offset
			cy := my + (dx/dist)*offset
			dc.MoveTo(x1, y1)
			dc.QuadraticTo(cx, cy, x2, y2)
			dc.Stroke()
		} else {
			dc.DrawLine(x1, y1, x2, y2)
			dc.Stroke()
		}

		m.drawEdges(dc, child, cam)
	}
}

func (m *Model) drawFileEdges(dc *gg.Context, node *DirNode, cam camera) {
	nx, ny := cam.worldToScreen(node.Body.Pos.X, node.Body.Pos.Y)
	for _, name := range node.SortedFileNames() {
		f := node.Files[name]
		if f.State == FileRemoved {
			continue
		}
		alpha := 0.06 + f.Heat*0.12
		dc.SetRGBA(0.4, 0.4, 0.4, alpha)
		dc.SetLineWidth(math.Max(0.3, 0.5*cam.scale))
		fx, fy := cam.worldToScreen(f.ScreenX, f.ScreenY)
		dc.DrawLine(nx, ny, fx, fy)
		dc.Stroke()
	}
	for _, child := range node.Children {
		m.drawFileEdges(dc, child, cam)
	}
}

func (m *Model) drawFiles(dc *gg.Context, node *DirNode, cam camera) {
	for _, name := range node.SortedFileNames() {
		f := node.Files[name]
		if f.State == FileRemoved {
			continue
		}

		x, y := cam.worldToScreen(f.ScreenX, f.ScreenY)

		r, g, b, _ := f.Color.RGBA()
		rf := float64(r) / 0xFFFF
		gf := float64(g) / 0xFFFF
		bf := float64(b) / 0xFFFF

		// Glow halo when hot
		if f.Heat > 0.1 {
			glowR := (fileNodeRadius + f.Heat*12.0) * cam.scale
			dc.SetRGBA(rf, gf, bf, f.Heat*0.3)
			dc.DrawCircle(x, y, glowR)
			dc.Fill()
		}

		// File node
		alpha := 0.3 + f.Heat*0.7
		if f.State == FileRemoving {
			alpha *= 0.5
			rf, gf, bf = 0.9, 0.2, 0.2
		}
		radius := fileNodeRadius * cam.scale
		if f.Heat > 0.5 {
			radius += f.Heat * 3.0 * cam.scale
		}
		dc.SetRGBA(rf, gf, bf, alpha)
		dc.DrawCircle(x, y, radius)
		dc.Fill()
	}
	for _, child := range node.Children {
		m.drawFiles(dc, child, cam)
	}
}

func (m *Model) drawDirNodes(dc *gg.Context, node *DirNode, cam camera) {
	if node.Name != "" || node.Parent != nil {
		x, y := cam.worldToScreen(node.Body.Pos.X, node.Body.Pos.Y)
		ds := depthScale(node.Depth)

		maxHeat := 0.0
		for _, f := range node.Files {
			if f.Heat > maxHeat {
				maxHeat = f.Heat
			}
		}

		// Dir glow
		if maxHeat > 0.1 {
			glowR := (dirNodeRadius*ds + maxHeat*8.0) * cam.scale
			dc.SetRGBA(0.4, 0.6, 0.9, maxHeat*0.25)
			dc.DrawCircle(x, y, glowR)
			dc.Fill()
		}

		alpha := 0.4 + maxHeat*0.6
		dc.SetRGBA(0.3, 0.5, 0.8, alpha)
		dc.DrawCircle(x, y, dirNodeRadius*ds*cam.scale)
		dc.Fill()
	}
	for _, child := range node.Children {
		m.drawDirNodes(dc, child, cam)
	}
}

func (m *Model) drawBeams(dc *gg.Context, cam camera) {
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
		dc.SetLineWidth(math.Max(0.5, 2.0*(1.0-a.Progress)*cam.scale))
		ux, uy := cam.worldToScreen(user.Body.Pos.X, user.Body.Pos.Y)
		fx, fy := cam.worldToScreen(file.ScreenX, file.ScreenY)
		dc.DrawLine(ux, uy, fx, fy)
		dc.Stroke()
	}
}

func (m *Model) drawUsers(dc *gg.Context, cam camera) {
	for _, u := range m.Users {
		if !u.Active {
			continue
		}
		x, y := cam.worldToScreen(u.Body.Pos.X, u.Body.Pos.Y)

		r, g, b, _ := u.Color.RGBA()
		rf := float64(r) / 0xFFFF
		gf := float64(g) / 0xFFFF
		bf := float64(b) / 0xFFFF

		// Glow
		dc.SetRGBA(rf, gf, bf, 0.2)
		dc.DrawCircle(x, y, (userRadius+5)*cam.scale)
		dc.Fill()

		// Circle
		dc.SetRGBA(rf, gf, bf, 0.9)
		dc.DrawCircle(x, y, userRadius*cam.scale)
		dc.Fill()
	}
}

func (m *Model) drawLabels(dc *gg.Context, node *DirNode, cam camera) {
	if node.Name != "" {
		x, y := cam.worldToScreen(node.Body.Pos.X, node.Body.Pos.Y)
		dc.SetRGBA(0.7, 0.8, 1.0, 0.85)
		dc.DrawStringAnchored(node.Name, x, y-dirNodeRadius*cam.scale-6, 0.5, 1.0)
	}
	for _, child := range node.Children {
		m.drawLabels(dc, child, cam)
	}
}

// drawFileLabels shows filenames on hot files.
func (m *Model) drawFileLabels(dc *gg.Context, node *DirNode, cam camera) {
	for _, name := range node.SortedFileNames() {
		f := node.Files[name]
		if f.Heat < 0.4 || f.State == FileRemoved {
			continue
		}
		x, y := cam.worldToScreen(f.ScreenX, f.ScreenY)
		alpha := math.Min(1.0, f.Heat)
		dc.SetRGBA(0.9, 0.9, 0.9, alpha*0.8)
		dc.DrawStringAnchored(f.Name, x+fileNodeRadius*cam.scale+3, y, 0, 0.5)
	}
	for _, child := range node.Children {
		m.drawFileLabels(dc, child, cam)
	}
}

func (m *Model) drawUserLabels(dc *gg.Context, cam camera) {
	for _, u := range m.Users {
		if !u.Active {
			continue
		}
		x, y := cam.worldToScreen(u.Body.Pos.X, u.Body.Pos.Y)
		dc.SetRGBA(1, 1, 1, 0.9)
		dc.DrawStringAnchored(u.Name, x, y-userRadius*cam.scale-5, 0.5, 1.0)
	}
}

func (m *Model) drawDateOverlay(dc *gg.Context, width, height int) {
	if m.Playback.CurrTime.IsZero() {
		return
	}

	dateStr := m.Playback.CurrTime.Format("2006-01-02")

	// Background bar
	barH := 28.0
	dc.SetRGBA(0, 0, 0, 0.7)
	dc.DrawRectangle(0, float64(height)-barH, float64(width), barH)
	dc.Fill()

	// Date text
	dc.SetRGB(0.9, 0.9, 0.9)
	dc.DrawStringAnchored(dateStr, 12, float64(height)-barH/2, 0, 0.5)

	// Progress bar
	progress := m.Playback.Progress()
	barX := 160.0
	barW := float64(width) - barX - 280
	if barW > 20 {
		dc.SetRGBA(0.2, 0.2, 0.25, 0.9)
		dc.DrawRoundedRectangle(barX, float64(height)-barH/2-6, barW, 12, 4)
		dc.Fill()

		if progress > 0 {
			dc.SetRGBA(0.3, 0.6, 1.0, 0.9)
			dc.DrawRoundedRectangle(barX, float64(height)-barH/2-6, barW*progress, 12, 4)
			dc.Fill()
		}
	}

	// Info
	activeUsers := 0
	for _, u := range m.Users {
		if u.Active {
			activeUsers++
		}
	}
	infoStr := fmt.Sprintf("%d files   %d users   %.1f d/s", len(m.Files), activeUsers, m.Playback.DaysPerSecond)
	if m.Playback.Paused {
		infoStr += "   PAUSED"
	}
	dc.SetRGB(0.6, 0.65, 0.7)
	dc.DrawStringAnchored(infoStr, float64(width)-12, float64(height)-barH/2, 1.0, 0.5)
}

func (m *Model) drawParticles(dc *gg.Context, cam camera) {
	for _, p := range m.Particles.Particles {
		x, y := cam.worldToScreen(p.Pos.X, p.Pos.Y)
		r, g, b, _ := p.Color.RGBA()
		rf := float64(r) / 0xFFFF
		gf := float64(g) / 0xFFFF
		bf := float64(b) / 0xFFFF

		t := p.Life / p.MaxLife // 1.0 → 0.0
		alpha := t * 0.8
		size := p.Size * t * cam.scale

		dc.SetRGBA(rf, gf, bf, alpha)
		dc.DrawCircle(x, y, size)
		dc.Fill()
	}
}

func (m *Model) drawLegend(dc *gg.Context, width int) {
	// Count files by extension
	extCounts := make(map[string]int)
	for _, f := range m.Files {
		if f.State != FileRemoved {
			extCounts[f.Extension]++
		}
	}
	if len(extCounts) == 0 {
		return
	}

	// Sort by count descending
	type extEntry struct {
		ext   string
		count int
	}
	var entries []extEntry
	for ext, count := range extCounts {
		entries = append(entries, extEntry{ext, count})
	}
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].count > entries[i].count {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Limit to top 15
	if len(entries) > 15 {
		entries = entries[:15]
	}

	// Draw background panel
	lineH := 18.0
	panelW := 140.0
	panelH := float64(len(entries))*lineH + 30
	px := float64(width) - panelW - 10
	py := 10.0

	dc.SetRGBA(0, 0, 0, 0.6)
	dc.DrawRoundedRectangle(px, py, panelW, panelH, 6)
	dc.Fill()

	dc.SetFontFace(fontSmall)
	dc.SetRGBA(0.7, 0.8, 0.9, 0.9)
	dc.DrawStringAnchored("File Types", px+panelW/2, py+12, 0.5, 0.5)

	for i, e := range entries {
		y := py + 26 + float64(i)*lineH

		// Color swatch
		c := config.ColorForExtension(e.ext)
		r, g, b, _ := c.RGBA()
		dc.SetRGBA(float64(r)/0xFFFF, float64(g)/0xFFFF, float64(b)/0xFFFF, 0.9)
		dc.DrawRoundedRectangle(px+8, y-4, 10, 10, 2)
		dc.Fill()

		// Label
		ext := e.ext
		if ext == "" {
			ext = "(none)"
		}
		dc.SetRGBA(0.8, 0.8, 0.8, 0.8)
		dc.DrawString(fmt.Sprintf("%s  %d", ext, e.count), px+24, y+5)
	}
}

func (m *Model) drawHelp(dc *gg.Context, width, height int) {
	lines := []string{
		"Controls:",
		"",
		"Space     Pause/Resume",
		"+/-       Speed up/down",
		"z/x       Zoom in/out",
		"Arrows    Pan camera",
		"Home      Reset camera",
		"l         Toggle legend",
		"?         Toggle help",
		"q         Quit",
	}

	lineH := 20.0
	panelW := 220.0
	panelH := float64(len(lines))*lineH + 20
	px := (float64(width) - panelW) / 2
	py := (float64(height) - panelH) / 2

	dc.SetRGBA(0, 0, 0, 0.8)
	dc.DrawRoundedRectangle(px, py, panelW, panelH, 8)
	dc.Fill()

	dc.SetRGBA(0.15, 0.15, 0.2, 0.9)
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(px, py, panelW, panelH, 8)
	dc.Stroke()

	dc.SetFontFace(fontRegular)
	for i, line := range lines {
		y := py + 16 + float64(i)*lineH
		if i == 0 {
			dc.SetRGBA(0.9, 0.9, 1.0, 1.0)
			dc.SetFontFace(fontBold)
			dc.DrawStringAnchored(line, px+panelW/2, y, 0.5, 0.5)
			dc.SetFontFace(fontRegular)
		} else {
			dc.SetRGBA(0.7, 0.75, 0.8, 0.9)
			dc.DrawString(line, px+16, y)
		}
	}
}

// applyBloom creates a bloom glow effect using a downscaled blur for performance.
func applyBloom(src image.Image, sigma, intensity float64) image.Image {
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Downscale dimensions
	sw := int(float64(w) * bloomScale)
	sh := int(float64(h) * bloomScale)
	if sw < 1 { sw = 1 }
	if sh < 1 { sh = 1 }

	// Extract bright pixels at reduced resolution
	bright := image.NewRGBA(image.Rect(0, 0, sw, sh))
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			// Sample from source at corresponding position
			srcX := bounds.Min.X + x*w/sw
			srcY := bounds.Min.Y + y*h/sh
			r, g, b, a := src.At(srcX, srcY).RGBA()
			luminance := (float64(r) + float64(g) + float64(b)) / (3.0 * 0xFFFF)
			if luminance > 0.25 {
				factor := math.Min(1.0, (luminance-0.25)/0.5)
				bright.SetRGBA(x, y, imgcolor.RGBA{
					R: uint8(float64(r>>8) * factor),
					G: uint8(float64(g>>8) * factor),
					B: uint8(float64(b>>8) * factor),
					A: uint8(a >> 8),
				})
			}
		}
	}

	// Blur at reduced resolution (much faster)
	blurred := imaging.Blur(bright, sigma*bloomScale)

	// Upscale back
	upscaled := imaging.Resize(blurred, w, h, imaging.Linear)

	// Additive composite
	result := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sr, sg, sb, sa := src.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			br, bg, bb, _ := upscaled.At(x, y).RGBA()

			rr := math.Min(float64(sr)/0xFFFF+float64(br)/0xFFFF*intensity, 1.0)
			rg := math.Min(float64(sg)/0xFFFF+float64(bg)/0xFFFF*intensity, 1.0)
			rb := math.Min(float64(sb)/0xFFFF+float64(bb)/0xFFFF*intensity, 1.0)
			ra := float64(sa) / 0xFFFF

			result.SetRGBA(x, y, imgcolor.RGBA{
				R: uint8(rr * 255),
				G: uint8(rg * 255),
				B: uint8(rb * 255),
				A: uint8(ra * 255),
			})
		}
	}

	return result
}

// depthScale returns a size multiplier based on tree depth (deeper = smaller).
func depthScale(depth int) float64 {
	return math.Pow(0.85, float64(depth))
}

func parseHexToRGB(hex string) imgcolor.Color {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return nil
	}
	var r, g, b uint8
	_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return nil
	}
	return imgcolor.RGBA{R: r, G: g, B: b, A: 255}
}
