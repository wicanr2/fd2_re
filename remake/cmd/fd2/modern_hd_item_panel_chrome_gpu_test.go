package main

import (
	"fmt"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func checkItemPanelChromeGamePixels(g *Game, pixels []byte, colors dialogueChromeSource) (int, error) {
	if !g.itemOpen {
		return 0, nil
	}
	frame := 11 - g.itemAnimStep
	if g.itemClosing {
		frame = g.itemAnimStep
	}
	pass, err := battle.NativeItemPanelFrameFor(frame)
	if err != nil {
		return 0, err
	}
	return checkItemPanelRegionChromePixels(pixels, colors, []battle.NativeItemPanelRegion{pass.Left, pass.Upper, pass.Bottom})
}

func checkItemPanelRegionChromePixels(pixels []byte, colors dialogueChromeSource, regions []battle.NativeItemPanelRegion) (int, error) {
	if len(pixels) != 1920*1200*4 {
		return 0, fmt.Errorf("missing full HD canvas")
	}
	// Independent fixed-source bounds; do not read the production panel list.
	full := []image.Rectangle{image.Rect(5, 7, 91, 93), image.Rect(92, 7, 315, 93), image.Rect(5, 94, 315, 196)}
	points := 0
	for _, r := range regions {
		if !r.Enabled || r.Width <= 0 || r.Height <= 0 {
			continue
		}
		source := image.Rect(r.SourceX, r.SourceY, r.SourceX+r.Width, r.SourceY+r.Height)
		var base image.Rectangle
		for _, rect := range full {
			if source.In(rect) {
				base = rect
				break
			}
		}
		if base.Empty() {
			return points, fmt.Errorf("unknown native source region %+v", r)
		}
		visible := image.Rect(r.DestX*6, r.DestY*6, (r.DestX+r.Width)*6, (r.DestY+r.Height)*6).Intersect(image.Rect(0, 0, 1920, 1200))
		artHeight := base.Dy() * 6
		if base.Min.Y == 94 {
			artHeight = 99 * 6 // Entry21 is shorter than the 102-row memmove span.
		}
		for y := visible.Min.Y; y < visible.Max.Y; y++ {
			for x := visible.Min.X; x < visible.Max.X; x++ {
				sx, sy := x-r.DestX*6+(r.SourceX-base.Min.X)*6, y-r.DestY*6+(r.SourceY-base.Min.Y)*6
				want, painted := itemPanelChromePixel(colors, sx, sy, base.Dx()*6, artHeight)
				if !painted {
					continue
				}
				at := (y*1920 + x) * 4
				for c := 0; c < 4; c++ {
					delta := int(pixels[at+c]) - int(want[c])
					if delta < -1 || delta > 1 {
						return points, fmt.Errorf("frame differs at%d,%d channel%d got%d want%d", x, y, c, pixels[at+c], want[c])
					}
				}
				points++
			}
		}
	}
	return points, nil
}
