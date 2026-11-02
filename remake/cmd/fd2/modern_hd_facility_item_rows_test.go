package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Find actual source items for all five original branches. No effect row or
// raw image is changed to obtain the test coverage.
func facilityItemTestIDs(t *testing.T, g *Game) []int {
	t.Helper()
	rows := g.nativeShopUI.effectRows
	ids := []int{-1, -1, -1, -1, -1}
	for id := 0; id < len(rows)/battle.NativeItemEffectRowSize; id++ {
		r := rows[id*battle.NativeItemEffectRowSize : (id+1)*battle.NativeItemEffectRowSize]
		kind := 4
		switch {
		case r[0] < 0x15:
			kind = 0
		case r[0] < 0x20:
			kind = 1
		case r[0] == 0x20 && r[13] == 5:
			kind = 2
		case r[0] == 0x20 && r[13] == 11:
			kind = 3
		}
		if ids[kind] < 0 {
			ids[kind] = id
		}
	}
	for _, id := range ids {
		if id < 0 {
			t.Fatal("actual source lacks required item branch")
		}
	}
	return append(ids, ids[0], ids[2], ids[3])
}

func facilityItemTestState(g *Game, mode string, variant int, ids []int, scroll bool) {
	if mode == "transfer_item" {
		facilityDialogueTestOwner(g, "church", mode, 0, 0)
		g.churchTransferSource = 0
	} else {
		shopDialogueTestState(g, variant, mode, 0, 0)
		g.camp.Node().Goods = nil
		for _, id := range ids {
			price, _ := battle.NativeFacilityItemListPrice(g.nativeShopUI.effectRows, id, battle.NativeFacilityFullPrice)
			g.camp.Node().Goods = append(g.camp.Node().Goods, campaign.Good{ID: id, Price: price})
		}
		g.nativeShopSellItemIDs = append([]int(nil), ids...)
	}
	// Deliberately permute compact slots. Raw inventory IDs and list ordinals
	// differ in both transfer owners.
	slots := []int{4, 2, 0, 7, 5, 3, 1, 6}
	inventory := make([]int, 8)
	for i, slot := range slots {
		inventory[slot] = ids[i]
	}
	g.partyRoster[0] = setNativeTransferInventory(g.partyRoster[0], inventory...)
	g.nativeShopTransferItems = append([]int(nil), slots...)
	g.churchTransferItems = append([]int(nil), slots...)
	if scroll {
		g.shopSel, g.nativeShopItemStart = 7, 2
		g.shopSellSlotSel, g.nativeShopSellItemTop = 7, 2
		g.nativeShopTransferSel, g.nativeShopTransferTop = 7, 2
		g.churchSel, g.churchItemStart = 7, 2
	}
}

func facilityItemTestCompose(g *Game) ([]byte, bool) {
	if g.camp.Node().Type == "church" {
		return g.composeNativeChurchTransferItemFrame()
	}
	return shopDialogueCompose(g)
}

// Independent fixture roles, after the original typed list/viewport mapping.
func facilityItemTestPlacements(ids []int, scroll bool) []struct {
	index  int
	origin image.Point
} {
	roles := map[int][2]int{ids[0]: {59, 64}, ids[1]: {60, 65}, ids[2]: {61, 66}, ids[3]: {61, 67}, ids[4]: {61, 0}}
	start := 0
	if scroll {
		start = 2
	}
	var result []struct {
		index  int
		origin image.Point
	}
	for row, id := range ids[start : start+6] {
		x, y := 10+148*(row%2), 119+26*(row/2)
		for _, p := range []struct {
			index  int
			origin image.Point
		}{{roles[id][0], image.Pt(x, y)}, {roles[id][1], image.Pt(x+95, y+2)}, {15, image.Pt(x+95, y+12)}} {
			if p.index != 0 {
				result = append(result, p)
			}
		}
	}
	return result
}

func TestModernHDFacilityItemRowsSourceAndFallback(t *testing.T) {
	set := facilityHDTestSet(t)
	g := facilityDialogueTestGame(t, set)
	ids := facilityItemTestIDs(t, g)
	for _, mode := range []string{"purchase", "sell_items", "transfer_items", "transfer_item"} {
		variants := []int{1, 3, 5}
		if mode == "transfer_item" {
			variants = []int{0}
		}
		for _, variant := range variants {
			for _, scroll := range []bool{false, true} {
				facilityItemTestState(g, mode, variant, ids, scroll)
				frame, ok := facilityItemTestCompose(g)
				if !ok {
					t.Fatal("actual compositor unavailable", mode, variant)
				}
				before := append([]byte(nil), frame...)
				party, _ := json.Marshal(g.partyRoster)
				source, ok := g.modernFacilityItemRowSource()
				start := 0
				if scroll {
					start = 2
				}
				if !ok || !bytes.Equal(intIDs(source.ids), intIDs(ids[start:start+6])) {
					t.Fatal("compact slot or viewport mapping changed", mode)
				}
				positions := facilityItemTestPlacements(ids, scroll)
				g.interfaceMode = playerInterfaceHD
				g.modernHDText = &modernHDTextFrame{}
				g.appendModernHDFacilityItemRows(frame)
				if len(g.modernHDText.serviceIcons) != len(positions) {
					t.Fatal("actual complete source refused", mode, variant)
				}
				h := g.facilityItemRowsHD
				actual := g.modernHDText.serviceIcons
				if g.prepareModernHDFacilityItemRows(source, set) != h {
					t.Fatal("identical source missed cache")
				}
				for i, pos := range positions {
					p := actual[i]
					if p.origin != pos.origin || p.icon != h.icons[pos.index] {
						t.Fatal("native origin or role changed", pos)
					}
					covered := append([]byte(nil), frame...)
					covered[pos.origin.Y*320+pos.origin.X] ^= 1
					g.modernHDText = &modernHDTextFrame{}
					g.appendModernHDFacilityItemRows(covered)
					if len(g.modernHDText.serviceIcons) != 0 {
						t.Fatal("partial source published rows", pos)
					}
				}
				for _, flag := range []string{"job", "help", "transient", "black", "white", "owner"} {
					g.modernHDText = &modernHDTextFrame{}
					g.helpVisible, g.transientUI = flag == "help", flag == "transient"
					g.nativeFullDACBlack, g.nativeFullDACWhite = flag == "black", flag == "white"
					if flag == "job" {
						g.nativeShopUIJob = &nativeClassUIJob{}
					}
					owner := g.camp.Node().Type
					if flag == "owner" {
						g.camp.Node().Type = "hotel"
					}
					g.appendModernHDFacilityItemRows(frame)
					if len(g.modernHDText.serviceIcons) != 0 {
						t.Fatal("covered owner retained rows", flag)
					}
					g.camp.Node().Type, g.nativeShopUIJob = owner, nil
				}
				g.helpVisible, g.transientUI, g.nativeFullDACBlack, g.nativeFullDACWhite = false, false, false, false
				after, _ := json.Marshal(g.partyRoster)
				if !bytes.Equal(before, frame) || !bytes.Equal(party, after) {
					t.Fatal("source frame or party changed")
				}
				g.modernHDText = nil
			}
		}
	}
	// Full source hashes and registered art are checked even after a cache hit.
	source, _ := g.modernFacilityItemRowSource()
	for _, index := range []int{59, 60, 61, 64, 65, 66, 67, 15} {
		bad := source
		bad.assets.RawCells = make(map[int]fdother.RawCell)
		for key, cell := range source.assets.RawCells {
			bad.assets.RawCells[key] = cell
		}
		cell := source.assets.RawCells[index]
		if index == 15 {
			cell = source.price
		}
		cell.Pixels = append([]byte(nil), cell.Pixels...)
		cell.Pixels[0] ^= 1
		if index == 15 {
			bad.price = cell
		} else {
			bad.assets.RawCells[index] = cell
		}
		if g.prepareModernHDFacilityItemRows(bad, set) != nil {
			t.Fatal("changed source reused material", index)
		}
	}
	for _, index := range []int{59, 60, 61} {
		icon := set.uiPanelIcons[index]
		delete(set.uiPanelIcons, index)
		if g.prepareModernHDFacilityItemRows(source, set) != nil {
			t.Fatal("missing registered art admitted", index)
		}
		set.uiPanelIcons[index] = icon
		bad := *icon
		bad.digest[0] ^= 1
		set.uiPanelIcons[index] = &bad
		if g.prepareModernHDFacilityItemRows(source, set) != nil {
			t.Fatal("changed registered art admitted", index)
		}
		set.uiPanelIcons[index] = icon
	}
	g.churchTransferItems = []int{8}
	if _, ok := g.modernFacilityItemRowSource(); ok {
		t.Fatal("invalid compact slot admitted")
	}
	palette := g.nativeClassUI.palette
	g.nativeClassUI.palette = append(color.Palette(nil), palette...)
	g.nativeClassUI.palette[73] = color.RGBA{1, 2, 3, 255}
	old := g.facilityItemRowsHD
	if h := g.prepareModernHDFacilityItemRows(source, set); h == nil || h == old {
		t.Fatal("palette change reused cached material")
	}
	g.nativeClassUI.palette[73] = nil
	if g.prepareModernHDFacilityItemRows(source, set) != nil {
		t.Fatal("incomplete palette admitted")
	}
	g.nativeClassUI.palette = palette
	old = g.prepareModernHDFacilityItemRows(source, set)
	g.font.userScale = 1.5
	if h := g.prepareModernHDFacilityItemRows(source, set); h == nil || h == old {
		t.Fatal("font scale reused material")
	}
	g.font = &Font{}
	if g.prepareModernHDFacilityItemRows(source, set) != nil {
		t.Fatal("missing font admitted")
	}
}

func intIDs(ids []int) []byte {
	raw, _ := json.Marshal(ids)
	return raw
}

func TestModernHDFacilityItemRowsFontMaterial(t *testing.T) {
	set := facilityHDTestSet(t)
	g := facilityDialogueTestGame(t, set)
	facilityItemTestState(g, "purchase", 1, facilityItemTestIDs(t, g), false)
	source, ok := g.modernFacilityItemRowSource()
	if !ok {
		t.Fatal("source unavailable")
	}
	h := g.prepareModernHDFacilityItemRows(source, set)
	if h == nil {
		t.Fatal("actual material unavailable")
	}
	palette, _ := preparationCursorPalette(g.nativeClassUI.palette)
	for _, label := range facilityItemRowLabels {
		want := image.NewRGBA(image.Rect(0, 0, label.width*6, 48))
		draw.Draw(want, want.Bounds(), image.NewUniform(palette[73]), image.Point{}, draw.Src)
		face, _ := g.font.faceFor(48)
		bounds, _ := font.BoundString(face, label.text)
		box := image.Rect(bounds.Min.X.Floor(), bounds.Min.Y.Floor(), bounds.Max.X.Ceil(), bounds.Max.Y.Ceil())
		if box.Dx() > want.Bounds().Dx() || box.Dy() > want.Bounds().Dy() {
			t.Fatal("base font no longer fits source rectangle", label.text)
		}
		d := font.Drawer{Dst: want, Src: image.NewUniform(palette[169]), Face: face, Dot: fixed.P(-box.Min.X, -box.Min.Y)}
		d.DrawString(label.text)
		if !bytes.Equal(want.Pix, h.icons[label.index].picture.Pix) {
			t.Fatal("material differs from independent font drawing", label.text)
		}
	}
}

func TestModernHDFacilityItemRowsRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_FACILITY_ITEM_ROWS_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	set := facilityHDTestSet(t)
	g := facilityDialogueTestGame(t, set)
	ids := facilityItemTestIDs(t, g)
	out := os.Getenv("FD2_TEST_FACILITY_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true}, captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	currentScroll := false
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		positions := facilityItemTestPlacements(ids, currentScroll)
		var placements []modernHDServiceIconPlacement
		if g.modernHDText != nil {
			placements = g.modernHDText.serviceIcons
		}
		hd := g.interfaceMode == playerInterfaceHD
		if len(placements) != 0 && !hd || hd && len(placements) != len(positions) {
			return fmt.Errorf("case%d wrong icon count %d", p.index, len(placements))
		}
		native, ok := facilityItemTestCompose(g)
		if !ok {
			return fmt.Errorf("case%d native source unavailable", p.index)
		}
		checked := 0
		for _, at := range positions {
			width, height := 24, 20
			if at.index >= 64 {
				width, height = 21, 8
			} else if at.index == 15 {
				width, height = 6, 8
			}
			for y := 0; y < height*6; y++ {
				for x := 0; x < width*6; x++ {
					value := native[(at.origin.Y+y/6)*320+at.origin.X+x/6]
					want := color.RGBAModel.Convert(g.nativeClassUI.palette[value]).(color.RGBA)
					if hd {
						want = g.facilityItemRowsHD.icons[at.index].picture.RGBAAt(x, y)
					}
					i := ((at.origin.Y*6+y+72)*1920 + at.origin.X*6 + x) * 4
					if !bytes.Equal(pixels[i:i+4], []byte{want.R, want.G, want.B, 255}) {
						return fmt.Errorf("case%d icon%d pixel%d,%d", p.index, at.index, x, y)
					}
					checked++
				}
			}
		}
		if g.modernHDText != nil {
			g.modernHDText.serviceIcons = nil
		}
		reference := ebiten.NewImage(1920, 1272)
		defer reference.Dispose()
		var geometry ebiten.GeoM
		geometry.Scale(3, 3)
		p.presentation.DrawFinalScreen(reference, offscreen, geometry)
		control := make([]byte, len(pixels))
		reference.ReadPixels(control)
		if g.modernHDText != nil {
			g.modernHDText.serviceIcons = placements
		}
		outside := append([]byte(nil), pixels...)
		for _, placement := range placements {
			size := placement.icon.picture.Bounds().Size()
			zero := make([]byte, size.X*4)
			for y := 0; y < size.Y; y++ {
				i := ((placement.origin.Y*6+y+72)*1920 + placement.origin.X*6) * 4
				copy(outside[i:i+len(zero)], zero)
				copy(control[i:i+len(zero)], zero)
			}
		}
		if !bytes.Equal(outside, control) {
			return fmt.Errorf("case%d changed bytes outside item roles", p.index)
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "owner": g.camp.Node().Type, "variant": g.nativeShopVariant, "shop_mode": g.nativeShopMode, "church_mode": g.churchMode, "locale": g.localeID, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil, "scroll": currentScroll, "item_role_pixels": checked, "all_outside_bytes_equal": true})
		return nil
	}
	add := func(mode string, variant, style int, locale string, scroll bool) {
		if style == 3 && locale == "zh-Hant" {
			p.captureCases[len(p.cases)] = true
		}
		p.cases = append(p.cases, func() {
			facilityItemTestState(g, mode, variant, ids, scroll)
			currentScroll = scroll
			var err error
			g.localeCatalog, err = loadOfficialLocale(locale)
			if err != nil {
				t.Fatal(err)
			}
			g.localeContent, err = loadOfficialLocaleContent(locale)
			if err != nil {
				t.Fatal(err)
			}
			g.localeEntities, err = loadOfficialLocaleEntities(locale)
			if err != nil {
				t.Fatal(err)
			}
			g.localeID = locale
			g.interfaceMode, g.modernStoryPortraits = playerInterfaceOriginal, nil
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
			if style >= 2 {
				g.modernStoryPortraits = set
			}
		})
	}
	for _, mode := range []string{"purchase", "sell_items", "transfer_items", "transfer_item"} {
		variants := []int{1, 3, 5}
		if mode == "transfer_item" {
			variants = []int{0}
		}
		for _, variant := range variants {
			for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
				for style := 0; style < 4; style++ {
					add(mode, variant, style, locale, false)
				}
			}
			for style := 0; style < 4; style++ {
				add(mode, variant, style, "zh-Hant", true)
			}
		}
	}
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.checks) != 200 {
		t.Fatal("incomplete item-row matrix", len(p.checks))
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-FACILITY-ITEM-ROWS-GPU", "scope": "three shops/three stable item modes and church transfer; four independent styles/four locales/scroll; complete opaque role RGBA and all outside bytes equal; no new original oracle claim", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
