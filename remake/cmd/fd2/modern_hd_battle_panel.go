package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"

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
	chrome      modernBattlePanelChrome
	background  byte
	digitInk    [2]byte
	font        *Font
	lastPalette [32]byte
	lastScale   float64
	picture     *image.RGBA
	key         string
}

func stageModernBattlePanelText(collectors []*[]*modernBattlePanelText) (*[]*modernBattlePanelText, func(), error) {
	if len(collectors) == 0 {
		return nil, func() {}, nil
	}
	if len(collectors) != 1 || collectors[0] == nil {
		return nil, nil, fmt.Errorf("modern HD panel: invalid collector")
	}
	staged := append([]*modernBattlePanelText(nil), (*collectors[0])...)
	return &staged, func() { *collectors[0] = staged }, nil
}

func bindModernBattlePanelFrames(panels []*modernBattlePanelText, groups ...[]nativeCompoundPresentedFrame) {
	for _, group := range groups {
		for _, frame := range group {
			bindModernBattlePanelText(panels, frame.hdSources)
		}
	}
}

func bindModernBattlePanelBank(panels []*modernBattlePanelText, bank modernHDCommandBankSource) {
	bindModernBattlePanelFrames(panels, bank.sourceFrames())
}

func (g *Game) renderPhysicalBattlePanel(scene *nativePhysicalScene, record, dst []byte, index int) error {
	return g.renderBattlePanelText(scene.panelAssets, record, dst, index, g.handlerChapter, &scene.hdPanelText)
}

func (g *Game) renderBattlePanelText(assets battle.NativeItemPanelDataAssets, record, dst []byte, index, chapter int, collected *[]*modernBattlePanelText) error {
	if err := g.renderLocalizedNativeBattlePanel(assets, record, dst, index, chapter); err != nil {
		return err
	}
	if collected == nil || g.font == nil || g.localeEntities == nil {
		return nil
	}
	x, y, err := battle.NativeBattlePanelOrigin(record, index, chapter)
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
	p := &modernBattlePanelText{font: f, name: name, background: assets.BattlePanel.Pixels[10*149+100]}
	for i, base := range []int{31, 42} {
		counts := [256]int{}
		frame := assets.Frames[base]
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
	p.chrome, err = freezeModernBattlePanelChrome(assets, p.values)
	if err != nil {
		return err
	}
	for _, previous := range *collected {
		if previous.native.X == x && previous.native.Y == y && previous.name == p.name && previous.values == p.values && previous.chrome == p.chrome && previous.font == p.font && bytes.Equal(previous.native.Pixels, p.native.Pixels) {
			return nil
		}
	}
	// Validate before publishing this scene or committing any combat state.
	if _, _, err := p.raster(g.nativeUIPalette); err != nil {
		return err
	}
	*collected = append(*collected, p)
	return nil
}

// Bind only a complete, byte-identical panel at its proven native origin.
// Partial scroll views inherit the explicit layer and clip from their producer.
func bindModernPhysicalPanelText(scene *nativePhysicalScene, sources *modernHDBattleSources) {
	if scene != nil {
		bindModernBattlePanelText(scene.hdPanelText, sources)
	}
}

func bindModernBattlePanelText(panels []*modernBattlePanelText, sources *modernHDBattleSources) {
	if sources == nil || len(sources.Base) != 64000 {
		return
	}
	cloned := false
	for _, p := range panels {
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
		for _, layer := range append(append([]modernHDBattleSource(nil), sources.BaseScenery...), sources.Layers...) {
			if layer.PanelText == p && layer.DX == 0 && layer.DY == 0 {
				found = true
				break
			}
		}
		if !found {
			if !cloned {
				sources.BaseScenery = append([]modernHDBattleSource(nil), sources.BaseScenery...)
				cloned = true
			}
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
	scale := 1.0
	if p.font != nil && p.font.userScale > 0 {
		scale = p.font.userScale
	}
	if p.picture != nil && key == p.lastPalette && scale == p.lastScale {
		return p.picture, p.key, nil
	}
	if p.font == nil || p.native.Width != 149 || p.native.Height != 42 || len(p.native.Pixels) != 149*42 {
		return nil, "", fmt.Errorf("modern HD panel: frozen source unavailable")
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 894, 252))
	if err := p.chrome.paint(canvas, palette, p.background); err != nil {
		return nil, "", err
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
			p.preserveNativePanelMarker(canvas, palette, image.Rect(v.x, v.y, 147, v.y+9))
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
	for _, field := range fields {
		r := image.Rectangle{Min: field.rect.Min.Mul(6), Max: field.rect.Max.Mul(6)}
		if err := drawModernBattlePanelInk(canvas, p.font, field.value, r, field.size, nativePaletteColor(palette, field.colour, color.RGBA{240, 244, 255, 255})); err != nil {
			return nil, "", err
		}
	}
	digest := sha256.Sum256(canvas.Pix)
	p.picture, p.lastPalette, p.key = canvas, key, fmt.Sprintf("panel-text/%x", digest)
	p.lastScale = scale
	return canvas, p.key, nil
}

func drawModernBattlePanelInk(dst *image.RGBA, f *Font, value string, rect image.Rectangle, maximum int, ink color.RGBA) error {
	maximum = int(f.effectiveScale(float64(maximum)))
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
