package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func TestNativeContinueBattlePublicationFromRealCurrentSnapshot(t *testing.T) {
	const (
		unitsPath    = "assets/maps/map0/map0_units.json"
		scenarioPath = "assets/scenarios/ch01.json"
	)
	savePath := filepath.Join(
		"../../../org_game/炎龍騎士團/FLAME2", "FD2.SAV",
	)
	stored, err := os.ReadFile(savePath)
	if err != nil {
		t.Skipf("player-provided FD2.SAV is absent: %v", err)
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil {
		t.Fatal(err)
	}
	assetState, err := battle.Load(assetPath(unitsPath))
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := battle.LoadScenario(assetPath(scenarioPath))
	if err != nil {
		t.Fatal(err)
	}
	input, err := fdsave.BuildContinueRuntimeInput(
		snapshot,
		fdsave.ContinueRuntimeContext{
			Chapter:            int(snapshot.Header.Chapter),
			FieldWidth:         assetState.W,
			FieldHeight:        assetState.H,
			SelectorGroupCount: 0x100,
			// The signed BIOS tick is deliberately supplied by the caller;
			// this deterministic contract test uses a valid zero seed and does
			// not claim title-clock or pixel-timing E2.
			TitleTimerTick: 0, HasTitleTimerTick: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	state := &battle.State{
		W: assetState.W, H: assetState.H,
		OwnDeploy: append([]battle.Cell(nil), assetState.OwnDeploy...),
		Cost:      append([]int(nil), assetState.Cost...),
		NativeCompositionEventBytes: append(
			[]byte(nil), assetState.NativeCompositionEventBytes...,
		),
		NativeMapEventGrid:    append([]byte(nil), assetState.NativeMapEventGrid...),
		HasNativeMapEventGrid: assetState.HasNativeMapEventGrid,
		NativeFieldEventSlots: append(
			[]int(nil), assetState.NativeFieldEventSlots...,
		),
		NativeFieldEvents: append(
			[]battle.NativeFieldEvent(nil), assetState.NativeFieldEvents...,
		),
		NativeFieldEventRules: append(
			[]battle.NativeFieldEventRule(nil), assetState.NativeFieldEventRules...,
		),
		NativeTileBlitModes:  append([]byte(nil), assetState.NativeTileBlitModes...),
		NativeTerrainControl: append([]byte(nil), assetState.NativeTerrainControl...),
		NativeTerrainMoveCodes: append(
			[]byte(nil), assetState.NativeTerrainMoveCodes...,
		),
	}
	if err := campaign.MaterializeNativeContinueFieldBoundary(
		state, input, int(snapshot.Header.Chapter),
	); err != nil {
		t.Fatal(err)
	}
	catalog, err := campaign.LoadNativeCharacterCatalog(
		assetPath("assets/data/native_character_catalog.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinueRuntimeUnits(
		state, input, catalog,
	); err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinueMapTiming(state, input); err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinuePendingGroups(
		state, input, int(snapshot.Header.Chapter), assetState, scenario, itemRows,
	); err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinueInteractiveBoundary(state, input); err != nil {
		t.Fatal(err)
	}

	graph := &campaign.Campaign{
		Start: "battle_ch01",
		Nodes: map[string]*campaign.Node{
			"battle_ch01": {
				Type: "battle", Units: unitsPath, Scenario: scenarioPath,
			},
		},
	}
	g := &Game{
		camp: campaign.NewRunner(graph),
		m: &MapData{
			W: assetState.W, H: assetState.H, TileW: 24, TileH: 24,
		},
		dialog: []battle.DialogLine{{Text: "stale"}},
	}
	if err := g.publishNativeContinueBattle(
		input, state, scenario, "battle_ch01", unitsPath, scenarioPath,
	); err != nil {
		t.Fatal(err)
	}
	if g.camp.NodeID() != "battle_ch01" || g.st != state || g.sc != scenario ||
		g.titlePhase != "" || len(g.dialog) != 0 || g.sel != nil ||
		g.curX != int(snapshot.Header.CursorX) ||
		g.curY != int(snapshot.Header.CursorY) {
		t.Fatalf("published CONTINUE state node=%q st=%p sc=%p title=%q cursor=(%d,%d)",
			g.camp.NodeID(), g.st, g.sc, g.titlePhase, g.curX, g.curY)
	}
}

func TestNativeContinueTitleCallerPublishesRealCurrentSnapshot(t *testing.T) {
	savePath := filepath.Join(
		"../../../org_game/炎龍騎士團/FLAME2", "FD2.SAV",
	)
	if _, err := os.Stat(savePath); err != nil {
		t.Skipf("player-provided FD2.SAV is absent: %v", err)
	}
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_NATIVE_TITLE_TICK", "0")
	g := &Game{
		camp:       campaign.NewRunner(graph),
		titlePhase: "menu",
	}
	if err := g.loadNativeContinueFromCurrentSnapshot(savePath); err != nil {
		t.Fatal(err)
	}
	if g.camp == nil || g.camp.NodeID() != "battle_ch01" ||
		g.st == nil || g.sc == nil || g.titlePhase != "" ||
		g.st.NativeRoundCounter != 1 || g.curX != 8 || g.curY != 17 ||
		g.st.NativeMapViewState.CameraX != 1 || g.st.NativeMapViewState.CameraY != 13 {
		t.Fatalf(
			"title CONTINUE publication node=%v title=%q state=%p scenario=%p round=%d cursor=(%d,%d) view=%+v",
			g.camp.NodeID(), g.titlePhase, g.st, g.sc, g.st.NativeRoundCounter,
			g.curX, g.curY, g.st.NativeMapViewState,
		)
	}
}

func TestNativeContinueTitleTimerSeedFailsClosed(t *testing.T) {
	t.Setenv("FD2_NATIVE_TITLE_TICK", "")
	if _, err := nativeContinueTitleTimerSeed(); err == nil {
		t.Fatal("missing signed BIOS tick was accepted")
	}
	t.Setenv("FD2_NATIVE_TITLE_TICK", "32768")
	if _, err := nativeContinueTitleTimerSeed(); err == nil {
		t.Fatal("out-of-range signed BIOS tick was accepted")
	}
}

func TestNativeContinueOpeningConfirmUsesExactSavedCursorOnce(t *testing.T) {
	g := &Game{
		st: &battle.State{
			W: 24, H: 24,
			HasNativeMapViewState: true,
			NativeMapViewState:    battle.NativeMapViewState{CursorX: 8, CursorY: 17},
		},
		curX: 8, curY: 17,
		nativeContinueOpeningConfirm: true,
	}
	g.confirm()
	if g.sel != nil || !g.nativeContinueCursorOverlay || !g.ring ||
		g.actionOverlayPhase != actionOverlayOpening || g.ringSel != 0 ||
		g.reach != nil || g.nativeContinueOpeningConfirm {
		t.Fatalf("first native CONTINUE confirm did not open action overlay: %+v", g)
	}

	// 已消費的旗標不會把第二次確認或其他狀態再導向 native overlay。
	g.resetActionOverlayLifecycle()
	g.moved, g.reach = false, nil
	g.confirm()
	if g.ring || g.sel != nil || g.moved || g.nativeContinueOpeningConfirm || g.nativeContinueCursorOverlay {
		t.Fatalf("consumed native CONTINUE bridge leaked into normal selection: %+v", g)
	}
}

func TestNativeContinueOpeningConfirmRejectsMovedCursor(t *testing.T) {
	unit := &battle.Unit{Camp: battle.Own, OnField: true, HP: 10, X: 7, Y: 17}
	g := &Game{
		st: &battle.State{
			W: 24, H: 24, Units: []*battle.Unit{unit},
			HasNativeMapViewState: true,
			NativeMapViewState:    battle.NativeMapViewState{CursorX: 8, CursorY: 17},
		},
		curX: 7, curY: 17,
		nativeContinueOpeningConfirm: true,
	}
	g.confirm()
	if g.ring || g.sel != unit || g.moved || g.nativeContinueOpeningConfirm || g.nativeContinueCursorOverlay {
		t.Fatalf("moved cursor must stay on normal selection path: %+v", g)
	}
}

func TestNativeContinueDownEndYesEntersAndCompletesEnemyPhase(t *testing.T) {
	state := &battle.State{W: 24, H: 24, Units: []*battle.Unit{
		{Camp: battle.Own, OnField: true, HP: 10, MaxHP: 10},
		{Camp: battle.Enemy, OnField: false, HP: 0, MaxHP: 10},
	}}
	g := &Game{
		st: state, sc: &battle.Scenario{}, ring: true, ringSel: 3,
		nativeContinueCursorOverlay: true,
	}
	if !g.beginNativeContinueEndTurn() {
		t.Fatal("verified Down→END route was rejected")
	}
	if g.aiBusy || g.nativeContinueEndTurnConfirm {
		t.Fatal("enemy phase or confirmation started before four close presents")
	}
	for present := 0; present < 4; present++ {
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	if !g.nativeContinueEndTurnConfirm || g.nativeContinueCursorOverlay || g.ring {
		t.Fatalf("END confirmation handoff = confirm:%v overlay:%v ring:%v",
			g.nativeContinueEndTurnConfirm, g.nativeContinueCursorOverlay, g.ring)
	}
	if g.msg == "" {
		t.Fatal("END confirmation has no visible player prompt")
	}

	g.confirmNativeContinueEndTurn()
	if g.aiBusy || g.nativeContinueEndTurnDelay != nativeContinueEndTurnDelayFrames ||
		g.msg != "好的，\n就結束本回合的行動吧！" {
		t.Fatalf("YES feedback boundary: ai=%v delay=%d msg=%q",
			g.aiBusy, g.nativeContinueEndTurnDelay, g.msg)
	}
	for frame := 1; frame < nativeContinueEndTurnDelayFrames; frame++ {
		g.stepNativeContinueEndTurn()
		if g.aiBusy {
			t.Fatalf("enemy phase started before 0xC8 ms boundary at frame %d", frame)
		}
	}
	g.stepNativeContinueEndTurn()
	if !g.aiBusy || g.banner != "ENEMY PHASE" || g.nativeContinueEndTurnDelay != 0 {
		t.Fatalf("delayed YES did not enter enemy phase: ai=%v banner=%q delay=%d",
			g.aiBusy, g.banner, g.nativeContinueEndTurnDelay)
	}
	g.aiStep()
	if g.aiBusy || state.Turn != 1 || g.banner != "PLAYER PHASE" {
		t.Fatalf("enemy phase did not return to player: ai=%v turn=%d banner=%q",
			g.aiBusy, state.Turn, g.banner)
	}
}

func TestNativeContinueEndTurnRemainsNarrowAndCancelable(t *testing.T) {
	state := &battle.State{W: 24, H: 24}
	for direction := 0; direction < 3; direction++ {
		g := &Game{st: state, ring: true, ringSel: direction, nativeContinueCursorOverlay: true}
		if g.beginNativeContinueEndTurn() {
			t.Fatalf("unverified direction %d opened END confirmation", direction)
		}
	}

	g := &Game{st: state, nativeContinueEndTurnConfirm: true}
	g.cancelNativeContinueEndTurn()
	if g.nativeContinueEndTurnConfirm || g.aiBusy || state.Turn != 0 {
		t.Fatalf("cancel mutated turn: confirm=%v ai=%v turn=%d",
			g.nativeContinueEndTurnConfirm, g.aiBusy, state.Turn)
	}
}
