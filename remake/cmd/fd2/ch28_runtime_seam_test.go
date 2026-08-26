package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func seedLateCampaignParty(t *testing.T, scenarioPath string, selected int) *Game {
	t.Helper()
	scenario, err := battle.LoadScenario(assetPath(scenarioPath))
	if err != nil {
		t.Fatal(err)
	}
	if selected <= 0 || len(scenario.Party) < selected {
		t.Fatalf("%s party=%d, want at least %d", scenarioPath, len(scenario.Party), selected)
	}
	order := make([]int, 0, len(scenario.Party))
	for _, unit := range scenario.Party {
		order = append(order, unit.Fig)
	}
	g := &Game{
		partyMembers:   make(map[int]bool, len(order)),
		partyJoinOrder: append([]int(nil), order...),
		partyDeploy:    make(map[int]bool, selected-1),
		partyRoster:    make(map[int]battle.Unit, len(order)),
	}
	for _, unit := range scenario.PartyUnits(nil) {
		g.partyMembers[unit.Fig] = true
		g.partyRoster[unit.Fig] = *unit
	}
	for _, id := range order[1:selected] {
		g.partyDeploy[id] = true
	}
	return g
}

func newChapter29RuntimeBattle(t *testing.T) *Game {
	t.Helper()
	g := seedLateCampaignParty(t, "assets/scenarios/ch29.json", 20)
	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	full.Start = "story_ch29"
	g.camp = campaign.NewRunner(full)
	g.enterNode()
	if g.loadErr != "" {
		t.Fatalf("story_ch29 entry: %s", g.loadErr)
	}
	if err := g.fastForwardShotCampaign(); err != nil {
		t.Fatalf("story_ch29 normal handler path: %v", err)
	}
	if g.loadErr != "" || g.camp.NodeID() != "battle_ch29" || g.st == nil || g.sc == nil {
		t.Fatalf("story_ch29 boundary node=%q state=%v scenario=%v err=%q", g.camp.NodeID(), g.st != nil, g.sc != nil, g.loadErr)
	}
	// Normal play draws at least one steady tactical frame before the result
	// seam; explicitly invoke that production compositor in this headless test.
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatalf("ch29 steady map frame: %v", err)
	}
	return g
}

// newChapter28RuntimeBattle loads the authored map/unit/scenario files.  It
// deliberately does not fabricate a party or call Runner.Advance: the test
// below must enter the postbattle handler through the production result
// confirmation seam.
func newChapter28RuntimeBattle(t *testing.T) *Game {
	t.Helper()
	scenario, err := battle.LoadScenario(assetPath("assets/scenarios/ch28.json"))
	if err != nil {
		t.Fatal(err)
	}
	// ch28.json is the authored roster; the map's twenty deployment cells are
	// the only direct-start party slice.  Seed the persistent records through
	// the same LOADCH boundary helper used by the production campaign, rather
	// than constructing a roster from guessed stats.
	order := make([]int, 0, len(scenario.Party))
	for _, unit := range scenario.Party {
		order = append(order, unit.Fig)
	}
	if len(order) < 20 {
		t.Fatalf("ch28 authored party=%d, want at least 20 deployment records", len(order))
	}
	g := seedLateCampaignParty(t, "assets/scenarios/ch28.json", 20)
	if err := g.loadMap(assetPath("assets/maps/map27")); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map27/map27_units.json", "assets/scenarios/ch28.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil {
		t.Fatalf("ch28 formal runtime setup err=%q state=%v scenario=%v", g.loadErr, g.st != nil, g.sc != nil)
	}
	deployed := append([]int{order[0]}, order[1:20]...)
	loaded := make([]*battle.Unit, 0, len(deployed))
	for _, id := range deployed {
		for _, unit := range g.st.Units {
			if unit != nil && unit.Fig == id && unit.Camp == battle.Own {
				loaded = append(loaded, unit)
				break
			}
		}
	}
	if len(loaded) != len(deployed) {
		t.Fatalf("ch28 runtime authored party records=%d, want %d", len(loaded), len(deployed))
	}
	if err := g.seedPersistentPartyFromLoadCH(deployed, loaded); err != nil {
		t.Fatalf("ch28 production party seed: %v", err)
	}
	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	full.Start = "battle_ch28"
	g.camp = campaign.NewRunner(full)
	return g
}

func TestLatePreHandlersReachTheirPlayerNumberedBattles(t *testing.T) {
	if os.Getenv("FD2_ORIGINAL_FDOTHER") == "" {
		t.Skip("late pre-handler indexed transitions require the read-only original FDOTHER/FDSHAP/FDICON bundle")
	}
	for _, tc := range []struct {
		name, start, wantBattle, scenario string
		wantRawRoster                     int
	}{
		{
			name: "player battle 28 uses raw index 27", start: "story_ch28", wantBattle: "battle_ch28",
			scenario: "assets/scenarios/ch28.json", wantRawRoster: 60,
		},
		{
			name: "player battle 29 uses raw index 28", start: "story_ch29", wantBattle: "battle_ch29",
			scenario: "assets/scenarios/ch29.json", wantRawRoster: 76,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := seedLateCampaignParty(t, tc.scenario, 20)
			full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
			if err != nil {
				t.Fatal(err)
			}
			full.Start = tc.start
			g.camp = campaign.NewRunner(full)
			g.enterNode()
			if g.loadErr != "" {
				t.Fatalf("%s handler entry: %s", tc.start, g.loadErr)
			}
			if len(g.storyRoster) != tc.wantRawRoster {
				t.Fatalf("%s raw LOADCH roster=%d want %d", tc.start, len(g.storyRoster), tc.wantRawRoster)
			}
			if err := g.fastForwardShotCampaign(); err != nil {
				t.Fatal(err)
			}
			if g.loadErr != "" || g.camp.NodeID() != tc.wantBattle || g.st == nil {
				t.Fatalf("%s boundary node=%q state=%v err=%q", tc.start, g.camp.NodeID(), g.st != nil, g.loadErr)
			}
			if tc.start == "story_ch29" {
				if g.sc == nil || !g.sc.RuntimeAppendGroups || len(g.st.Units) != 76 || len(g.st.Roster) != 20 {
					t.Fatalf("ch29 adopted topology runtime=%v units=%d roster=%d", g.sc != nil && g.sc.RuntimeAppendGroups, len(g.st.Units), len(g.st.Roster))
				}
				for index, unit := range g.st.Units {
					if unit == nil {
						t.Fatalf("ch29 runtime slot%d is nil", index)
					}
					if index < 20 {
						if unit.Camp != battle.Own {
							t.Fatalf("ch29 persistent slot%d camp=%v, want own", index, unit.Camp)
						}
						continue
					}
					if unit.Group != 8 {
						t.Fatalf("ch29 runtime slot%d group=%d, want group8", index, unit.Group)
					}
				}
				for _, unit := range g.st.Units {
					if unit != nil && (unit.Group == 1 || unit.Group == 2 || unit.Group == 3 ||
						(unit.Group >= 4 && unit.Group <= 7) || unit.Group == 9) {
						t.Fatalf("ch29 source-only/dynamic group%d was materialized at handoff", unit.Group)
					}
				}
			}
		})
	}
}

func TestChapter27PreReactivatesOnlyNonzeroHPPartyRecords(t *testing.T) {
	actors := make([]battle.Unit, 20)
	for i := range actors {
		actors[i] = battle.Unit{
			Camp: battle.Own, HP: 10, OnField: false,
			NativeRecordByte5: 1, HasNativeRecordByte5: true,
		}
	}
	actors[7].HP = 0
	g := newBeatTestGame(t, []campaign.Beat{{Op: "reactivate_nonzero_hp", Source: "0x33cea", Count: 20}})
	g.storyActors = actors
	g.beatAdvance()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	for i := range g.storyActors {
		wantActive := i != 7
		if g.storyActors[i].OnField != wantActive || (wantActive && g.storyActors[i].NativeRecordByte5 != 0) || (!wantActive && g.storyActors[i].NativeRecordByte5 != 1) {
			t.Fatalf("slot%d active/raw5=%v/%d want %v", i, g.storyActors[i].OnField, g.storyActors[i].NativeRecordByte5, wantActive)
		}
	}
}

func TestChapter27PreReactivationPreflightIsAtomic(t *testing.T) {
	actors := make([]battle.Unit, 20)
	for i := range actors {
		actors[i] = battle.Unit{Camp: battle.Own, HP: 10, NativeRecordByte5: 1, HasNativeRecordByte5: true}
	}
	actors[19].HasNativeRecordByte5 = false
	g := newBeatTestGame(t, []campaign.Beat{{Op: "reactivate_nonzero_hp", Source: "0x33cea", Count: 20}})
	g.storyActors = actors
	g.beatAdvance()
	if g.loadErr == "" {
		t.Fatal("missing raw byte+5 provenance was accepted")
	}
	for i := 0; i < 19; i++ {
		if g.storyActors[i].OnField || g.storyActors[i].NativeRecordByte5 != 1 {
			t.Fatalf("preflight failure partially mutated slot%d", i)
		}
	}
}

// TestChapter28BattleResultRunsCh27PostToPreparation29 verifies the complete
// authored ch28 result boundary.  It uses faithful mode and the production
// confirmation method, then consumes the real dialogue pages before letting
// the handler's sync_party/set_chapter beats finish.  No native renderer or
// guessed runtime frontier is substituted because ch27_post has no such
// operation.
func TestChapter28BattleResultRunsCh27PostToPreparation29(t *testing.T) {
	g := newChapter28RuntimeBattle(t)
	if g.camp.NodeID() != "battle_ch28" {
		t.Fatalf("formal battle cursor=%q, want battle_ch28", g.camp.NodeID())
	}

	// The authored scenario is the runtime input.  Marking the already loaded
	// battle result is the same seam used after the real enemy phase; it does
	// not alter units, party membership, or handler data.
	g.result = "win"
	if !g.confirmBattleResult() || g.result != "" {
		t.Fatalf("battle result confirmation failed: node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
	if g.camp.NodeID() != "postbattle_ch28_persist" {
		t.Fatalf("ch28 result entered %q, want postbattle_ch28_persist", g.camp.NodeID())
	}
	if g.loadErr != "" || len(g.beats) != 3 {
		t.Fatalf("ch27_post admission beats=%d err=%q", len(g.beats), g.loadErr)
	}

	// Consume the actual ch28 dialogue through dlgAdvance, including any page
	// scroll delay, rather than clearing the dialog queue as a test shortcut.
	for steps := 0; len(g.dialog) > 0 && steps < 100; steps++ {
		// Update() decrements this timer once per frame.  Keep the same
		// production timer semantics here; tick() intentionally covers only
		// beat jobs and does not own the dialogue lifecycle.
		for g.dlgScrollT > 0 {
			g.dlgScrollT--
		}
		if g.loadErr != "" {
			t.Fatalf("ch27_post dialogue setup failed: %s", g.loadErr)
		}
		g.dlgAdvance()
	}
	if len(g.dialog) != 0 {
		t.Fatalf("ch27_post dialogue did not finish: remaining=%d page=%d scroll=%d", len(g.dialog), g.dlgPage, g.dlgScrollT)
	}
	// The dialog beat remains the active beat until the same callback used by
	// the normal input path advances it.  This consumes sync_party and then
	// set_chapter without bypassing either authored operation.
	g.beatAdvance()
	if g.loadErr != "" {
		t.Fatalf("ch27_post sync/set-chapter failed: beat=%d err=%q", g.beatIdx, g.loadErr)
	}

	// The handler's final beat starts the normal story fade.  Advance it by
	// ticking the real fade state; the callback enters the editable preparation
	// boundary and clears transient battle state.
	for ticks := 0; g.camp.NodeID() == "postbattle_ch28_persist" && ticks < 120; ticks++ {
		g.tick(1)
	}
	if g.loadErr != "" {
		t.Fatalf("ch27_post transition failed: %s", g.loadErr)
	}
	if g.camp.NodeID() != "preparation_ch29" || g.handlerChapter != 28 {
		t.Fatalf("ch28 postbattle boundary node=%q chapter=%d beat=%d/%d fade=%v", g.camp.NodeID(), g.handlerChapter, g.beatIdx, len(g.beats), g.fade != nil)
	}
	if g.st != nil {
		t.Fatal("preparation_ch29 retained transient battle state")
	}
	if len(g.partyRoster) == 0 {
		t.Fatal("ch27_post sync_party did not publish any authored party record")
	}
	for id, unit := range g.partyRoster {
		if unit.Camp != battle.Own {
			t.Fatalf("persistent roster id=%d contains non-player camp=%v", id, unit.Camp)
		}
	}
}

func TestChapter29BattleResultColdLoadsPreparation30AndFeedsFinalEnding(t *testing.T) {
	fdotherPath := os.Getenv("FD2_ORIGINAL_FDOTHER")
	if fdotherPath == "" {
		t.Skip("ch28 post indexed presenter requires the read-only original FDOTHER/FDSHAP/FDICON bundle")
	}
	// 地圖 renderer 與 ending loader 歷史上使用不同環境鍵；兩者都指向
	// 同一個玩家唯讀 archive，避免測試環境碰巧預設其中一個鍵才通過。
	t.Setenv("FD2_FDOTHER", fdotherPath)
	t.Setenv("FD2_ANI", filepath.Join(filepath.Dir(fdotherPath), "ANI.DAT"))
	t.Setenv("FD2_MUTE", "1")
	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Cleanup(func() { userDataDirCached = oldCache })

	g := newChapter29RuntimeBattle(t)
	if len(g.st.Units) != 76 || g.camp.NodeID() != "battle_ch29" {
		t.Fatalf("ch29 start node=%q slots=%d", g.camp.NodeID(), len(g.st.Units))
	}
	wantView := battle.NativeMapViewState{
		CameraX: 9, CameraY: 56, CursorX: 15, CursorY: 63,
		VisibleCursorX: 6, VisibleCursorY: 7,
	}
	if !g.st.HasNativeMapViewState || g.st.NativeMapViewState != wantView ||
		!g.st.HasNativeMapRangeModeState || g.st.NativeMapRangeMode != 0 ||
		!g.st.HasNativeMapHUDState || g.st.NativeMapHUDState.DisplayGateB != 1 {
		t.Fatalf("ch29 native presentation view=%+v range=%d hud=%+v flags=%v/%v/%v",
			g.st.NativeMapViewState, g.st.NativeMapRangeMode, g.st.NativeMapHUDState,
			g.st.HasNativeMapViewState, g.st.HasNativeMapRangeModeState, g.st.HasNativeMapHUDState)
	}
	g.result = "win"
	if !g.confirmBattleResult() || g.result != "" || g.camp.NodeID() != "postbattle_ch29_persist" {
		t.Fatalf("ch29 result confirmation node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
	if g.loadErr != "" || g.approximatePostbattle || len(g.beats) == 0 {
		t.Fatalf("ch28 post admission beats=%d approximate=%v err=%q", len(g.beats), g.approximatePostbattle, g.loadErr)
	}
	err := g.fastForwardShotCampaign()
	if err == nil || !strings.Contains(err.Error(), "reached non-battle node=\"preparation_ch30\"") {
		t.Fatalf("ch28 post fast-forward boundary err=%v node=%q", err, g.camp.NodeID())
	}
	if g.loadErr != "" || g.camp.NodeID() != "preparation_ch30" || g.st != nil || g.handlerChapter != 29 {
		t.Fatalf("ch29 post boundary node=%q chapter=%d battle=%v err=%q", g.camp.NodeID(), g.handlerChapter, g.st != nil, g.loadErr)
	}
	if len(g.partyRoster) == 0 {
		t.Fatal("ch28 post sync_party did not publish persistent records")
	}
	wantRoster := clonePartyRoster(g.partyRoster)
	wantOrder := append([]int(nil), g.partyJoinOrder...)
	g.saveGameToSlot(0)
	if !strings.Contains(g.msg, "preparation_ch30") {
		t.Fatalf("preparation_ch30 save was not created: %q", g.msg)
	}
	// Cold-load through a new Game and campaign runner. Reusing the old Game
	// would let un-serialized runtime fields accidentally satisfy the final
	// preparation or ending admission contract.
	coldCampaign, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	g = &Game{camp: campaign.NewRunner(coldCampaign)}
	g.loadGameFromSlot(0)
	if g.loadErr != "" || g.camp.NodeID() != "preparation_ch30" || g.st != nil ||
		g.handlerChapter != 29 || !reflect.DeepEqual(g.partyRoster, wantRoster) ||
		!reflect.DeepEqual(g.partyJoinOrder, wantOrder) {
		t.Fatalf("preparation_ch30 save/load mismatch: node=%q chapter=%d roster=%#v order=%v battle=%v err=%q msg=%q",
			g.camp.NodeID(), g.handlerChapter, g.partyRoster, g.partyJoinOrder, g.st != nil, g.loadErr, g.msg)
	}
	if g.acceptTownDeparturePrompt() || !g.prepSelecting {
		t.Fatal("preparation_ch30 skipped its 19-member selection pass")
	}
	for i := 0; i < g.prepLimit; i++ {
		if g.prepSel != i {
			t.Fatalf("preparation_ch30 cursor=%d before selection %d; Return did not follow the original auto-advance order", g.prepSel, i)
		}
		if !g.togglePreparationSelection() {
			t.Fatalf("preparation_ch30 could not select roster index %d", i)
		}
		if i+1 < g.prepLimit && g.prepSel != i+1 {
			t.Fatalf("preparation_ch30 cursor=%d after selection %d, want %d", g.prepSel, i, i+1)
		}
	}
	if g.prepSelecting || !g.prepConfirm || g.preparationSelected() != 19 {
		t.Fatalf("preparation_ch30 selection state selecting=%v confirm=%v selected=%d", g.prepSelecting, g.prepConfirm, g.preparationSelected())
	}
	if got := g.camp.Advance("confirm"); got != "story_ch30" {
		t.Fatalf("preparation_ch30 confirm=%q, want story_ch30", got)
	}
	g.enterNode()
	if g.loadErr != "" || g.camp.NodeID() != "story_ch30" || len(g.beats) == 0 {
		t.Fatalf("story_ch30 entry node=%q beats=%d err=%q", g.camp.NodeID(), len(g.beats), g.loadErr)
	}
	if len(g.storyActors) != 27 {
		t.Fatalf("story_ch30 LOADCH runtime=%d, want selected leader+19 and seven group0 records", len(g.storyActors))
	}
	if err := g.fastForwardShotCampaign(); err != nil {
		t.Fatalf("story_ch30→battle_ch30: %v", err)
	}
	if g.loadErr != "" || g.camp.NodeID() != "battle_ch30" || g.st == nil || g.sc == nil {
		t.Fatalf("battle_ch30 boundary node=%q state=%v scenario=%v err=%q", g.camp.NodeID(), g.st != nil, g.sc != nil, g.loadErr)
	}
	deployed := 0
	campCounts := make(map[battle.Camp]int)
	groupCounts := make(map[int]int)
	var progressed *battle.Unit
	progressedRosterID := -1
	for _, unit := range g.st.Units {
		if unit == nil {
			continue
		}
		campCounts[unit.Camp]++
		groupCounts[unit.Group]++
		if unit.Camp != battle.Own {
			continue
		}
		deployed++
		if progressed != nil {
			continue
		}
		rawIdentity, ok := unit.NativeIdentity, unit.HasNativeIdentity
		if !ok && unit.HasNativeRecordByte8 {
			rawIdentity, ok = int(unit.NativeRecordByte8), true
		}
		if !ok {
			continue
		}
		for id, roster := range g.partyRoster {
			rosterIdentity, rosterOK := roster.NativeIdentity, roster.HasNativeIdentity
			if !rosterOK && roster.HasNativeRecordByte8 {
				rosterIdentity, rosterOK = int(roster.NativeRecordByte8), true
			}
			if rosterOK && rosterIdentity == rawIdentity {
				progressed, progressedRosterID = unit, id
				break
			}
		}
	}
	if deployed != 20 || progressed == nil || progressedRosterID < 0 {
		t.Fatalf("battle_ch30 cold-load deployment=%d progressed=%v rosterID=%d camps=%v groups=%v partyDeploy=%v",
			deployed, progressed != nil, progressedRosterID, campCounts, groupCounts, g.partyDeploy)
	}
	progressed.Lv++
	progressed.Exp = 77.5
	wantFinalLevel := progressed.Lv
	for _, unit := range g.st.Units {
		if unit != nil && unit.Camp == battle.Enemy {
			unit.HP, unit.OnField = 0, false
		}
	}
	// 只縮短敵軍全滅條件；回合結束仍走玩家可見的空游標操作面板、END、
	// YES 回覆與完整索引畫面生命週期，不可用 endTurn helper 繞過介面。
	// 上方冷讀刻意以最小 Game 驗證序列化邊界；正常程式啟動則會先由
	// loadMap 建立原生資產包。此處透過同一正式 loader 補回該啟動前置，
	// 不直接注入 HUD、selector 或畫面 buffer。
	if err := g.loadMap(assetPath("assets/maps/map29")); err != nil {
		t.Fatalf("battle_ch30 native map assets: %v", err)
	}
	if !nativeMapAssetsAvailable(g.nativeMapAssets) {
		t.Fatal("battle_ch30 native map asset bundle unavailable")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatalf("battle_ch30 steady native frame: %v (cursor=%d,%d map=%v hud=%v cycle=%v selector=%v range=%v/%d)",
			err, g.curX, g.curY, g.m != nil, g.st.HasNativeMapHUDState,
			g.st.HasNativeMapCycleState, g.st.NativeMapSelectorCache != nil,
			g.st.HasNativeMapRangeModeState, g.st.NativeMapRangeMode)
	}
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Fatalf("battle_ch30 END preparation assets: %v", err)
	}
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatalf("battle_ch30 END class assets: %v", err)
	}
	g.ring, g.ringSel, g.nativeSystemCursorOverlay = true, 3, true
	if !g.beginNativeSystemEndTurn() {
		t.Fatal("battle_ch30 player END route was rejected")
	}
	for present := 0; present < 4; present++ {
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	for steps := 0; g.nativeClassUIJob != nil; steps++ {
		if steps >= 256 {
			t.Fatal("battle_ch30 END opening did not settle within 256 frames")
		}
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	g.confirmNativeSystemEndTurn()
	choiceJob := g.nativeClassUIJob
	for steps := 0; g.nativeClassUIJob == choiceJob; steps++ {
		if steps >= 256 {
			t.Fatal("battle_ch30 END confirmation did not publish its response within 256 frames")
		}
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	for steps := 0; g.nativeClassUIJob != nil; steps++ {
		if steps >= 256 {
			t.Fatal("battle_ch30 END response did not settle within 256 frames")
		}
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	for g.nativeSystemEndTurnDelay > 0 {
		g.stepNativeSystemEndTurn()
	}
	for steps := 0; g.nativeClassUIJob != nil; steps++ {
		if steps >= 256 {
			t.Fatal("battle_ch30 ENEMY PHASE banner did not settle within 256 frames")
		}
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if !g.aiBusy {
		t.Fatal("battle_ch30 END→YES did not enter the enemy phase")
	}
	g.aiStep()
	if g.aiBusy || g.result != "win" || g.camp.NodeID() != "battle_ch30" {
		t.Fatalf("battle_ch30 result seam busy=%v result=%q node=%q err=%q", g.aiBusy, g.result, g.camp.NodeID(), g.loadErr)
	}
	if !g.confirmBattleResult() || g.loadErr != "" || g.result != "" ||
		g.camp.NodeID() != "ending" || g.nativeEnding == nil || !g.nativeEnding.campaignSourceBound {
		t.Fatalf("battle_ch30→ending node=%q result=%q preview=%v err=%q notice=%q", g.camp.NodeID(), g.result, g.nativeEnding != nil, g.loadErr, g.endingNotice)
	}
	if got := g.partyRoster[progressedRosterID]; got.Lv != wantFinalLevel || got.Exp != 77.5 {
		t.Fatalf("final battle progress did not reach persistent ending roster: id=%d unit=%#v", progressedRosterID, got)
	}
	for _, elapsed := range []int{0, 1000, 2500, 0, 256, 2000} {
		if _, err := g.nativeEnding.player.Advance(elapsed); err != nil {
			t.Fatal(err)
		}
	}
	if !g.nativeEnding.player.ResumeBlockedDialogue() {
		t.Fatal("final ending first native dialogue gate was unavailable")
	}
	if _, err := g.nativeEnding.player.Advance(5000); err != nil || !g.nativeEnding.player.ResumeBlockedDialogue() {
		t.Fatalf("final ending second native dialogue gate: %v", err)
	}
	if _, err := g.nativeEnding.player.Advance(7500); err != nil || !g.nativeEnding.atNativeMontageGate() {
		t.Fatalf("final ending montage gate err=%v preview=%#v", err, g.nativeEnding)
	}
	if err := g.startCampaignNativeMontage(); err != nil {
		t.Fatal(err)
	}
	gotMontage := g.nativeEnding.montage
	if gotMontage == nil || len(gotMontage.Units) != len(g.partyJoinOrder) {
		gotRecords := 0
		if gotMontage != nil {
			gotRecords = len(gotMontage.Units)
		}
		t.Fatalf("cold-loaded ending montage records=%d join order=%d", gotRecords, len(g.partyJoinOrder))
	}
}
