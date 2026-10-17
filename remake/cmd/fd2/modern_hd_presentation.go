package main

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const modernHDWidth, modernHDHeight = 1920, 1200

// The game retains its input/layout coordinates. Only the complete HD theme
// opts into the final display layer; faithful and legacy packs keep Ebiten's
// existing presentation path.
type modernHDPresentation struct{ *Game }

type modernHDOverlay struct {
	portrait *modernStoryPortraitFrame
	rect     image.Rectangle // Existing 640x400 layout coordinates.
}

type modernHDImageTarget interface {
	DrawImage(*ebiten.Image, *ebiten.DrawImageOptions)
}

func drawModernHDFinal(target modernHDImageTarget, base *ebiten.Image,
	geometry ebiten.GeoM, overlay *modernHDOverlay, brightness float64) {
	target.DrawImage(base, &ebiten.DrawImageOptions{GeoM: geometry})
	if overlay == nil || overlay.portrait == nil || overlay.portrait.hdImage == nil {
		return
	}
	portrait := overlay.portrait
	if portrait.hdGPU == nil {
		portrait.hdGPU = ebiten.NewImageFromImage(portrait.hdImage)
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(overlay.rect.Dx())/float64(portrait.hdGPU.Bounds().Dx()),
		float64(overlay.rect.Dy())/float64(portrait.hdGPU.Bounds().Dy()))
	op.GeoM.Translate(float64(overlay.rect.Min.X), float64(overlay.rect.Min.Y))
	op.GeoM.Concat(geometry)
	value := float32(math.Max(0, math.Min(1, brightness)))
	op.ColorScale.Scale(value, value, value, 1)
	target.DrawImage(portrait.hdGPU, op)
}

func (p *modernHDPresentation) DrawFinalScreen(screen ebiten.FinalScreen,
	offscreen *ebiten.Image, geometry ebiten.GeoM) {
	g := p.Game
	overlay := g.modernHDOverlay
	if g.helpVisible || g.nativePlayerStatus != nil || g.transientUI ||
		g.nativeFullDACWhite || g.nativeFullDACBlack ||
		(g.nativeDefeat == nil && g.titlePhase == "" && g.nativePaletteRamp == nil && nativeDACIsBlack(g.nativeMapDAC)) {
		overlay = nil
	}
	brightness := 1.0
	if g.fade != nil && g.fade.total > 0 {
		brightness = float64(g.fade.t) / float64(g.fade.total)
		if g.fade.out {
			brightness = 1 - brightness
		}
	}
	drawModernHDFinal(screen, offscreen, geometry, overlay, brightness)
}
