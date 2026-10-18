package main

import (
	"bytes"
	"image"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestModernHDItemSelectionUsesRawSlotsAndKeepsIndexedCache(t *testing.T) {
	g, screen, unit := newModernHDStatusGame(t)
	if !g.prepareNativeItemPanel(unit) {
		t.Fatal("item panel unavailable")
	}
	legacy := make([]byte, 64000)
	if err := battle.RenderNativeItemPanelResources(separatedAssetPath(""), separatedAssetPath("portraits"), g.nativeItemPanelRecord, legacy); err != nil || !bytes.Equal(legacy, g.nativeItemPanelBase) {
		t.Fatalf("legacy base changed: %v", err)
	}
	defer prepareHDChurchText(t, g, screen)()
	g.itemAnimStep = 11
	for selection, raw := range []int{0, 2} {
		g.itemSel = selection
		if !g.refreshNativeItemPanel(unit) || g.nativeItemPanelText.selectedRawSlot != raw {
			t.Fatalf("compact %d did not select raw %d", selection, raw)
		}
		panel := g.nativeItemPanel
		indexed := append([]byte(nil), g.nativeItemPanelBase...)
		data := g.nativeItemPanelText
		art := append([]byte(nil), data.data.base...)
		g.modernHDText.runs = []modernHDTextRun{
			{text: "outside", clip: image.Rect(0, 0, 8, 8)},
			{text: "covered", clip: image.Rect(30, 250, 60, 280)},
		}
		if !g.drawNativeItemPanel(screen) {
			t.Fatal("HD panel not drawn")
		}
		outside, selected := false, false
		for _, r := range g.modernHDText.runs {
			if r.text == "covered" {
				t.Fatal("covered text leaked through the item panel")
			}
			outside = outside || r.text == "outside"
			selected = selected || r.color == modernHDNativeTextColor(g.nativeUIPalette, 201)
		}
		if !outside || !selected || len(g.modernHDText.runs) <= 1 || panel != g.nativeItemPanel || !bytes.Equal(indexed, g.nativeItemPanelBase) ||
			!bytes.Equal(art, data.data.base) || g.nativeItemPanelText != data || g.font.indexedHD != nil {
			t.Fatal("missing selection/outer text or HD mutated the indexed snapshot")
		}
	}
	g.modernHDText.runs = nil
	g.itemAnimStep = 0
	if !g.drawNativeItemPanel(screen) || len(g.modernHDText.runs) != 0 {
		t.Fatal("HD text leaked into indexed opening")
	}
	g.clearNativeItemPanel()
	if g.nativeItemPanelText != nil {
		t.Fatal("closed snapshot retained")
	}
}

func TestModernHDItemEmptyAndEquipRollbackKeepSnapshot(t *testing.T) {
	g, screen := newNativeShopRecipientPathGame(t)
	g.partyRoster[0] = setNativeTransferInventory(g.partyRoster[0], 0, 1)
	g.shopItemTypes[0], g.shopItemTypes[1] = 0, 0
	g.shopEquipTypes[1] = []int{0}
	g.shopItemStats = map[int]campaign.ItemStats{0: {Type: 0, AP: 1}, 1: {Type: 0, AP: 3}}
	drainNativeShopProductionJobs(t, g, screen)
	enterNativeShopEquipPanel(t, g, 0)
	drainNativeShopProductionJobs(t, g, screen)
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	drainNativeShopProductionJobs(t, g, screen)
	settleNativeShopEquipPanelOpening(g)
	old := g.nativeItemPanelText
	g.handleNativeShopInputState(nativeShopTransferInput{delta: 2})
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	if g.nativeItemPanelText == old || g.nativeItemPanelText.selectedRawSlot != 1 || g.partyRoster[0].AP != 32 {
		t.Fatal("successful equip did not publish its fresh snapshot")
	}
	before := cloneNativeShopUnit(g.partyRoster[0])
	old, panel := g.nativeItemPanelText, g.nativeItemPanel
	g.nativeItemEffectRows = []byte{0}
	if g.applyNativeShopEquipSelection() || g.nativeItemPanelText != old || g.nativeItemPanel != panel || !reflect.DeepEqual(before, g.partyRoster[0]) {
		t.Fatal("failed equip published partial display/state")
	}
	u := setNativeTransferInventory(before)
	if !g.prepareNativeItemPanelMode(&u, true) || g.nativeItemPanelText.selectedRawSlot != -1 {
		t.Fatal("empty panel did not publish an unselected snapshot")
	}
	defer prepareHDChurchText(t, g, screen)()
	g.itemAnimStep = 11
	if !g.drawNativeItemPanel(screen) || len(g.modernHDText.runs) == 0 {
		t.Fatal("empty panel lost status text")
	}
}

func TestModernHDItemOcclusionKeepsExactlyTheOuterArea(t *testing.T) {
	covered := image.Rect(20, 30, 80, 70)
	g := &Game{modernHDText: &modernHDTextFrame{runs: []modernHDTextRun{{text: "same", clip: image.Rect(0, 0, 100, 100)}}}}
	g.clearModernHDTextRegion(covered)
	area := 0
	for i, r := range g.modernHDText.runs {
		if !r.clip.Intersect(covered).Empty() || r.text != "same" {
			t.Fatal("opaque region or text changed")
		}
		area += r.clip.Dx() * r.clip.Dy()
		for _, other := range g.modernHDText.runs[:i] {
			if !r.clip.Intersect(other.clip).Empty() {
				t.Fatal("outer text would be rendered twice")
			}
		}
	}
	if area != 10000-60*40 {
		t.Fatal("outer text area was lost")
	}
}
