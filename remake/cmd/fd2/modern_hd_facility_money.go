package main

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

var facilityMoneyRect = image.Rect(5, 95, 68, 110)
var facilityMoneyDigits = image.Rect(16, 98, 64, 107)

type modernHDFacilityMoneyPanel struct {
	owner             string
	variant, resource int
	cell              fdother.RawCell
	palette           [256]color.RGBA
	font              *Font
	presented         bool
	foreground        byte
	rolling           bool
	artHD             bool
}
type modernHDMoneyCache struct {
	palette [256]color.RGBA
	image   *ebiten.Image
}

func facilityMoneySourceMatches(frame []byte, c fdother.RawCell) bool {
	if len(frame) != 64000 || c.Width != 63 || c.Height != 15 || len(c.Pixels) != 945 {
		return false
	}
	sum := sha256.Sum256(c.Pixels)
	if hex.EncodeToString(sum[:]) != "3ae75118a0206611cf5e3b8bc90821fffad3ace38c99407e33cef7c238816837" {
		return false
	}
	for y := 0; y < 15; y++ {
		for x := 0; x < 63; x++ {
			at := image.Pt(5+x, 95+y)
			if !at.In(facilityMoneyDigits) && frame[at.Y*320+at.X] != c.Pixels[y*63+x] {
				return false
			}
		}
	}
	return true
}
func (g *Game) appendModernHDFacilityMoney(frame []byte, palette color.Palette) {
	if g.modernHDText == nil {
		return
	}
	g.modernHDText.money = nil
	if g.camp == nil || g.camp.Node() == nil || g.nativeClassUI == nil || g.font == nil || g.font.sf == nil ||
		g.helpVisible || g.transientUI || g.nativeFullDACBlack || g.nativeFullDACWhite {
		return
	}
	d := &modernHDFacilityMoneyPanel{owner: g.camp.Node().Type, font: g.font}
	switch d.owner {
	case "shop":
		switch g.nativeShopMode {
		case "menu", "purchase", "confirm", "insufficient", "recipient_consumable", "recipient_equipment", "recipient_full", "no_recipient",
			"equip_confirm", "equip_roster", "equip_panel", "sell_roster", "sell_items", "sell_empty", "sell_confirm", "success", "sell_success",
			"transfer_intro", "transfer_empty", "transfer_dest_prompt", "transfer_full", "transfer_source", "transfer_dest", "transfer_items":
		default:
			return
		}
		a, _, _, ok := g.nativeShopState()
		if !ok || a == nil || g.camp.Node().NativeHubVariant != g.nativeShopVariant || a.ResourceID != map[int]int{1: 12, 3: 29, 5: 63}[g.nativeShopVariant] {
			return
		}
		d.variant, d.resource = g.nativeShopVariant, a.ResourceID
		d.cell = fdother.RawCell{Width: a.Decoration.Width, Height: a.Decoration.Height, Pixels: a.Decoration.Pixels}
	case "church":
		switch g.churchMode {
		case "menu", "class", "class_confirm", "revive", "revive_confirm", "revive_empty", "revive_insufficient",
			"status_roster", "status_commands", "status_view", "transfer_source", "transfer_dest", "transfer_item", "transfer_full":
		default:
			return
		}
		if len(g.nativeClassUI.entries) <= 1 {
			return
		}
		c := g.nativeClassUI.entries[1]
		d.resource = 14
		d.cell = fdother.RawCell{Width: c.Width, Height: c.Height, Pixels: c.Pixels}
	default:
		return
	}
	if !facilityMoneySourceMatches(frame, d.cell) {
		return
	}
	colors, ok := preparationCursorPalette(palette)
	if !ok {
		return
	}
	copy(d.palette[:], colors)
	d.cell.Pixels = append([]byte(nil), d.cell.Pixels...)

	best := -1
	for y := 4; y < 12; y++ {
		for x := 4; x < 10; x++ {
			index := d.cell.Pixels[y*63+x]
			if index == 73 {
				continue
			}
			ink := d.palette[index]
			brightness := int(ink.R) + int(ink.G) + int(ink.B)
			if brightness > best {
				d.foreground, best = index, brightness
			}
		}
	}
	if best < 0 {
		return
	}
	d.rolling = d.owner == "shop" && g.nativeShopUIJob != nil && g.nativeShopUIJob.goldText != nil && len(g.nativeShopUIJob.timeline) != 0
	g.modernHDText.money = d
}

// Match the existing HD panel contract: three one-HD-pixel bevel bands and
// three-by-three corners, retaining native colors and the original full bounds.
func (d *modernHDFacilityMoneyPanel) picture() *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, 378, 90))
	ink := func(index int) color.RGBA { return d.palette[d.cell.Pixels[index]] }
	fill := func(r image.Rectangle, c color.RGBA) {
		draw.Draw(canvas, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
	fill(canvas.Bounds(), d.palette[73])
	for band := 0; band < 3; band++ {
		fill(image.Rect(3, band, 375, band+1), ink(band*63+20))
		fill(image.Rect(375+band, 3, 376+band, 87), ink(4*63+60+band))
		fill(image.Rect(3, 87+band, 375, 88+band), ink((12+band)*63+20))
		fill(image.Rect(band, 3, band+1, 87), ink(4*63+band))
	}
	for _, corner := range []struct{ to, from image.Point }{
		{image.Pt(0, 0), image.Pt(0, 0)}, {image.Pt(375, 0), image.Pt(60, 0)},
		{image.Pt(0, 87), image.Pt(0, 12)}, {image.Pt(375, 87), image.Pt(60, 12)},
	} {
		for y := 0; y < 3; y++ {
			for x := 0; x < 3; x++ {
				canvas.SetRGBA(corner.to.X+x, corner.to.Y+y, ink((corner.from.Y+y)*63+corner.from.X+x))
			}
		}
	}
	hole := facilityMoneyDigits.Sub(facilityMoneyRect.Min)
	hole = image.Rectangle{Min: hole.Min.Mul(6), Max: hole.Max.Mul(6)}
	draw.Draw(canvas, hole, image.Transparent, image.Point{}, draw.Src)
	return canvas
}
func (d *modernHDFacilityMoneyPanel) currencyRun() modernHDTextRun {
	return modernHDTextRun{font: d.font, text: "$", x: 18, y: 198, pixels: 20, color: d.palette[d.foreground], clip: image.Rect(18, 198, 30, 214), inkTop: true}
}
func (p *modernHDPresentation) drawModernHDFacilityMoney(screen ebiten.FinalScreen, geometry ebiten.GeoM, brightness float64) {
	frame := p.Game.modernHDText
	if frame == nil || frame.money == nil {
		return
	}
	d := frame.money
	d.presented = false
	if p.moneyChrome == nil || p.moneyChrome.palette != d.palette {
		if p.moneyChrome != nil && p.moneyChrome.image != nil {
			p.moneyChrome.image.Dispose()
		}
		p.moneyChrome = &modernHDMoneyCache{palette: d.palette, image: ebiten.NewImageFromImage(d.picture())}
	}
	options := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	options.GeoM.Scale(1.0/3, 1.0/3)
	options.GeoM.Translate(10, 190)
	options.GeoM.Concat(geometry)
	value := float32(math.Max(0, math.Min(1, brightness)))
	options.ColorScale.Scale(value, value, value, 1)
	screen.DrawImage(p.moneyChrome.image, options)
	d.presented, d.artHD = true, p.Game.modernStoryPortraits != nil
}
