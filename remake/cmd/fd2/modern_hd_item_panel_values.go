package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"golang.org/x/image/font/sfnt"
)

type modernHDItemPanelValues struct {
	seal    [32]byte
	picture *image.RGBA
	gpu     *ebiten.Image
}

type modernHDItemPanelLabel struct {
	text string
	rect image.Rectangle // Native 320x200 coordinates, measured from entry20.
}

func modernHDItemPanelLabels() []modernHDItemPanelLabel {
	return []modernHDItemPanelLabel{
		{"LV·", image.Rect(139, 33, 157, 41)}, {"EX·", image.Rect(139, 44, 157, 52)},
		{"MU·", image.Rect(138, 55, 157, 63)}, {"DX·", image.Rect(99, 55, 117, 63)},
		{"HIT·", image.Rect(96, 67, 117, 75)}, {"EV·", image.Rect(99, 79, 117, 87)},
		{"AP·", image.Rect(139, 67, 157, 75)}, {"DP·", image.Rect(139, 79, 157, 87)},
		{"HP·", image.Rect(177, 32, 197, 40)}, {"MP·", image.Rect(177, 51, 197, 59)},
		{"/", image.Rect(286, 41, 293, 49)}, {"/", image.Rect(286, 59, 293, 67)},
	}
}

func modernHDItemPanelValuesSeal(d *nativeUnitStatusData, palette color.Palette) [32]byte {
	if d == nil {
		return [32]byte{}
	}
	var cells []fdother.RawCell
	for i := 23; i <= 30; i++ {
		cells = append(cells, d.assets.RawCells[i])
	}
	raw, _ := json.Marshal([]any{d.record, d.assets.ItemPanelUpper, cells, palette})
	return sha256.Sum256(raw)
}

// This is the existing signed native bar rule, clipped only at presentation.
func modernHDItemPanelBarWidth(current, maximum int) int {
	if current == 0 || maximum == 0 {
		return 0
	}
	length := 101*current/maximum + 1
	if length < 1 {
		length = 1
	}
	if length > 102 {
		length = 102
	}
	return (length + 1) * 6
}

func prepareModernHDItemPanelValues(d *nativeUnitStatusData, palette color.Palette, f *Font) (*modernHDItemPanelValues, error) {
	if d == nil || len(d.record) != 80 || f == nil || len(palette) != 256 || d.assets.ItemPanelUpper.Width != 223 || d.assets.ItemPanelUpper.Height != 86 || len(d.assets.ItemPanelUpper.Pixels) != 223*86 {
		return nil, errors.New("modern HD item values: source unavailable")
	}
	seal := modernHDItemPanelValuesSeal(d, palette)
	if d.itemValues != nil {
		if d.itemValues.seal != seal {
			return nil, errors.New("modern HD item values: changed frozen source")
		}
		return d.itemValues, nil
	}
	for i := 23; i <= 30; i++ {
		cell := d.assets.RawCells[i]
		if cell.Width != 1 || cell.Height != 5 || len(cell.Pixels) != 5 {
			return nil, fmt.Errorf("modern HD item values: bar cell%d unavailable", i)
		}
	}
	rgba := func(index byte) (color.RGBA, error) {
		if palette[index] == nil {
			return color.RGBA{}, errors.New("modern HD item values: palette unavailable")
		}
		r, g, b, a := palette[index].RGBA()
		if a != 65535 {
			return color.RGBA{}, errors.New("modern HD item values: translucent source")
		}
		return color.RGBA{byte(r >> 8), byte(g >> 8), byte(b >> 8), 255}, nil
	}
	// Entry13 belongs to dialogue. Entry20 has its own panel background.
	background := d.assets.ItemPanelUpper.Pixels[70*223+100]
	indices := []byte{background, d.assets.ItemPanelUpper.Pixels[224], d.assets.RawCells[24].Pixels[1], d.assets.RawCells[27].Pixels[1], d.assets.RawCells[29].Pixels[1]}
	colors := make([]color.RGBA, len(indices))
	for i, index := range indices {
		var err error
		colors[i], err = rgba(index)
		if err != nil {
			return nil, err
		}
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 223*6, 86*6))
	var buffer sfnt.Buffer
	for _, label := range modernHDItemPanelLabels() {
		for _, ch := range label.text {
			index, err := f.sf.GlyphIndex(&buffer, ch)
			if err != nil || index == 0 {
				return nil, errors.New("modern HD item values: missing label glyph")
			}
		}
	}
	local := func(r image.Rectangle) image.Rectangle {
		r = r.Sub(image.Pt(92, 7))
		return image.Rectangle{Min: r.Min.Mul(6), Max: r.Max.Mul(6)}
	}
	fill := func(r image.Rectangle, c color.RGBA) {
		draw.Draw(canvas, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
	for _, label := range modernHDItemPanelLabels() {
		r := local(label.rect)
		fill(r, colors[0])
		var ink color.RGBA
		best := -1
		for y := label.rect.Min.Y; y < label.rect.Max.Y; y++ {
			for x := label.rect.Min.X; x < label.rect.Max.X; x++ {
				index := d.assets.ItemPanelUpper.Pixels[(y-7)*223+x-92]
				if index == background {
					continue
				}
				c, err := rgba(index)
				if err != nil {
					return nil, err
				}
				if brightness := int(c.R) + int(c.G) + int(c.B); brightness > best {
					ink, best = c, brightness
				}
			}
		}
		if best < 0 {
			return nil, errors.New("modern HD item values: source label unavailable")
		}
		if err := drawModernBattlePanelInk(canvas, f, label.text, r, 48, ink); err != nil {
			return nil, err
		}
	}
	for i, y := range []int{33, 52} {
		fill(local(image.Rect(197, y-1, 302, y+6)), colors[0])
		r := local(image.Rect(198, y, 301, y+5))
		fill(r, colors[4])
		offset := 64 + i*4
		current := int(int16(binary.LittleEndian.Uint16(d.record[offset:])))
		maximum := int(int16(binary.LittleEndian.Uint16(d.record[offset+2:])))
		if width := modernHDItemPanelBarWidth(current, maximum); width != 0 {
			fill(image.Rect(r.Min.X, r.Min.Y, r.Min.X+width, r.Max.Y), colors[2+i])
		}
		for _, edge := range []image.Rectangle{image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1), image.Rect(r.Min.X, r.Max.Y-1, r.Max.X, r.Max.Y), image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y), image.Rect(r.Max.X-1, r.Min.Y, r.Max.X, r.Max.Y)} {
			fill(edge, colors[1])
		}
	}
	return &modernHDItemPanelValues{seal: seal, picture: canvas}, nil
}
