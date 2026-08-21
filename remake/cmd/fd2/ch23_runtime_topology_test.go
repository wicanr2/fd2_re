package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func newChapter23RuntimeBattle(t *testing.T) (*Game, []int) {
	t.Helper()
	order := []int{0, 9, 4, 30, 1, 8, 2, 10, 13, 12, 5, 6, 11, 14, 17, 15}
	g := &Game{
		partyMembers:   make(map[int]bool, len(order)),
		partyJoinOrder: append([]int(nil), order...),
		partyDeploy:    make(map[int]bool, len(order)-1),
	}
	for _, id := range order {
		g.partyMembers[id] = true
	}
	for _, id := range order[1:] {
		g.partyDeploy[id] = true
	}
	if err := g.loadMap("assets/maps/map22"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map22/map22_units.json", "assets/scenarios/ch23.json")
	return g, order
}

func TestChapter23PersistentPartyPrecedesAllAuthoredMapGroups(t *testing.T) {
	g, _ := newChapter23RuntimeBattle(t)
	if g.loadErr != "" || g.st == nil || g.sc == nil || !g.sc.RuntimeAppendGroups {
		t.Fatalf("chapter23 setup err=%q state=%v scenario=%v", g.loadErr, g.st != nil, g.sc != nil && g.sc.RuntimeAppendGroups)
	}
	if got := len(g.st.Units); got != 86 {
		t.Fatalf("chapter23 runtime frontier=%d, want persistent16+map70=86", got)
	}
	for slot, unit := range g.st.Units[:16] {
		if unit == nil || unit.Camp != battle.Own {
			t.Fatalf("chapter23 persistent slot%d=%#v", slot, unit)
		}
	}
	for slot := 16; slot <= 17; slot++ {
		unit := g.st.Units[slot]
		if unit == nil || unit.Group != 0 {
			t.Fatalf("chapter23 raw group0 slot%d=%#v", slot, unit)
		}
	}
	wantGroups := map[int]int{0: 2, 1: 24, 2: 6, 3: 6, 4: 6, 5: 6, 6: 6, 7: 6, 8: 4, 9: 4}
	gotGroups := make(map[int]int, len(wantGroups))
	for _, unit := range g.st.Units[16:] {
		if unit != nil {
			gotGroups[unit.Group]++
		}
	}
	for group, want := range wantGroups {
		if got := gotGroups[group]; got != want {
			t.Fatalf("chapter23 raw group%d count=%d, want %d (all=%v)", group, got, want, gotGroups)
		}
	}
	if len(g.st.PendingGroups) != 0 {
		t.Fatalf("chapter23 current authored approximation left pending groups: %v", g.st.PendingGroups)
	}
}

func TestChapter23BattleResultRunsBoundPostbattleAndReachesPreparation24SaveBoundary(t *testing.T) {
	base := filepath.Join("..", "..", "..", "org_game", "炎龍騎士團", "FLAME2")
	for _, archive := range []string{"FDFIELD.DAT", "FDSHAP.DAT", "FDOTHER.DAT", "FDICON.DAT"} {
		if _, err := os.Stat(filepath.Join(base, archive)); err != nil {
			t.Skipf("player-provided original %s is absent: %v", archive, err)
		}
	}
	t.Setenv("FD2_ORIGINAL_FDFIELD", filepath.Join(base, "FDFIELD.DAT"))
	t.Setenv("FD2_ORIGINAL_FDSHAP", filepath.Join(base, "FDSHAP.DAT"))
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDICON", filepath.Join(base, "FDICON.DAT"))
	g, order := newChapter23RuntimeBattle(t)
	if g.loadErr != "" || g.st == nil || len(g.st.Units) != 86 {
		t.Fatalf("chapter23 production setup err=%q units=%d", g.loadErr, len(g.st.Units))
	}
	if err := g.seedPersistentPartyFromLoadCH(order, g.st.Units[:len(order)]); err != nil {
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
		t.Fatal("chapter23 map has no empty cursor cell")
	}
	g.curX, g.curY = emptyX, emptyY
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CursorX: emptyX, CursorY: emptyY, VisibleCursorX: emptyX, VisibleCursorY: emptyY,
	}); err != nil || !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatalf("chapter23 native view setup err=%v", err)
	}
	if _, ok := g.nativeMapHUDInput(); !ok {
		t.Fatal("chapter23 native map input unavailable for production handler")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	full.Start = "battle_ch23"
	g.camp = campaign.NewRunner(full)
	g.result = "win"
	if !g.confirmBattleResult() || g.result != "" || g.camp.NodeID() != "postbattle_ch23_persist" {
		t.Fatalf("chapter23 result handoff node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
	maxSlots := len(g.st.Units)
	for frame := 0; frame < 50000 && g.camp.NodeID() != "preparation_ch24"; frame++ {
		if g.transitionReveal != nil {
			g.transitionReveal.drawn = true
			g.transitionReveal.ticks = 0
			g.stepTransitionReveal()
		}
		if g.native2189A != nil {
			g.native2189A.drawn = true
			g.stepNative2189A()
		}
		if g.indexedTransition != nil {
			g.indexedTransition.drawn = true
			g.stepNativeIndexedTransition()
		}
		if g.nativePaletteRamp != nil {
			g.nativePaletteRamp.drawn = true
			g.stepNativePaletteRamp()
		}
		if len(g.dialog) != 0 {
			g.dialog = nil
			g.beatAdvance()
		}
		g.tick(1)
		if g.st != nil && len(g.st.Units) > maxSlots {
			maxSlots = len(g.st.Units)
		}
		if g.loadErr != "" {
			t.Fatalf("ch22_post stopped at beat %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
		}
	}
	if g.camp.NodeID() != "preparation_ch24" || g.handlerChapter != 23 || g.st != nil || maxSlots != 86 {
		t.Fatalf("chapter23 post boundary node=%q chapter=%d state=%v maxSlots=%d", g.camp.NodeID(), g.handlerChapter, g.st != nil, maxSlots)
	}
	if len(g.partyRoster) != len(order) {
		t.Fatalf("chapter23 synced roster=%d, want %d", len(g.partyRoster), len(order))
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	g.saveGameToSlot(1)
	if g.msg != "已存檔(槽位2：preparation_ch24)" {
		t.Fatalf("preparation24 save message=%q", g.msg)
	}
	g.camp.Cur = "postbattle_ch23_persist"
	g.partyMembers, g.partyJoinOrder, g.partyRoster = nil, nil, nil
	g.loadGameFromSlot(1)
	if g.camp.NodeID() != "preparation_ch24" || len(g.partyRoster) != len(order) {
		t.Fatalf("preparation24 save/load node=%q roster=%d order=%v", g.camp.NodeID(), len(g.partyRoster), g.partyJoinOrder)
	}
}
