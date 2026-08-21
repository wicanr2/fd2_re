package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func newChapter24RuntimeBattle(t *testing.T) (*Game, []int) {
	t.Helper()
	base := "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("ch24 production boundary regression requires the read-only original asset bundle")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	// Keep the exact LOADCH identity order authored by ch24.json.  This is
	// deliberately not copied from the adjacent chapter: slots 1 and 2 are
	// 亞雷斯(4), 悠妮(9) here, and the runtime rejects a reordered fixture.
	order := []int{0, 4, 9, 30, 1, 8, 2, 10, 13, 12, 5, 6, 11, 14, 17, 15, 18, 16, 21, 7, 27, 25, 28, 24, 23, 20, 22}
	g := &Game{
		partyMembers: make(map[int]bool, len(order)), partyJoinOrder: append([]int(nil), order...),
		partyDeploy: make(map[int]bool, 15),
	}
	for _, id := range order {
		g.partyMembers[id] = true
	}
	for _, id := range order[1:16] {
		g.partyDeploy[id] = true
	}
	if err := g.loadMap("assets/maps/map23"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map23/map23_units.json", "assets/scenarios/ch24.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil || len(g.st.Units) != 86 {
		t.Fatalf("ch24 setup err=%q state=%v scenario=%v units=%d", g.loadErr, g.st != nil, g.sc != nil, len(g.st.Units))
	}
	deployed := append([]int{order[0]}, order[1:16]...)
	// Raw map23 keeps its 70 FDFIELD records first; opening spawn_party appends
	// the 16 LOADCH records into the matching trailing deploy records.  Preserve
	// that proven 70+16 topology instead of treating enemy slot 0 as the leader.
	partyStart := len(g.st.Units) - len(deployed)
	if err := g.seedPersistentPartyFromLoadCH(deployed, g.st.Units[partyStart:]); err != nil {
		t.Fatal(err)
	}
	// A normal player battle has already sent this construction-order roster
	// through the native selector cache before the victory screen.  Direct test
	// setup must materialize that renderer provenance explicitly; the postbattle
	// adapter must not invent it after entry.
	if err := g.st.AppendNativeMapSelectorBatch(g.st.Units); err != nil {
		t.Fatal(err)
	}
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
		t.Fatal("ch24 map has no empty cursor cell")
	}
	g.curX, g.curY = emptyX, emptyY
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CursorX: emptyX, CursorY: emptyY, VisibleCursorX: emptyX, VisibleCursorY: emptyY,
	}); err != nil || !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatalf("ch24 native view setup err=%v", err)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	return g, deployed
}

func TestChapter24BattleResultPostbattlePreparationSaveLoadUsesProductionBoundaries(t *testing.T) {
	g, deployed := newChapter24RuntimeBattle(t)
	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	full.Start = "battle_ch24"
	g.camp = campaign.NewRunner(full)
	g.result = "win"
	if !g.confirmBattleResult() || g.result != "" {
		t.Fatalf("ch24 result confirmation failed: node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
	for step := 0; step < 2000 && g.camp.NodeID() != "preparation_ch25"; step++ {
		if len(g.dialog) != 0 {
			g.dialog = nil
			g.beatAdvance()
			continue
		}
		if g.nativeCh23Loop != nil {
			g.nativeCh23Loop.drawn = true
			g.nativeCh23Loop.waitFrames = 0
			g.stepNativeCh23LoopAt(g.nativeMapClock.last.Add(nativeBIOSTickPeriod))
			continue
		}
		g.tick(1)
		if g.loadErr != "" {
			t.Fatalf("ch23_post stopped at beat %d/%d node=%q: %s", g.beatIdx, len(g.beats), g.camp.NodeID(), g.loadErr)
		}
	}
	if g.camp.NodeID() != "preparation_ch25" || g.handlerChapter != 24 {
		t.Fatalf("ch24 post boundary node=%q chapter=%d beat=%d/%d", g.camp.NodeID(), g.handlerChapter, g.beatIdx, len(g.beats))
	}
	if g.st != nil || len(g.partyRoster) != len(deployed) {
		t.Fatalf("preparation_ch25 state=%v roster=%d, want cleared battle and %d records", g.st != nil, len(g.partyRoster), len(deployed))
	}

	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Cleanup(func() { userDataDirCached = oldCache })
	g.saveGameToSlot(0)
	if _, err := os.Stat(saveSlotPath(0)); err != nil {
		t.Fatalf("preparation_ch25 save was not written: %v", err)
	}
	wantRoster := g.partyRoster
	wantOrder := append([]int(nil), g.partyJoinOrder...)
	wantDeploy := g.partyDeploy
	g.partyRoster, g.partyJoinOrder, g.partyDeploy = nil, nil, nil
	g.st = &battle.State{}
	g.loadGameFromSlot(0)
	if g.loadErr != "" || g.camp.NodeID() != "preparation_ch25" || g.st != nil || g.handlerChapter != 24 ||
		!reflect.DeepEqual(g.partyRoster, wantRoster) || !reflect.DeepEqual(g.partyJoinOrder, wantOrder) ||
		!reflect.DeepEqual(g.partyDeploy, wantDeploy) {
		t.Fatalf("preparation_ch25 reload node=%q err=%q battle=%v chapter=%d roster=%d", g.camp.NodeID(), g.loadErr, g.st != nil, g.handlerChapter, len(g.partyRoster))
	}
}
