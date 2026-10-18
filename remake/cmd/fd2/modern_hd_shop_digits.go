package main

import (
	"image"
	"image/color"
	"strconv"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// Only the verified decimal banks are replaced. Overflow and other pictorial
// markers keep their source sprites. These copies exist for one Draw only.
func (g *Game) beginModernHDShopDigits() func() {
	if !g.modernHDShopText() || g.nativeClassUI == nil || g.nativeShopUI == nil {
		return func() {}
	}
	shared := g.nativeClassUI
	oldDigits := shared.digits
	oldFrames := g.nativeShopUI.itemAssets.Frames
	shared.digits = append([]fdother.Frame(nil), oldDigits...)
	frames := make(map[int]fdother.Frame, len(oldFrames))
	for id, frame := range oldFrames {
		frames[id] = frame
	}
	for digit, frame := range oldDigits {
		shared.digits[digit] = g.modernHDDecimalSprite(frame, digit)
	}
	for _, base := range []int{31, 42, 119} {
		for digit := 0; digit < 10; digit++ {
			if frame, ok := frames[base+digit]; ok {
				frames[base+digit] = g.modernHDDecimalSprite(frame, digit)
			}
		}
	}
	g.nativeShopUI.itemAssets.Frames = frames
	return func() { shared.digits = oldDigits; g.nativeShopUI.itemAssets.Frames = oldFrames }
}

func (g *Game) modernHDDecimalSprite(source fdother.Frame, digit int) fdother.Frame {
	// Production separated frames have an explicit binary mask. An unverified
	// or archive-only representation retains its ordinary sprite renderer.
	if len(source.Indexed) != source.Width*source.Height || len(source.Mask) != len(source.Indexed) {
		return source
	}
	var foreground byte
	best := -1
	for i, index := range source.Indexed {
		if source.Mask[i] != 255 {
			continue
		}
		c := nativePaletteColor(g.nativeClassUI.palette, index, color.RGBA{})
		brightness := int(c.R) + int(c.G) + int(c.B)
		if brightness > best {
			foreground = index
			best = brightness
		}
	}
	if best < 0 {
		return source
	}
	placeholder, err := source.TextPlaceholder(func(x, y int) {
		capture := g.font.indexedHD
		if capture == nil {
			return
		}
		clip := image.Rect(x*2, y*2, (x+source.Width)*2, (y+source.Height)*2)
		kept := capture.runs[:0]
		for _, run := range capture.runs {
			if !run.inkTop || run.clip != clip {
				kept = append(kept, run)
			}
		}
		capture.runs = append(kept, modernHDTextRun{font: g.font, text: strconv.Itoa(digit),
			x: float64(x * 2), y: float64(y * 2), pixels: 20, color: modernHDNativeTextColor(capture.palette, foreground), clip: clip, inkTop: true})
	})
	if err != nil {
		return source
	}
	return placeholder
}
