package main

import (
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// The native item/status panels select DATO with raw record byte +7. Each
// source crop follows its actual memmove destination, including partial entry.
func (g *Game) addModernHDPanelPortrait(record []byte, regions []battle.NativeItemPanelRegion) {
	if g.modernStoryPortraits == nil || !g.modernStoryPortraits.hasHD || len(record) != 80 {
		return
	}
	layout := battle.NativeItemPanelBaseLayoutFor()
	portrait := g.modernStoryPortraits.portraits[int(record[layout.PortraitRecordOffset])]
	if portrait == nil || portrait.hdImage == nil {
		return
	}
	x, y := layout.PortraitDestination.X, layout.PortraitDestination.Y
	base := image.Rect(x*2, y*2, (x+80)*2, (y+80)*2)
	for _, region := range regions {
		if !region.Enabled || region.Width <= 0 || region.Height <= 0 {
			continue
		}
		source := image.Rect(region.SourceX*2, region.SourceY*2,
			(region.SourceX+region.Width)*2, (region.SourceY+region.Height)*2)
		visible := source.Intersect(base)
		if visible.Empty() {
			continue
		}
		offset := image.Pt((region.DestX-region.SourceX)*2, (region.DestY-region.SourceY)*2)
		g.modernHDPanelPortraits = append(g.modernHDPanelPortraits, &modernHDOverlay{
			portrait: portrait, rect: base.Add(offset), clip: visible.Add(offset),
		})
	}
}
