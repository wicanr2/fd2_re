package main

import (
	"image"
	"image/color"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// A job owns its background and window plan even if the live balance or theme
// changes. No HD font or callback is installed while Update builds the job.
type nativeGoldRollText struct {
	background [48 * 9]byte
	windows    []campaign.NativeGoldRollWindow
	foreground byte
	credit     bool
}

func (g *Game) snapshotNativeGoldRoll(assets *campaign.NativeShopAssets, oldGold, newGold int, credit bool) *nativeGoldRollText {
	if assets == nil || g.nativeClassUI == nil || len(g.nativeClassUI.digits) != 10 ||
		assets.GoldRollStrip.Width != 6 || assets.GoldRollStrip.Height != 99 || len(assets.GoldRollStrip.Pixels) != 6*99 {
		return nil
	}
	windows, err := campaign.PlanNativeGoldRoll(oldGold, newGold, credit)
	if err != nil || len(windows) == 0 {
		return nil
	}
	digits := make([]fdother.Frame, 10)
	for i, source := range g.nativeClassUI.digits {
		width := 6
		if i == 1 {
			width = 5 // FDOTHER #5 frame 32; the counter still advances six pixels.
		}
		if source.Width != width || source.Height != 8 {
			return nil
		}
		digits[i], err = source.TextPlaceholder(func(int, int) {})
		if err != nil {
			return nil
		}
	}
	bare, err := campaign.ComposeNativeShopBareScene(assets, digits, oldGold)
	if err != nil {
		return nil
	}
	d := &nativeGoldRollText{windows: windows, credit: credit}
	for row := 0; row < 9; row++ {
		offset := campaign.NativeShopGoldRollOffset + row*320
		copy(d.background[row*48:(row+1)*48], bare[offset:offset+48])
	}
	best := -1
	for _, index := range assets.GoldRollStrip.Pixels {
		c := nativePaletteColor(g.nativeClassUI.palette, index, color.RGBA{})
		brightness := int(c.R) + int(c.G) + int(c.B)
		if index != 0 && brightness > best {
			d.foreground, best = index, brightness
		}
	}
	if best < 0 {
		return nil
	}
	return d
}

func (d *nativeGoldRollText) compose(source []byte, index int, font *Font, palette color.Palette) ([]byte, []modernHDTextRun, bool) {
	if d == nil || len(source) != 320*200 || index < 0 || index >= len(d.windows) || font == nil {
		return nil, nil, false
	}
	window := d.windows[index]
	for _, row := range window.Rows {
		if row < 0 || row > 90 {
			return nil, nil, false
		}
	}
	frame := append([]byte(nil), source...)
	for row := 0; row < 9; row++ {
		offset := campaign.NativeShopGoldRollOffset + row*320
		copy(frame[offset:offset+48], d.background[row*48:(row+1)*48])
	}
	var runs []modernHDTextRun
	for digit, row := range window.Rows {
		clip := image.Rect(32+digit*12, 196, 44+digit*12, 214)
		value, offset := (row/9)%10, row%9
		appendDigit := func(value, y int) {
			runs = append(runs, modernHDTextRun{font: font, text: strconv.Itoa(value),
				x: float64(clip.Min.X), y: float64(y * 2), pixels: 20,
				color: modernHDNativeTextColor(palette, d.foreground), clip: clip, inkTop: true})
		}
		appendDigit(value, 99-offset)
		if offset != 0 {
			appendDigit((value+1)%10, 108-offset)
		}
	}
	return frame, runs, true
}

func (g *Game) drawModernHDGoldRoll(screen *ebiten.Image, job *nativeClassUIJob, index int) bool {
	if g.modernHDText == nil || job.goldText == nil || len(job.goldText.windows) != len(job.timeline) {
		return false
	}
	step := job.timeline[index]
	frame, runs, ok := job.goldText.compose(step.frame, index, g.font, step.palette)
	if !ok {
		return false
	}
	if !g.drawModernHDShopFrame(screen, frame, step.palette, step.shopEffectMask) {
		g.presentNativeClassFrameWithPalette(screen, frame, step.palette)
	}
	g.modernHDText.runs = append(g.modernHDText.runs, runs...)
	return true
}

func (job *nativeClassUIJob) timelinePosition() (index int, total time.Duration) {
	elapsed := job.elapsed
	index = len(job.timeline) - 1
	selected := false
	for i, step := range job.timeline {
		total += step.duration
		if !selected {
			if elapsed < step.duration {
				index, selected = i, true
			} else {
				elapsed -= step.duration
			}
		}
	}
	return
}
