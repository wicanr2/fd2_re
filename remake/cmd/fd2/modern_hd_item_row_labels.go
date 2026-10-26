package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"golang.org/x/image/font/sfnt"
)

type modernHDItemRowLabels struct {
	seal            [32]byte
	items, commands *modernHDItemPanelValues
}

type modernHDItemRowLabel struct {
	source int
	rect   image.Rectangle
}

func modernHDItemRowLabelsSeal(d *nativeUnitStatusData, palette color.Palette) [32]byte {
	if d == nil {
		return [32]byte{}
	}
	var cells []fdother.RawCell
	for _, index := range []int{64, 65, 66, 67, 92} {
		cells = append(cells, d.assets.RawCells[index])
	}
	raw, _ := json.Marshal([]any{d.record, d.itemRows, d.commandIDs, d.commandBook, d.assets.ItemPanelBottom, cells, palette})
	return sha256.Sum256(raw)
}

func prepareModernHDItemRowLabels(d *nativeUnitStatusData, palette color.Palette, f *Font) (*modernHDItemRowLabels, error) {
	if d == nil || len(d.record) != 80 || len(palette) != 256 || f == nil || d.assets.ItemPanelBottom.Width != 310 || d.assets.ItemPanelBottom.Height != 99 || len(d.assets.ItemPanelBottom.Pixels) != 310*99 {
		return nil, errors.New("modern HD row labels: source unavailable")
	}
	seal := modernHDItemRowLabelsSeal(d, palette)
	if d.rowLabels != nil {
		if d.rowLabels.seal != seal {
			return nil, errors.New("modern HD row labels: changed frozen source")
		}
		return d.rowLabels, nil
	}
	labels := map[int]string{64: "+AP", 65: "+DP", 66: "+HP", 67: "+MP", 92: "-MP"}
	background := d.assets.ItemPanelBottom.Pixels[70*310+100]
	var buffer sfnt.Buffer
	for index, text := range labels {
		cell := d.assets.RawCells[index]
		width := 21
		if index == 92 {
			width = 20
		}
		if cell.Width != width || cell.Height != 8 || len(cell.Pixels) != width*8 || cell.Pixels[0] != background {
			return nil, fmt.Errorf("modern HD row labels: source cell%d differs", index)
		}
		for _, ch := range text {
			glyph, err := f.sf.GlyphIndex(&buffer, ch)
			if err != nil || glyph == 0 {
				return nil, errors.New("modern HD row labels: missing glyph")
			}
		}
	}
	cells, err := battle.NativeItemSelectorCells(d.record, 0, -1, d.itemRows)
	if err != nil {
		return nil, err
	}
	var items, commands []modernHDItemRowLabel
	for _, cell := range cells {
		if cell.HasStatValue && cell.StatIcon >= 64 && cell.StatIcon <= 67 {
			x, y := cell.LabelX+68, cell.LabelY+4
			items = append(items, modernHDItemRowLabel{cell.StatIcon, image.Rect(x, y, x+21, y+8)})
		}
	}
	for i, id := range d.commandIDs {
		if id < 0 || id >= len(d.commandBook) || d.commandBook[id].ID != id {
			return nil, errors.New("modern HD row labels: command source unavailable")
		}
		x, y := 68+100*(i/4), 108+22*(i%4)
		commands = append(commands, modernHDItemRowLabel{92, image.Rect(x, y, x+20, y+8)})
	}
	paint := func(fields []modernHDItemRowLabel) (*modernHDItemPanelValues, error) {
		if len(fields) == 0 {
			return nil, nil
		}
		canvas := image.NewRGBA(image.Rect(0, 0, 310*6, 99*6))
		rgba := func(index byte) (color.RGBA, error) {
			if palette[index] == nil {
				return color.RGBA{}, errors.New("modern HD row labels: palette unavailable")
			}
			r, g, b, a := palette[index].RGBA()
			if a != 65535 {
				return color.RGBA{}, errors.New("modern HD row labels: translucent source")
			}
			return color.RGBA{byte(r >> 8), byte(g >> 8), byte(b >> 8), 255}, nil
		}
		bg, err := rgba(background)
		if err != nil {
			return nil, err
		}
		for _, field := range fields {
			if !field.rect.In(image.Rect(5, 94, 315, 193)) {
				return nil, errors.New("modern HD row labels: field outside source panel")
			}
			local := field.rect.Sub(image.Pt(5, 94))
			r := image.Rectangle{Min: local.Min.Mul(6), Max: local.Max.Mul(6)}
			draw.Draw(canvas, r, image.NewUniform(bg), image.Point{}, draw.Src)
			var ink color.RGBA
			best := -1
			for _, index := range d.assets.RawCells[field.source].Pixels {
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
			if best < 0 {
				return nil, errors.New("modern HD row labels: source ink unavailable")
			}
			if err := drawModernBattlePanelInk(canvas, f, labels[field.source], r, 48, ink); err != nil {
				return nil, err
			}
		}
		return &modernHDItemPanelValues{seal: seal, picture: canvas}, nil
	}
	itemArt, err := paint(items)
	if err != nil {
		return nil, err
	}
	commandArt, err := paint(commands)
	if err != nil {
		return nil, err
	}
	return &modernHDItemRowLabels{seal: seal, items: itemArt, commands: commandArt}, nil
}
