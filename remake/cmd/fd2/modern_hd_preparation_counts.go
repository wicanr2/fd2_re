package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"golang.org/x/image/font/sfnt"
)

const preparationCountFrameSHA = "7768d4b208a32a48a0fc92998dd7f8f67e356e45156154d45828c3fd692b50a6"
const preparationCountMaskSHA = "ca901af74c6f663d21624279a281d0361cbab4cac12203b8b4cc624e08a27df8"

type modernHDPreparationCounts struct {
	key    [32]byte
	font   *Font
	chrome [2]modernHDDialogueChrome
	labels *modernHDItemPanelValues
	head   *modernUIEmbeddedIcon
}

func preparationCountSourceMatches(frame fdother.Frame) bool {
	if frame.Width != 86 || frame.Height != 86 || len(frame.Indexed) != 86*86 || len(frame.Mask) != 86*86 {
		return false
	}
	pixels, mask := sha256.Sum256(frame.Indexed), sha256.Sum256(frame.Mask)
	return hex.EncodeToString(pixels[:]) == preparationCountFrameSHA && hex.EncodeToString(mask[:]) == preparationCountMaskSHA
}

// These captions are embedded in the pinned original Frame137. Decimal values
// still come from the original two digit banks through their Draw-only adapter.
func (g *Game) prepareModernHDPreparationCounts(ui *nativePreparationUIAssets) *modernHDPreparationCounts {
	if g.modernHDText == nil || g.font == nil || g.font.sf == nil || g.nativeClassUI == nil || ui == nil || ui.roster == nil ||
		!g.prepSelecting || g.prepConfirm || g.prepRecordSlots || g.nativeClassUIJob != nil ||
		g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != "preparation" ||
		g.prepSel < 0 || g.prepSel >= len(g.prepIDs) ||
		!preparationCountSourceMatches(ui.roster.UpperLeft) {
		return nil
	}
	frame, palette := ui.roster.UpperLeft, g.nativeClassUI.palette
	head, err := modernPreparationCountIcon(ui, g.interfaceAssetTheme())
	if err != nil {
		return nil
	}
	var headDigest [32]byte
	if head != nil {
		headDigest = head.digest
	}
	raw, err := json.Marshal([]any{frame.Indexed, frame.Mask, ui.dialogue, palette, headDigest})
	if err != nil {
		return nil
	}
	key := sha256.Sum256(raw)
	if h := g.prepCountsHD; h != nil && h.key == key && h.font == g.font {
		return h
	}
	chrome, err := freezeModernHDDialogueChrome(ui.dialogue, palette)
	if err != nil {
		return nil
	}
	pal := func(index byte) (color.RGBA, bool) {
		if len(palette) != 256 || palette[index] == nil {
			return color.RGBA{}, false
		}
		c := color.RGBAModel.Convert(palette[index]).(color.RGBA)
		return c, c.A == 255
	}
	background := frame.Indexed[4*86+4]
	bg, ok := pal(background)
	if !ok {
		return nil
	}
	picture := image.NewRGBA(image.Rect(0, 0, 86*6, 86*6))
	for _, label := range []struct {
		text string
		rect image.Rectangle
		size int
	}{
		{"出戰人數", image.Rect(10, 4, 74, 20), 84},
		{"剩餘人數", image.Rect(10, 42, 74, 58), 84},
		{"×", image.Rect(45, 28, 53, 36), 48},
		{"×", image.Rect(45, 66, 53, 74), 48},
	} {
		var buffer sfnt.Buffer
		for _, ch := range label.text {
			index, err := g.font.sf.GlyphIndex(&buffer, ch)
			if err != nil || index == 0 {
				return nil
			}
		}
		best := -1
		var ink color.RGBA
		for y := label.rect.Min.Y; y < label.rect.Max.Y; y++ {
			for x := label.rect.Min.X; x < label.rect.Max.X; x++ {
				at := y*86 + x
				index := frame.Indexed[at]
				if index == background || frame.Mask[at] != 255 {
					continue
				}
				c, ok := pal(index)
				if !ok {
					return nil
				}
				if n := int(c.R) + int(c.G) + int(c.B); n > best {
					best, ink = n, c
				}
			}
		}
		if best < 0 {
			return nil
		}
		rect := image.Rectangle{Min: label.rect.Min.Mul(6), Max: label.rect.Max.Mul(6)}
		draw.Draw(picture, rect, image.NewUniform(bg), image.Point{}, draw.Src)
		if err := drawModernBattlePanelInk(picture, g.font, label.text, rect, label.size, ink); err != nil {
			return nil
		}
	}
	if head != nil {
		for _, source := range []image.Rectangle{head.source, head.source.Add(image.Pt(0, 38))} {
			rect := image.Rectangle{Min: source.Min.Mul(6), Max: source.Max.Mul(6)}
			draw.Draw(picture, rect, image.NewUniform(bg), image.Point{}, draw.Src)
			draw.Draw(picture, rect, head.picture, image.Point{}, draw.Over)
		}
	}
	h := &modernHDPreparationCounts{key: key, font: g.font, chrome: [2]modernHDDialogueChrome{*chrome, *chrome}, labels: &modernHDItemPanelValues{seal: key, picture: picture}, head: head}
	g.prepCountsHD = h
	return h
}

func (g *Game) appendModernHDPreparationCounts(ui *nativePreparationUIAssets) {
	if h := g.prepareModernHDPreparationCounts(ui); h != nil {
		g.modernHDText.panels = append(g.modernHDText.panels,
			modernHDDialoguePanel{chrome: &h.chrome[0], values: h.labels, rect: image.Rect(5, 7, 91, 93), status: true},
			modernHDDialoguePanel{chrome: &h.chrome[1], rect: image.Rect(5, 94, 315, 193), status: true})
	}
}
