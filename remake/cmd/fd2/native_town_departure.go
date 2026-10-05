package main

import (
	"errors"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type nativeTownDepartureFrame struct {
	indexed []byte
	dac     []byte
	palette color.Palette
}

type nativeTownDepartureJob struct {
	frames []nativeTownDepartureFrame
	step   int
	drawn  bool
	then   func()
}

// startNativeTownDeparture prebuilds every indexed/DAC step before publishing
// the owner. The saved town caller is also restored after roster selection.
// Contract: fd2_town_departure_20261005.json, spec.
func (g *Game) startNativeTownDeparture(then func()) error {
	if g.camp == nil || g.camp.Node() == nil || g.nativeTownDeparture != nil || g.nativePaletteRamp != nil || !nativeMapAssetsAvailable(g.nativeMapAssets) {
		return errors.New("native town departure owner or palette unavailable")
	}
	n := g.camp.Node()
	town := g.camp.C.Nodes[n.Cancel]
	if n.Type != "preparation" || town == nil || town.Type != "town" || town.NativeTownVariant == nil || len(g.prepPromptSource) != 320*200 {
		return errors.New("native town departure caller source unavailable")
	}
	job := &nativeTownDepartureJob{then: then}
	for k := 1; k <= 11; k++ {
		step, delta := k, 4*k
		if k == 11 {
			step, delta = 10, 64
		}
		frame, err := campaign.ComposeNativeTownDepartureFrame(g.prepPromptSource, *town.NativeTownVariant, 2, step)
		if err != nil {
			return err
		}
		dac := make([]byte, 768)
		if err := fdother.ApplyVGAPaletteSubtraction(dac, g.nativeMapAssets.PaletteDAC, 0, 255, delta); err != nil {
			return err
		}
		palette, err := fdother.VGAPaletteFromDAC(dac)
		if err != nil {
			return err
		}
		job.frames = append(job.frames, nativeTownDepartureFrame{frame, dac, palette})
	}
	g.nativeTownDeparture = job
	return nil
}

func (g *Game) stepNativeTownDeparture() {
	j := g.nativeTownDeparture
	if j == nil || !j.drawn {
		return
	}
	j.step++
	j.drawn = false
	if j.step < len(j.frames) {
		return
	}
	g.nativeMapDAC = append(g.nativeMapDAC[:0], j.frames[len(j.frames)-1].dac...)
	clear(g.nativeMapVGA)
	g.nativeTownDeparture, g.nativeTownLoadPending = nil, true
	if j.then != nil {
		j.then()
	}
}

func (g *Game) drawNativeTownDeparture(screen *ebiten.Image) {
	j := g.nativeTownDeparture
	f := j.frames[j.step]
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), f.palette)
	copy(img.Pix, f.indexed)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	j.drawn = true
}

// finishNativePreparationDeparture is shared by the small-roster prompt and
// final roster YES. Standalone preparation has no town departure caller.
func (g *Game) finishNativePreparationDeparture(then func()) {
	if g.camp != nil && g.camp.Node() != nil && g.camp.Node().Cancel != "" && g.nativeTownUI != nil {
		if err := g.startNativeTownDeparture(then); err != nil {
			g.loadErr = "native town departure: " + err.Error()
		}
		return
	}
	then()
}
