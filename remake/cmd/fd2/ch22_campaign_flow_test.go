package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func newChapter22RuntimeBattle(t *testing.T) (*Game, []int) {
	t.Helper()
	// 原版記錄0固定為隊長；後續15筆是可編輯整備選擇，邊界與第20戰
	// production regression 相同，不是直接跳章或除錯隊伍。
	order := []int{0, 9, 4, 30, 1, 8, 2, 10, 13, 12, 5, 6, 11, 14, 17, 15, 18, 19, 20}
	g := &Game{
		partyMembers:   make(map[int]bool, len(order)),
		partyJoinOrder: append([]int(nil), order...),
		partyDeploy:    make(map[int]bool, 15),
	}
	for _, id := range order {
		g.partyMembers[id] = true
	}
	for _, id := range order[1:16] {
		g.partyDeploy[id] = true
	}
	if err := g.loadMap("assets/maps/map21"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map21/map21_units.json", "assets/scenarios/ch22.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil {
		t.Fatalf("ch22 setup err=%q state=%v scenario=%v", g.loadErr, g.st != nil, g.sc != nil)
	}
	if !g.sc.RuntimeAppendGroups || len(g.sc.Party) != 16 || len(g.st.Units) != 66 {
		t.Fatalf("ch22 initial frontier runtime=%v party=%d units=%d", g.sc.RuntimeAppendGroups, len(g.sc.Party), len(g.st.Units))
	}
	deployed := append([]int{order[0]}, order[1:16]...)
	if err := g.seedPersistentPartyFromLoadCH(deployed, g.st.Units[:len(deployed)]); err != nil {
		t.Fatal(err)
	}
	return g, deployed
}

func appendChapter22Groups(t *testing.T, g *Game, groups ...int) {
	t.Helper()
	want := 66
	for _, group := range groups {
		n, err := g.st.AppendGroupWithNativePlacement(group, 0)
		if err != nil || n == 0 {
			t.Fatalf("ch22 native group%d append n=%d err=%v", group, n, err)
		}
		want += n
		if len(g.st.Units) != want {
			t.Fatalf("ch22 group%d frontier=%d, want %d", group, len(g.st.Units), want)
		}
	}
}

func advanceChapter22NativeDialogue(t *testing.T, g *Game, seen map[[2]int]bool) {
	t.Helper()
	if len(g.dialog) == 0 {
		return
	}
	current := g.dialog[len(g.dialog)-1]
	if current.NativeDialogue == nil || current.Upper == nil ||
		current.NativeDialogue.SourceDAT != "FDTXT_022" ||
		current.NativeDialogue.StringIndex < 4 || current.NativeDialogue.StringIndex > 6 ||
		len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) < 5 {
		t.Fatalf("ch21_post dialog lost indexed lifecycle: %#v", current)
	}
	seen[[2]int{current.NativeDialogue.StringIndex, current.NativeDialogue.Utterance}] = true
	if g.nativeStoryDialogueAtInputWait() &&
		!g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
		t.Fatal("ch21_post formal story input was rejected")
	}
}

func TestChapter22RuntimeAppendGroupsBuildsProvenFrontiers(t *testing.T) {
	g, _ := newChapter22RuntimeBattle(t)
	if got := len(g.st.PendingGroups); got != 3 || !g.st.PendingGroups[1] || !g.st.PendingGroups[2] || !g.st.PendingGroups[3] {
		t.Fatalf("ch22 pending groups=%v, want 1/2/3", g.st.PendingGroups)
	}
	appendChapter22Groups(t, g, 1, 2, 3)
}

func TestChapter22PostbattleBindingReachesPreparation23ForMaterializedFrontiers(t *testing.T) {
	if os.Getenv("FD2_ORIGINAL_FDOTHER") == "" {
		t.Skip("ch22 indexed transition regression requires the read-only original FDOTHER/FDSHAP/FDICON bundle")
	}
	for _, groups := range [][]int{{1, 2}, {1, 2, 3}} {
		name := "groups_1_2"
		if len(groups) == 3 {
			name = "groups_1_2_3"
		}
		t.Run(name, func(t *testing.T) {
			g, deployed := newChapter22RuntimeBattle(t)
			appendChapter22Groups(t, g, groups...)
			if len(g.st.Units) != 73 && len(g.st.Units) != 79 {
				t.Fatalf("postbattle frontier=%d is outside production binding", len(g.st.Units))
			}

			// 原始 indexed transition 會消費戰場游標／視圖與合成器資產；把游標
			// 具體放在空格，避免虛構單位目標或依賴 GUI 點擊。
			emptyX, emptyY, found := 0, 0, false
			for y := 0; y < g.st.H && !found; y++ {
				for x := 0; x < g.st.W; x++ {
					if g.st.UnitAt(x, y) == nil {
						emptyX, emptyY, found = x, y, true
						break
					}
				}
			}
			if !found {
				t.Fatal("ch22 map has no empty cursor cell")
			}
			g.curX, g.curY = emptyX, emptyY
			if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
				CursorX: emptyX, CursorY: emptyY, VisibleCursorX: emptyX, VisibleCursorY: emptyY,
			}); err != nil || !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
				t.Fatalf("ch22 native view setup err=%v", err)
			}
			if _, ok := g.nativeMapHUDInput(); !ok {
				t.Fatalf("ch22 native map input unavailable assets=%v view=%v hud=%v cycle=%v cache=%v cur=(%d,%d) map=%dx%d",
					nativeMapAssetsAvailable(g.nativeMapAssets), g.st.HasNativeMapViewState, g.st.HasNativeMapHUDState,
					g.st.HasNativeMapCycleState, g.st.NativeMapSelectorCache != nil, g.curX, g.curY, g.m.W, g.m.H)
			}
			if err := g.composeNativeMapFrame(); err != nil {
				t.Fatal(err)
			}

			beats, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch21_post.json"))
			if err != nil || len(issues) != 0 || len(beats) == 0 || beats[0].Op != "runtime_context" {
				t.Fatalf("ch21_post compile err=%v issues=%#v first=%#v", err, issues, beats)
			}
			full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
			if err != nil {
				t.Fatal(err)
			}
			full.Start = "postbattle_ch22_persist"
			g.camp = campaign.NewRunner(full)
			// 由正式節點入口保存 battle.State 的原生視圖，再讓
			// runtime_context 領回；直接掛 beats 會跳過產品路徑的視圖交接。
			g.enterNode()
			if g.loadErr != "" || len(g.beats) != len(beats) {
				t.Fatalf("ch21_post formal entry err=%q beats=%d want=%d", g.loadErr, len(g.beats), len(beats))
			}
			seen := make(map[[2]int]bool, 11)
			for frame := 0; frame < 30000 && g.camp.NodeID() != "preparation_ch23"; frame++ {
				if g.nativePaletteRamp != nil {
					g.nativePaletteRamp.drawn = true
				}
				if g.indexedTransition != nil {
					g.indexedTransition.drawn = true
				}
				advanceChapter22NativeDialogue(t, g, seen)
				if err := g.Update(); err != nil {
					t.Fatal(err)
				}
				if g.loadErr != "" {
					t.Fatalf("ch21_post stopped at beat %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
				}
			}
			if g.camp.NodeID() != "preparation_ch23" || g.handlerChapter != 22 {
				op := "<out-of-range>"
				if g.beatIdx >= 0 && g.beatIdx < len(g.beats) {
					op = g.beats[g.beatIdx].Op
				}
				t.Fatalf("ch21_post boundary node=%q chapter=%d beat=%d/%d op=%s transition=%v palette=%v delay=%d dialog=%d", g.camp.NodeID(), g.handlerChapter, g.beatIdx, len(g.beats), op, g.indexedTransition != nil, g.nativePaletteRamp != nil, g.beatDelay, len(g.dialog))
			}
			if len(g.partyRoster) != len(deployed) {
				t.Fatalf("ch21_post synced roster=%d, want %d", len(g.partyRoster), len(deployed))
			}
			if len(seen) != 11 {
				t.Fatalf("ch21_post native dialogues=%d, want 11", len(seen))
			}
		})
	}
}

func TestChapter22BattleResultPreparationSaveLoadUsesProductionBoundaries(t *testing.T) {
	fdother := os.Getenv("FD2_ORIGINAL_FDOTHER")
	if fdother == "" {
		t.Skip("ch22 production boundary regression requires the read-only original FDOTHER/FDSHAP/FDICON bundle")
	}
	if _, err := os.Stat(fdother); err != nil {
		t.Skipf("original FDOTHER.DAT is unavailable: %v", err)
	}
	g, deployed := newChapter22RuntimeBattle(t)
	// The bound ch22 post handler has two verified runtime frontiers: after
	// groups 1/2 (73 slots) or after group 3 (79 slots).  Use the smaller
	// proven frontier here; inventing a different runtime length would make the
	// production handler reject the state, correctly.
	appendChapter22Groups(t, g, 1, 2)
	// The indexed transition consumes the battle cursor/view provenance.  Use
	// the same empty-cell setup as the existing ch22 binding regression rather
	// than fabricating a coordinate or allowing the handler to guess one.
	emptyX, emptyY, found := 0, 0, false
	for y := 0; y < g.st.H && !found; y++ {
		for x := 0; x < g.st.W; x++ {
			if g.st.UnitAt(x, y) == nil {
				emptyX, emptyY, found = x, y, true
				break
			}
		}
	}
	if !found {
		t.Fatal("ch22 map has no empty cursor cell")
	}
	g.curX, g.curY = emptyX, emptyY
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CursorX: emptyX, CursorY: emptyY, VisibleCursorX: emptyX, VisibleCursorY: emptyY,
	}); err != nil || !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatalf("ch22 native view setup err=%v", err)
	}
	if _, ok := g.nativeMapHUDInput(); !ok {
		t.Fatal("ch22 native map input unavailable for production handler")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	full.Start = "battle_ch22"
	g.camp = campaign.NewRunner(full)

	// The battle fixture is already materialized above.  Only the player's
	// normal result confirmation is injected; the test must not call Runner
	// Advance or the postbattle handler directly.
	g.result = "win"
	if !g.confirmBattleResult() || g.result != "" {
		t.Fatalf("production battle-result confirmation failed: node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
	seen := make(map[[2]int]bool, 11)
	for frame := 0; frame < 30000 && g.camp.NodeID() != "preparation_ch23"; frame++ {
		if g.nativePaletteRamp != nil {
			g.nativePaletteRamp.drawn = true
		}
		if g.indexedTransition != nil {
			g.indexedTransition.drawn = true
		}
		advanceChapter22NativeDialogue(t, g, seen)
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if g.loadErr != "" {
			t.Fatalf("production ch22 result flow stopped at beat %d/%d node=%q: %s", g.beatIdx, len(g.beats), g.camp.NodeID(), g.loadErr)
		}
	}
	if g.camp.NodeID() != "preparation_ch23" || g.handlerChapter != 22 {
		t.Fatalf("production ch22 result boundary node=%q chapter=%d beat=%d/%d", g.camp.NodeID(), g.handlerChapter, g.beatIdx, len(g.beats))
	}
	if len(seen) != 11 {
		t.Fatalf("production ch21_post native dialogues=%d, want 11", len(seen))
	}
	if g.st != nil || len(g.partyRoster) != len(deployed) {
		t.Fatalf("preparation boundary state=%v roster=%d, want cleared battle and %d persistent records", g.st != nil, len(g.partyRoster), len(deployed))
	}

	// Save only at the editable preparation node, then reload through the same
	// node-boundary API.  Keep the file in a per-test XDG directory so this
	// regression never touches a developer save.
	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Cleanup(func() { userDataDirCached = oldCache })
	g.saveGameToSlot(0)
	if _, err := os.Stat(saveSlotPath(0)); err != nil {
		t.Fatalf("preparation save was not written: %v", err)
	}
	wantRoster := g.partyRoster
	wantOrder := append([]int(nil), g.partyJoinOrder...)
	wantDeploy := g.partyDeploy
	g.partyRoster = nil
	g.partyJoinOrder = nil
	g.partyDeploy = nil
	g.st = &battle.State{}
	g.loadGameFromSlot(0)
	if g.loadErr != "" || g.camp.NodeID() != "preparation_ch23" {
		t.Fatalf("preparation reload failed: node=%q err=%q", g.camp.NodeID(), g.loadErr)
	}
	if g.st != nil || g.handlerChapter != 22 || !reflect.DeepEqual(g.partyRoster, wantRoster) ||
		!reflect.DeepEqual(g.partyJoinOrder, wantOrder) || !reflect.DeepEqual(g.partyDeploy, wantDeploy) {
		t.Fatalf("preparation reload state=%#v order=%v deploy=%v battle=%v chapter=%d", g.partyRoster, g.partyJoinOrder, g.partyDeploy, g.st != nil, g.handlerChapter)
	}
}

func TestChapter22PreHandlerReachesBattle23WithLoadCHView(t *testing.T) {
	if os.Getenv("FD2_ORIGINAL_FDOTHER") == "" {
		t.Skip("ch22_pre indexed transition regression requires the read-only original FDOTHER/FDSHAP/FDICON bundle")
	}
	g, _ := newChapter22RuntimeBattle(t)
	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	full.Start = "story_ch23"
	g.camp = campaign.NewRunner(full)
	g.enterNode()
	if g.loadErr != "" {
		t.Fatalf("ch22_pre entry stopped at beat %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
	}
	if !g.hasStoryNativeMapView || len(g.storyRoster) != 70 {
		t.Fatalf("ch22_pre LOADCH view=%#v has=%v roster=%d, want zero view and 70 raw records", g.storyNativeMapView, g.hasStoryNativeMapView, len(g.storyRoster))
	}
	if err := g.fastForwardShotCampaign(); err != nil {
		t.Fatal(err)
	}
	if g.loadErr != "" || g.camp.NodeID() != "battle_ch23" {
		t.Fatalf("ch22_pre boundary node=%q beat=%d/%d err=%q", g.camp.NodeID(), g.beatIdx, len(g.beats), g.loadErr)
	}
	if g.st == nil || len(g.st.Units) == 0 {
		t.Fatal("battle_ch23 was reached without a materialized battle state")
	}
}
