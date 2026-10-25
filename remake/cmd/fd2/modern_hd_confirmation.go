package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"golang.org/x/image/font/sfnt"
)

var modernHDConfirmationIndices = [...]int{16, 17, 48, 49, 51, 52}

type modernHDConfirmationCell struct {
	height                  int
	background, ink, shadow color.RGBA
}

type modernHDConfirmation struct {
	cells  map[int]*modernHDConfirmationCell
	labels map[string][2]string
	fonts  map[string]*Font
	bounds map[string][2]image.Rectangle
	images map[modernHDConfirmationImageKey]*ebiten.Image
}

type modernHDConfirmationImageKey struct {
	cell          *modernHDConfirmationCell
	locale        string
	option, pulse int
}

type modernHDConfirmationPlacement struct {
	cell        *modernHDConfirmationCell
	sourceIndex int
	rect        image.Rectangle // Native 320x200 coordinates.
	option      int
	pulse       int
	owner       *modernHDConfirmation
	locale      string
}

func freezeModernHDConfirmation(cells []fdother.RawCell, palette color.Palette, font *Font) (*modernHDConfirmation, error) {
	if len(cells) <= 52 || len(palette) != 256 {
		return nil, errors.New("modern HD confirmation: source unavailable")
	}
	result := &modernHDConfirmation{cells: map[int]*modernHDConfirmationCell{}, labels: map[string][2]string{}, fonts: map[string]*Font{}, bounds: map[string][2]image.Rectangle{}, images: map[modernHDConfirmationImageKey]*ebiten.Image{}}
	for _, index := range modernHDConfirmationIndices {
		cell := cells[index]
		height := 16
		if index == 16 || index == 17 {
			height = 20
		}
		if cell.Width != 24 || cell.Height != height || len(cell.Pixels) != 24*height {
			return nil, fmt.Errorf("modern HD confirmation: cell%d shape differs", index)
		}
		colorAt := func(x, y int) (color.RGBA, error) {
			ink := palette[cell.Pixels[y*24+x]]
			if ink == nil {
				return color.RGBA{}, errors.New("modern HD confirmation: palette unavailable")
			}
			r, g, b, a := ink.RGBA()
			if a != 65535 {
				return color.RGBA{}, errors.New("modern HD confirmation: source color is not opaque")
			}
			return color.RGBA{byte(r >> 8), byte(g >> 8), byte(b >> 8), 255}, nil
		}
		art := &modernHDConfirmationCell{height: height}
		var err error
		if art.background, err = colorAt(3, 3); err != nil {
			return nil, err
		}
		if art.ink, err = colorAt(12, 0); err != nil {
			return nil, err
		}
		if art.shadow, err = colorAt(12, 1); err != nil {
			return nil, err
		}
		result.cells[index] = art
	}
	if font == nil || font.sf == nil {
		return nil, errors.New("modern HD confirmation: font unavailable")
	}
	for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
		catalog, err := loadOfficialLocale(locale)
		if err != nil {
			return nil, err
		}
		f := font
		if regional := font.hdFonts[locale]; regional != nil {
			f = regional
		}
		if f.sf == nil {
			return nil, errors.New("modern HD confirmation: regional font unavailable")
		}
		face, _ := f.faceFor(40)
		if face == nil {
			return nil, errors.New("modern HD confirmation: font face unavailable")
		}
		var labels [2]string
		var bounds [2]image.Rectangle
		for option, key := range []string{"common.yes", "common.no"} {
			label, err := catalog.Format(key)
			if err != nil || label == "" {
				return nil, fmt.Errorf("modern HD confirmation: %s/%s unavailable", locale, key)
			}
			var buffer sfnt.Buffer
			for _, rune := range label {
				glyph, err := f.sf.GlyphIndex(&buffer, rune)
				if err != nil || glyph == 0 {
					return nil, errors.New("modern HD confirmation: label glyph unavailable")
				}
			}
			bounds[option] = text.BoundString(face, label)
			// The smallest native cell is 24x16. Include a one-HD-pixel
			// shadow and the selected six-HD-pixel downward pulse.
			b := bounds[option]
			left, top := (144-b.Dx())/2, (96-b.Dy())/2
			ink := image.Rect(left, top, left+b.Dx()+1, top+b.Dy()+7)
			if b.Empty() || !ink.In(image.Rect(12, 12, 132, 84)) {
				return nil, fmt.Errorf("modern HD confirmation: %s label exceeds text-safe area", locale)
			}
			labels[option] = label
		}
		result.labels[locale], result.bounds[locale], result.fonts[locale] = labels, bounds, f
	}
	return result, nil
}

func (c *modernHDConfirmationCell) patch() *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, 144, c.height*6))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(c.ink), image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds().Inset(3), image.NewUniform(c.background), image.Point{}, draw.Src)
	// Thin inner lower/right shadow uses the same native cell's shade.
	for _, rect := range []image.Rectangle{image.Rect(3, c.height*6-4, 141, c.height*6-3), image.Rect(140, 3, 141, c.height*6-3)} {
		draw.Draw(canvas, rect, image.NewUniform(c.shadow), image.Point{}, draw.Src)
	}
	return canvas
}

func (c *modernHDConfirmation) placements(kind fullRewardPhase, index, choice, pulse int) ([]modernHDConfirmationPlacement, error) {
	spread, indices := 16, [2]int{48, 51}
	switch kind {
	case fullRewardChoice:
		if choice < 0 || choice > 1 || pulse < 0 || pulse > 1 {
			return nil, errors.New("modern HD confirmation: invalid selection")
		}
		indices[choice] += pulse
	case fullRewardOpening:
		if index < 6 {
			return nil, nil
		}
		if index > 9 {
			return nil, errors.New("modern HD confirmation: opening index outside source")
		}
		spread, indices, pulse = (index-5)*4, [2]int{16, 17}, 0
	case fullRewardChoiceClosing:
		if index < 0 || index > 3 {
			return nil, errors.New("modern HD confirmation: closing index outside source")
		}
		spread, indices, pulse = 12-index*4, [2]int{16, 17}, 0
	default:
		return nil, nil
	}
	var result []modernHDConfirmationPlacement
	for option, index := range indices {
		cell := c.cells[index]
		if cell == nil {
			return nil, errors.New("modern HD confirmation: frozen cell unavailable")
		}
		x := 248 - spread
		if option == 1 {
			x = 248 + spread
		}
		shift := 0
		if kind == fullRewardChoice && option == choice {
			shift = pulse
		}
		result = append(result, modernHDConfirmationPlacement{cell: cell, sourceIndex: index, rect: image.Rect(x, 168, x+24, 168+cell.height), option: option, pulse: shift})
	}
	return result, nil
}

func (g *Game) appendModernHDConfirmation(c *modernHDConfirmation, kind fullRewardPhase, index, choice, pulse int) error {
	if g.modernHDText == nil {
		return nil
	}
	if c == nil {
		return errors.New("modern HD confirmation: frozen source unavailable")
	}
	placements, err := c.placements(kind, index, choice, pulse)
	if err != nil {
		return err
	}
	locale := g.localeID
	if locale == "" {
		locale = "zh-Hant"
	}
	if _, ok := c.labels[locale]; !ok {
		return errors.New("modern HD confirmation: unsupported locale")
	}
	for _, p := range placements {
		p.owner, p.locale = c, locale
		g.modernHDText.choices = append(g.modernHDText.choices, p)
	}
	return nil
}

// Rasterize at the target HD size before composing each button. This retains
// native left-then-right occlusion when the opening/closing cells overlap.
func (c *modernHDConfirmation) buttonImage(p modernHDConfirmationPlacement) *ebiten.Image {
	key := modernHDConfirmationImageKey{cell: p.cell, locale: p.locale, option: p.option, pulse: p.pulse}
	if cached := c.images[key]; cached != nil {
		return cached
	}
	result := ebiten.NewImageFromImage(p.cell.patch())
	face, _ := c.fonts[p.locale].faceFor(40)
	label, bounds := c.labels[p.locale][p.option], c.bounds[p.locale][p.option]
	x := (144-bounds.Dx())/2 - bounds.Min.X
	baseline := (p.rect.Dy()*6-bounds.Dy())/2 + p.pulse*6 - bounds.Min.Y
	target := result.SubImage(result.Bounds().Inset(12)).(*ebiten.Image)
	for layer, ink := range []color.RGBA{p.cell.shadow, p.cell.ink} {
		offset := 1 - layer
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x+offset), float64(baseline+offset))
		op.ColorScale.ScaleWithColor(ink)
		text.DrawWithOptions(target, label, face, op)
	}
	c.images[key] = result
	return result
}

func (p *modernHDPresentation) drawModernHDConfirmation(screen ebiten.FinalScreen, geometry ebiten.GeoM, brightness float64) {
	if p.Game.modernHDText == nil {
		return
	}
	for _, button := range p.Game.modernHDText.choices {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(1.0/3, 1.0/3)
		op.GeoM.Translate(float64(button.rect.Min.X*2), float64(button.rect.Min.Y*2))
		op.GeoM.Concat(geometry)
		v := float32(math.Max(0, math.Min(1, brightness)))
		op.ColorScale.Scale(v, v, v, 1)
		screen.DrawImage(button.owner.buttonImage(button), op)
	}
}
