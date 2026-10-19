package main

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const modernHDWidth, modernHDHeight = 1920, 1200

// Every theme shares a final display layer so F2 can activate HD after startup.
// Faithful rendering retains the pinned Ebiten default screen filter.
type modernHDPresentation struct {
	*Game
	screenShader                  *ebiten.Shader
	textSurface                   *ebiten.Image
	faithfulWidth, faithfulHeight int
}

func (p *modernHDPresentation) Update() error {
	before := p.modernStoryPortraits != nil && p.modernStoryPortraits.hasHD
	w, h := ebiten.WindowSize()
	if err := p.Game.Update(); err != nil {
		return err
	}
	after := p.modernStoryPortraits != nil && p.modernStoryPortraits.hasHD
	if before == after || ebiten.IsFullscreen() {
		return nil
	}
	if after {
		p.faithfulWidth, p.faithfulHeight = w, h
		ebiten.SetWindowSize(modernHDWidth, modernHDHeight)
	} else {
		if p.faithfulWidth <= 0 || p.faithfulHeight <= 0 {
			p.faithfulWidth, p.faithfulHeight = logicalW*2, logicalH*2
		}
		ebiten.SetWindowSize(p.faithfulWidth, p.faithfulHeight)
	}
	return nil
}

type modernHDOverlay struct {
	portrait *modernStoryPortraitFrame
	rect     image.Rectangle // Existing 640x400 layout coordinates.
	mirrorX  bool
}

type modernHDImageTarget interface {
	DrawImage(*ebiten.Image, *ebiten.DrawImageOptions)
}

func drawModernHDOverlay(target modernHDImageTarget,
	geometry ebiten.GeoM, overlay *modernHDOverlay, brightness float64) {
	if overlay == nil || overlay.portrait == nil || overlay.portrait.hdImage == nil {
		return
	}
	portrait := overlay.portrait
	if portrait.hdGPU == nil {
		portrait.hdGPU = ebiten.NewImageFromImage(portrait.hdImage)
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	sx := float64(overlay.rect.Dx()) / float64(portrait.hdGPU.Bounds().Dx())
	x := float64(overlay.rect.Min.X)
	if overlay.mirrorX {
		sx = -sx
		x = float64(overlay.rect.Max.X)
	}
	op.GeoM.Scale(sx,
		float64(overlay.rect.Dy())/float64(portrait.hdGPU.Bounds().Dy()))
	op.GeoM.Translate(x, float64(overlay.rect.Min.Y))
	op.GeoM.Concat(geometry)
	value := float32(math.Max(0, math.Min(1, brightness)))
	op.ColorScale.Scale(value, value, value, 1)
	target.DrawImage(portrait.hdGPU, op)
}

func (p *modernHDPresentation) DrawFinalScreen(screen ebiten.FinalScreen,
	offscreen *ebiten.Image, geometry ebiten.GeoM) {
	if p.Game.modernHDBattleVisible && p.Game.modernHDBattleGPU != nil {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(1.0/3, 1.0/3)
		op.GeoM.Concat(geometry)
		screen.DrawImage(p.Game.modernHDBattleGPU.World, op)
	}
	if p.Game.modernHDWorldVisible && p.Game.modernHDMapGPU != nil {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(1.0/3, 1.0/3)
		op.GeoM.Concat(geometry)
		screen.DrawImage(p.Game.modernHDMapGPU.World, op)
	}
	p.drawDefaultFinalScreen(screen, offscreen, geometry)
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
	drawModernHDOverlay(screen, geometry, overlay, brightness)
	if !g.nativeFullDACWhite && !g.nativeFullDACBlack &&
		!(g.nativeDefeat == nil && g.titlePhase == "" && g.nativePaletteRamp == nil && nativeDACIsBlack(g.nativeMapDAC)) {
		p.drawModernHDText(screen, geometry, brightness)
	}
}
