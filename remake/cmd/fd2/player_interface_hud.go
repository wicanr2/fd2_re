package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

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
	// This native producer writes after the complete map, including its HUD.
	if job := g.nativeFieldEvent61; job != nil && job.hd != nil && job.frame >= 0 && job.frame < len(job.frames) {
		patch := image.NewRGBA(image.Rect(0, 0, 8, 2))
		for y := 0; y < 2; y++ {
			for x := 0; x < 8; x++ {
				patch.Set(x, y, job.hd.scene.palette[job.frames[job.frame][(151+y)*320+36+x]])
			}
		}
		texture := ebiten.NewImageFromImage(patch)
		defer texture.Dispose()
		write := &ebiten.DrawImageOptions{}
		write.GeoM.Scale(6, 6)
		write.GeoM.Translate(216, 906)
		r.World.DrawImage(texture, write)
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(1.0/3, 1.0/3)
	op.GeoM.Concat(geometry)
	screen.DrawImage(r.World, op)
}
