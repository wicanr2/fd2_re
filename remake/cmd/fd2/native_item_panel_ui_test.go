package main

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func nativeItemPanelTestUnit() *battle.Unit {
	return &battle.Unit{
		BattleFig: 0, NativeIdentity: 0, HasNativeIdentity: true,
		NativeRecordByte6: 1, HasNativeRecordByte6: true,
		NativeRecordRace: 1, HasNativeRecordRace: true,
		NativeRecordClass: 1, HasNativeRecordClass: true,
		Lv: 12, MV: 5, Exp: 34, DX: 56,
		HP: 80, MaxHP: 100, MP: 20, MaxMP: 40,
		AP: 123, DP: 98, HIT: 76, EV: 54,
		InventorySlots:       []int{0, 0xff, 79, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags: []int{0x40, 0x80, 0, 0x80, 0x80, 0x80, 0x80, 0x80},
	}
}

func TestNativeItemRawSlotsCompactsDisplayWithoutMovingRecords(t *testing.T) {
	unit := nativeItemPanelTestUnit()
	got := nativeItemRawSlots(unit)
	if len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("raw slots=%v", got)
	}
}

func TestPrepareNativeItemPanelUsesSeparatedTextFontAndLMI1Entries(t *testing.T) {
	t.Setenv("FD2_ORIGINAL_FDOTHER", "")
	t.Setenv("FD2_ORIGINAL_FDTXT", t.TempDir()+"/missing-FDTXT.DAT")
	t.Setenv("FD2_ASSET_PACK", "../../generated-assets/fd2-original-b97caf22")
	t.Setenv("FD2_ORIGINAL_DATO", "")
	g := &Game{sel: nativeItemPanelTestUnit(), itemOpen: true}
	g.nativeUIPalette = loadNativeUIPalette()
	if !g.prepareNativeItemPanel(g.sel) || g.nativeItemPanel == nil {
		t.Fatal("native item panel did not prepare")
	}
	screen := ebiten.NewImage(640, 400)
	if !g.drawNativeItemPanel(screen) {
		t.Fatal("native item panel frame 11 did not draw")
	}
	if g.itemAnimStep != 0 {
		t.Fatalf("initial animation step=%d", g.itemAnimStep)
	}
	for step := 0; step < 11; step++ {
		if !g.stepNativeItemPanelAnimation() {
			t.Fatalf("opening step %d did not block input", step)
		}
	}
	if g.itemAnimStep != 11 || g.stepNativeItemPanelAnimation() {
		t.Fatalf("opened animation step=%d", g.itemAnimStep)
	}
	g.beginNativeItemPanelClose()
	if !g.itemClosing || g.itemAnimStep != 0 {
		t.Fatalf("closing state=%v/%d", g.itemClosing, g.itemAnimStep)
	}
	for step := 0; step < 12; step++ {
		if !g.stepNativeItemPanelAnimation() {
			t.Fatalf("closing step %d did not block input", step)
		}
	}
	if g.itemOpen || !g.ring || g.nativeItemPanel != nil {
		t.Fatalf("closed state item=%v ring=%v panel=%v", g.itemOpen, g.ring, g.nativeItemPanel)
	}
}

func TestCampaignChapterOnePartyPreparesNativeItemPanel(t *testing.T) {
	sc, err := battle.LoadScenario("../../assets/scenarios/ch01.json")
	if err != nil {
		t.Fatal(err)
	}
	units := sc.PartyUnits(nil)
	if len(units) == 0 {
		t.Fatal("chapter one has no party")
	}
	unit := units[0]
	if _, err := battle.NativeItemPanelRecordForUnit(unit); err != nil {
		t.Fatalf("normal campaign party lacks native item-panel provenance: %v", err)
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", "")
	t.Setenv("FD2_ORIGINAL_FDTXT", t.TempDir()+"/missing-FDTXT.DAT")
	t.Setenv("FD2_ASSET_PACK", "../../generated-assets/fd2-original-b97caf22")
	t.Setenv("FD2_ORIGINAL_DATO", "")
	g := &Game{sel: unit, itemOpen: true, nativeUIPalette: loadNativeUIPalette()}
	if !g.prepareNativeItemPanel(unit) || g.nativeItemPanel == nil {
		t.Fatal("normal campaign party did not prepare native item panel")
	}
}

func TestApplyNativeImmediateItemUsesTwoStageSelfTargetAndEndsAction(t *testing.T) {
	unit := nativeItemPanelTestUnit()
	unit.Inventory = []int{198, 0x20}
	unit.Equipped = []bool{false, true}
	unit.InventorySlots = []int{198, 0x20, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	unit.NativeInventoryFlags = []int{0, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	unit.OnField, unit.EquipmentBaseSet, unit.BaseAP = true, true, 20
	g := &Game{
		st: &battle.State{
			W: 1, H: 1, Units: []*battle.Unit{unit},
			NativeCompositionEventBytes: []byte{0},
		},
		sel: unit, moved: true, itemOpen: true,
		shopItemStats: map[int]campaign.ItemStats{0x20: {AP: 2}},
	}
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	applied, err := g.applyNativeImmediateItem(0, 198)
	if err != nil || !applied {
		t.Fatalf("applied=%v err=%v", applied, err)
	}
	if unit.BaseAP != 29 || unit.AP != 31 || !unit.Acted ||
		!unit.HasNativeRecordByte5 || unit.NativeRecordByte5&0x80 == 0 ||
		len(unit.Inventory) != 1 || unit.Inventory[0] != 0x20 ||
		g.sel != nil || g.itemOpen || g.moved {
		t.Fatalf("native immediate item transaction incomplete: unit=%#v game=%#v", unit, g)
	}
}

func TestApplyNativeImmediateCapacityItemKeepsCurrentHP(t *testing.T) {
	unit := nativeItemPanelTestUnit()
	unit.HP, unit.MaxHP = 40, 100
	unit.Inventory = []int{94}
	unit.Equipped = []bool{false}
	unit.InventorySlots = []int{94, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	unit.NativeInventoryFlags = []int{0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	unit.OnField = true
	g := &Game{
		st: &battle.State{
			W: 1, H: 1, Units: []*battle.Unit{unit},
			NativeCompositionEventBytes: []byte{0},
		},
		sel: unit, moved: true, itemOpen: true,
	}
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	applied, err := g.applyNativeImmediateItem(0, 94)
	if err != nil || !applied {
		t.Fatalf("applied=%v err=%v", applied, err)
	}
	if unit.HP != 40 || unit.MaxHP != 120 || len(unit.Inventory) != 0 ||
		!unit.Acted || unit.NativeRecordByte5&0x80 == 0 || g.sel != nil {
		t.Fatalf("native capacity item transaction incomplete: unit=%#v", unit)
	}
}

func TestNativeHPRestoreTargetTransactionUsesProcessRNGAndConsumesSource(t *testing.T) {
	actor := nativeItemPanelTestUnit()
	actor.X, actor.Y, actor.OnField = 1, 1, true
	actor.NativeRecordByte5, actor.HasNativeRecordByte5 = 0, true
	actor.Inventory = []int{192}
	actor.Equipped = []bool{false}
	actor.InventorySlots = []int{192, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	actor.NativeInventoryFlags = []int{0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}

	target := nativeItemPanelTestUnit()
	target.X, target.Y, target.OnField = 1, 2, true
	target.NativeIdentity = 1
	target.NativeRecordByte5, target.HasNativeRecordByte5 = 0, true
	target.HP, target.MaxHP = 20, 100

	g := &Game{
		st: &battle.State{
			W: 3, H: 3, Units: []*battle.Unit{actor, target},
			NativeCompositionEventBytes: make([]byte, 9),
			NativeTileBlitModes:         make([]byte, 9),
		},
		sel: actor, moved: true, itemOpen: true,
	}
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	targeting, err := g.beginNativeTargetItem(0, 192)
	if err != nil || !targeting || !g.nativeItemTargeting || g.itemOpen {
		t.Fatalf("targeting=%v state=%v itemOpen=%v err=%v", targeting, g.nativeItemTargeting, g.itemOpen, err)
	}
	candidates := g.nativeItemSelectionTargets()
	foundTarget := false
	for _, candidate := range candidates {
		foundTarget = foundTarget || candidate == target
	}
	if !foundTarget {
		t.Fatalf("native item selection candidates=%v", candidates)
	}
	applied, err := g.applyNativeTargetItem(target)
	if err != nil || !applied {
		t.Fatalf("applied=%v err=%v", applied, err)
	}
	wantState := fdother.NativeRNGStep(0)
	wantHP := 20 + 40*9/10 + int(wantState%100)*40/1000
	if target.HP != wantHP || g.nativeRNGState != wantState ||
		len(actor.Inventory) != 0 || actor.NativeInventoryFlags[0]&0x80 == 0 ||
		!actor.Acted || actor.NativeRecordByte5&0x80 == 0 ||
		g.sel != nil || g.nativeItemTargeting {
		t.Fatalf("native HP transaction actor=%#v targetHP=%d rng=%#x game=%#v", actor, target.HP, g.nativeRNGState, g)
	}
}

func TestNativeType5HPRestoreKeepsItemPanelWithoutApplicableTarget(t *testing.T) {
	actor := nativeItemPanelTestUnit()
	actor.X, actor.Y, actor.OnField = 1, 1, true
	actor.HP, actor.MaxHP = 100, 100
	target := nativeItemPanelTestUnit()
	target.X, target.Y, target.OnField = 1, 2, true
	target.NativeIdentity = 1
	target.HP, target.MaxHP = 100, 100
	state := &battle.State{
		W: 3, H: 3, Units: []*battle.Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 9),
		NativeTileBlitModes:         []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	}
	if !state.MaterializeNativeMapRangeMode(1) {
		t.Fatal("baseline selector rejected")
	}
	g := &Game{st: state, sel: actor, moved: true, itemOpen: true}
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	beforeField := append([]byte(nil), state.NativeTileBlitModes...)
	started, err := g.beginNativeTargetItem(2, 192)
	if err != nil || started {
		t.Fatalf("full-HP type5 target started=%v err=%v", started, err)
	}
	if !g.itemOpen || g.nativeItemTargeting || state.NativeMapRangeMode != 1 ||
		!slices.Equal(beforeField, state.NativeTileBlitModes) {
		t.Fatalf("full-HP gate crossed modal boundary: itemOpen=%v targeting=%v selector=%d field=%v",
			g.itemOpen, g.nativeItemTargeting, state.NativeMapRangeMode, state.NativeTileBlitModes)
	}

	target.HP = 99
	started, err = g.beginNativeTargetItem(2, 192)
	if err != nil || !started || !g.nativeItemTargeting || g.itemOpen {
		t.Fatalf("injured target entry started=%v targeting=%v itemOpen=%v err=%v",
			started, g.nativeItemTargeting, g.itemOpen, err)
	}
}

func TestNativeMarkerClearTargetTransactionSyncsTransientAndRNG(t *testing.T) {
	actor := nativeItemPanelTestUnit()
	actor.X, actor.Y, actor.OnField = 1, 1, true
	actor.NativeRecordByte5, actor.HasNativeRecordByte5 = 0, true
	actor.Inventory = []int{196}
	actor.Equipped = []bool{false}
	actor.InventorySlots = []int{196, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	actor.NativeInventoryFlags = []int{0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}

	target := nativeItemPanelTestUnit()
	target.X, target.Y, target.OnField = 1, 2, true
	target.NativeIdentity = 1
	target.NativeRecordByte5, target.HasNativeRecordByte5 = 0, true
	target.HP, target.MaxHP = 20, 100
	target.NativeTransient[3] = 4 // raw record +0x25, type-6 marker

	g := &Game{
		st: &battle.State{
			W: 3, H: 3, Units: []*battle.Unit{actor, target},
			NativeCompositionEventBytes: make([]byte, 9),
			NativeTileBlitModes:         make([]byte, 9),
		},
		sel: actor, moved: true, itemOpen: true,
	}
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	if targeting, err := g.beginNativeTargetItem(0, 196); err != nil || !targeting {
		t.Fatalf("targeting=%v err=%v", targeting, err)
	}
	if applied, err := g.applyNativeTargetItem(target); err != nil || !applied {
		t.Fatalf("applied=%v err=%v", applied, err)
	}
	wantState := fdother.NativeRNGStep(0)
	if target.NativeTransient[3] != 0 || target.HP != 29 ||
		g.nativeRNGState != wantState || len(actor.Inventory) != 0 ||
		!actor.Acted || actor.NativeRecordByte5&0x80 == 0 {
		t.Fatalf("marker-clear actor=%#v target=%#v rng=%#x", actor, target, g.nativeRNGState)
	}
}

func TestNativeRetainedStatItemsSyncDerivedWordsAndMarkers(t *testing.T) {
	for _, tc := range []struct {
		itemID                  int
		marker                  int
		wantAP, wantHIT, wantEV int
	}{
		{itemID: 210, marker: 2, wantAP: 123, wantHIT: 91, wantEV: 69},
		{itemID: 214, marker: 0, wantAP: 142, wantHIT: 76, wantEV: 54},
	} {
		unit := nativeItemPanelTestUnit()
		unit.X, unit.Y, unit.OnField = 1, 1, true
		unit.NativeRecordByte5, unit.HasNativeRecordByte5 = 0, true
		unit.Inventory = []int{tc.itemID}
		unit.Equipped = []bool{false}
		unit.InventorySlots = []int{tc.itemID, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
		unit.NativeInventoryFlags = []int{0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
		unit.NativeTransient[tc.marker] = 0
		g := &Game{
			st: &battle.State{
				W: 3, H: 3, Units: []*battle.Unit{unit},
				NativeCompositionEventBytes: make([]byte, 9),
				NativeTileBlitModes:         make([]byte, 9),
			},
			sel: unit, moved: true, itemOpen: true,
		}
		var err error
		g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
		if err != nil {
			t.Fatal(err)
		}
		if targeting, err := g.beginNativeTargetItem(0, tc.itemID); err != nil || !targeting {
			t.Fatalf("item %d targeting=%v err=%v", tc.itemID, targeting, err)
		}
		if applied, err := g.applyNativeTargetItem(unit); err != nil || !applied {
			t.Fatalf("item %d applied=%v err=%v", tc.itemID, applied, err)
		}
		if unit.AP != tc.wantAP || unit.HIT != tc.wantHIT || unit.EV != tc.wantEV ||
			unit.NativeTransient[tc.marker] < 2 || unit.NativeTransient[tc.marker] > 5 ||
			g.nativeRNGState != fdother.NativeRNGStep(0) ||
			len(unit.Inventory) != 1 || unit.Inventory[0] != tc.itemID {
			t.Fatalf("item %d unit=%#v rng=%#x", tc.itemID, unit, g.nativeRNGState)
		}
	}
}

func TestNativeRetainedMarkerApplicationSyncsDamageAndThreeRNGSteps(t *testing.T) {
	actor := nativeItemPanelTestUnit()
	actor.X, actor.Y, actor.OnField, actor.Camp = 1, 1, true, battle.Own
	actor.NativeRecordByte5, actor.HasNativeRecordByte5 = 0, true
	actor.Inventory = []int{212}
	actor.Equipped = []bool{false}
	actor.InventorySlots = []int{212, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	actor.NativeInventoryFlags = []int{0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}

	target := nativeItemPanelTestUnit()
	target.X, target.Y, target.OnField, target.Camp = 1, 2, true, battle.Enemy
	target.NativeIdentity = 1
	target.NativeRecordByte5, target.HasNativeRecordByte5 = 0, true
	target.HP, target.MaxHP = 50, 100
	target.NativeTransient[4] = 0 // type-14 raw marker +0x26

	g := &Game{
		st: &battle.State{
			W: 3, H: 3, Units: []*battle.Unit{actor, target},
			NativeCompositionEventBytes: make([]byte, 9),
			NativeTileBlitModes:         make([]byte, 9),
		},
		sel: actor, moved: true, itemOpen: true,
	}
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	if targeting, err := g.beginNativeTargetItem(0, 212); err != nil || !targeting {
		t.Fatalf("targeting=%v err=%v", targeting, err)
	}
	if applied, err := g.applyNativeTargetItem(target); err != nil || !applied {
		t.Fatalf("applied=%v err=%v", applied, err)
	}
	state := fdother.NativeRNGStep(fdother.NativeRNGStep(fdother.NativeRNGStep(0)))
	if target.HP != 41 || target.NativeTransient[4] < 2 || target.NativeTransient[4] > 5 ||
		g.nativeRNGState != state || len(actor.Inventory) != 1 ||
		actor.Inventory[0] != 212 || !actor.Acted {
		t.Fatalf("marker application actor=%#v target=%#v rng=%#x", actor, target, g.nativeRNGState)
	}
}

func TestNativeCommandDamageItemFailsClosedWithoutIndexedPresentation(t *testing.T) {
	actor := nativeItemPanelTestUnit()
	actor.X, actor.Y, actor.OnField, actor.Camp = 0, 0, true, battle.Own
	actor.NativeRecordByte5, actor.HasNativeRecordByte5 = 0, true
	actor.Inventory = []int{56}
	actor.Equipped = []bool{false}
	actor.InventorySlots = []int{56, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	actor.NativeInventoryFlags = []int{0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}

	target := nativeItemPanelTestUnit()
	target.X, target.Y, target.OnField, target.Camp = 1, 0, true, battle.Enemy
	target.NativeIdentity, target.ClassID = 1, 1
	target.NativeRecordByte5, target.HasNativeRecordByte5 = 0, true
	target.HP, target.MaxHP = 200, 200
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[0] = battle.NativeCommandRecord{ID: 0, Damage: 100, Hit: 100}
	g := &Game{
		st: &battle.State{
			W: 2, H: 1, Units: []*battle.Unit{actor, target},
			NativeCompositionEventBytes: make([]byte, 2),
			NativeTileBlitModes:         make([]byte, 2),
		},
		sel: actor, moved: true, itemOpen: true,
		nativeCommandBook:        book,
		nativeCommandResistances: map[int]int{1: 10},
	}
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	if targeting, err := g.beginNativeTargetItem(0, 56); err != nil || !targeting {
		t.Fatalf("targeting=%v err=%v", targeting, err)
	}
	if applied, err := g.applyNativeTargetItem(target); err == nil || applied {
		t.Fatalf("missing indexed presentation applied=%v err=%v", applied, err)
	}
	if target.HP != 200 || g.nativeRNGState != 0 ||
		len(actor.Inventory) != 1 || actor.Inventory[0] != 56 || actor.Acted ||
		actor.NativeRecordByte5&0x80 != 0 || !g.nativeItemTargeting {
		t.Fatalf("failed command item mutated actor=%#v target=%#v rng=%#x targeting=%v", actor, target, g.nativeRNGState, g.nativeItemTargeting)
	}
}

func TestNativeRelocationUsesSeparateDestinationCursorAndMode6Legality(t *testing.T) {
	g := completeNativeUnitPresentGame(t)
	actor := g.st.Units[0]
	actor.X, actor.Y, actor.OnField, actor.Camp = 0, 0, true, battle.Own
	actor.NativeIdentity = 24
	actor.HasNativeIdentity = true
	actor.NativeRecordByte5, actor.HasNativeRecordByte5 = 0, true
	actor.MP, actor.MaxMP = 30, 40
	actor.Inventory = []int{101}
	actor.Equipped = []bool{false}
	actor.InventorySlots = []int{101, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	actor.NativeInventoryFlags = []int{0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}

	target := g.st.Units[1]
	target.X, target.Y, target.OnField, target.Camp = 1, 0, true, battle.Own
	if !target.SetNativeMapCoordinatesRaw(1, 0) {
		t.Fatal("target raw map coordinates unavailable")
	}
	target.NativeIdentity = 1
	target.HasNativeIdentity = true
	target.NativeRecordClass = 1
	target.HasNativeRecordClass = true
	target.NativeRecordByte5, target.HasNativeRecordByte5 = 0, true
	target.InventorySlots = []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	target.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	g.st.Units = g.st.Units[:2]

	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[23].MPCost = 20
	g.sel, g.moved, g.itemOpen = actor, true, true
	g.nativeCommandBook = book
	g.st.NativeCompositionEventBytes = make([]byte, g.st.W*g.st.H)
	g.st.NativeTerrainMoveCodes = make([]byte, g.st.W*g.st.H)
	g.st.NativeTileBlitModes[target.Y*g.st.W+target.X] = 0
	g.m.NativeTileBlitModes[target.Y*g.st.W+target.X] = 0
	for index := range g.st.NativeTerrainMoveCodes {
		g.st.NativeTerrainMoveCodes[index] = 2
	}
	g.nativeMapDAC = append([]byte(nil), g.nativeMapAssets.PaletteDAC...)
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CursorX: 1, CursorY: 0, VisibleCursorX: 1, VisibleCursorY: 0,
	}); err != nil {
		t.Fatal(err)
	}
	g.camX, g.camY = 0, 0
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	var err error
	g.nativeCommandPaletteFlash, err = battle.LoadNativeCommandPaletteFlashTable("../../assets/data/native_command_palette_flash.json")
	if err != nil {
		t.Fatal(err)
	}
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	if targeting, err := g.beginNativeTargetItem(0, 101); err != nil || !targeting {
		t.Fatalf("targeting=%v err=%v", targeting, err)
	}
	wantFirstSelector := int(g.nativeItemEffectRows[101*battle.NativeItemEffectRowSize+0x12]) + 2
	if g.st.NativeMapRangeMode != wantFirstSelector {
		t.Fatalf("type23 first-stage selector=%d, want row+0x12+2 == %d", g.st.NativeMapRangeMode, wantFirstSelector)
	}
	if applied, err := g.applyNativeTargetItem(target); err != nil || applied ||
		!g.nativeItemRelocating || g.nativeItemTargeting {
		t.Fatalf("target stage applied=%v relocating=%v targeting=%v err=%v", applied, g.nativeItemRelocating, g.nativeItemTargeting, err)
	}
	allFF := make([]byte, g.st.W*g.st.H)
	for index := range allFF {
		allFF[index] = 0xff
	}
	if g.st.NativeMapRangeMode != 1 || !slices.Equal(g.st.NativeTileBlitModes, allFF) {
		t.Fatalf("destination stage selector/grid=%d/%#v", g.st.NativeMapRangeMode, g.st.NativeTileBlitModes)
	}
	if !g.cancelNativeItemTargetModal() || !g.itemOpen ||
		g.nativeItemTargeting || g.nativeItemRelocating ||
		g.st.NativeMapRangeMode != 1 ||
		!slices.Equal(g.st.NativeTileBlitModes, allFF) {
		t.Fatalf("destination cancel did not return to item panel: %#v", g)
	}
	if targeting, err := g.beginNativeTargetItem(0, 101); err != nil || !targeting {
		t.Fatalf("retargeting=%v err=%v", targeting, err)
	}
	if applied, err := g.applyNativeTargetItem(target); err != nil || applied ||
		!g.nativeItemRelocating {
		t.Fatalf("second target stage applied=%v relocating=%v err=%v", applied, g.nativeItemRelocating, err)
	}
	destinations := g.nativeRelocationDestinations()
	if !destinations[battle.Cell{X: 2, Y: 2}] ||
		destinations[battle.Cell{X: actor.X, Y: actor.Y}] {
		t.Fatalf("mode6 destinations=%v", destinations)
	}
	t.Setenv("FD2_MUTE", "1")
	introPixels := g.nativeMapAssets.FDOTHER6[0x72].Pixels
	g.nativeMapAssets.FDOTHER6[0x72].Pixels = nil
	if applied, err := g.applyNativeRelocationDestination(8, 7); err == nil || applied ||
		g.nativeModifierPresentation != nil || g.nativeUnitPresent != nil ||
		target.X != 1 || target.Y != 0 || actor.MP != 30 || actor.Acted {
		t.Fatalf("missing renderer crossed atomic boundary: applied=%v err=%v actor=%#v target=%#v", applied, err, actor, target)
	}
	g.nativeMapAssets.FDOTHER6[0x72].Pixels = introPixels
	if applied, err := g.applyNativeRelocationDestination(8, 7); err != nil || !applied {
		t.Fatalf("destination applied=%v err=%v", applied, err)
	}
	if g.nativeModifierPresentation == nil || target.X != 1 || target.Y != 0 || actor.MP != 30 || actor.Acted {
		t.Fatalf("relocation crossed palette boundary: actor=%#v target=%#v", actor, target)
	}
	for phase := 0; phase < 8; phase++ {
		if !g.drawNativeCommandModifierPresentation(ebiten.NewImage(640, 400)) {
			t.Fatalf("command23 palette phase %d did not draw", phase)
		}
		g.stepNativeCommandModifierPresentation()
	}
	if g.nativeModifierPresentation != nil || g.nativeUnitPresent == nil || target.X != 1 || target.Y != 0 || actor.MP != 30 {
		t.Fatalf("command23 did not enter disappear presentation atomically: actor=%#v target=%#v", actor, target)
	}
	sawOffMap := false
	for steps := 0; g.nativeUnitPresent != nil && steps < 120; steps++ {
		stepNativeUnitPresentNow(g)
		if target.X == 0xff && target.Y == 0xff {
			sawOffMap = true
		}
	}
	if g.nativeUnitPresent != nil || !sawOffMap || target.X != 8 || target.Y != 7 || actor.MP != 10 ||
		len(actor.Inventory) != 1 || actor.Inventory[0] != 101 ||
		!actor.Acted || g.sel != nil || g.nativeItemRelocating ||
		g.st.NativeMapRangeMode != 1 ||
		!slices.Equal(g.st.NativeTileBlitModes, allFF) {
		t.Fatalf("relocation actor=%#v target=%#v game=%#v", actor, target, g)
	}
}
