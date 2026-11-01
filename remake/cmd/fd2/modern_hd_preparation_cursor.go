package main

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/draw"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

type modernPreparationCursor struct {
	key     [32]byte
	picture *image.RGBA
	gpu     *ebiten.Image
}

type modernPreparationNativeCursorUnit struct {
	key     [32]byte
	picture *image.RGBA
	gpu     *ebiten.Image
}

func preparationCursorSourceMatches(sprite fdicon.Sprite) bool {
	match := func(raw []byte, want string) bool {
		sum := sha256.Sum256(raw)
		return len(raw) == 24*24 && hex.EncodeToString(sum[:]) == want
	}
	return match(sprite.Pixels, "81b0eb08a1950bf99690c0366df3b9a90d194bd84a2498f0a6ca7d9b88aedf2e") &&
		match(sprite.Mask, "329fd24c50c8cf6420bcfbff2fddaf24ae6f86a935175804c1737e9c11a164e7") &&
		match(sprite.RemapMask, "2bedf3eb1b34094c6b569bed31705229e709264a78ef7e935c5f026a4692b149")
}

func preparationCursorPalette(palette color.Palette) ([]color.RGBA, bool) {
	if len(palette) != 256 {
		return nil, false
	}
	result := make([]color.RGBA, 256)
	for i, c := range palette {
		if c == nil {
			return nil, false
		}
		result[i] = color.RGBAModel.Convert(c).(color.RGBA)
		if result[i].A != 255 {
			return nil, false
		}
	}
	return result, true
}

// The approved native-color thin bevel keeps the original four corner extents.
// It is a code-native UI material, independent of character artwork.
func preparationCursorPatch(sprite fdicon.Sprite, palette []color.RGBA) *image.RGBA {
	picture := image.NewRGBA(image.Rect(0, 0, 144, 144))
	at := func(x, y int) color.RGBA { return palette[sprite.Pixels[y*24+x]] }
	for band := 0; band < 3; band++ {
		for x := 0; x < 144; x++ {
			if x >= 60 && x < 84 {
				continue
			}
			picture.SetRGBA(x, band, at(4, band))
			picture.SetRGBA(x, 143-band, at(4, 23-band))
		}
		for y := 0; y < 144; y++ {
			if y >= 54 && y < 78 {
				continue
			}
			picture.SetRGBA(band, y, at(band, 4))
			picture.SetRGBA(143-band, y, at(23-band, 4))
		}
	}
	for _, c := range [][4]int{{0, 0, 0, 0}, {141, 0, 21, 0}, {0, 141, 0, 21}, {141, 141, 21, 21}} {
		for y := 0; y < 3; y++ {
			for x := 0; x < 3; x++ {
				picture.SetRGBA(c[0]+x, c[1]+y, at(c[2]+x, c[3]+y))
			}
		}
	}
	return picture
}

func (g *Game) prepareModernHDPreparationCursor() *modernPreparationCursor {
	if g.interfaceMode != playerInterfaceHD || g.nativeClassUI == nil || g.nativePreparationUI == nil ||
		g.nativePreparationUI.roster == nil || g.nativePreparationUI.roster.Cursor == nil ||
		len(g.nativePreparationUI.roster.Cursor.Sprites) == 0 {
		return nil
	}
	sprite := g.nativePreparationUI.roster.Cursor.Sprites[0]
	if !preparationCursorSourceMatches(sprite) {
		return nil
	}
	palette, ok := preparationCursorPalette(g.nativeClassUI.palette)
	if !ok {
		return nil
	}
	raw := append([]byte(nil), sprite.Pixels...)
	raw = append(raw, sprite.Mask...)
	raw = append(raw, sprite.RemapMask...)
	for _, c := range palette {
		raw = append(raw, c.R, c.G, c.B, c.A)
	}
	key := sha256.Sum256(raw)
	if old := g.prepCursorHD; old != nil && old.key == key {
		return old
	}
	c := &modernPreparationCursor{key: key, picture: preparationCursorPatch(sprite, palette)}
	if old := g.prepCursorHD; old != nil && old.gpu != nil {
		old.gpu.Dispose()
	}
	g.prepCursorHD = c
	return c
}

// Native art is already in the indexed underlay. Replaying just this slot above
// the new cursor restores exactly the original cursor-before-unit occlusion.
func (g *Game) prepareModernHDPreparationNativeCursorUnit(sprite fdicon.Sprite, selected bool) *modernPreparationNativeCursorUnit {
	palette, ok := preparationCursorPalette(g.nativeClassUI.palette)
	if !ok || len(sprite.Pixels) != 24*24 || len(sprite.Mask) != 24*24 {
		return nil
	}
	picture := image.NewRGBA(image.Rect(0, 0, 144, 144))
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			i := y*24 + x
			if sprite.Mask[i] == 0 {
				continue
			}
			index := sprite.Pixels[i]
			if !selected {
				index = (index & 7) + 0x18
			}
			draw.Draw(picture, image.Rect(x*6, y*6, (x+1)*6, (y+1)*6), image.NewUniform(palette[index]), image.Point{}, draw.Src)
		}
	}
	key := sha256.Sum256(picture.Pix)
	if old := g.prepNativeCursorUnitHD; old != nil && old.key == key {
		return old
	}
	unit := &modernPreparationNativeCursorUnit{key: key, picture: picture}
	if old := g.prepNativeCursorUnitHD; old != nil && old.gpu != nil {
		old.gpu.Dispose()
	}
	g.prepNativeCursorUnitHD = unit
	return unit
}

func drawPreparationCursorTexture(screen ebiten.FinalScreen, texture *ebiten.Image, rect image.Rectangle, geometry ebiten.GeoM, brightness float64) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(1.0/3, 1.0/3)
	op.GeoM.Translate(float64(rect.Min.X)/3, float64(rect.Min.Y)/3)
	op.GeoM.Concat(geometry)
	op.ColorScale.Scale(float32(brightness), float32(brightness), float32(brightness), 1)
	screen.DrawImage(texture, op)
}
