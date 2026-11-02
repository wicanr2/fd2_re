package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"golang.org/x/image/font/sfnt"
)

var shopComparisonLabels = [5]struct {
	text   string
	width  int
	height int
	sha    string
}{
	{"AP", 13, 8, "28d736020fda195eb673add76fa5ae9b06168abea68eef57b7b1026197eb8487"},
	{"DP", 13, 8, "90e27bd889c745c5700bd59bd3d02ef319b00630890f7b2062e3dd51817e34e3"},
	{"HIT", 16, 8, "d3358ec250e6275b29cc60b56cc779a4801ff4a66ac27cca4af7956773887c5b"},
	{"EV", 13, 8, "1aa48ba38281a24199d2ba0d3d6dd930fe2377459180d0e2dc954947d3bcb3d9"},
	{"→", 7, 6, "23932878b128c12253a300e887bbebc6b47d27d907092361a22f92fcc061d392"},
}

type modernHDShopComparison struct {
	key   [32]byte
	font  *Font
	icons [5]*modernHDServiceIcon
}

type shopComparisonPlacement struct {
	cell   int
	origin image.Point
}

func shopComparisonPlacements(rows int) []shopComparisonPlacement {
	var result []shopComparisonPlacement
	for row := 0; row < rows; row++ {
		y := 117 + 26*row
		for _, p := range []shopComparisonPlacement{
			{0, image.Pt(122, y+3)}, {1, image.Pt(122, y+12)},
			{2, image.Pt(196, y+3)}, {3, image.Pt(196, y+12)},
			{4, image.Pt(157, y+4)}, {4, image.Pt(157, y+13)},
			{4, image.Pt(234, y+4)}, {4, image.Pt(234, y+13)},
		} {
			result = append(result, p)
		}
	}
	return result
}

func (g *Game) prepareModernHDShopComparison(cells [5]fdother.LMI1Entry) *modernHDShopComparison {
	if g.font == nil || g.font.sf == nil || g.nativeClassUI == nil {
		return nil
	}
	palette, ok := preparationCursorPalette(g.nativeClassUI.palette)
	if !ok {
		return nil
	}
	for i, label := range shopComparisonLabels {
		cell := cells[i]
		sum := sha256.Sum256(cell.Pixels)
		if cell.Width != label.width || cell.Height != label.height || len(cell.Pixels) != label.width*label.height ||
			hex.EncodeToString(sum[:]) != label.sha {
			return nil
		}
	}
	raw, err := json.Marshal([]any{cells, palette, g.font.effectiveScale(48)})
	if err != nil {
		return nil
	}
	key := sha256.Sum256(raw)
	if h := g.shopComparisonHD; h != nil && h.key == key && h.font == g.font {
		return h
	}
	h := &modernHDShopComparison{key: key, font: g.font}
	for i, label := range shopComparisonLabels {
		var buffer sfnt.Buffer
		for _, ch := range label.text {
			index, err := g.font.sf.GlyphIndex(&buffer, ch)
			if err != nil || index == 0 {
				return nil
			}
		}
		background := cells[i].Pixels[0]
		ink, best := palette[background], -1
		for _, index := range cells[i].Pixels {
			if index == background {
				continue
			}
			c := palette[index]
			if light := int(c.R) + int(c.G) + int(c.B); light > best {
				ink, best = c, light
			}
		}
		if best < 0 {
			return nil
		}
		picture := image.NewRGBA(image.Rect(0, 0, label.width*6, label.height*6))
		draw.Draw(picture, picture.Bounds(), image.NewUniform(palette[background]), image.Point{}, draw.Src)
		if err := drawModernBattlePanelInk(picture, g.font, label.text, picture.Bounds(), label.height*6, ink); err != nil {
			return nil
		}
		h.icons[i] = &modernHDServiceIcon{picture: picture, digest: sha256.Sum256(picture.Pix)}
	}
	g.shopComparisonHD = h
	return h
}

// Compare all complete opaque source rectangles before publishing any labels.
// The existing small-image painter keeps these vector glyphs independent of F2.
func (g *Game) appendModernHDShopComparison(frame []byte) {
	if g.modernHDText == nil || len(frame) != 64000 {
		return
	}
	source, ok := g.modernFacilityRosterSource()
	if !ok || source.owner != "shop" || source.mode != "recipient_equipment" {
		return
	}
	assets, _, _, ok := g.nativeShopState()
	if !ok || assets == nil {
		return
	}
	h := g.prepareModernHDShopComparison(assets.CompareCells)
	if h == nil {
		return
	}
	positions := shopComparisonPlacements(len(source.ids))
	for _, p := range positions {
		cell := assets.CompareCells[p.cell]
		for y := 0; y < cell.Height; y++ {
			at := (p.origin.Y+y)*320 + p.origin.X
			if !bytes.Equal(frame[at:at+cell.Width], cell.Pixels[y*cell.Width:(y+1)*cell.Width]) {
				return
			}
		}
	}
	for _, p := range positions {
		g.modernHDText.serviceIcons = append(g.modernHDText.serviceIcons,
			modernHDServiceIconPlacement{icon: h.icons[p.cell], origin: p.origin})
	}
}
