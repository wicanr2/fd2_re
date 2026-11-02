package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"golang.org/x/image/font/sfnt"
)

var facilityItemRowCategories = []modernLMIHDDescriptor{
	{Index: 59, Width: 24, Height: 20, PixelsSHA256: "924ccec7fb6b235945c8fe6849d3e23a0371c69ac7fa05d6d3258eef217e859f"},
	{Index: 60, Width: 24, Height: 20, PixelsSHA256: "e30156258340ca382a651007bbc6a3eb9526ba4a06ee7bbea9666e2e36cdbb62"},
	{Index: 61, Width: 24, Height: 20, PixelsSHA256: "406893316358233bc1ac97442c4d7f265f8e785b78c8e0996b88cc26e292a9d1"},
}

var facilityItemRowLabels = []struct {
	index int
	text  string
	width int
	sha   string
}{
	{64, "+AP", 21, "486252ca6298087c31b5f3be81e3f6caedd7e53acbe7bfbf53b9627ae7a91024"},
	{65, "+DP", 21, "5817807474e22435305cfd6ea01735afdd803ab3085aea70d0bde9892fd02010"},
	{66, "+HP", 21, "811fd7f03545d1bd961a8ea633afc0a5d8a2e7b156ff775b67e44f7afc34dfc1"},
	{67, "+MP", 21, "e16ad4cb92472f537551505afce6632fca04ab029cb5d2bf54f86c9a8cee49ef"},
	{15, "$", 6, "99dc9cc4c66cc7eb8d41176489540835171ce49d29808888646235abebc9734b"},
}

type modernHDFacilityItemRows struct {
	key   [32]byte
	font  *Font
	icons map[int]*modernHDServiceIcon
}

type facilityItemRowSource struct {
	assets          battle.NativeItemPanelDataAssets
	price           fdother.RawCell
	rows            []byte
	ids             []int // Visible raw item IDs, after the owner's compact-slot mapping.
	start, selected int
}

type facilityItemRowPlacement struct {
	index  int
	origin image.Point
}

func (s facilityItemRowSource) cell(index int) fdother.RawCell {
	if index == 15 {
		return s.price
	}
	return s.assets.RawCells[index]
}

func (s facilityItemRowSource) validSources() bool {
	for _, source := range facilityItemRowCategories {
		if modernUIRawDescription(source.Index, s.cell(source.Index)) != source {
			return false
		}
	}
	for _, source := range facilityItemRowLabels {
		cell := s.cell(source.index)
		sum := sha256.Sum256(cell.Pixels)
		if cell.Width != source.width || cell.Height != 8 || len(cell.Pixels) != source.width*8 || hex.EncodeToString(sum[:]) != source.sha {
			return false
		}
	}
	return true
}

func (s facilityItemRowSource) frozen() *facilityItemRowSource {
	copySource := s
	copySource.ids = append([]int(nil), s.ids...)
	copySource.rows = append([]byte(nil), s.rows...)
	copySource.price.Pixels = append([]byte(nil), s.price.Pixels...)
	cells := make(map[int]fdother.RawCell)
	for _, index := range []int{59, 60, 61, 64, 65, 66, 67} {
		cell := s.cell(index)
		cell.Pixels = append([]byte(nil), cell.Pixels...)
		cells[index] = cell
	}
	copySource.assets = battle.NativeItemPanelDataAssets{RawCells: cells}
	return &copySource
}

func (g *Game) modernFacilityItemRowSource() (s facilityItemRowSource, ok bool) {
	if g.camp == nil || g.camp.Node() == nil || g.nativeClassUI == nil ||
		g.nativeClassUIJob != nil || g.nativeChurchUIJob != nil || g.nativeShopUIJob != nil ||
		g.helpVisible || g.transientUI || g.nativePlayerStatus != nil ||
		g.nativeFullDACBlack || g.nativeFullDACWhite {
		return
	}
	var ids, slots []int
	var selected, start, sourceID int
	switch g.camp.Node().Type {
	case "shop":
		a, _, _, valid := g.nativeShopState()
		if !valid || a == nil || g.nativeShopUI == nil ||
			g.camp.Node().NativeHubVariant != g.nativeShopVariant ||
			a.ResourceID != map[int]int{1: 12, 3: 29, 5: 63}[g.nativeShopVariant] {
			return s, false
		}
		s.assets, s.price, s.rows = g.nativeShopUI.itemAssets, a.PriceCell, g.nativeShopUI.effectRows
		switch g.nativeShopMode {
		case "purchase":
			for _, good := range g.camp.ShopGoods() {
				ids = append(ids, good.ID)
			}
			selected, start = g.shopSel, g.nativeShopItemStart
		case "sell_items":
			ids, selected, start = g.nativeShopSellItemIDs, g.shopSellSlotSel, g.nativeShopSellItemTop
		case "transfer_items":
			slots, sourceID = g.nativeShopTransferItems, g.nativeShopTransferSource
			selected, start = g.nativeShopTransferSel, g.nativeShopTransferTop
		default:
			return s, false
		}
	case "church":
		if g.churchMode != "transfer_item" || g.churchTransferSource < 0 {
			return s, false
		}
		var err error
		s.assets, err = battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
		if err != nil {
			return s, false
		}
		s.rows, err = battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
		if err != nil {
			return s, false
		}
		s.price = g.nativeClassUI.priceCell
		slots, sourceID = g.churchTransferItems, g.churchTransferSource
		selected, start = g.churchSel, g.churchItemStart
	default:
		return s, false
	}
	if slots != nil {
		unit, exists := g.partyRoster[sourceID]
		if !exists {
			return s, false
		}
		for _, slot := range slots {
			if slot < 0 || slot >= len(unit.Inventory) {
				return s, false
			}
			ids = append(ids, unit.Inventory[slot])
		}
	}
	origin, visible := campaign.NativeTwoColumnWindow(len(ids), selected, start)
	if visible == 0 || origin != start {
		return s, false
	}
	s.ids = append([]int(nil), ids[start:start+visible]...)
	s.start, s.selected = start, selected
	return s, true
}

func (g *Game) prepareModernHDFacilityItemRows(s facilityItemRowSource, set *modernStoryPortraitSet) *modernHDFacilityItemRows {
	if set == nil || len(set.uiPanelIcons) != 10 || g.font == nil || g.font.sf == nil || g.nativeClassUI == nil || !s.validSources() {
		return nil
	}
	palette, ok := preparationCursorPalette(g.nativeClassUI.palette)
	if !ok {
		return nil
	}
	cells := make(map[int]fdother.RawCell)
	var artDigests [][32]byte
	for _, index := range []int{59, 60, 61} {
		cell, icon := s.assets.RawCells[index], set.uiPanelIcons[index]
		if cell.Width != 24 || cell.Height != 20 || len(cell.Pixels) != 480 || icon == nil ||
			icon.native != modernUIRawDescription(index, cell) || icon.picture == nil ||
			icon.picture.Bounds() != image.Rect(0, 0, 144, 120) || sha256.Sum256(icon.picture.Pix) != icon.digest {
			return nil
		}
		cells[index] = cell
		artDigests = append(artDigests, icon.digest)
	}
	for _, label := range facilityItemRowLabels {
		cell := s.assets.RawCells[label.index]
		if label.index == 15 {
			cell = s.price
		}
		sum := sha256.Sum256(cell.Pixels)
		if cell.Width != label.width || cell.Height != 8 || len(cell.Pixels) != label.width*8 ||
			hex.EncodeToString(sum[:]) != label.sha {
			return nil
		}
		cells[label.index] = cell
	}
	raw, err := json.Marshal([]any{cells, artDigests, palette, g.font.effectiveScale(48)})
	if err != nil {
		return nil
	}
	key := sha256.Sum256(raw)
	if h := g.facilityItemRowsHD; h != nil && h.key == key && h.font == g.font {
		return h
	}
	h := &modernHDFacilityItemRows{key: key, font: g.font, icons: make(map[int]*modernHDServiceIcon)}
	for _, index := range []int{59, 60, 61} {
		icon := set.uiPanelIcons[index]
		h.icons[index] = &modernHDServiceIcon{picture: icon.picture, digest: icon.digest}
	}
	for _, label := range facilityItemRowLabels {
		var buffer sfnt.Buffer
		for _, ch := range label.text {
			glyph, err := g.font.sf.GlyphIndex(&buffer, ch)
			if err != nil || glyph == 0 {
				return nil
			}
		}
		cell := cells[label.index]
		background := cell.Pixels[0]
		ink, best := palette[background], -1
		for _, index := range cell.Pixels {
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
		picture := image.NewRGBA(image.Rect(0, 0, label.width*6, 48))
		draw.Draw(picture, picture.Bounds(), image.NewUniform(palette[background]), image.Point{}, draw.Src)
		if err := drawModernBattlePanelInk(picture, g.font, label.text, picture.Bounds(), 48, ink); err != nil {
			return nil
		}
		h.icons[label.index] = &modernHDServiceIcon{picture: picture, digest: sha256.Sum256(picture.Pix)}
	}
	g.facilityItemRowsHD = h
	return h
}

// This native proof also binds jobs created while original art/UI are active.
// Every source and complete opaque destination must match before publication.
func (s facilityItemRowSource) placements(frame []byte) ([]facilityItemRowPlacement, bool) {
	if len(frame) != 64000 || !s.validSources() || len(s.ids) == 0 || len(s.ids) > 6 {
		return nil, false
	}
	var placements []facilityItemRowPlacement
	for i, id := range s.ids {
		if id < 0 || id >= len(s.rows)/battle.NativeItemEffectRowSize {
			return nil, false
		}
		row := s.rows[id*battle.NativeItemEffectRowSize : (id+1)*battle.NativeItemEffectRowSize]
		category, stat := 61, 0
		switch {
		case row[0] < 0x15:
			category, stat = 59, 64
		case row[0] < 0x20:
			category, stat = 60, 65
		case row[0] == 0x20 && row[13] == 5:
			stat = 66
		case row[0] == 0x20 && row[13] == 11:
			stat = 67
		}
		x, y := 10+148*(i%2), 119+26*(i/2)
		for _, p := range []facilityItemRowPlacement{{category, image.Pt(x, y)}, {stat, image.Pt(x+95, y+2)}, {15, image.Pt(x+95, y+12)}} {
			if p.index == 0 {
				continue
			}
			cell := s.cell(p.index)
			for line := 0; line < cell.Height; line++ {
				at := (p.origin.Y+line)*320 + p.origin.X
				if !bytes.Equal(frame[at:at+cell.Width], cell.Pixels[line*cell.Width:(line+1)*cell.Width]) {
					return nil, false
				}
			}
			placements = append(placements, p)
		}
	}
	return placements, true
}

func (g *Game) modernHDFacilityItemRowPlacements(frame []byte, s facilityItemRowSource, set *modernStoryPortraitSet) ([]modernHDServiceIconPlacement, bool) {
	plan, ok := s.placements(frame)
	if !ok {
		return nil, false
	}
	h := g.prepareModernHDFacilityItemRows(s, set)
	if h == nil {
		return nil, false
	}
	var placements []modernHDServiceIconPlacement
	for _, p := range plan {
		placements = append(placements, modernHDServiceIconPlacement{icon: h.icons[p.index], origin: p.origin})
	}
	return placements, true
}

// The original ??? marker, names, numbers and transaction state remain owned
// by RenderNativeFacilityItemRows and its existing text/decimal adapters.
func (g *Game) appendModernHDFacilityItemRows(frame []byte) {
	if g.modernHDText == nil {
		return
	}
	s, ok := g.modernFacilityItemRowSource()
	if !ok {
		return
	}
	placements, ok := g.modernHDFacilityItemRowPlacements(frame, s, g.interfaceAssetTheme())
	if !ok {
		return
	}
	g.modernHDText.serviceIcons = append(g.modernHDText.serviceIcons, placements...)
}
