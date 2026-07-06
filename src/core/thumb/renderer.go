/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package thumb

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"

	"github.com/fogleman/gg"
	xdraw "golang.org/x/image/draw"
)

// Canvas dimensions.
const (
	canvasW float64 = 1920
	canvasH float64 = 1080

	// Card (poster-style panel that frames the album art + title).
	cardW = 1140.0
	cardH = 600.0
	cardY = 96.0
	cardR = 40.0

	// Album art inside the card.
	thumbW = 1030.0
	thumbH = 480.0
	thumbR = 27.0
)

// cardX and thumbX are derived from the constants above; declared as vars
// (rather than consts) because they are not integral and Go disallows
// implicit-truncating constant-to-int conversions later in the renderer.
var (
	cardX  = (canvasW - cardW) / 2
	thumbX = cardX + (cardW-thumbW)/2
	thumbY = cardY + 24.0
)

// palette drives the accent colours (neon/accent/background wash) for a
// single thumbnail render, echoing the vibrant, poster-style colour pops
// used across a rotation of looks rather than a single fixed theme.
type palette struct {
	neon   color.RGBA
	accent color.RGBA
	bg     color.RGBA
}

var palettes = []palette{
	{neon: color.RGBA{R: 0, G: 230, B: 180, A: 255}, accent: color.RGBA{R: 0, G: 120, B: 255, A: 255}, bg: color.RGBA{R: 10, G: 20, B: 40, A: 255}},
	{neon: color.RGBA{R: 255, G: 60, B: 180, A: 255}, accent: color.RGBA{R: 160, G: 0, B: 255, A: 255}, bg: color.RGBA{R: 30, G: 0, B: 30, A: 255}},
	{neon: color.RGBA{R: 255, G: 200, B: 0, A: 255}, accent: color.RGBA{R: 255, G: 90, B: 0, A: 255}, bg: color.RGBA{R: 30, G: 15, B: 0, A: 255}},
	{neon: color.RGBA{R: 60, G: 255, B: 100, A: 255}, accent: color.RGBA{R: 0, G: 180, B: 255, A: 255}, bg: color.RGBA{R: 0, G: 25, B: 15, A: 255}},
	{neon: color.RGBA{R: 255, G: 70, B: 70, A: 255}, accent: color.RGBA{R: 255, G: 190, B: 0, A: 255}, bg: color.RGBA{R: 30, G: 5, B: 5, A: 255}},
	{neon: color.RGBA{R: 140, G: 80, B: 255, A: 255}, accent: color.RGBA{R: 255, G: 80, B: 160, A: 255}, bg: color.RGBA{R: 15, G: 0, B: 30, A: 255}},
}

// pickPalette deterministically selects a palette from the track's identity
// so repeated renders of the *same* track look the same, while different
// tracks get visual variety.
func pickPalette(seed string) palette {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	return palettes[int(h.Sum32())%len(palettes)]
}

// render produces a 1920×1080 PNG thumbnail and returns it as a byte slice.
func render(d *TrackData, art image.Image) ([]byte, error) {
	gc := gg.NewContext(int(canvasW), int(canvasH))
	pal := pickPalette(d.Name + d.Channel)

	drawBackground(gc, art, pal)
	drawCard(gc, art, pal)
	titleBottomY := drawTitleBlock(gc, d, pal)
	pbY := titleBottomY + 30
	drawProgressBar(gc, 240, pbY, canvasW-480, pal, d.Status)
	drawPillRow(gc, d, pbY+34)
	drawEqualizer(gc, 56, canvasH-130, pal, d.Name, d.Status)
	drawTopBadges(gc, d, pal)
	drawBottomBar(gc, d, pal)
	drawVignette(gc)

	var buf bytes.Buffer
	if err := gc.EncodePNG(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ── Background ───────────────────────────────────────────────────────────────

func drawBackground(gc *gg.Context, art image.Image, pal palette) {
	// Heavily blurred, darkened album art as a textured backdrop.
	if art != nil {
		tiny := image.NewRGBA(image.Rect(0, 0, 40, 22))
		xdraw.BiLinear.Scale(tiny, tiny.Bounds(), art, art.Bounds(), xdraw.Over, nil)
		blurred := image.NewRGBA(image.Rect(0, 0, int(canvasW), int(canvasH)))
		xdraw.BiLinear.Scale(blurred, blurred.Bounds(), tiny, tiny.Bounds(), xdraw.Over, nil)
		gc.DrawImage(blurred, 0, 0)
	} else {
		grad := gg.NewLinearGradient(0, 0, canvasW, canvasH)
		grad.AddColorStop(0, pal.bg)
		grad.AddColorStop(1, pal.accent)
		gc.SetFillStyle(grad)
		gc.DrawRectangle(0, 0, canvasW, canvasH)
		gc.Fill()
	}

	// Palette colour wash for contrast + brand cohesion.
	washR, washG, washB, _ := pal.bg.RGBA()
	gc.SetRGBA(float64(washR>>8)/255, float64(washG>>8)/255, float64(washB>>8)/255, 0.65)
	gc.DrawRectangle(0, 0, canvasW, canvasH)
	gc.Fill()

	// Radial darkening from the edges toward the centre (vignette wash).
	grad := gg.NewRadialGradient(canvasW/2, canvasH/2, 0, canvasW/2, canvasH/2, canvasW*0.75)
	grad.AddColorStop(0, color.RGBA{A: 0})
	grad.AddColorStop(1, color.RGBA{A: 190})
	gc.SetFillStyle(grad)
	gc.DrawRectangle(0, 0, canvasW, canvasH)
	gc.Fill()

	drawNoise(gc)
}

func drawNoise(gc *gg.Context) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 4200; i++ {
		x := rng.Float64() * canvasW
		y := rng.Float64() * canvasH
		a := rng.Float64() * 0.035
		gc.SetRGBA(1, 1, 1, a)
		gc.DrawRectangle(x, y, 1, 1)
		gc.Fill()
	}
}

// ── Card ─────────────────────────────────────────────────────────────────────

// drawCard renders the central glassmorphism card: a multi-layer neon glow
// border, a translucent glass fill, and the album art (rounded, rimmed)
// centred inside it — mirroring the reference poster layout.
func drawCard(gc *gg.Context, art image.Image, pal palette) {
	nr, ng, nb := norm(pal.neon)

	// Outer multi-layer glow, spread outward from the card edge.
	for spread := 34.0; spread >= 4; spread -= 8 {
		alpha := 0.16 * (spread / 34)
		gc.SetRGBA(nr, ng, nb, alpha)
		gc.SetLineWidth(6)
		gc.DrawRoundedRectangle(cardX-spread, cardY-spread, cardW+spread*2, cardH+spread*2, cardR+spread*0.5)
		gc.Stroke()
	}

	// Glass fill.
	gc.SetRGBA(1, 1, 1, 0.05)
	gc.DrawRoundedRectangle(cardX, cardY, cardW, cardH, cardR)
	gc.Fill()

	// Glow-tinted border.
	gc.SetRGBA(nr, ng, nb, 0.55)
	gc.SetLineWidth(2.5)
	gc.DrawRoundedRectangle(cardX, cardY, cardW, cardH, cardR)
	gc.Stroke()

	// Album art (or placeholder) with a soft dark rim for depth.
	gc.Push()
	gc.DrawRoundedRectangle(thumbX-3, thumbY-3, thumbW+6, thumbH+6, thumbR+3)
	gc.Clip()
	gc.SetRGBA(0, 0, 0, 0.55)
	gc.DrawRoundedRectangle(thumbX-3, thumbY-3, thumbW+6, thumbH+6, thumbR+3)
	gc.Fill()
	gc.Pop()

	gc.Push()
	gc.DrawRoundedRectangle(thumbX, thumbY, thumbW, thumbH, thumbR)
	gc.Clip()
	if art != nil {
		sized := cropAndResize(art, int(thumbW), int(thumbH))
		gc.DrawImage(sized, int(thumbX), int(thumbY))
	} else {
		drawDefaultArt(gc, thumbX, thumbY, thumbW, thumbH, pal)
	}
	gc.Pop()

	// Bright gradient rim on top of the art.
	rimGrad := gg.NewLinearGradient(thumbX, thumbY, thumbX+thumbW, thumbY+thumbH)
	rimGrad.AddColorStop(0, pal.neon)
	rimGrad.AddColorStop(0.5, pal.accent)
	rimGrad.AddColorStop(1, pal.neon)
	gc.SetStrokeStyle(rimGrad)
	gc.SetLineWidth(4)
	gc.DrawRoundedRectangle(thumbX, thumbY, thumbW, thumbH, thumbR)
	gc.Stroke()
}

// drawDefaultArt renders a premium gradient placeholder with a music emblem
// when no album art could be fetched.
func drawDefaultArt(gc *gg.Context, x, y, w, h float64, pal palette) {
	grad := gg.NewLinearGradient(x, y, x+w, y+h)
	grad.AddColorStop(0, pal.bg)
	grad.AddColorStop(0.5, pal.accent)
	grad.AddColorStop(1, pal.neon)
	gc.SetFillStyle(grad)
	gc.DrawRectangle(x, y, w, h)
	gc.Fill()

	cx := x + w/2
	cy := y + h/2
	size := math.Min(w, h)

	gc.SetRGBA(1, 1, 1, 0.10)
	gc.DrawCircle(cx, cy, size*0.34)
	gc.Fill()
	gc.SetRGBA(1, 1, 1, 0.07)
	gc.DrawCircle(cx, cy, size*0.24)
	gc.Fill()

	ns := size * 0.12
	gc.SetRGBA(1, 1, 1, 0.35)
	gc.DrawEllipse(cx-ns*1.5, cy+ns*1.2, ns*1.1, ns*0.80)
	gc.Fill()
	gc.DrawEllipse(cx+ns*0.35, cy+ns*0.5, ns*1.1, ns*0.80)
	gc.Fill()

	gc.SetLineWidth(ns * 0.30)
	gc.DrawLine(cx-ns*0.4, cy+ns*1.2, cx-ns*0.4, cy-ns*1.0)
	gc.Stroke()
	gc.DrawLine(cx+ns*1.45, cy+ns*0.5, cx+ns*1.45, cy-ns*1.7)
	gc.Stroke()
	gc.DrawLine(cx-ns*0.4, cy-ns*1.0, cx+ns*1.45, cy-ns*1.7)
	gc.Stroke()
}

// cropAndResize center-crops src to the w:h aspect ratio then scales to w×h.
func cropAndResize(src image.Image, w, h int) image.Image {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()

	targetRatio := float64(w) / float64(h)
	srcRatio := float64(sw) / float64(sh)

	var crop image.Rectangle
	if srcRatio > targetRatio {
		newW := int(float64(sh) * targetRatio)
		off := (sw - newW) / 2
		crop = image.Rect(sb.Min.X+off, sb.Min.Y, sb.Min.X+off+newW, sb.Max.Y)
	} else {
		newH := int(float64(sw) / targetRatio)
		off := (sh - newH) / 2
		crop = image.Rect(sb.Min.X, sb.Min.Y+off, sb.Max.X, sb.Min.Y+off+newH)
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, crop, xdraw.Over, nil)
	return dst
}

// ── Title block ──────────────────────────────────────────────────────────────

// drawTitleBlock renders the song title on a dark readability pill beneath
// the card, followed by the artist/channel line. Returns the Y coordinate
// just below the block so subsequent elements can be positioned relative
// to it.
func drawTitleBlock(gc *gg.Context, d *TrackData, pal palette) float64 {
	setTitleFont(gc)
	maxW := canvasW - 220
	title := truncateStr(d.Name, maxW, gc)
	tw, th := gc.MeasureString(title)
	tx := (canvasW - tw) / 2
	ty := cardY + cardH + 44

	pad := 18.0
	gc.SetRGBA(0, 0, 0, 0.55)
	gc.DrawRoundedRectangle(tx-pad, ty-th, tw+pad*2, th+pad*1.4, 14)
	gc.Fill()

	drawShadowedText(gc, tx, ty, title, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	nextY := ty + 14
	if d.Channel != "" {
		setArtistFont(gc)
		channel := truncateStr(d.Channel, maxW, gc)
		cw, _ := gc.MeasureString(channel)
		nr, ng, nb := norm(pal.neon)
		gc.SetRGBA(nr, ng, nb, 0.9)
		gc.DrawString(channel, (canvasW-cw)/2, nextY+38)
		nextY += 46
	}
	return nextY
}

func drawShadowedText(gc *gg.Context, x, y float64, text string, fill color.RGBA) {
	gc.SetRGBA(0, 0, 0, 0.7)
	for _, o := range [][2]float64{{-2, -2}, {2, -2}, {-2, 2}, {2, 2}, {0, 3}, {3, 0}} {
		gc.DrawString(text, x+o[0], y+o[1])
	}
	r, g, b, _ := fill.RGBA()
	gc.SetRGBA(float64(r>>8)/255, float64(g>>8)/255, float64(b>>8)/255, 1)
	gc.DrawString(text, x, y)
}

// ── Progress bar ─────────────────────────────────────────────────────────────

// drawProgressBar renders a gradient-filled playback progress track with a
// glowing knob, echoing the reference poster's animated-look scrubber.
func drawProgressBar(gc *gg.Context, x, y, w float64, pal palette, status string) {
	h := 9.0
	gc.SetRGBA(1, 1, 1, 0.14)
	gc.DrawRoundedRectangle(x, y, w, h, h/2)
	gc.Fill()

	pct := 0.42
	if strings.EqualFold(status, "stopped") {
		pct = 0
	}
	fw := w * pct
	if fw > 1 {
		grad := gg.NewLinearGradient(x, y, x+fw, y)
		grad.AddColorStop(0, pal.neon)
		grad.AddColorStop(1, pal.accent)
		gc.SetFillStyle(grad)
		gc.DrawRoundedRectangle(x, y, fw, h, h/2)
		gc.Fill()
	}

	knobX := x + fw
	gc.SetRGBA(1, 1, 1, 1)
	gc.DrawCircle(knobX, y+h/2, 10)
	gc.Fill()
	nr, ng, nb := norm(pal.neon)
	gc.SetRGBA(nr, ng, nb, 1)
	gc.DrawCircle(knobX, y+h/2, 6)
	gc.Fill()
}

// ── Pill metadata row ────────────────────────────────────────────────────────

// drawPillRow renders the centred row of rounded "pill" badges showing
// views, duration and requester — mirroring the reference layout.
func drawPillRow(gc *gg.Context, d *TrackData, y float64) {
	setSmallFont(gc)

	views := d.Views
	if views == "" {
		views = "—"
	}
	pills := []string{
		fmt.Sprintf("\U0001F441  %s", views),
		fmt.Sprintf("\u23F1  %s", formatDuration(d.Duration)),
		fmt.Sprintf("\u25B6  %s", truncateStr(d.User, 260, gc)),
	}

	totalW := 0.0
	widths := make([]float64, len(pills))
	for i, p := range pills {
		w, _ := gc.MeasureString(p)
		widths[i] = w + 28*2
		totalW += widths[i] + 18
	}
	totalW -= 18

	cx := (canvasW - totalW) / 2
	for i, p := range pills {
		drawPill(gc, cx, y, p, widths[i])
		cx += widths[i] + 18
	}
}

func drawPill(gc *gg.Context, x, y float64, text string, w float64) {
	_, th := gc.MeasureString(text)
	h := th + 18

	gc.SetRGBA(1, 1, 1, 0.10)
	gc.DrawRoundedRectangle(x, y, w, h, h/2)
	gc.Fill()
	gc.SetRGBA(1, 1, 1, 0.35)
	gc.SetLineWidth(1.4)
	gc.DrawRoundedRectangle(x, y, w, h, h/2)
	gc.Stroke()

	gc.SetRGBA(1, 1, 1, 1)
	gc.DrawString(text, x+28, y+h-13)
}

// ── Equalizer ────────────────────────────────────────────────────────────────

// drawEqualizer renders a small animated-look bar equalizer, colour-graded
// across the palette's neon → accent range.
func drawEqualizer(gc *gg.Context, x, y float64, pal palette, seed, status string) {
	const bars = 8
	const bw = 12.0
	const gap = 8.0
	const maxH = 60.0

	heights := spectrumHeights(seed, bars)
	baseAlpha := 0.85
	if strings.EqualFold(status, "paused") || strings.EqualFold(status, "stopped") {
		baseAlpha = 0.35
	}

	nr, ng, nb := norm(pal.neon)
	ar, ag, ab := norm(pal.accent)
	for i, hv := range heights {
		t := float64(i) / float64(bars-1)
		r := nr + (ar-nr)*t
		g := ng + (ag-ng)*t
		b := nb + (ab-nb)*t
		bh := hv * maxH
		bx := x + float64(i)*(bw+gap)
		gc.SetRGBA(r, g, b, baseAlpha)
		gc.DrawRoundedRectangle(bx, y-bh, bw, bh, 4)
		gc.Fill()
	}
}

// spectrumHeights returns deterministic bar heights (0–1) seeded from s.
func spectrumHeights(s string, count int) []float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))

	heights := make([]float64, count)
	for i := range heights {
		heights[i] = 0.25 + 0.75*rng.Float64()
	}
	return heights
}

// ── Top badges ───────────────────────────────────────────────────────────────

// drawTopBadges renders the top-left status/brand badge and the top-right
// decorative music-note glyph.
func drawTopBadges(gc *gg.Context, d *TrackData, pal palette) {
	setBadgeFont(gc)

	label := "\U0001F3B5  Zefron Music"
	badgeColor := pal.neon
	if strings.EqualFold(d.Status, "stopped") {
		label = "\u26AB  ENDED"
		badgeColor = color.RGBA{R: 220, G: 30, B: 30, A: 255}
	} else if d.QueuePos == 1 {
		label = "\u25CF  NOW PLAYING"
		badgeColor = color.RGBA{R: 0, G: 200, B: 80, A: 255}
	}
	drawFilledBadge(gc, 32, 32, label, badgeColor)

	setTitleFont(gc)
	nr, ng, nb := norm(pal.neon)
	drawShadowedText(gc, canvasW-108, 88, "\u266A", color.RGBA{
		R: uint8(nr * 255), G: uint8(ng * 255), B: uint8(nb * 255), A: 255,
	})
}

func drawFilledBadge(gc *gg.Context, x, y float64, text string, bg color.RGBA) {
	tw, th := gc.MeasureString(text)
	pw := tw + 28*2
	ph := th + 18

	r, g, b, _ := bg.RGBA()
	gc.SetRGBA(float64(r>>8)/255, float64(g>>8)/255, float64(b>>8)/255, 0.85)
	gc.DrawRoundedRectangle(x, y, pw, ph, ph/2)
	gc.Fill()
	gc.SetRGBA(1, 1, 1, 0.28)
	gc.SetLineWidth(1.2)
	gc.DrawRoundedRectangle(x, y, pw, ph, ph/2)
	gc.Stroke()

	gc.SetRGBA(1, 1, 1, 1)
	gc.DrawString(text, x+28, y+ph-13)
}

// ── Bottom bar ───────────────────────────────────────────────────────────────

// drawBottomBar renders the footer strip: a queue-position badge on the
// left, the source platform on the right, and the brand name centred.
func drawBottomBar(gc *gg.Context, d *TrackData, pal palette) {
	barY := canvasH - 81
	barH := 69.0

	gc.SetRGBA(0, 0, 0, 0.55)
	gc.DrawRectangle(0, barY, canvasW, barH)
	gc.Fill()

	setBadgeFont(gc)
	nr, ng, nb := norm(pal.accent)
	accentBadge := color.RGBA{R: uint8(nr * 255), G: uint8(ng * 255), B: uint8(nb * 255), A: 255}
	drawFilledBadge(gc, 26, barY+12, fmt.Sprintf("\u2B50  %s", formatQueue(d.QueuePos)), accentBadge)

	platLabel := fmt.Sprintf("\U0001F4E2  %s", platformLabel(d.Platform))
	pw, _ := gc.MeasureString(platLabel)
	nr2, ng2, nb2 := norm(pal.neon)
	neonBadge := color.RGBA{R: uint8(nr2 * 255), G: uint8(ng2 * 255), B: uint8(nb2 * 255), A: 255}
	drawFilledBadge(gc, canvasW-pw-28*2-26, barY+12, platLabel, neonBadge)

	setBrandFont(gc)
	brand := "Zefron Music"
	bw, _ := gc.MeasureString(brand)
	gc.SetRGBA(0.86, 0.86, 0.86, 0.7)
	gc.DrawString(brand, (canvasW-bw)/2, barY+barH/2+8)
}

// ── Vignette ─────────────────────────────────────────────────────────────────

func drawVignette(gc *gg.Context) {
	top := gg.NewLinearGradient(0, 0, 0, 110)
	top.AddColorStop(0, color.RGBA{A: 88})
	top.AddColorStop(1, color.RGBA{A: 0})
	gc.SetFillStyle(top)
	gc.DrawRectangle(0, 0, canvasW, 110)
	gc.Fill()
}

// ── Colour helpers ───────────────────────────────────────────────────────────

// norm returns a color.RGBA's channels normalised to the 0–1 range gg expects.
func norm(c color.RGBA) (float64, float64, float64) {
	return float64(c.R) / 255, float64(c.G) / 255, float64(c.B) / 255
}

// ── Font helpers ──────────────────────────────────────────────────────────────

func setTitleFont(gc *gg.Context) {
	if f := boldFace(48); f != nil {
		gc.SetFontFace(f)
	}
}

func setArtistFont(gc *gg.Context) {
	if f := italicFace(30); f != nil {
		gc.SetFontFace(f)
	}
}

func setBadgeFont(gc *gg.Context) {
	if f := boldFace(22); f != nil {
		gc.SetFontFace(f)
	}
}

func setBrandFont(gc *gg.Context) {
	if f := regularFace(20); f != nil {
		gc.SetFontFace(f)
	}
}

func setSmallFont(gc *gg.Context) {
	if f := regularFace(22); f != nil {
		gc.SetFontFace(f)
	}
}

// ── Text utilities ────────────────────────────────────────────────────────────

// truncateStr shortens s so it fits within maxWidth, appending "…" when cut.
func truncateStr(s string, maxWidth float64, gc *gg.Context) string {
	w, _ := gc.MeasureString(s)
	if w <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		if cw, _ := gc.MeasureString(string(runes) + "\u2026"); cw <= maxWidth {
			return string(runes) + "\u2026"
		}
		runes = runes[:len(runes)-1]
	}
	return "\u2026"
}

// formatDuration converts seconds to M:SS or H:MM:SS.
func formatDuration(sec int) string {
	if sec <= 0 {
		return "0:00"
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// formatQueue formats the queue-position string.
func formatQueue(pos int) string {
	switch {
	case pos <= 0:
		return "Empty"
	case pos == 1:
		return "Now Playing"
	default:
		return fmt.Sprintf("#%d Queued", pos)
	}
}
