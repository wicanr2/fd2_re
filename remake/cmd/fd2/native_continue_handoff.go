package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// nativeContinueBattleSource is resolved from the editable campaign graph,
// not from a guessed chapter-to-map formula.  A current-runtime save is only
// accepted when exactly one battle node owns the matching scenario chapter.
type nativeContinueBattleSource struct {
	nodeID       string
	unitsPath    string
	scenarioPath string
	nodeMap      string
}

func (g *Game) resolveNativeContinueBattleSource(chapter int) (nativeContinueBattleSource, error) {
	if g == nil || g.camp == nil || g.camp.C == nil {
		return nativeContinueBattleSource{}, errors.New("原版續戰：戰役圖尚未載入")
	}
	if chapter < 0 || chapter >= 30 {
		return nativeContinueBattleSource{}, fmt.Errorf("原版續戰：章節索引 %d 超出原版範圍", chapter)
	}
	var matches []nativeContinueBattleSource
	for nodeID, node := range g.camp.C.Nodes {
		if node == nil || node.Type != "battle" || node.Units == "" ||
			node.Scenario == "" || node.Map == "" {
			continue
		}
		scenario, err := battle.LoadScenario(assetPath(node.Scenario))
		if err != nil || scenario.Chapter != chapter+1 {
			continue
		}
		matches = append(matches, nativeContinueBattleSource{
			nodeID: nodeID, unitsPath: node.Units, scenarioPath: node.Scenario,
			nodeMap: node.Map,
		})
	}
	if len(matches) != 1 {
		return nativeContinueBattleSource{}, fmt.Errorf(
			"原版續戰：章節 %d 對應戰鬥節點數=%d，拒絕猜測",
			chapter, len(matches),
		)
	}
	return matches[0], nil
}

func nativeContinueTitleTimerSeed() (int, error) {
	raw := os.Getenv("FD2_NATIVE_TITLE_TICK")
	if raw == "" {
		return 0, errors.New(
			"原版續戰：缺少 FD2_NATIVE_TITLE_TICK（必須由標題呼叫端提供 signed BIOS tick）",
		)
	}
	tick, err := strconv.Atoi(raw)
	if err != nil || tick < -0x8000 || tick > 0x7fff {
		return 0, errors.New("原版續戰：FD2_NATIVE_TITLE_TICK 不在 signed 16-bit 範圍")
	}
	return tick, nil
}

// loadNativeContinueFromCurrentSnapshot owns the title CONTINUE application
// caller.  It resolves an exact editable battle source, builds every typed
// adapter on private state, and publishes only after the complete handoff is
// valid.  Missing save, timer, assets, or an ambiguous chapter mapping keeps
// the title active and never mutates the current battle.
func (g *Game) loadNativeContinueFromCurrentSnapshot(path string) error {
	if path == "" {
		return errors.New("原版續戰：未提供 FD2_NATIVE_SAVE")
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("原版續戰：讀取 FD2.SAV：%w", err)
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		return fmt.Errorf("原版續戰：解碼 FD2.SAV：%w", err)
	}
	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil {
		return fmt.Errorf("原版續戰：current-runtime 快照：%w", err)
	}
	timer, err := nativeContinueTitleTimerSeed()
	if err != nil {
		return err
	}
	source, err := g.resolveNativeContinueBattleSource(int(snapshot.Header.Chapter))
	if err != nil {
		return err
	}

	// All loading and adapter writes below happen on a private Game/state.  The
	// live title remains untouched until publishNativeContinueBattle succeeds.
	candidate := *g
	if err := candidate.loadMap(source.nodeMap); err != nil {
		return fmt.Errorf("原版續戰：地圖資產：%w", err)
	}
	state, err := battle.Load(assetPath(source.unitsPath))
	if err != nil {
		return fmt.Errorf("原版續戰：單位資產：%w", err)
	}
	// pending-group validation compares the untouched authored FDFIELD
	// topology with the progressively materialized CONTINUE state.  Keep a
	// separate load so adapter writes cannot make that proof self-referential.
	assetState, err := battle.Load(assetPath(source.unitsPath))
	if err != nil {
		return fmt.Errorf("原版續戰：單位資產基準：%w", err)
	}
	scenario, err := battle.LoadScenario(assetPath(source.scenarioPath))
	if err != nil {
		return fmt.Errorf("原版續戰：劇本資產：%w", err)
	}
	if scenario.Chapter != int(snapshot.Header.Chapter)+1 ||
		scenario.Map < 0 || state.W != candidate.m.W || state.H != candidate.m.H ||
		assetState.W != state.W || assetState.H != state.H {
		return errors.New("原版續戰：戰場資產與 current-runtime 章節不一致")
	}
	if err := candidate.bindNativeFutureItemRows(state); err != nil {
		return fmt.Errorf("原版續戰：future item rows：%w", err)
	}
	if err := candidate.bindNativeMovementCostRows(state); err != nil {
		return fmt.Errorf("原版續戰：movement rows：%w", err)
	}
	candidate.bindCommandLearn(state)
	candidate.bindNativeCommandBook(state)
	candidate.bindNativeCommandResistances(state)
	input, err := fdsave.BuildContinueRuntimeInput(
		snapshot,
		fdsave.ContinueRuntimeContext{
			Chapter:            int(snapshot.Header.Chapter),
			FieldWidth:         state.W,
			FieldHeight:        state.H,
			SelectorGroupCount: 0x100,
			TitleTimerTick:     timer,
			HasTitleTimerTick:  true,
		},
	)
	if err != nil {
		return fmt.Errorf("原版續戰：typed input：%w", err)
	}
	catalog, err := campaign.LoadNativeCharacterCatalog(
		assetPath("assets/data/native_character_catalog.json"),
	)
	if err != nil {
		return fmt.Errorf("原版續戰：角色目錄：%w", err)
	}
	if err := campaign.MaterializeNativeContinueFieldBoundary(
		state, input, int(snapshot.Header.Chapter),
	); err != nil {
		return err
	}
	if err := campaign.MaterializeNativeContinueRuntimeUnits(state, input, catalog); err != nil {
		return err
	}
	if err := campaign.MaterializeNativeContinueMapTiming(state, input); err != nil {
		return err
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	)
	if err != nil {
		return fmt.Errorf("原版續戰：item rows：%w", err)
	}
	if err := campaign.MaterializeNativeContinuePendingGroups(
		state, input, int(snapshot.Header.Chapter), assetState, scenario, itemRows,
	); err != nil {
		return err
	}
	if err := campaign.MaterializeNativeContinueInteractiveBoundary(state, input); err != nil {
		return err
	}
	if err := candidate.publishNativeContinueBattle(
		input, state, scenario, source.nodeID, source.unitsPath, source.scenarioPath,
	); err != nil {
		return err
	}
	candidate.nativeCurrentSavePlain = append([]byte(nil), plain...)
	*g = candidate
	return nil
}

// publishNativeContinueBattle is the application-owned half of the native
// CONTINUE boundary.  It consumes a state that has already passed all typed
// save/resource adapters and publishes it to Game in one shallow transaction;
// it never calls resetBattle or Scenario.Setup, so saved turn/runtime order is
// not replaced by an authored battle opening.
//
// The title caller is intentionally separate: it must provide the signed BIOS
// timer seed and resolve the exact campaign node.  Missing either input keeps
// title CONTINUE fail-closed; this E1 path does not claim BIOS timing parity.
func (g *Game) publishNativeContinueBattle(
	input fdsave.ContinueRuntimeInput,
	state *battle.State,
	scenario *battle.Scenario,
	nodeID, unitsPath, scenarioPath string,
) error {
	if g == nil {
		return fmt.Errorf("native CONTINUE battle handoff: nil game")
	}
	if err := campaign.ValidateNativeContinueBattleHandoff(
		input, state, scenario,
	); err != nil {
		return err
	}
	options := fdother.NativeSystemOptions{
		Raw53AF9: input.Header.Raw53AF9, Raw51AAB: input.Header.HUDGateA,
		Raw51E61: input.Header.Raw51E61, Raw51E62: input.Header.Raw51E62,
	}
	if err := options.Validate(); err != nil {
		return fmt.Errorf("native CONTINUE battle handoff: system options: %w", err)
	}
	if g.m == nil || g.m.W != state.W || g.m.H != state.H {
		return fmt.Errorf("native CONTINUE battle handoff: map asset is unavailable")
	}
	if g.camp == nil || g.camp.C == nil {
		return fmt.Errorf("native CONTINUE battle handoff: campaign graph is unavailable")
	}
	node := g.camp.C.Nodes[nodeID]
	if node == nil || node.Type != "battle" {
		return fmt.Errorf("native CONTINUE battle handoff: node %q is not a battle", nodeID)
	}
	if unitsPath == "" || scenarioPath == "" ||
		node.Units != unitsPath || node.Scenario != scenarioPath {
		return fmt.Errorf("native CONTINUE battle handoff: campaign asset identity mismatch")
	}

	runner := *g.camp
	runner.Flags = make(map[string]bool, len(g.camp.Flags))
	for key, value := range g.camp.Flags {
		runner.Flags[key] = value
	}
	runner.Cur = nodeID

	candidate := *g
	candidate.camp = &runner
	candidate.st = state
	candidate.sc = scenario
	candidate.titlePhase = ""
	candidate.storyBG = false
	candidate.storyActors = nil
	candidate.storyRoster = nil
	candidate.storyCompositionEventBytes = nil
	candidate.storySpawned = nil
	candidate.storyRosterPath = ""
	candidate.storyPartyScenario = ""
	candidate.storyNativeMapView = battle.NativeMapViewState{}
	candidate.hasStoryNativeMapView = false
	candidate.dialog = nil
	candidate.dlgShown, candidate.dlgPhase, candidate.dlgT = dlgNone, 0, 0
	candidate.dlgUpper = nil
	candidate.dlgPage, candidate.dlgScrollT, candidate.dlgScrollFrom = 0, 0, 0
	candidate.beats, candidate.beatIdx, candidate.beatDelay = nil, -1, 0
	candidate.camPan, candidate.focusJob, candidate.actJob = nil, nil, nil
	candidate.fade, candidate.transitionReveal = nil, nil
	candidate.indexedTransition, candidate.spawnIntroTransition = nil, nil
	candidate.nativeTurnStaging = nil
	candidate.nativeFieldEvent61 = nil
	candidate.nativeAIIdleRecovery = nil
	candidate.battleEvent, candidate.battleEventDelay = nil, 0
	candidate.atk, candidate.walk = nil, nil
	candidate.sel, candidate.reach = nil, nil
	candidate.moved, candidate.result = false, ""
	candidate.msg, candidate.loadErr = "", ""
	candidate.nativeChapterRestore = nil
	candidate.nativeSystemOptions = &options
	candidate.gold = int(input.Header.Currency)
	candidate.restoreNativeMapHUDGateA(options.Raw51AAB)
	candidate.nativeMapWork, candidate.nativeMapVGA = nil, nil
	candidate.nativeMapClock.Reset()
	candidate.resetActionOverlayLifecycle()
	// 原版 E2 current-runtime 畫面與 0x16f55 的關聯目前是強推論：游標仍在
	// 儲存位置時，第一個 Return 應直接開 action overlay。這個一次性旗標
	// 只保留該 E2 首次輸入邊界；後續空游標面板由共用 0x117E7 owner 處理。
	candidate.nativeContinueOpeningConfirm = true
	if !candidate.syncNativeMapView() {
		return fmt.Errorf("native CONTINUE battle handoff: map view publication failed")
	}
	candidate.prevCurX, candidate.prevCurY = candidate.curX, candidate.curY
	*g = candidate
	return nil
}

// consumeNativeContinueOpeningConfirm 僅辨識剛由原版 CONTINUE 發布的
// 第一個玩家確認，而且游標必須還在 save header 原樣記錄的位置。它只記錄
// 該 E2 首次輸入；共用空游標面板本身另由 0x117E7 直接指令證實。
func (g *Game) consumeNativeContinueOpeningConfirm() bool {
	if g == nil || !g.nativeContinueOpeningConfirm {
		return false
	}
	g.nativeContinueOpeningConfirm = false
	if g.st == nil || !g.st.HasNativeMapViewState {
		return false
	}
	view := g.st.NativeMapViewState
	return g.curX == view.CursorX && g.curY == view.CursorY
}
