package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func nativeEvent61PlayerGame(t *testing.T, items ...int) (*Game, *battle.Unit) {
	t.Helper()
	fdotherPath, err := filepath.Abs(
		"../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT unavailable")
	}
	assetPack, err := filepath.Abs("../../generated-assets/fd2-original-b97caf22")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(assetPack, "animations", "fdother_045_event61", "bank.json")); err != nil {
		t.Skip("separated event61 pack unavailable")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_ASSET_PACK", assetPack)
	t.Setenv("FD2_CAMPAIGN", "assets/scenarios/campaign_full.json")
	t.Setenv("FD2_CAMP_NODE", "battle_ch26")
	// Other native-map families are migrated separately and still need the
	// player archive in this broad battle fixture. The event61 owners themselves
	// are required to load only the separated pack.
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if g.st == nil || len(g.st.Units) == 0 {
		t.Fatal("chapter26 battle state unavailable")
	}
	view := g.st.NativeMapViewState
	hud := g.st.NativeMapHUDState
	if !g.st.HasNativeMapViewState || !g.st.HasNativeMapHUDState ||
		view.CameraX != 9 || view.CameraY != 39 ||
		view.CursorX != 15 || view.CursorY != 46 ||
		view.VisibleCursorX != 6 || view.VisibleCursorY != 7 ||
		hud.DisplayGateA != 1 || hud.DisplayGateB != 1 ||
		hud.AnchorX != 1 {
		t.Fatalf("chapter26 pre-handler runtime view=%#v hud=%#v", view, hud)
	}
	trigger := g.st.Units[0]
	trigger.Acted = false
	trigger.NativeRecordWord42 = uint16(trigger.MaxHP)
	trigger.HasNativeRecordWord42 = true
	trigger.Inventory = append([]int(nil), items...)
	trigger.Equipped = make([]bool, len(items))
	trigger.InventorySlots = make([]int, 8)
	trigger.NativeInventoryFlags = make([]int, 8)
	for i := range trigger.InventorySlots {
		trigger.InventorySlots[i] = 0xff
		trigger.NativeInventoryFlags[i] = 0x80
	}
	for i, item := range items {
		trigger.InventorySlots[i] = item
		trigger.NativeInventoryFlags[i] = 0
	}
	if !g.positionScreenshotCursor(1, 46) {
		t.Fatal("chapter26 native cursor could not reach event61")
	}
	trigger.X, trigger.Y = 1, 46
	g.sel, g.curX, g.curY = trigger, 1, 46
	g.selOrigX, g.selOrigY = 1, 46
	g.moved = true
	return g, trigger
}

func TestNativeEvent61MissingItemRunsOnlyEditableFDTXT2(t *testing.T) {
	g, trigger := nativeEvent61PlayerGame(t, 0x20)
	g.finishSelectedWait()
	if g.battleEvent == nil || len(g.dialog) != 1 ||
		g.dialog[0].Text != "那是什麼奇怪的東西?頭部還開著?" {
		t.Fatalf("missing item dialogue=%#v run=%#v", g.dialog, g.battleEvent)
	}
	if g.nativeFieldEvent61 != nil || g.st.NativeEventState[12] != 0 ||
		!reflect.DeepEqual(trigger.Inventory, []int{0x20}) {
		t.Fatal("missing item path mutated or started presentation")
	}
	g.dialog = nil
	g.advanceBattleEvent()
	if g.battleEvent != nil || g.sel != nil || !trigger.Acted {
		t.Fatal("missing item path did not finish the successful wait action")
	}
}

func TestNativeEvent61AttackWaitsForPresentationCompletion(t *testing.T) {
	g, trigger := nativeEvent61PlayerGame(t, 0x20)
	// The map-26 fixture's native battle selector is 113, whose FIGANI PNG
	// resource is not yet exported. Use the already paired player resources so
	// this test isolates selector1 deferral rather than silently approving a
	// missing presentation asset.
	trigger.BattleFig = 4
	target := &battle.Unit{
		Name: "測試敵兵", BattleFig: 96, Camp: battle.Enemy, X: 2, Y: 46,
		HP: 20, MaxHP: 20, OnField: true,
	}
	g.st.Units = append(g.st.Units, target)
	g.curX, g.curY = target.X, target.Y
	g.confirm()
	if g.atk == nil || g.atk.after == nil {
		t.Fatal("successful attack did not retain the deferred selector1 owner")
	}
	if g.battleEvent != nil {
		t.Fatal("attack committed selector1 before the full-screen presentation")
	}
	g.finishAttackPresentation()
	if g.atk != nil || g.battleEvent == nil || len(g.dialog) != 1 ||
		g.dialog[0].Text != "那是什麼奇怪的東西?頭部還開著?" {
		t.Fatalf(
			"post-presentation event=%#v dialog=%#v acted=%v",
			g.battleEvent, g.dialog, trigger.Acted,
		)
	}
	g.dialog = nil
	g.advanceBattleEvent()
	if g.battleEvent != nil || !trigger.Acted {
		t.Fatal("attack action did not finish after selector1 dialogue")
	}
}

func TestNativeEvent61ImmediateItemRunsAfterSuccessfulMutation(t *testing.T) {
	g, trigger := nativeEvent61PlayerGame(t, 198)
	g.st.NativeCompositionEventBytes = make([]byte, g.st.W*g.st.H)
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix(
		"../../assets/data/native_item_effect_rows.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := g.applyNativeImmediateItem(0, 198)
	if err != nil || !applied {
		t.Fatalf("immediate item applied=%v err=%v", applied, err)
	}
	if len(trigger.Inventory) != 0 ||
		trigger.Acted || g.battleEvent == nil || len(g.dialog) != 1 {
		t.Fatalf(
			"post-item inventory=%v acted=%v event=%#v dialog=%#v",
			trigger.Inventory, trigger.Acted, g.battleEvent, g.dialog,
		)
	}
	g.dialog = nil
	g.advanceBattleEvent()
	if g.battleEvent != nil || !trigger.Acted || g.sel != nil {
		t.Fatal("immediate item action did not finish after selector1 dialogue")
	}
}

func TestNativeEvent61MaterializedRuntimePresentsCommitsAndPersistsWold(t *testing.T) {
	g, trigger := nativeEvent61PlayerGame(t, 0xD0, 0x20)
	view := g.st.NativeMapViewState
	hud := g.st.NativeMapHUDState
	if !g.st.HasNativeMapViewState || !g.st.HasNativeMapHUDState ||
		view.CameraX != 0 || view.CameraY != 39 ||
		view.CursorX != 1 || view.CursorY != 46 ||
		view.VisibleCursorX != 1 || view.VisibleCursorY != 7 ||
		hud.AnchorX != 0xf2 {
		t.Fatalf("event61 campaign runtime view=%#v hud=%#v", view, hud)
	}
	g.finishSelectedWait()
	if g.battleEvent == nil || len(g.dialog) != 1 ||
		g.dialog[0].Text != "這機兵的頭部怎麼開著?這個金屬盒子..好像滿適合的,應該是這樣放進去.....咦!" {
		t.Fatalf("success dialogue3=%#v run=%#v", g.dialog, g.battleEvent)
	}
	g.dialog = nil
	g.advanceBattleEvent()
	job := g.nativeFieldEvent61
	if job == nil || len(job.frames) != 59 || job.frame != 0 {
		t.Fatalf("event61 job=%#v loadErr=%q", job, g.loadErr)
	}
	for frame := 0; frame < 59; frame++ {
		if g.nativeFieldEvent61 == nil ||
			g.nativeFieldEvent61.frame != frame {
			t.Fatalf("frame %d job=%#v", frame, g.nativeFieldEvent61)
		}
		g.nativeFieldEvent61.drawn = true
		if !g.nativeFieldEvent61.hasTick {
			g.stepNativeFieldEvent61Tick(100)
		}
		g.stepNativeFieldEvent61Tick(102 + frame*2)
	}
	if g.nativeFieldEvent61 != nil || g.st.NativeEventState[12] != 1 ||
		!reflect.DeepEqual(trigger.Inventory, []int{0x20}) ||
		!g.partyMembers[31] || len(g.partyJoinOrder) == 0 ||
		g.partyJoinOrder[len(g.partyJoinOrder)-1] != 31 {
		t.Fatalf(
			"commit state=%d inventory=%v members=%v order=%v job=%#v",
			g.st.NativeEventState[12], trigger.Inventory,
			g.partyMembers, g.partyJoinOrder, g.nativeFieldEvent61,
		)
	}
	if _, ok := g.partyRoster[31]; !ok {
		t.Fatal("JOIN31 did not persist the materialized Wold record")
	}
	if g.battleEvent == nil || len(g.dialog) != 1 {
		t.Fatalf("FDTXT4 did not begin: run=%#v dialog=%#v", g.battleEvent, g.dialog)
	}
	for g.battleEvent != nil {
		g.dialog = nil
		g.advanceBattleEvent()
	}
	if g.sel != nil || !trigger.Acted {
		t.Fatal("final FDTXT4 did not return to the completed wait action")
	}
}

func TestNativeSystemGroupMarchPausesForEvent61PresentationAndResumes(t *testing.T) {
	g, trigger := nativeEvent61PlayerGame(t, 0xD0, 0x20)
	trigger.X, trigger.Y = 2, 46
	trigger.NativeMapPresentation.X = 2
	trigger.NativeMapPresentation.Y = 46
	step := battle.NativeSystemGroupMarchStep{
		UnitIndex: 0,
		Path:      []battle.Cell{{X: 2, Y: 46}, {X: 1, Y: 46}},
		Events: []battle.NativeSystemGroupMarchEvent{{
			PathIndex: 1, EventID: 61, TextIndex: 3, Presentation: true,
		}},
	}
	plan := battle.NativeSystemGroupMarchPlan{
		Destination: battle.Cell{X: 1, Y: 46}, Steps: []battle.NativeSystemGroupMarchStep{step},
	}
	if !g.preflightNativeSystemGroupMarchEvents(plan) {
		t.Fatal("event61 presentation assets failed group-march preflight")
	}
	g.nativeSystemGroupMarch = &plan
	g.startNextNativeSystemGroupMarchStep()
	for tick := 0; tick < 7; tick++ {
		g.stepBattleWalk()
	}
	if g.walk == nil || !g.walk.nativeGroupMarchPaused || g.battleEvent == nil ||
		g.nativeFieldEvent61 != nil || g.st.NativeEventState[12] != 0 {
		t.Fatalf("event61 did not pause at text3: walk=%#v event=%v job=%#v state=%d", g.walk, g.battleEvent != nil, g.nativeFieldEvent61, g.st.NativeEventState[12])
	}
	g.dialog = nil
	g.advanceBattleEvent()
	if g.nativeFieldEvent61 == nil || len(g.nativeFieldEvent61.frames) != 59 {
		t.Fatalf("event61 presentation did not start: job=%#v err=%q", g.nativeFieldEvent61, g.loadErr)
	}
	for frame := 0; frame < 59; frame++ {
		g.nativeFieldEvent61.drawn = true
		if !g.nativeFieldEvent61.hasTick {
			g.stepNativeFieldEvent61Tick(100)
		}
		g.stepNativeFieldEvent61Tick(102 + frame*2)
	}
	for g.battleEvent != nil {
		g.dialog = nil
		g.advanceBattleEvent()
	}
	if g.walk == nil || g.walk.nativeGroupMarchPaused || g.st.NativeEventState[12] != 1 ||
		!g.partyMembers[31] || !reflect.DeepEqual(trigger.Inventory, []int{0x20}) {
		t.Fatalf("event61 did not resume after commit: walk=%#v state=%d members=%v inventory=%v err=%q", g.walk, g.st.NativeEventState[12], g.partyMembers, trigger.Inventory, g.loadErr)
	}
	g.stepBattleWalk()
	if g.walk != nil || g.nativeSystemGroupMarch != nil || !trigger.Acted ||
		trigger.NativeRecordByte5&0x80 == 0 {
		t.Fatalf("group march did not finish after event61: walk=%#v plan=%#v trigger=%+v", g.walk, g.nativeSystemGroupMarch, trigger)
	}
}

func TestNativeEvent61ProductionOwnersRejectMissingSeparatedBank(t *testing.T) {
	g, trigger := nativeEvent61PlayerGame(t, 0xD0, 0x20)
	t.Setenv("FD2_ASSET_PACK", t.TempDir())
	step := battle.NativeSystemGroupMarchStep{
		UnitIndex: 0,
		Events: []battle.NativeSystemGroupMarchEvent{{
			PathIndex: 0, EventID: 61, TextIndex: 3, Presentation: true,
		}},
	}
	if g.preflightNativeSystemGroupMarchEvents(battle.NativeSystemGroupMarchPlan{
		Steps: []battle.NativeSystemGroupMarchStep{step},
	}) {
		t.Fatal("group-march preflight accepted a missing event61 separated bank")
	}
	plan, err := battle.PlanNativeFieldEvent61(g.st, trigger, trigger.X, trigger.Y)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.beginNativeFieldEvent61Presentation(plan, nil); err == nil {
		t.Fatal("event61 presentation accepted a missing separated bank")
	}
	if g.nativeFieldEvent61 != nil || g.st.NativeEventState[12] != 0 ||
		!reflect.DeepEqual(trigger.Inventory, []int{0xD0, 0x20}) {
		t.Fatal("missing event61 bank partially published the presentation or mutation")
	}
}
