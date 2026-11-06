package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// Frozen with the panel, so later settlement cannot change a bar already shown.
// Colors retain the source palette; geometry is the HD presentation contract.
type modernBattlePanelChrome struct {
	ready         bool
	border, track byte
	ink           [2]byte
	width         [2]int
	frame         [149 * 42]byte
}

func freezeModernBattlePanelChrome(assets battle.NativeItemPanelDataAssets, values battle.NativeBattlePanelValues) (modernBattlePanelChrome, error) {
	if assets.BattlePanel.Width != 149 || assets.BattlePanel.Height != 42 || len(assets.BattlePanel.Pixels) != 149*42 {
		return modernBattlePanelChrome{}, fmt.Errorf("modern HD panel: chrome source unavailable")
	}
	for _, index := range []int{24, 27, 29} {
		cell, ok := assets.RawCells[index]
		if !ok || cell.Width != 1 || cell.Height != 5 || len(cell.Pixels) != 5 {
			return modernBattlePanelChrome{}, fmt.Errorf("modern HD panel: chrome raw cell %d unavailable", index)
		}
	}
	c := modernBattlePanelChrome{
		ready: true, border: assets.BattlePanel.Pixels[149+1], track: assets.RawCells[29].Pixels[1],
		ink: [2]byte{assets.RawCells[24].Pixels[1], assets.RawCells[27].Pixels[1]},
	}
	copy(c.frame[:], assets.BattlePanel.Pixels)
	for i, bar := range [][2]int{{values.HP, values.MaxHP}, {values.MP, values.MaxMP}} {
		if bar[0] == 0 || bar[1] == 0 {
			continue
		}
		// The native writer emits a left cap, repeated body, then a right cap.
		// Even a negative length still emits both caps. Clip only presentation.
		length := 101*bar[0]/bar[1] + 1
		if length < 1 {
			length = 1
		}
		c.width[i] = (length + 1) * 6
		if c.width[i] > 103*6 {
			c.width[i] = 103 * 6
		}
	}
	return c, nil
}

func (c modernBattlePanelChrome) paint(canvas *image.RGBA, palette color.Palette, background byte) error {
	if !c.ready || canvas.Bounds() != image.Rect(0, 0, 894, 252) {
		return fmt.Errorf("modern HD panel: frozen chrome unavailable")
	}
	fill := func(r image.Rectangle, index byte) {
		draw.Draw(canvas, r, image.NewUniform(palette[index]), image.Point{}, draw.Src)
	}
	stroke := func(r image.Rectangle, width int, index byte) {
		fill(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+width), index)
		fill(image.Rect(r.Min.X, r.Max.Y-width, r.Max.X, r.Max.Y), index)
		fill(image.Rect(r.Min.X, r.Min.Y, r.Min.X+width, r.Max.Y), index)
		fill(image.Rect(r.Max.X-width, r.Min.Y, r.Max.X, r.Max.Y), index)
	}
	fill(canvas.Bounds(), background)
	// FDOTHER#5/22 has three distinct perimeter pixels, including the
	// asymmetric corner highlights and lower shadow. Keep the complete source
	// bevel instead of collapsing it to one uniform outline.
	for y := 0; y < 42; y++ {
		for x := 0; x < 149; x++ {
			if x < 3 || x >= 146 || y < 3 || y >= 39 {
				fill(image.Rect(x*6, y*6, (x+1)*6, (y+1)*6), c.frame[y*149+x])
			}
		}
	}
	fill(image.Rect(24, 120, 870, 122), c.border)
	for i, y := range []int{22, 31} {
		r := image.Rect(21*6, y*6, 124*6, (y+6)*6)
		fill(r, c.track)
		if c.width[i] > 0 {
			body := image.Rect(r.Min.X, r.Min.Y, r.Min.X+c.width[i], r.Max.Y)
			fill(body, c.ink[i])
			base := color.RGBAModel.Convert(palette[c.ink[i]]).(color.RGBA)
			for row := 0; row < 6; row++ {
				light := func(v uint8) uint8 { return uint8(int(v) + (255-int(v))*(6-row)/15) }
				dark := func(v uint8) uint8 { return uint8(int(v) * (12 + row) / 18) }
				for _, edge := range []struct {
					y      int
					colour color.RGBA
				}{
					{r.Min.Y + row, color.RGBA{light(base.R), light(base.G), light(base.B), base.A}},
					{r.Max.Y - 1 - row, color.RGBA{dark(base.R), dark(base.G), dark(base.B), base.A}},
				} {
					draw.Draw(canvas, image.Rect(body.Min.X, edge.y, body.Max.X, edge.y+1), image.NewUniform(edge.colour), image.Point{}, draw.Src)
				}
			}
		}
		stroke(r, 2, c.border)
	}
	return nil
}

func (p *modernBattlePanelText) preserveNativePanelMarker(canvas *image.RGBA, palette color.Palette, r image.Rectangle) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			draw.Draw(canvas, image.Rect(x*6, y*6, (x+1)*6, (y+1)*6), image.NewUniform(palette[p.native.Pixels[y*149+x]]), image.Point{}, draw.Src)
		}
	}
}
