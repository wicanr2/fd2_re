package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func itemPanelValuesColors(d *nativeUnitStatusData, palette color.Palette) map[string]color.RGBA {
	result := map[string]color.RGBA{}
	for name, index := range map[string]byte{"background": d.assets.ItemPanelUpper.Pixels[70*223+100], "border": d.assets.ItemPanelUpper.Pixels[224], "hp": d.assets.RawCells[24].Pixels[1], "mp": d.assets.RawCells[27].Pixels[1], "track": d.assets.RawCells[29].Pixels[1]} {
		result[name] = color.RGBAModel.Convert(palette[index]).(color.RGBA)
	}
	return result
}

func checkItemPanelValuesPixels(pixels []byte, d *nativeUnitStatusData, palette color.Palette, regions []battle.NativeItemPanelRegion) (int, error) {
	if d == nil || len(d.record) != 80 || len(pixels) != 1920*1200*4 {
		return 0, fmt.Errorf("missing values source or canvas")
	}
	colors := itemPanelValuesColors(d, palette)
	count := 0
	for _, region := range regions {
		if !region.Enabled || region.SourceX < 92 || region.SourceY < 7 || region.SourceX+region.Width > 315 || region.SourceY+region.Height > 93 {
			continue
		}
		copyRect := image.Rect(region.SourceX*6, region.SourceY*6, (region.SourceX+region.Width)*6, (region.SourceY+region.Height)*6)
		for i, y := range []int{33, 52} {
			outer := image.Rect(197*6, (y-1)*6, 302*6, (y+6)*6)
			bar := image.Rect(198*6, y*6, 301*6, (y+5)*6)
			current := int(int16(binary.LittleEndian.Uint16(d.record[64+i*4:])))
			maximum := int(int16(binary.LittleEndian.Uint16(d.record[66+i*4:])))
			length := 0
			if current != 0 && maximum != 0 {
				body := 101*current/maximum + 1
				if body < 1 {
					body = 1
				}
				length = (body + 1) * 6
				if length > 618 {
					length = 618
				}
			}
			visible := outer.Intersect(copyRect)
			for sy := visible.Min.Y; sy < visible.Max.Y; sy++ {
				for sx := visible.Min.X; sx < visible.Max.X; sx++ {
					x, yy := sx+(region.DestX-region.SourceX)*6, sy+(region.DestY-region.SourceY)*6
					if x < 0 || yy < 0 || x >= 1920 || yy >= 1200 {
						continue
					}
					want := colors["background"]
					if image.Pt(sx, sy).In(bar) {
						want = colors["track"]
						if sx-bar.Min.X < length {
							want = colors[[]string{"hp", "mp"}[i]]
						}
						if sx == bar.Min.X || sx == bar.Max.X-1 || sy == bar.Min.Y || sy == bar.Max.Y-1 {
							want = colors["border"]
						}
					}
					at := (yy*1920 + x) * 4
					for c, v := range []byte{want.R, want.G, want.B, want.A} {
						delta := int(pixels[at+c]) - int(v)
						if delta < -1 || delta > 1 {
							return count, fmt.Errorf("bar%d at%d,%d channel%d got%d want%d", i, x, yy, c, pixels[at+c], v)
						}
					}
					count++
				}
			}
		}
	}
	return count, nil
}

func checkItemPanelValuesGamePixels(g *Game, pixels []byte) (int, error) {
	if !g.itemOpen {
		return 0, nil
	}
	if g.nativeItemPanelText == nil {
		return 0, fmt.Errorf("missing item values owner")
	}
	frame := 11 - g.itemAnimStep
	if g.itemClosing {
		frame = g.itemAnimStep
	}
	pass, err := battle.NativeItemPanelFrameFor(frame)
	if err != nil {
		return 0, err
	}
	return checkItemPanelValuesPixels(pixels, &g.nativeItemPanelText.data, g.nativeUIPalette, []battle.NativeItemPanelRegion{pass.Left, pass.Upper, pass.Bottom})
}
