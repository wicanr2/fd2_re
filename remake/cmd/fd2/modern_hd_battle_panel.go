package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Frozen at the native panel write, before settlement or a later HP step.
// The native image is also the exact source identity; no glyph recognition.
type modernBattlePanelText struct {
	native      figani.Frame
	name        string
	values      battle.NativeBattlePanelValues
	background  byte
	digitInk    [2]byte
	font        *Font
	lastPalette [32]byte
	picture     *image.RGBA
	key         string
}

func (g *Game) renderPhysicalBattlePanel(scene *nativePhysicalScene, record, dst []byte, index int) error {
	if err := g.renderLocalizedNativeBattlePanel(scene.panelAssets, record, dst, index, g.handlerChapter); err != nil {
		return err
	}
	if g.font == nil || g.localeEntities == nil {
		return nil
	}
	x, y, err := battle.NativeBattlePanelOrigin(record, index, g.handlerChapter)
	if err != nil {
		return err
	}
	name, err := g.localeEntities.BattleName(int(record[8]))
	if err != nil {
		return nil // An unresolved optional name must not change native reachability.
	}
	f := g.font
	if regional := f.hdFonts[g.localeID]; regional != nil {
		f = regional
	}
	p := &modernBattlePanelText{font: f, name: name, background: scene.panelAssets.BattlePanel.Pixels[10*149+100]}
	for i, base := range []int{31, 42} {
		counts := [256]int{}
		frame := scene.panelAssets.Frames[base]
		for k, pixel := range frame.Indexed {
			if frame.Mask[k] != 0 && pixel != p.background {
				counts[pixel]++
			}
		}
		brightness := uint32(0)
		for pixel, count := range counts {
			r, green, b, _ := g.nativeUIPalette[pixel].RGBA()
			light := 30*r + 59*green + 11*b
			if count > 0 && light > brightness {
				p.digitInk[i] = byte(pixel)
				brightness = light
			}
		}
	}
	p.native = figani.Frame{X: x, Y: y, Width: 149, Height: 42, Pixels: make([]byte, 149*42), Mask: bytes.Repeat([]byte{255}, 149*42)}
	for row := 0; row < 42; row++ {
		copy(p.native.Pixels[row*149:(row+1)*149], dst[(y+row)*320+x:(y+row)*320+x+149])
	}
	word := func(offset int) int { return int(int16(binary.LittleEndian.Uint16(record[offset:]))) }
	p.values = battle.NativeBattlePanelValues{Level: int(record[0x21]), HP: word(0x40), MaxHP: word(0x42), MP: word(0x44), MaxMP: word(0x46)}
	// Validate before publishing this scene or committing any combat state.
	if _, _, err := p.raster(g.nativeUIPalette); err != nil {
		return err
	}
	for _, previous := range scene.hdPanelText {
		if previous.native.X == x && previous.native.Y == y && bytes.Equal(previous.native.Pixels, p.native.Pixels) {
			return nil
		}
	}
	scene.hdPanelText = append(scene.hdPanelText, p)
	return nil
}

// Bind only a complete, byte-identical panel at its proven native origin.
// Partial scroll views inherit the explicit layer and clip from their producer.
func bindModernPhysicalPanelText(scene *nativePhysicalScene, sources *modernHDBattleSources) {
	if scene == nil || sources == nil || len(sources.Base) != 64000 {
		return
	}
	for _, p := range scene.hdPanelText {
		f := p.native
		matches := true
		for row := 0; row < f.Height; row++ {
			if !bytes.Equal(f.Pixels[row*f.Width:(row+1)*f.Width], sources.Base[(f.Y+row)*320+f.X:(f.Y+row)*320+f.X+f.Width]) {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		found := false
		for _, layer := range sources.BaseScenery {
			if layer.PanelText == p {
				found = true
				break
			}
		}
		if !found {
			sources.BaseScenery = append(sources.BaseScenery, modernHDBattleSource{Frame: f, Resource: -1, Fill: -1, PanelText: p})
		}
	}
}

func (p *modernBattlePanelText) raster(palette color.Palette) (image.Image, string, error) {
	raw, err := modernHDBattlePalette(palette)
	if err != nil {
		return nil, "", err
	}
	key := sha256.Sum256(raw)
	if p.picture != nil && key == p.lastPalette {
		return p.picture, p.key, nil
	}
	if p.font == nil || p.native.Width != 149 || p.native.Height != 42 || len(p.native.Pixels) != 149*42 {
		return nil, "", fmt.Errorf("modern HD panel: frozen source unavailable")
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 894, 252))
	for y := 0; y < 42; y++ {
		for x := 0; x < 149; x++ {
			c := palette[p.native.Pixels[y*149+x]]
			draw.Draw(canvas, image.Rect(x*6, y*6, (x+1)*6, (y+1)*6), image.NewUniform(c), image.Point{}, draw.Src)
		}
	}
	type field struct {
		value  string
		rect   image.Rectangle
		size   int
		colour byte
	}
	fields := []field{{p.name, image.Rect(5, 4, 114, 20), 96, 0xcd}, {"LV·", image.Rect(115, 4, 131, 13), 48, 0xcd},
		{"HP", image.Rect(3, 21, 20, 30), 48, 0xcd}, {"MP", image.Rect(3, 30, 20, 39), 48, 0xcd}}
	for _, v := range []struct{ value, max, width, x, y int }{
		{p.values.Level, 99, 2, 132, 4}, {p.values.HP, 999, 3, 126, 21}, {p.values.MP, 999, 3, 126, 30},
	} {
		if v.value > v.max {
			continue
		} // Keep the original overflow glyph.
		value := v.value
		if value < 0 {
			value = 0
		}
		ink := p.digitInk[0]
		if v.y == 21 && p.values.HP != p.values.MaxHP || v.y == 30 && p.values.MP != p.values.MaxMP {
			ink = p.digitInk[1]
		}
		fields = append(fields, field{fmt.Sprintf("%0*d", v.width, value), image.Rect(v.x, v.y, v.x+v.width*6+1, v.y+9), 48, ink})
	}
	for i, field := range fields {
		r := image.Rectangle{Min: field.rect.Min.Mul(6), Max: field.rect.Max.Mul(6)}
		erase := r
		if i == 0 {
			// BlitNativeGlyph writes shadow at (x-1,y+1), including row20.
			// Clear the full former name and LV-label band; leave LV digits.
			erase = image.Rect(4*6, 4*6, 132*6, 21*6)
		}
		draw.Draw(canvas, erase, image.NewUniform(palette[p.background]), image.Point{}, draw.Src)
		if err := drawModernBattlePanelInk(canvas, p.font, field.value, r, field.size, modernHDNativeTextColor(palette, field.colour)); err != nil {
			return nil, "", err
		}
	}
	digest := sha256.Sum256(canvas.Pix)
	p.picture, p.lastPalette, p.key = canvas, key, fmt.Sprintf("panel-text/%x", digest)
	return canvas, p.key, nil
}

func drawModernBattlePanelInk(dst *image.RGBA, f *Font, value string, rect image.Rectangle, maximum int, ink color.RGBA) error {
	for size := maximum; size >= 24; size-- {
		face, _ := f.faceFor(size)
		if face == nil {
			return fmt.Errorf("modern HD panel: font unavailable")
		}
		bounds, _ := font.BoundString(face, value)
		box := image.Rect(bounds.Min.X.Floor(), bounds.Min.Y.Floor(), bounds.Max.X.Ceil(), bounds.Max.Y.Ceil())
		if box.Dx() > rect.Dx() || box.Dy() > rect.Dy() {
			continue
		}
		pen := fixed.P(rect.Min.X-box.Min.X, rect.Min.Y-box.Min.Y)
		d := font.Drawer{Dst: dst.SubImage(rect).(*image.RGBA), Src: image.NewUniform(ink), Face: face, Dot: pen}
		d.DrawString(value)
		return nil
	}
	return fmt.Errorf("modern HD panel: text exceeds safe rectangle: %q", value)
}
