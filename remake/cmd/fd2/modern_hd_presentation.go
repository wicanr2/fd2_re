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
	screenShader                   *ebiten.Shader
	textSurface                    *ebiten.Image
	faithfulWidth, faithfulHeight  int
	toolbar                        bool
	floatingToolbar                bool
	toolbarTicks                   int
	gameSurface, toolbarSurface    *ebiten.Image
	interfaceHUDGPU                *modernHDMapGPU
	appliedPlayerSpeed             int
	poster                         playerPosterPresentation
	posterMenuSurface, posterPixel *ebiten.Image
}

func (p *modernHDPresentation) Update() error {
	before := p.Game.hdPresentationEnabled()
	beforeWidth := p.layoutWidth()
	w, h := ebiten.WindowSize()
	p.poster.consumeInput = false
	if !p.Game.playerPosterControlsAllowed() {
		p.poster.menuOpen = false
	}
	if p.toolbar {
		p.Game.playerViewportLeft = p.posterOffset()
		p.Game.playerViewportTop = p.toolbarOffset()
		p.updateToolbarVisibility()
		p.Game.playerToolbarTouchBlocked = p.floatingToolbar && p.toolbarTicks > 0
		p.queueToolbarInput()
		p.stepPosterControls()
	}
	p.Game.playerOverlayBlock = image.Rectangle{}
	if p.poster.menuOpen {
		p.Game.playerOverlayBlock = playerPosterMenu
	}
	if !p.poster.consumeInput {
		if err := p.Game.Update(); err != nil {
			return err
		}
		if p.Game.playerPosterCycleRequested {
			p.cyclePosters()
			p.Game.playerPosterCycleRequested = false
		}
	}
	if !p.Game.playerPosterControlsAllowed() {
		p.poster.menuOpen = false
	}
	speed := normalizePlayerSpeed(p.Game.playerSpeed)
	if p.appliedPlayerSpeed != speed {
		if p.appliedPlayerSpeed > 0 {
			p.toolbarTicks = (p.toolbarTicks*speed + p.appliedPlayerSpeed - 1) / p.appliedPlayerSpeed
		}
		ebiten.SetTPS(60 * speed)
		p.appliedPlayerSpeed = speed
	}
	after := p.Game.hdPresentationEnabled()
	if ebiten.IsFullscreen() {
		return nil
	}
	if before != after && after {
		p.faithfulWidth, p.faithfulHeight = w, h*beforeWidth/logicalW
		ebiten.SetWindowSize(modernHDWidth, p.presentationHeight(modernHDWidth))
	} else if before != after {
		if p.faithfulWidth <= 0 || p.faithfulHeight <= 0 {
			p.faithfulWidth, p.faithfulHeight = logicalW*2, logicalH*2
			p.faithfulHeight += p.toolbarOffset() * 2
		}
		height := p.faithfulHeight
		if p.posterOffset() != 0 {
			height = p.presentationHeight(p.faithfulWidth)
		}
		ebiten.SetWindowSize(p.faithfulWidth, height)
	} else if beforeWidth != p.layoutWidth() {
		ebiten.SetWindowSize(w, p.presentationHeight(w))
	}
	return nil
}

func (p *modernHDPresentation) presentationHeight(width int) int {
	return int(math.Round(float64(width) * float64(logicalH+p.toolbarOffset()) / float64(p.layoutWidth())))
}

type modernHDOverlay struct {
	portrait *modernStoryPortraitFrame
	rect     image.Rectangle // Existing 640x400 layout coordinates.
	clip     image.Rectangle // Optional destination clip for moving native panels.
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
	source := portrait.hdGPU
	rect := overlay.rect
	if !overlay.clip.Empty() {
		clipped := rect.Intersect(overlay.clip)
		if clipped.Empty() {
			return
		}
		left, right := clipped.Min.X-rect.Min.X, clipped.Max.X-rect.Min.X
		if overlay.mirrorX {
			left, right = rect.Dx()-right, rect.Dx()-left
		}
		source = source.SubImage(image.Rect(left*source.Bounds().Dx()/rect.Dx(),
			(clipped.Min.Y-rect.Min.Y)*source.Bounds().Dy()/rect.Dy(),
			right*source.Bounds().Dx()/rect.Dx(),
			(clipped.Max.Y-rect.Min.Y)*source.Bounds().Dy()/rect.Dy())).(*ebiten.Image)
		rect = clipped
	}
	sx := float64(rect.Dx()) / float64(source.Bounds().Dx())
	x := float64(rect.Min.X)
	if overlay.mirrorX {
		sx = -sx
		x = float64(rect.Max.X)
	}
	op.GeoM.Scale(sx,
		float64(rect.Dy())/float64(source.Bounds().Dy()))
	op.GeoM.Translate(x, float64(rect.Min.Y))
	op.GeoM.Concat(geometry)
	value := float32(math.Max(0, math.Min(1, brightness)))
	op.ColorScale.Scale(value, value, value, 1)
	target.DrawImage(source, op)
}

func (p *modernHDPresentation) DrawFinalScreen(screen ebiten.FinalScreen,
	offscreen *ebiten.Image, geometry ebiten.GeoM) {
	contentGeometry := geometry
	if p.toolbarOffset() > 0 || p.posterOffset() > 0 {
		contentGeometry.Reset()
		contentGeometry.Translate(float64(p.posterOffset()), float64(p.toolbarOffset()))
		contentGeometry.Concat(geometry)
	}
	p.drawPosters(screen, geometry)
	p.drawModernFacilityScene(screen, contentGeometry)
	if p.Game.modernHDBattleVisible && p.Game.modernHDBattleGPU != nil {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(1.0/3, 1.0/3)
		op.GeoM.Concat(contentGeometry)
		screen.DrawImage(p.Game.modernHDBattleGPU.World, op)
	}
	if p.Game.modernHDWorldVisible && p.Game.modernHDMapGPU != nil {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(1.0/3, 1.0/3)
		op.GeoM.Concat(contentGeometry)
		screen.DrawImage(p.Game.modernHDMapGPU.World, op)
	}
	p.drawDefaultFinalScreen(screen, offscreen, geometry)
	p.drawIndependentInterfaceHUD(screen, contentGeometry)
	g := p.Game
	overlay := g.modernHDOverlay
	if g.helpVisible || g.nativePlayerStatus != nil || (g.transientUI && !g.modernHDTransientExpiryPortraitAllowed()) ||
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
	if !g.helpVisible && !g.nativeFullDACWhite && !g.nativeFullDACBlack &&
		!(g.nativeDefeat == nil && g.titlePhase == "" && g.nativePaletteRamp == nil && nativeDACIsBlack(g.nativeMapDAC)) {
		p.drawModernHDDialogueChrome(screen, contentGeometry, brightness)
		p.drawModernHDConfirmation(screen, contentGeometry, brightness)
		p.drawModernHDSystemInfo(screen, contentGeometry, brightness)
	}
	drawModernHDOverlay(screen, contentGeometry, overlay, brightness)
	if !g.helpVisible && !g.transientUI && !g.nativeFullDACWhite && !g.nativeFullDACBlack &&
		!(g.nativeDefeat == nil && g.titlePhase == "" && g.nativePaletteRamp == nil && nativeDACIsBlack(g.nativeMapDAC)) {
		for _, portrait := range g.modernHDPanelPortraits {
			drawModernHDOverlay(screen, contentGeometry, portrait, brightness)
		}
	}
	if !g.nativeFullDACWhite && !g.nativeFullDACBlack &&
		!(g.nativeDefeat == nil && g.titlePhase == "" && g.nativePaletteRamp == nil && nativeDACIsBlack(g.nativeMapDAC)) {
		p.drawModernHDText(screen, contentGeometry, brightness)
	}
	if p.toolbarVisible() {
		toolbarGeometry := ebiten.GeoM{}
		toolbarGeometry.Translate(float64(p.posterOffset()), 0)
		toolbarGeometry.Concat(geometry)
		p.drawThemeToolbar(screen, toolbarGeometry)
		p.drawPosterMenu(screen, toolbarGeometry)
	}
}
