package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
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

func nativeSystemOverlayTestCells() []*ebiten.Image {
	cells := make([]*ebiten.Image, nativeActionOverlayCellCount)
	for _, index := range []int{12, 15, 18, 21} {
		cells[index] = ebiten.NewImage(1, 1)
	}
	return cells
}

func TestNativeContinueOpeningConfirmHandsOffToSharedSystemOverlay(t *testing.T) {
	g := &Game{
		m: &MapData{W: 24, H: 24, TileW: 24, TileH: 24},
		st: &battle.State{
			W: 24, H: 24,
			HasNativeMapViewState: true,
			NativeMapViewState:    battle.NativeMapViewState{CursorX: 8, CursorY: 17},
		},
		curX: 8, curY: 17,
		nativeContinueOpeningConfirm: true,
		nativeActionCells:            nativeSystemOverlayTestCells(),
	}
	g.confirm()
	if g.sel != nil || !g.nativeSystemCursorOverlay || !g.ring ||
		g.actionOverlayPhase != actionOverlayOpening || g.ringSel != 0 ||
		g.reach != nil || g.nativeContinueOpeningConfirm {
		t.Fatalf("first native CONTINUE confirm did not open action overlay: %+v", g)
	}

	// 一次性 CONTINUE 旗標已消費，但 0x117E7 的共用空游標 owner 仍會
	// 在下一次確認重新開啟同一面板。
	g.resetActionOverlayLifecycle()
	g.moved, g.reach = false, nil
	g.confirm()
	if !g.ring || g.sel != nil || g.moved || g.nativeContinueOpeningConfirm || !g.nativeSystemCursorOverlay {
		t.Fatalf("shared empty-cursor owner was not reusable after CONTINUE bridge: %+v", g)
	}
}

func TestNativeContinueOpeningConfirmRejectsMovedCursor(t *testing.T) {
	unit := &battle.Unit{Camp: battle.Own, OnField: true, HP: 10, X: 7, Y: 17}
	g := &Game{
		m: &MapData{W: 24, H: 24, TileW: 24, TileH: 24},
		st: &battle.State{
			W: 24, H: 24, Units: []*battle.Unit{unit},
			HasNativeMapViewState: true,
			NativeMapViewState:    battle.NativeMapViewState{CursorX: 8, CursorY: 17},
		},
		curX: 7, curY: 17,
		nativeContinueOpeningConfirm: true,
		nativeActionCells:            nativeSystemOverlayTestCells(),
	}
	g.confirm()
	if g.ring || g.sel != unit || g.moved || g.nativeContinueOpeningConfirm || g.nativeSystemCursorOverlay {
		t.Fatalf("moved cursor must stay on normal selection path: %+v", g)
	}
}

func TestNativeSystemDownEndYesEntersAndCompletesEnemyPhase(t *testing.T) {
	state := &battle.State{W: 24, H: 24, Units: []*battle.Unit{
		{Camp: battle.Own, OnField: true, HP: 10, MaxHP: 10},
		{Camp: battle.Enemy, OnField: false, HP: 0, MaxHP: 10},
	}}
	g := &Game{
		st: state, sc: &battle.Scenario{}, ring: true, ringSel: 3,
		nativeSystemCursorOverlay: true,
	}
	base := "../../../org_game/炎龍騎士團/FLAME2"
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	var err error
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Skipf("player-provided original UI assets are absent: %v", err)
	}
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeMapVGA = make([]byte, 320*200)
	if !g.beginNativeSystemEndTurn() {
		t.Fatal("verified Down→END route was rejected")
	}
	if g.aiBusy || g.nativeSystemEndTurnConfirm {
		t.Fatal("enemy phase or confirmation started before four close presents")
	}
	for present := 0; present < 4; present++ {
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	if !g.nativeSystemEndTurnConfirm || g.nativeSystemCursorOverlay || g.ring ||
		g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 10 {
		t.Fatalf("END confirmation handoff = confirm:%v overlay:%v ring:%v",
			g.nativeSystemEndTurnConfirm, g.nativeSystemCursorOverlay, g.ring)
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}

	g.confirmNativeSystemEndTurn()
	if g.aiBusy || g.nativeSystemEndTurnDelay != 0 || g.nativeClassUIJob == nil ||
		len(g.nativeClassUIJob.frames) != 4 {
		t.Fatalf("YES choice-close boundary: ai=%v delay=%d job=%#v",
			g.aiBusy, g.nativeSystemEndTurnDelay, g.nativeClassUIJob)
	}
	choiceJob := g.nativeClassUIJob
	for g.nativeClassUIJob == choiceJob {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) == 0 {
		t.Fatal("YES choice close did not start progressive response")
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.nativeSystemEndTurnDelay != nativeSystemEndTurnDelayFrames {
		t.Fatalf("YES response delay=%d", g.nativeSystemEndTurnDelay)
	}
	for frame := 1; frame < nativeSystemEndTurnDelayFrames; frame++ {
		g.stepNativeSystemEndTurn()
		if g.aiBusy {
			t.Fatalf("enemy phase started before 0xC8 ms boundary at frame %d", frame)
		}
	}
	g.stepNativeSystemEndTurn()
	if g.aiBusy || g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 5 ||
		len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatalf("dialogue close boundary: ai=%v job=%#v", g.aiBusy, g.nativeClassUIJob)
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if !g.aiBusy || g.banner != "ENEMY PHASE" || g.nativeSystemEndTurnUI != nil {
		t.Fatalf("restored YES did not enter enemy phase: ai=%v banner=%q ui=%#v",
			g.aiBusy, g.banner, g.nativeSystemEndTurnUI)
	}
	g.aiStep()
	if g.aiBusy || state.Turn != 1 || g.banner != "PLAYER PHASE" {
		t.Fatalf("enemy phase did not return to player: ai=%v turn=%d banner=%q",
			g.aiBusy, state.Turn, g.banner)
	}
}

func TestNativeSystemEndTurnRemainsNarrowAndCancelable(t *testing.T) {
	state := &battle.State{W: 24, H: 24}
	for direction := 0; direction < 3; direction++ {
		g := &Game{st: state, ring: true, ringSel: direction, nativeSystemCursorOverlay: true}
		if g.beginNativeSystemEndTurn() {
			t.Fatalf("unverified direction %d opened END confirmation", direction)
		}
	}

	g := &Game{st: state, nativeSystemEndTurnConfirm: true}
	g.cancelNativeSystemEndTurn()
	if g.nativeSystemEndTurnConfirm || g.aiBusy || state.Turn != 0 {
		t.Fatalf("cancel mutated turn: confirm=%v ai=%v turn=%d",
			g.nativeSystemEndTurnConfirm, g.aiBusy, state.Turn)
	}
}

func TestNativeSystemEndTurnFailsClosedBeforeOverlayMutation(t *testing.T) {
	g := &Game{
		st: &battle.State{W: 24, H: 24}, ring: true, ringSel: 3,
		nativeSystemCursorOverlay: true,
	}
	if g.beginNativeSystemEndTurn() || !g.ring || !g.nativeSystemCursorOverlay ||
		g.actionOverlayPhase != "" || g.nativeSystemEndTurnUI != nil {
		t.Fatalf("missing indexed assets mutated END overlay: %+v", g)
	}
}

func TestNativeSystemEndTurnNoClosesAndRestoresWithoutTurnMutation(t *testing.T) {
	state := &battle.State{W: 24, H: 24}
	g := &Game{
		st: state, sc: &battle.Scenario{}, ring: true, ringSel: 3,
		nativeSystemCursorOverlay: true,
	}
	base := "../../../org_game/炎龍騎士團/FLAME2"
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	var err error
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Skipf("player-provided original UI assets are absent: %v", err)
	}
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeMapVGA = make([]byte, 320*200)
	if !g.beginNativeSystemEndTurn() {
		t.Fatal("verified END route was rejected")
	}
	for present := 0; present < 4; present++ {
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	g.cancelNativeSystemEndTurn()
	if g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 4 {
		t.Fatalf("NO did not start four choice-close frames: %#v", g.nativeClassUIJob)
	}
	choiceJob := g.nativeClassUIJob
	for g.nativeClassUIJob == choiceJob {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) == 0 {
		t.Fatal("NO choice close did not start progressive response")
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	for g.nativeSystemEndTurnDelay > 0 {
		g.stepNativeSystemEndTurn()
	}
	if g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 5 ||
		len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatalf("NO did not start dialogue close and restore: %#v", g.nativeClassUIJob)
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.nativeSystemEndTurnUI != nil || g.aiBusy || state.Turn != 0 {
		t.Fatalf("NO mutated turn: ui=%#v ai=%v turn=%d", g.nativeSystemEndTurnUI, g.aiBusy, state.Turn)
	}
}
