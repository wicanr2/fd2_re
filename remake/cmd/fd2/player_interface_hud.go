package main

import "github.com/hajimehoshi/ebiten/v2"

// An original-art world keeps its native framebuffer. Only the frozen HUD
// rectangle is rendered at HD resolution; its unit and terrain stay original.
func (p *modernHDPresentation) drawIndependentInterfaceHUD(screen ebiten.FinalScreen, geometry ebiten.GeoM) {
	g := p.Game
	if !g.nativeInterfaceHUDVisible || g.nativeInterfaceHUD == nil || g.modernStoryPortraits != nil ||
		!g.modernHDInterfaceEnabled() || g.helpVisible || g.nativeFullDACWhite || g.nativeFullDACBlack ||
		g.nativePlayerStatus != nil || g.nativeItemPanel != nil || len(g.dialog) != 0 || g.nativeSystemEndTurnUI != nil {
		return
	}
	if p.interfaceHUDGPU == nil {
		p.interfaceHUDGPU = &modernHDMapGPU{World: ebiten.NewImage(modernHDWidth, modernHDHeight)}
	}
	r := p.interfaceHUDGPU
	r.World.Clear()
	r.drawHUD(&modernHDMapFrame{HUDPanel: g.nativeInterfaceHUD}, modernHDMapViewport())
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(1.0/3, 1.0/3)
	op.GeoM.Concat(geometry)
	screen.DrawImage(r.World, op)
}
