package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernHDMapHUDPanel struct {
	Rect                image.Rectangle
	Base, Above         *image.RGBA
	Terrain, NativeUnit *image.RGBA
	TerrainHD           bool
}

type modernHDMapHUDArt struct {
	Key         [32]byte
	Font        *Font
	Base, Above *image.RGBA
}

func modernHUDGlyphInk(frame fdother.Frame, palette color.Palette) (color.RGBA, error) {
	if frame.Width <= 0 || frame.Height <= 0 || frame.Width > 18 || frame.Height > 8 {
		return color.RGBA{}, errors.New("modern HD map HUD: invalid glyph geometry")
	}
	pixels, err := modernHDOverwriteLayer(frame.Width*frame.Height, palette, func(dst []byte) error {
		return frame.BlitAt(dst, frame.Width, 0, -1)
	})
	if err != nil {
		return color.RGBA{}, err
	}
	var ink color.RGBA
	best := -1
	for _, c := range pixels {
		if c.A == 255 && int(c.R)+int(c.G)+int(c.B) > best {
			ink, best = c, int(c.R)+int(c.G)+int(c.B)
		}
	}
	if best < 0 {
		return ink, errors.New("modern HD map HUD: glyph has no source ink")
	}
	return ink, nil
}

func prepareModernHUDArt(in indexedmap.NativeFrameInput, palette color.Palette, f *Font, previous *modernHDMapHUDArt) (*modernHDMapHUDArt, error) {
	if f == nil || f.sf == nil || len(palette) != 256 || in.Frames.Panel.Width != 69 || in.Frames.Panel.Height != 34 {
		return nil, errors.New("modern HD map HUD: incomplete panel or font")
	}
	panel, err := modernHDOverwriteLayer(69*34, palette, func(dst []byte) error {
		return in.Frames.Panel.BlitAt(dst, 69, 0, -1)
	})
	if err != nil {
		return nil, err
	}
	for _, c := range panel {
		if c.A != 255 {
			return nil, errors.New("modern HD map HUD: panel is not opaque")
		}
	}
	colours := make([]color.RGBA, 256)
	for i, c := range palette {
		if c == nil {
			return nil, errors.New("modern HD map HUD: missing palette colour")
		}
		r, g, b, a := c.RGBA()
		if a != 65535 {
			return nil, errors.New("modern HD map HUD: nonopaque palette")
		}
		colours[i] = color.RGBA{byte(r >> 8), byte(g >> 8), byte(b >> 8), 255}
	}
	u := in.HUD.OptionalUnit
	var hp any
	if u != nil {
		hp = [2]uint16{u.Current, u.Maximum}
	}
	encoded, _ := json.Marshal([]any{in.Frames, colours, in.HUD.TerrainControl, hp})
	key := sha256.Sum256(encoded)
	if previous != nil && previous.Key == key && previous.Font == f {
		return previous, nil
	}
	base, above := image.NewRGBA(image.Rect(0, 0, 414, 204)), image.NewRGBA(image.Rect(0, 0, 414, 204))
	fill := func(rect image.Rectangle, ink color.RGBA) {
		draw.Draw(base, rect, image.NewUniform(ink), image.Point{}, draw.Src)
	}
	stroke := func(rect image.Rectangle, ink color.RGBA) {
		fill(image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+3), ink)
		fill(image.Rect(rect.Min.X, rect.Max.Y-3, rect.Max.X, rect.Max.Y), ink)
		fill(image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+3, rect.Max.Y), ink)
		fill(image.Rect(rect.Max.X-3, rect.Min.Y, rect.Max.X, rect.Max.Y), ink)
	}
	at := func(x, y int) color.RGBA { return panel[y*69+x] }
	fill(base.Bounds(), at(0, 1))
	fill(image.Rect(6, 6, 408, 198), at(1, 1))
	fill(image.Rect(12, 12, 402, 192), at(40, 3))
	stroke(image.Rect(12, 12, 402, 192), at(2, 2))
	fill(image.Rect(30, 24, 186, 180), at(6, 4))
	fill(image.Rect(36, 30, 180, 174), at(6, 5))
	fill(image.Rect(30, 30, 33, 180), at(5, 5))
	fill(image.Rect(30, 177, 180, 180), at(5, 29))
	paint := func(dst *image.RGBA, text string, rect image.Rectangle, ink color.RGBA) error {
		return drawModernBattlePanelInk(dst, f, text, image.Rectangle{Min: rect.Min.Mul(6), Max: rect.Max.Mul(6)}, 48, ink)
	}
	for i, label := range []string{"A", "D"} {
		y := 8 + 11*i
		ink, best := color.RGBA{}, -1
		background, shadow := at(40, 3), at(6, 4)
		for row := y; row < y+8; row++ {
			for x := 35; x < 41; x++ {
				c := at(x, row)
				if c != background && c != shadow && int(c.R)+int(c.G)+int(c.B) > best {
					ink, best = c, int(c.R)+int(c.G)+int(c.B)
				}
			}
		}
		// A palette flash may make ink and background identical.
		if best < 0 {
			ink = at(38, y)
		}
		if err := paint(base, label, image.Rect(35, y, 41, y+8), ink); err != nil {
			return nil, err
		}
	}
	ap, dp, err := indexedmap.NativeMapHUDTerrainAPDP(in.HUD.TerrainControl)
	if err != nil {
		return nil, err
	}
	for i, value := range []int{ap, dp} {
		y, sign, glyph := 8+11*i, "+", in.Frames.PositiveSign
		if value < 0 {
			value, sign, glyph = -value, "-", in.Frames.NegativeSign
		}
		ink, err := modernHUDGlyphInk(glyph, palette)
		if err != nil {
			return nil, err
		}
		if err := paint(base, sign, image.Rect(43, y, 49, y+glyph.Height), ink); err != nil {
			return nil, err
		}
		for place, digit := range []int{value / 10, value % 10} {
			ink, err := modernHUDGlyphInk(in.Frames.Digits[digit], palette)
			if err != nil {
				return nil, err
			}
			x := 51 + place*6
			if err := paint(base, fmt.Sprint(digit), image.Rect(x, y, x+6, y+8), ink); err != nil {
				return nil, err
			}
		}
	}
	if u != nil {
		digits, overflow := in.Frames.Digits, in.Frames.HPEqualOverflow
		if u.Current != u.Maximum {
			digits, overflow = in.Frames.HPMismatchDigits, in.Frames.HPMismatchOverflow
		}
		value := fmt.Sprintf("%03d", u.Current)
		if u.Current > 999 {
			value = "???"
		}
		for i, ch := range value {
			glyph := overflow
			if ch != '?' {
				glyph = digits[int(ch-'0')]
			}
			ink, err := modernHUDGlyphInk(glyph, palette)
			if err != nil {
				return nil, err
			}
			x := 9 + i*6
			if err := paint(above, string(ch), image.Rect(x, 21, x+6, 29), ink); err != nil {
				return nil, err
			}
		}
	}
	return &modernHDMapHUDArt{Key: key, Font: f, Base: base, Above: above}, nil
}

func modernHUDNativeSlot(palette color.Palette, render func([]byte) error) (*image.RGBA, error) {
	pixels, err := modernHDOverwriteLayer(24*24, palette, render)
	if err != nil {
		return nil, err
	}
	image := image.NewRGBA(image.Rect(0, 0, 144, 144))
	for y := 0; y < 144; y++ {
		for x := 0; x < 144; x++ {
			image.SetRGBA(x, y, pixels[(y/6)*24+x/6])
		}
	}
	return image, nil
}

func (g *Game) prepareModernHDMapHUDPanel(in indexedmap.NativeFrameInput, palette color.Palette) (*modernHDMapHUDPanel, error) {
	if !in.HUD.DisplayGateA || !in.HUD.DisplayGateB || g.font == nil {
		return nil, nil
	}
	layout, err := fdicon.NativeMapHUDLayoutFor(in.HUD.AnchorX, fdicon.NativeMapStride)
	if err != nil {
		return nil, err
	}
	f := g.font
	if regional := f.hdFonts[g.localeID]; regional != nil {
		f = regional
	}
	art, err := prepareModernHUDArt(in, palette, f, g.modernHDMapHUDArt)
	if err != nil {
		return nil, err
	}
	tile := in.HUD.TerrainDescriptor
	if in.HUDTerrain == nil || tile < 0 || tile > 1023 || tile >= len(in.HUDTerrain.Sprites) {
		return nil, errors.New("modern HD map HUD: terrain source absent")
	}
	terrain, err := modernHUDNativeSlot(palette, func(dst []byte) error {
		return in.HUDTerrain.Sprites[tile].BlitAt(dst, 24, 0, 0)
	})
	if err != nil {
		return nil, err
	}
	hd := false
	if g.modernStoryPortraits != nil && g.nativeMapAssets != nil {
		mapID := g.nativeMapAssets.MapIndex
		if asset := g.modernStoryPortraits.terrainHD[mapID][tile]; asset != nil {
			if err := g.modernStoryPortraits.validateTerrainHD(mapID, in.HUDTerrain); err != nil {
				return nil, err
			}
			ctx, err := terrainColourContext(mapID, g.nativeMapAssets.Palette, palette, modernHDIdentityLUT())
			if err != nil {
				return nil, err
			}
			terrain, hd = asset.shade(ctx, true), true
		}
	}
	var nativeUnit *image.RGBA
	if u := in.HUD.OptionalUnit; u != nil {
		if _, err := fdicon.NativeMapHUDUnitFrameIndex(u.SelectorSlot, u.RawState); err != nil {
			return nil, err
		}
		if in.HUDUnits == nil || in.HUDCache == nil {
			return nil, errors.New("modern HD map HUD: optional unit source absent")
		}
		cycle := u.RawState
		if cycle == 3 {
			cycle = 1
		}
		sprite, err := in.HUDUnits.SpriteForNativeSlot(in.HUDCache, u.SelectorSlot, 0, cycle)
		if err != nil {
			return nil, err
		}
		nativeUnit, err = modernHUDNativeSlot(palette, func(dst []byte) error { return sprite.BlitAt(dst, 24, 0, 0) })
		if err != nil {
			return nil, err
		}
	}
	x, y := layout.Frame%456+4, layout.Frame/456+4
	g.modernHDMapHUDArt = art
	return &modernHDMapHUDPanel{Rect: image.Rect(x*6, y*6, (x+69)*6, (y+34)*6), Base: art.Base, Above: art.Above,
		Terrain: terrain, TerrainHD: hd, NativeUnit: nativeUnit}, nil
}

func clearModernHDHUDPanelPixels(ui *image.RGBA, panel *modernHDMapHUDPanel) {
	if panel == nil {
		return
	}
	for y := panel.Rect.Min.Y / 6; y < panel.Rect.Max.Y/6; y++ {
		for x := panel.Rect.Min.X / 6; x < panel.Rect.Max.X/6; x++ {
			if image.Pt(x, y).In(ui.Bounds()) {
				ui.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
}
