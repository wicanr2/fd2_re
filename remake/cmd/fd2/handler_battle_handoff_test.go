package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestChapterZeroHandlerRosterCarriesIntoBattleWithoutReplay(t *testing.T) {
	source, err := battle.Load("../../assets/maps/map0/map0_units.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := battle.LoadScenario("../../assets/scenarios/ch01.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := reorderScenarioParty(sc, []int{0, 9, 4, 30}); err != nil {
		t.Fatal(err)
	}
	sc.Setup(source)
	if len(source.Units) != 12 {
		t.Fatalf("source runtime units=%d, want party4+groups1/2", len(source.Units))
	}
	for slot := 0; slot < 4; slot++ {
		u := source.Units[slot]
		u.SetMapPlacement(u.X, u.Y-6, 2)
	}

	g := &Game{
		m:                  &MapData{W: source.W, H: source.H, TileW: 24, TileH: 24},
		storyActors:        make([]battle.Unit, len(source.Units)),
		storyRoster:        make([]battle.Unit, len(source.Roster)),
		storyRosterPath:    "assets/maps/map0/map0_units.json",
		storyPartyScenario: "assets/scenarios/ch01.json",
	}
	for i, unit := range source.Units {
		g.storyActors[i] = *unit
	}
	for i, unit := range source.Roster {
		g.storyRoster[i] = *unit
	}
	g.resetBattle(g.storyRosterPath, g.storyPartyScenario)
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if len(g.dialog) != 0 {
		t.Fatalf("on_battle_start dialogue replayed: %v", g.dialog)
	}
	if len(g.st.Units) != 12 {
		t.Fatalf("adopted runtime units=%d", len(g.st.Units))
	}
	for slot, wantY := range []int{14, 15, 16, 17} {
		u := g.st.Units[slot]
		if u.Y != wantY || !u.HasMapSelectorSlot || !u.HasNativeMapPresentation {
			t.Fatalf("slot%d=%#v want y%d with native presentation", slot, u, wantY)
		}
	}
	if !g.st.PendingGroups[3] || !g.st.PendingGroups[4] ||
		!g.st.PendingGroups[5] || !g.st.PendingGroups[6] ||
		!g.st.PendingGroups[7] {
		t.Fatalf("pending groups=%v", g.st.PendingGroups)
	}
}

func TestDirectBattleStartStillUsesDeploymentState(t *testing.T) {
	g := &Game{m: &MapData{W: 24, H: 24, TileW: 24, TileH: 24}}
	g.resetBattle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json")
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	// LOADCH 之後 FDFIELD 佈陣格 i 對到 runtime 槽 i（第六章 r3 seq 77），
	// ch01.json 的 deploy_cells 依 FDFIELD 順序 (7,20)(10,21)(8,22)(11,23)。
	for slot, wantY := range []int{20, 21, 22, 23} {
		if got := g.st.Units[slot].Y; got != wantY {
			t.Fatalf("direct slot%d y=%d want deploy y%d", slot, got, wantY)
		}
	}
	if len(g.dialog) != 1 {
		t.Fatalf("direct opening dialogue=%v", g.dialog)
	}
}
