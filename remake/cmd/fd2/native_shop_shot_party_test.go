package main

import (
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestParseNativeShopEquipmentRecipientShotState(t *testing.T) {
	for _, tc := range []struct {
		spec                                string
		good, selection, start, cycle, gold int
		ok                                  bool
	}{
		{spec: "0,0,0,0,1000", gold: 1000, ok: true},
		{spec: "3,2,1,2,99999999", good: 3, selection: 2, start: 1, cycle: 2, gold: 99999999, ok: true},
		{spec: "-1,0,0,0,1000"},
		{spec: "0,-1,0,0,1000"},
		{spec: "0,0,-1,0,1000"},
		{spec: "0,0,0,-1,1000"},
		{spec: "0,0,0,3,1000"},
		{spec: "0,0,0,0,-1"},
		{spec: "0,0,0,0,100000000"},
		{spec: "0,0,0,0"},
		{spec: "0,0,0,0,1,2"},
		{spec: "x,0,0,0,1"},
	} {
		good, selection, start, cycle, gold, ok :=
			parseNativeShopEquipmentRecipientShotState(tc.spec)
		if good != tc.good || selection != tc.selection ||
			start != tc.start || cycle != tc.cycle ||
			gold != tc.gold || ok != tc.ok {
			t.Fatalf(
				"parseNativeShopEquipmentRecipientShotState(%q)=(%d,%d,%d,%d,%d,%v), want (%d,%d,%d,%d,%d,%v)",
				tc.spec, good, selection, start, cycle, gold, ok,
				tc.good, tc.selection, tc.start, tc.cycle, tc.gold, tc.ok,
			)
		}
	}
}

func TestParseNativeShopSellShotState(t *testing.T) {
	for _, tc := range []struct {
		spec                                string
		mode                                string
		unit, selection, start, cycle, gold int
		ok                                  bool
	}{
		{spec: "roster,0,0,0,0,0", mode: "roster", ok: true},
		{spec: "items,1,2,0,2,99999999", mode: "items", unit: 1, selection: 2, cycle: 2, gold: 99999999, ok: true},
		{spec: "unknown,0,0,0,0,0"},
		{spec: "roster,-1,0,0,0,0"},
		{spec: "items,0,0,0,3,0"},
		{spec: "items,0,0,0,0,100000000"},
		{spec: "items,0,0,0,0"},
	} {
		mode, unit, selection, start, cycle, gold, ok :=
			parseNativeShopSellShotState(tc.spec)
		if mode != tc.mode || unit != tc.unit || selection != tc.selection ||
			start != tc.start || cycle != tc.cycle || gold != tc.gold || ok != tc.ok {
			t.Fatalf(
				"parseNativeShopSellShotState(%q)=(%q,%d,%d,%d,%d,%d,%v), want (%q,%d,%d,%d,%d,%d,%v)",
				tc.spec, mode, unit, selection, start, cycle, gold, ok,
				tc.mode, tc.unit, tc.selection, tc.start, tc.cycle, tc.gold, tc.ok,
			)
		}
	}
}

func TestParseNativeShopSellConfirmShotState(t *testing.T) {
	for _, tc := range []struct {
		spec                            string
		unit, item, choice, pulse, gold int
		ok                              bool
	}{
		{spec: "0,0,0,0,0", ok: true},
		{spec: "1,2,1,3,99999999", unit: 1, item: 2, choice: 1, pulse: 3, gold: 99999999, ok: true},
		{spec: "-1,0,0,0,0"},
		{spec: "0,0,2,0,0"},
		{spec: "0,0,0,4,0"},
		{spec: "0,0,0,0,100000000"},
		{spec: "0,0,0,0"},
	} {
		unit, item, choice, pulse, gold, ok :=
			parseNativeShopSellConfirmShotState(tc.spec)
		if unit != tc.unit || item != tc.item || choice != tc.choice ||
			pulse != tc.pulse || gold != tc.gold || ok != tc.ok {
			t.Fatalf(
				"parseNativeShopSellConfirmShotState(%q)=(%d,%d,%d,%d,%d,%v), want (%d,%d,%d,%d,%d,%v)",
				tc.spec, unit, item, choice, pulse, gold, ok,
				tc.unit, tc.item, tc.choice, tc.pulse, tc.gold, tc.ok,
			)
		}
	}
}

func TestNativeShopEquipmentRecipientShotUsesBindingPartyProjection(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(base, "DATO.DAT"))

	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	shop, err := loadNativeShopUIAssets(shared)
	if err != nil {
		t.Fatal(err)
	}
	c, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	runner := campaign.NewRunner(c)
	runner.Cur = "shop_ch02_weapon"
	types, equip, err := campaign.LoadShopEligibility(
		assetPath("assets/data/item.json"),
		assetPath("assets/data/class_equip_types.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := campaign.LoadItemStats(assetPath("assets/data/item.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{
		shotPath:      "recipient.png",
		camp:          runner,
		nativeClassUI: shared,
		nativeShopUI:  shop,
		nativeUIPalette: append(
			color.Palette(nil), shared.palette...,
		),
		shopItemTypes:  types,
		shopEquipTypes: equip,
		shopItemStats:  stats,
	}
	if err := g.materializeShotPartyFromBinding(
		"assets/cutscenes/bindings/ch00_pre.json",
	); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.partyJoinOrder, []int{0, 9, 4, 30}) {
		t.Fatalf("binding party order=%v", g.partyJoinOrder)
	}
	if !g.setupNativeShop() {
		t.Fatal("ch02 weapon shop did not admit native owner")
	}
	g.nativeShopUIJob = nil
	if !g.setNativeShopEquipmentRecipientShotState(0, 0, 0, 1, 1000) {
		t.Fatal("equipment recipient state rejected verified binding party")
	}
	if g.nativeShopRecipientCycle != 1 {
		t.Fatalf("recipient cycle=%d", g.nativeShopRecipientCycle)
	}
	if !reflect.DeepEqual(g.shopRecipients, []int{0, 9, 4}) {
		t.Fatalf("equipment recipients=%v, want original visible order", g.shopRecipients)
	}
	wantStats := map[int]struct {
		current, candidate [4]int
	}{
		0: {current: [4]int{16, 12, 97, 2}, candidate: [4]int{16, 6, 97, 2}},
		9: {current: [4]int{11, 7, 86, 1}, candidate: [4]int{11, 4, 86, 1}},
		4: {current: [4]int{26, 6, 92, 2}, candidate: [4]int{26, 6, 92, 2}},
	}
	for _, id := range g.shopRecipients {
		unit := g.partyRoster[id]
		record, err := campaign.NativeShopEquipmentRecordForUnit(&unit)
		if err != nil {
			t.Fatal(err)
		}
		current, err := campaign.NativeShopEquipmentCurrentStats(record)
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := campaign.NativeShopEquipmentCandidateStats(
			record, 0x80, g.nativeShopUI.effectRows,
		)
		if err != nil {
			t.Fatal(err)
		}
		if current != wantStats[id].current ||
			candidate != wantStats[id].candidate {
			t.Fatalf(
				"recipient %d stats current=%v candidate=%v want=%v/%v",
				id, current, candidate,
				wantStats[id].current, wantStats[id].candidate,
			)
		}
	}
	frame, ok := g.composeNativeShopEquipmentRecipient()
	if !ok || len(frame) != 320*200 {
		t.Fatal("equipment recipient did not pass final compositor admission")
	}

	oldMode, oldSelection, oldStart, oldGold :=
		g.nativeShopMode, g.shopRecipientSel, g.nativeShopRecipientStart, g.gold
	if g.setNativeShopEquipmentRecipientShotState(0, 3, 0, 0, 1000) ||
		g.nativeShopMode != oldMode ||
		g.shopRecipientSel != oldSelection ||
		g.nativeShopRecipientStart != oldStart || g.gold != oldGold {
		t.Fatal("out-of-range recipient state did not fail atomically")
	}
}

func TestNativeShopSellShotUsesBindingPartyProjection(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(base, "DATO.DAT"))

	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	shop, err := loadNativeShopUIAssets(shared)
	if err != nil {
		t.Fatal(err)
	}
	c, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	runner := campaign.NewRunner(c)
	runner.Cur = "shop_ch02_weapon"
	types, equip, err := campaign.LoadShopEligibility(
		assetPath("assets/data/item.json"),
		assetPath("assets/data/class_equip_types.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := campaign.LoadItemStats(assetPath("assets/data/item.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{
		shotPath: "sell.png", camp: runner,
		nativeClassUI: shared, nativeShopUI: shop,
		shopItemTypes: types, shopEquipTypes: equip, shopItemStats: stats,
	}
	if err := g.materializeShotPartyFromBinding(
		"assets/cutscenes/bindings/ch00_pre.json",
	); err != nil {
		t.Fatal(err)
	}
	if !g.setupNativeShop() {
		t.Fatal("ch02 weapon shop did not admit native owner")
	}
	g.nativeShopUIJob = nil
	if !g.setNativeShopSellShotState("roster", 0, 0, 0, 0, 0) ||
		g.nativeShopMode != "sell_roster" {
		t.Fatal("sell roster shot rejected verified binding party")
	}
	if frame, ok := g.composeNativeShopSellRoster(); !ok || len(frame) != 320*200 {
		t.Fatal("sell roster shot did not pass final compositor admission")
	}
	g.nativeShopMode = "menu"
	if !g.setNativeShopSellShotState("items", 0, 0, 0, 0, 0) ||
		g.nativeShopMode != "sell_items" {
		t.Fatal("sell item shot rejected verified binding party")
	}
	if !reflect.DeepEqual(g.nativeShopSellItemIDs, []int{0, 132, 192}) {
		t.Fatalf("sell items=%v", g.nativeShopSellItemIDs)
	}
	if frame, ok := g.composeNativeShopSellItems(); !ok || len(frame) != 320*200 {
		t.Fatal("sell item shot did not pass final compositor admission")
	}
	g.nativeShopMode = "menu"
	if g.setNativeShopSellShotState("roster", 0, 1, 0, 0, 0) {
		t.Fatal("sell roster shot accepted divergent unit/selection")
	}
	if g.nativeShopMode != "menu" {
		t.Fatal("rejected sell shot did not roll back atomically")
	}
	if !g.setNativeShopSellConfirmShotState(0, 0, 0, 2, 0) ||
		g.nativeShopMode != "sell_confirm" ||
		g.nativeShopSellConfirmSel != 0 || g.nativeShopUIPulse != 2 {
		t.Fatal("sell confirmation shot rejected verified raw item")
	}
	if frame, ok := g.composeNativeShopSellConfirmation(); !ok || len(frame) != 320*200 {
		t.Fatal("sell confirmation shot did not pass final compositor admission")
	}
	g.nativeShopMode = "menu"
	if g.setNativeShopSellConfirmShotState(0, 99, 0, 0, 0) {
		t.Fatal("sell confirmation shot accepted missing raw item")
	}
	if g.nativeShopMode != "menu" {
		t.Fatal("rejected sell confirmation did not roll back atomically")
	}
}

func TestShotPartyBindingRequiresScreenshotAndCompletePartyLoadCH(t *testing.T) {
	g := &Game{}
	if err := g.materializeShotPartyFromBinding(
		"assets/cutscenes/bindings/ch00_pre.json",
	); err == nil {
		t.Fatal("party binding accepted non-screenshot runtime")
	}
	g.shotPath = "shot.png"
	if err := g.materializeShotPartyFromBinding(
		"assets/cutscenes/bindings/ch00_post.json",
	); err == nil {
		t.Fatal("party binding accepted handler without complete party LOADCH")
	}
}
