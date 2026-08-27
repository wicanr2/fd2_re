package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func driveNativeBattleIntro(t *testing.T, g *Game) int {
	t.Helper()
	frames := 0
	for g.spawnIntroTransition != nil {
		frames++
		g.spawnIntroTransition.drawn = true
		g.stepNativeSpawnIntro()
		if frames > 12 {
			t.Fatal("native battle intro exceeded twelve presentation passes")
		}
	}
	if frames != 12 || g.actJob == nil {
		t.Fatalf("native battle intro frames=%d acting=%v, want 12 then acting", frames, g.actJob != nil)
	}
	return frames
}

func driveNativeBattleActing(t *testing.T, g *Game) {
	t.Helper()
	for ticks := 0; g.actJob != nil; ticks++ {
		if ticks > 1000 {
			t.Fatal("native battle following acting did not complete")
		}
		g.stepActJob()
	}
}

func TestNativeBattleIntroCallRequiresExactEventCallerProvenance(t *testing.T) {
	gate, eventID := 0, 1
	action := battle.Action{
		Type: "spawn_group", Groups: []int{4}, Camp: "enemy", NativeEventID: &eventID,
		NativeSpawns: []battle.NativeSpawnCall{{
			Group: 4, Via: "spawn_group_with_intro", Source: "0x342ce", RawPlacementGate: &gate,
			FollowingActing: &battle.NativeFollowingActing{Resource: 3, Source: "0x342e7"},
		}},
	}
	if call, ok, err := nativeBattleIntroCall(action); err != nil || !ok || call.Group != 4 {
		t.Fatalf("exact event1 provenance rejected: call=%#v ok=%v err=%v", call, ok, err)
	}
	action.NativeSpawns[0].Source = "0x342cf"
	if _, _, err := nativeBattleIntroCall(action); err == nil {
		t.Fatal("changed spawn caller address was accepted")
	}
	action.NativeSpawns[0].Source = "0x342ce"
	action.NativeSpawns[0].FollowingActing.Resource = 4
	if _, _, err := nativeBattleIntroCall(action); err == nil {
		t.Fatal("changed following ACTING resource was accepted")
	}
}

func TestChapter1GlobalIntroEventsPresentThenRunExactFollowingActing(t *testing.T) {
	const originalBase = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(originalBase, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)

	st, err := battle.Load(assetPath("assets/maps/map0/map0_units.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc, err := battle.LoadScenario(assetPath("assets/scenarios/ch01.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	g := &Game{st: st, sc: sc, sfxSpawnIntro: []byte{1}}
	if err := g.bindNativeFutureItemRows(st); err != nil {
		t.Fatal(err)
	}
	if err := g.loadMap("assets/maps/map0"); err != nil {
		t.Fatal(err)
	}

	// Event0 establishes the exact 14-slot frontier consumed by ACTING(3).
	st.Turn = 3
	g.finishTurn()
	if len(g.dialog) != 1 || len(st.Units) != 14 {
		t.Fatalf("turn3 frontier/dialogue units=%d dialog=%#v", len(st.Units), g.dialog)
	}
	g.dialog = nil
	g.advanceBattleEvent()
	if st.Turn != 4 || g.battleEvent != nil {
		t.Fatalf("turn3 completion turn=%d event=%#v", st.Turn, g.battleEvent)
	}
	st.NativeMapCycleState.Idle, st.NativeMapCycleState.Moving = 2, 3
	st.NativeTerrainPhaseState.Phase = 5
	st.NativeTerrainFlipState.Value = 1
	st.NativeUnitPixelShiftState.Value = 1

	// A missing editable acting resource must fail before constructor state,
	// selector cache, or the turn continuation is published.
	badScenario, err := battle.LoadScenario(assetPath("assets/scenarios/ch01.json"))
	if err != nil {
		t.Fatal(err)
	}
	badScenario.NativeActingResources = "assets/cutscenes/acting/missing.json"
	g.sc = badScenario
	beforeRoster, beforeCache := len(st.Roster), st.NativeMapSelectorCache
	g.finishTurn()
	if g.loadErr == "" || len(st.Units) != 14 || len(st.Roster) != beforeRoster ||
		st.NativeMapSelectorCache != beforeCache || st.Turn != 4 || g.battleEvent != nil {
		t.Fatalf(
			"failed intro mutated state: err=%q units=%d roster=%d/%d cache_same=%v turn=%d event=%#v",
			g.loadErr, len(st.Units), len(st.Roster), beforeRoster,
			st.NativeMapSelectorCache == beforeCache, st.Turn, g.battleEvent,
		)
	}
	g.loadErr = ""
	sc, err = battle.LoadScenario(assetPath("assets/scenarios/ch01.json"))
	if err != nil {
		t.Fatal(err)
	}
	g.sc = sc

	// Event1 constructs group4 before pass0, presents exactly 12 frames, then
	// executes the independent 0x342E7 ACTING(3) on slots14..17.
	g.finishTurn()
	if len(st.Units) != 18 || g.spawnIntroTransition == nil || g.actJob != nil || st.Turn != 4 {
		t.Fatalf("event1 start units=%d intro=%v acting=%v turn=%d", len(st.Units), g.spawnIntroTransition != nil, g.actJob != nil, st.Turn)
	}
	if st.NativeMapCycleState.Idle != 2 || st.NativeMapCycleState.Moving != 3 ||
		st.NativeTerrainPhaseState.Phase != 5 || st.NativeTerrainFlipState.Value != 1 ||
		st.NativeUnitPixelShiftState.Value != 1 {
		t.Fatalf("event1 reset battle animation phase: cycle=%#v terrain=%#v flip=%#v shift=%#v",
			st.NativeMapCycleState, st.NativeTerrainPhaseState,
			st.NativeTerrainFlipState, st.NativeUnitPixelShiftState)
	}
	driveNativeBattleIntro(t, g)
	before3 := [4][2]int{}
	for i := range before3 {
		before3[i] = [2]int{st.Units[14+i].X, st.Units[14+i].Y}
	}
	driveNativeBattleActing(t, g)
	want3 := [4][2]int{{-4, -2}, {-4, -2}, {-2, -4}, {-4, -2}}
	for i, delta := range want3 {
		u := st.Units[14+i]
		if u.X != before3[i][0]+delta[0] || u.Y != before3[i][1]+delta[1] {
			t.Fatalf("ACTING(3) slot%d=(%d,%d), start=%v delta=%v", 14+i, u.X, u.Y, before3[i], delta)
		}
	}
	if st.Turn != 5 || g.battleEvent != nil {
		t.Fatalf("event1 completion turn=%d event=%#v", st.Turn, g.battleEvent)
	}

	// Event2 repeats the same presentation boundary, then ACTING(4), and only
	// after that exposes the authored boss dialogue.
	g.finishTurn()
	if len(st.Units) != 23 || g.spawnIntroTransition == nil || st.Turn != 5 {
		t.Fatalf("event2 start units=%d intro=%v turn=%d", len(st.Units), g.spawnIntroTransition != nil, st.Turn)
	}
	driveNativeBattleIntro(t, g)
	before4 := [5][2]int{}
	for i := range before4 {
		before4[i] = [2]int{st.Units[18+i].X, st.Units[18+i].Y}
	}
	driveNativeBattleActing(t, g)
	want4 := [5][2]int{{1, -1}, {0, -1}, {1, 0}, {0, -1}, {1, 0}}
	for i, delta := range want4 {
		u := st.Units[18+i]
		if u.X != before4[i][0]+delta[0] || u.Y != before4[i][1]+delta[1] {
			t.Fatalf("ACTING(4) slot%d=(%d,%d), start=%v delta=%v", 18+i, u.X, u.Y, before4[i], delta)
		}
	}
	if len(g.dialog) != 1 || g.dialog[0].Speaker != 71 || st.Turn != 5 || g.battleEvent == nil {
		t.Fatalf("event2 following dialogue=%#v turn=%d event=%#v", g.dialog, st.Turn, g.battleEvent)
	}
	g.dialog = nil
	g.advanceBattleEvent()
	if st.Turn != 6 || g.battleEvent != nil {
		t.Fatalf("event2 completion turn=%d event=%#v", st.Turn, g.battleEvent)
	}
}

func TestChapter3Turn3BattleEventBlocksTurnUntilOriginalSequenceCompletes(t *testing.T) {
	st, err := battle.Load(assetPath("assets/maps/map2/map2_units.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc, err := battle.LoadScenario(assetPath("assets/scenarios/ch03.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	st.Turn = 3
	st.Units[0].Poisoned, st.Units[0].PoisonTurns = true, 2

	g := &Game{m: &MapData{W: 40, H: 40, TileW: 24, TileH: 24}, st: st, sc: sc}
	if err := g.bindNativeFutureItemRows(st); err != nil {
		t.Fatal(err)
	}
	g.finishTurn()
	if len(st.Units) != 27 || g.battleEvent == nil || g.camPan == nil {
		t.Fatalf("event did not execute SPAWN then block on first PAN: units=%d run=%#v pan=%#v", len(st.Units), g.battleEvent, g.camPan)
	}
	if st.Turn != 3 || st.Units[0].PoisonTurns != 2 {
		t.Fatalf("turn/status advanced before staging: turn=%d poison=%d", st.Turn, st.Units[0].PoisonTurns)
	}
	g.finishTurn() // re-entry while blocked must be a no-op
	if st.Turn != 3 || len(st.Units) != 27 {
		t.Fatalf("finishTurn re-entry duplicated event: turn=%d units=%d", st.Turn, len(st.Units))
	}

	for i := 0; i < 3; i++ {
		g.stepCamPan()
	}
	if g.camX != 72 || g.camY != 0 || g.camPan != nil || g.battleEventDelay != 48 {
		t.Fatalf("first PAN/delay = cam(%v,%v) pan=%#v delay=%d, want (72,0)/nil/48", g.camX, g.camY, g.camPan, g.battleEventDelay)
	}
	for i := 0; i < 47; i++ {
		g.stepBattleEventDelay()
	}
	if g.battleEventDelay != 1 || g.camPan != nil || st.Turn != 3 {
		t.Fatalf("800ms wait ended early: delay=%d pan=%#v turn=%d", g.battleEventDelay, g.camPan, st.Turn)
	}
	g.stepBattleEventDelay()
	if g.camPan == nil || g.camPan.toX != 72 || g.camPan.toY != 408 {
		t.Fatalf("second PAN target=%#v, want pixel (72,408)", g.camPan)
	}
	for i := 0; i < 17; i++ {
		g.stepCamPan()
	}
	if g.camX != 72 || g.camY != 408 || g.camPan != nil || g.battleEventDelay != 12 {
		t.Fatalf("second PAN/delay = cam(%v,%v) pan=%#v delay=%d, want (72,408)/nil/12", g.camX, g.camY, g.camPan, g.battleEventDelay)
	}
	for i := 0; i < 12; i++ {
		g.stepBattleEventDelay()
	}
	if len(g.dialog) != 1 || g.dialog[0].Speaker != 77 || g.dialog[0].Text != "鐵諾,你果然很耐命!怪不得頭子一定要我親自來看看....不過,你的好運也到此為止了!" {
		t.Fatalf("first authored dialogue played out of order: %#v", g.dialog)
	}
	if st.Turn != 3 || st.Units[0].PoisonTurns != 2 {
		t.Fatalf("turn/status advanced before dialogue completion: turn=%d poison=%d", st.Turn, st.Units[0].PoisonTurns)
	}

	wantSpeakers := []int{77, 2, 77, 8, 2, 8, 77}
	for i, speaker := range wantSpeakers {
		if len(g.dialog) != 1 || g.dialog[0].Speaker != speaker {
			t.Fatalf("dialogue %d speaker=%#v, want %d", i, g.dialog, speaker)
		}
		g.dialog = nil
		g.advanceBattleEvent()
	}
	if g.battleEvent != nil || st.Turn != 4 || st.Units[0].PoisonTurns != 1 {
		t.Fatalf("sequence completion = run=%#v turn=%d poison=%d, want nil/4/1", g.battleEvent, st.Turn, st.Units[0].PoisonTurns)
	}
}

func TestBattleEventNativeSpawnFailureDoesNotAdvanceTurnContinuation(t *testing.T) {
	gate, eventID := 1, 3
	st := &battle.State{Turn: 6}
	sc := &battle.Scenario{RuntimeAppendGroups: true}
	action := battle.Action{
		Type: "spawn_group", Groups: []int{6}, NativeEventID: &eventID,
		NativeSpawns: []battle.NativeSpawnCall{{
			Group: 6, Via: "spawn_group", Source: "0x34397", RawPlacementGate: &gate,
		}},
	}
	continued := false
	g := &Game{st: st, sc: sc}
	g.startBattleEvent([]battle.Action{action}, func() { continued = true })

	if g.loadErr == "" || continued || st.Turn != 6 || g.battleEvent != nil {
		t.Fatalf(
			"失敗事件仍前進：err=%q continued=%v turn=%d run=%#v",
			g.loadErr, continued, st.Turn, g.battleEvent,
		)
	}
}

func triggerChapter7Event26(t *testing.T, g *Game) {
	t.Helper()
	if g == nil || g.st == nil || len(g.st.Units) <= 27 {
		t.Fatal("chapter7 event26 runtime frontier is absent")
	}
	trigger := g.st.Units[0]
	if trigger == nil || !trigger.HasNativeRecordByte6 || trigger.NativeRecordByte6 == 0 {
		t.Fatalf("chapter7 event26 trigger=%#v", trigger)
	}
	trigger.SetMapPlacement(10, 13, 0)
	done := false
	g.walk = &walkAnim{
		u: trigger,
		path: []battle.Cell{
			{X: 10, Y: 13},
			{X: 9, Y: 13},
		},
		then: func() { done = true },
	}
	for steps := 0; g.walk != nil && steps < 8; steps++ {
		g.stepBattleWalk()
	}
	if !done || g.walk != nil || g.loadErr != "" || g.st.NativeEventState[16] != 1 {
		t.Fatalf(
			"event26 done=%v walk=%v err=%q state16=%d",
			done, g.walk != nil, g.loadErr, g.st.NativeEventState[16],
		)
	}
	for index := 9; index <= 27; index++ {
		unit := g.st.Units[index]
		if unit == nil || !unit.HasNativeRecordByte34 || unit.NativeRecordByte34&0x0f != 0 {
			t.Fatalf("event26 unit%d=%#v", index, unit)
		}
	}
}

func TestChapter7Event26RejectsWrongTriggerProvenance(t *testing.T) {
	tests := []struct {
		name        string
		triggerSlot int
		from        battle.Cell
		to          battle.Cell
	}{
		{
			name:        "raw_byte6_zero",
			triggerSlot: 9,
			from:        battle.Cell{X: 10, Y: 13},
			to:          battle.Cell{X: 9, Y: 13},
		},
		{
			name:        "outside_event_cells",
			triggerSlot: 0,
			from:        battle.Cell{X: 8, Y: 13},
			to:          battle.Cell{X: 7, Y: 13},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &Game{}
			if err := g.loadMap("assets/maps/map6"); err != nil {
				t.Fatal(err)
			}
			g.resetBattle("assets/maps/map6/map6_units.json", "assets/scenarios/ch07.json")
			if g.loadErr != "" || len(g.st.Units) != 34 {
				t.Fatalf("chapter7 setup err=%q units=%d", g.loadErr, len(g.st.Units))
			}
			trigger := g.st.Units[tc.triggerSlot]
			if trigger == nil || !trigger.HasNativeRecordByte6 {
				t.Fatalf("trigger slot%d=%#v", tc.triggerSlot, trigger)
			}
			if tc.name == "raw_byte6_zero" && trigger.NativeRecordByte6 != 0 {
				t.Fatalf("slot%d raw +6=%d, want zero fixture", tc.triggerSlot, trigger.NativeRecordByte6)
			}
			before := make([]byte, 19)
			for index := 9; index <= 27; index++ {
				unit := g.st.Units[index]
				if unit == nil || !unit.HasNativeRecordByte34 {
					t.Fatalf("slot%d lacks raw +0x34 provenance", index)
				}
				before[index-9] = unit.NativeRecordByte34
			}
			trigger.SetMapPlacement(tc.from.X, tc.from.Y, 0)
			g.walk = &walkAnim{
				u:    trigger,
				path: []battle.Cell{tc.from, tc.to},
			}
			for steps := 0; g.walk != nil && steps < 8; steps++ {
				g.stepBattleWalk()
			}
			if g.walk != nil || g.loadErr != "" || g.st.NativeEventState[16] != 0 {
				t.Fatalf("rejected event26 walk=%v err=%q state16=%d", g.walk != nil, g.loadErr, g.st.NativeEventState[16])
			}
			for index := 9; index <= 27; index++ {
				if got := g.st.Units[index].NativeRecordByte34; got != before[index-9] {
					t.Fatalf("rejected event26 mutated slot%d raw +0x34=%#x, want %#x", index, got, before[index-9])
				}
			}
		})
	}
}

func TestChapter7Event25FailsClosedWithoutFieldEvent26(t *testing.T) {
	g := &Game{}
	if err := g.loadMap("assets/maps/map6"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map6/map6_units.json", "assets/scenarios/ch07.json")
	if g.loadErr != "" || len(g.st.Units) != 34 {
		t.Fatalf("chapter7 setup err=%q units=%d", g.loadErr, len(g.st.Units))
	}
	g.st.Turn = 10
	g.finishTurn()
	if g.battleEvent != nil || len(g.st.Units) != 34 ||
		g.st.NativeEventState[16] != 0 || g.st.NativeEventState[17] != 0 || g.st.Turn != 11 {
		t.Fatalf(
			"ungated event25 run=%v units=%d state16=%d state17=%d turn=%d",
			g.battleEvent != nil, len(g.st.Units), g.st.NativeEventState[16],
			g.st.NativeEventState[17], g.st.Turn,
		)
	}
}

func TestChapter7Event25BuildsSlot43ThenCommitsState17(t *testing.T) {
	g := &Game{}
	if err := g.loadMap("assets/maps/map6"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map6/map6_units.json", "assets/scenarios/ch07.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil {
		t.Fatalf("chapter7 setup err=%q state=%v scenario=%v", g.loadErr, g.st != nil, g.sc != nil)
	}
	if !g.sc.RuntimeAppendGroups || len(g.st.Units) != 34 || !g.st.PendingGroups[2] {
		t.Fatalf("chapter7 opening units=%d pending=%v runtime_append=%v", len(g.st.Units), g.st.PendingGroups, g.sc.RuntimeAppendGroups)
	}
	triggerChapter7Event26(t, g)
	g.st.Turn = 10
	g.finishTurn()
	if len(g.st.Units) != 44 || g.camPan == nil || g.st.NativeEventState[17] != 0 {
		t.Fatalf("event25 spawn units=%d pan=%v state17=%d", len(g.st.Units), g.camPan != nil, g.st.NativeEventState[17])
	}
	if slot43 := g.st.Units[43]; slot43 == nil || slot43.Group != 2 || slot43.Camp != battle.Ally || slot43.Fig != 12 ||
		!slot43.HasNativeRecordByte5 || slot43.NativeRecordByte5&1 != 0 {
		t.Fatalf("event25 slot43=%#v", slot43)
	}
	for steps := 0; g.camPan != nil && steps < 100; steps++ {
		g.stepCamPan()
	}
	if g.camPan != nil || g.actJob == nil || g.camX != 16*24 || g.camY != 10*24 {
		t.Fatalf("event25 pan/acting pan=%v acting=%v cam=(%v,%v)", g.camPan != nil, g.actJob != nil, g.camX, g.camY)
	}
	driveNativeBattleActing(t, g)
	wantSpeakers := []int{12, -1, 12, -1, 1, 13, 0, 4, 13}
	for i, speaker := range wantSpeakers {
		if len(g.dialog) != 1 || g.dialog[0].Speaker != speaker {
			t.Fatalf("event25 dialogue %d=%#v, want speaker %d", i, g.dialog, speaker)
		}
		if g.st.NativeEventState[17] != 0 {
			t.Fatalf("event25 state17 committed before dialogue %d completed", i)
		}
		g.dialog = nil
		g.advanceBattleEvent()
	}
	if g.battleEvent != nil || g.st.NativeEventState[17] != 1 || g.st.Turn != 11 {
		t.Fatalf("event25 completion run=%v state17=%d turn=%d", g.battleEvent != nil, g.st.NativeEventState[17], g.st.Turn)
	}
}

func TestChapter7PostBranchesOnKeliRawInactiveStateThenEntersTown8(t *testing.T) {
	base := "../../../org_game/炎龍騎士團/FLAME2"
	for _, archive := range []string{"FDFIELD.DAT", "FDSHAP.DAT", "FDOTHER.DAT", "FDICON.DAT", "FDTXT.DAT", "DATO.DAT"} {
		if _, err := os.Stat(filepath.Join(base, archive)); err != nil {
			t.Skipf("player-provided original %s is absent: %v", archive, err)
		}
	}
	t.Setenv("FD2_ORIGINAL_FDFIELD", filepath.Join(base, "FDFIELD.DAT"))
	t.Setenv("FD2_ORIGINAL_FDSHAP", filepath.Join(base, "FDSHAP.DAT"))
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDICON", filepath.Join(base, "FDICON.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(base, "DATO.DAT"))
	t.Setenv("FD2_MUTE", "1")
	for _, tc := range []struct {
		name       string
		rawByte5   byte
		wantJoined bool
		wantIndex  int
		wantDialog int
	}{
		{name: "active_joins", rawByte5: 0, wantJoined: true, wantIndex: 4, wantDialog: 8},
		{name: "inactive_does_not_join", rawByte5: 1, wantJoined: false, wantIndex: 5, wantDialog: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := &Game{}
			if err := g.loadMap("assets/maps/map6"); err != nil {
				t.Fatal(err)
			}
			g.resetBattle("assets/maps/map6/map6_units.json", "assets/scenarios/ch07.json")
			if g.loadErr != "" || len(g.st.Units) != 34 {
				t.Fatalf("chapter7 setup err=%q units=%d", g.loadErr, len(g.st.Units))
			}
			triggerChapter7Event26(t, g)
			g.st.Turn = 10
			g.finishTurn()
			for steps := 0; g.camPan != nil && steps < 100; steps++ {
				g.stepCamPan()
			}
			driveNativeBattleActing(t, g)
			for g.battleEvent != nil {
				if len(g.dialog) != 0 {
					g.dialog = nil
					g.advanceBattleEvent()
					continue
				}
				g.advanceBattleEvent()
			}
			if g.st.NativeEventState[17] != 1 || len(g.st.Units) != 44 {
				t.Fatalf("event25 boundary state17=%d units=%d", g.st.NativeEventState[17], len(g.st.Units))
			}
			g.st.Units[43].NativeRecordByte5 = tc.rawByte5

			order := []int{0, 4, 9, 30, 1, 8, 2, 10, 13}
			g.partyMembers = make(map[int]bool, len(order))
			for _, id := range order {
				g.partyMembers[id] = true
			}
			g.partyJoinOrder = append([]int(nil), order...)
			if err := g.seedPersistentPartyFromLoadCH(order, g.st.Units[:len(order)]); err != nil {
				t.Fatal(err)
			}
			g.curX, g.curY = 0, 0
			if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{}); err != nil ||
				!g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
				t.Fatalf("chapter7 native view setup err=%v", err)
			}
			if err := g.composeNativeMapFrame(); err != nil {
				t.Fatal(err)
			}
			beats, issues, err := campaign.CompileHandlerBinding(
				assetPath("assets/cutscenes/bindings/ch06_post.json"),
			)
			if err != nil || len(issues) != 0 {
				t.Fatalf("ch06_post compile err=%v issues=%#v", err, issues)
			}
			c := &campaign.Campaign{
				Start: "postbattle_ch07_persist",
				Nodes: map[string]*campaign.Node{
					"postbattle_ch07_persist": {Type: "cutscene", Next: "town_ch08"},
					"town_ch08":               {Type: "town"},
				},
			}
			g.camp = campaign.NewRunner(c)
			g.beats, g.beatIdx, g.storyBG = beats, -1, true
			g.beatAdvance()
			seen := make(map[int]bool, tc.wantDialog)
			for frame := 0; frame < 30000 && g.camp.NodeID() != "town_ch08"; frame++ {
				if len(g.dialog) != 0 {
					current := g.dialog[len(g.dialog)-1]
					if current.NativeDialogue == nil || current.Upper == nil ||
						current.NativeDialogue.SourceDAT != "FDTXT_007" || current.NativeDialogue.StringIndex != tc.wantIndex ||
						len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) != 5 {
						t.Fatalf("ch06_post dialog lost indexed lifecycle: %#v", current)
					}
					seen[current.NativeDialogue.Utterance] = true
					if g.nativeStoryDialogueAtInputWait() && !g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
						t.Fatal("ch06_post formal story input was rejected")
					}
				}
				if err := g.Update(); err != nil {
					t.Fatal(err)
				}
				if g.loadErr != "" {
					t.Fatalf("ch06_post stopped at %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
				}
			}
			if g.camp.NodeID() != "town_ch08" || g.partyMembers[12] != tc.wantJoined || len(seen) != tc.wantDialog {
				t.Fatalf("ch06_post node=%q members=%v wantJoined=%v dialogs=%d/%d", g.camp.NodeID(), g.partyMembers, tc.wantJoined, len(seen), tc.wantDialog)
			}
			joined, ok := g.partyRoster[12]
			if tc.wantJoined {
				if !ok || !joined.HasNativeIdentity || joined.NativeIdentity != 12 ||
					!joined.HasNativeRecordByte5 || joined.NativeRecordByte5 != 0 {
					t.Fatalf("Keli persistent record=%#v", joined)
				}
			} else if ok {
				t.Fatalf("inactive Keli unexpectedly persisted=%#v", joined)
			}
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			userDataDirCached = ""
			g.saveGameToSlot(0)
			g.camp.Cur = "postbattle_ch07_persist"
			g.loadGameFromSlot(0)
			if g.camp.NodeID() != "town_ch08" || g.partyMembers[12] != tc.wantJoined {
				t.Fatalf("town08 load node=%q members=%v wantJoined=%v", g.camp.NodeID(), g.partyMembers, tc.wantJoined)
			}
		})
	}
}

func TestChapter8PostJoinsLornaPersistsPartyAndEntersTown9(t *testing.T) {
	base := "../../../org_game/炎龍騎士團/FLAME2"
	for _, archive := range []string{"FDFIELD.DAT", "FDSHAP.DAT", "FDOTHER.DAT", "FDICON.DAT", "FDTXT.DAT", "DATO.DAT"} {
		if _, err := os.Stat(filepath.Join(base, archive)); err != nil {
			t.Skipf("player-provided original %s is absent: %v", archive, err)
		}
	}
	t.Setenv("FD2_ORIGINAL_FDFIELD", filepath.Join(base, "FDFIELD.DAT"))
	t.Setenv("FD2_ORIGINAL_FDSHAP", filepath.Join(base, "FDSHAP.DAT"))
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDICON", filepath.Join(base, "FDICON.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(base, "DATO.DAT"))
	t.Setenv("FD2_MUTE", "1")
	for _, frontier := range []int{29, 41} {
		t.Run(fmt.Sprintf("frontier_%d", frontier), func(t *testing.T) {
			g := &Game{}
			if err := g.loadMap("assets/maps/map7"); err != nil {
				t.Fatal(err)
			}
			g.resetBattle("assets/maps/map7/map7_units.json", "assets/scenarios/ch08.json")
			if g.loadErr != "" || g.st == nil || g.sc == nil || !g.sc.RuntimeAppendGroups || len(g.st.Units) != 29 {
				t.Fatalf("chapter8 setup err=%q units=%d runtime_append=%v", g.loadErr, len(g.st.Units), g.sc != nil && g.sc.RuntimeAppendGroups)
			}
			if frontier == 41 {
				for group := 2; group <= 7; group++ {
					if n, err := g.st.AppendGroupWithNativePlacement(group, 0); err != nil || n != 2 {
						t.Fatalf("event27 group%d append=%d err=%v", group, n, err)
					}
				}
			}
			if len(g.st.Units) != frontier {
				t.Fatalf("chapter8 runtime frontier=%d, want %d", len(g.st.Units), frontier)
			}
			if slot28 := g.st.Units[28]; slot28 == nil || !slot28.HasNativeRecordByte8 || slot28.NativeRecordByte8 != 5 {
				t.Fatalf("chapter8 slot28 must retain raw JOIN5 identity: %#v", slot28)
			}
			order := make([]int, 0, 10)
			g.partyMembers = make(map[int]bool, 10)
			for slot := 0; slot < 10; slot++ {
				unit := g.st.Units[slot]
				if unit == nil || !unit.HasNativeIdentity {
					t.Fatalf("chapter8 party slot%d lacks raw identity: %#v", slot, unit)
				}
				order = append(order, unit.NativeIdentity)
				g.partyMembers[unit.NativeIdentity] = true
			}
			g.partyJoinOrder = append([]int(nil), order...)
			if err := g.seedPersistentPartyFromLoadCH(order, g.st.Units[:10]); err != nil {
				t.Fatal(err)
			}
			g.curX, g.curY = 0, 0
			if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{}); err != nil ||
				!g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
				t.Fatalf("chapter8 native view setup err=%v", err)
			}
			if err := g.composeNativeMapFrame(); err != nil {
				t.Fatal(err)
			}
			beats, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch07_post.json"))
			if err != nil || len(issues) != 0 {
				t.Fatalf("ch07_post compile err=%v issues=%#v", err, issues)
			}
			g.camp = campaign.NewRunner(&campaign.Campaign{Start: "postbattle_ch08_persist", Nodes: map[string]*campaign.Node{
				"postbattle_ch08_persist": {Type: "cutscene", Next: "town_ch09"}, "town_ch09": {Type: "town"},
			}})
			g.beats, g.beatIdx, g.storyBG = beats, -1, true
			g.beatAdvance()
			seen := make(map[int]bool, 8)
			for frame := 0; frame < 30000 && g.camp.NodeID() != "town_ch09"; frame++ {
				if len(g.dialog) != 0 {
					current := g.dialog[len(g.dialog)-1]
					if current.NativeDialogue == nil || current.Upper == nil || current.NativeDialogue.SourceDAT != "FDTXT_008" ||
						len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) != 5 {
						t.Fatalf("ch07_post dialog lost indexed lifecycle: %#v", current)
					}
					seen[current.NativeDialogue.Utterance+current.NativeDialogue.StringIndex*10] = true
					if g.nativeStoryDialogueAtInputWait() && !g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
						t.Fatal("ch07_post formal story input was rejected")
					}
				}
				if err := g.Update(); err != nil {
					t.Fatal(err)
				}
				if g.loadErr != "" {
					t.Fatalf("ch07_post stopped at %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
				}
			}
			if g.camp.NodeID() != "town_ch09" || !g.partyMembers[5] || g.handlerChapter != 8 || len(seen) != 8 {
				t.Fatalf("ch07_post node=%q members=%v chapter=%d dialogues=%d", g.camp.NodeID(), g.partyMembers, g.handlerChapter, len(seen))
			}
			if g.nativeFullDACBlack {
				t.Fatal("town_ch09 must clear terminal blackout")
			}
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			userDataDirCached = ""
			g.saveGameToSlot(0)
			g.camp.Cur = "postbattle_ch08_persist"
			g.loadGameFromSlot(0)
			if g.camp.NodeID() != "town_ch09" || !g.partyMembers[5] {
				t.Fatalf("town09 load node=%q members=%v", g.camp.NodeID(), g.partyMembers)
			}
		})
	}
}

func TestChapter10PostRunsExactPaletteAndDirectPatchBeforeTown11(t *testing.T) {
	originalBase := "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(originalBase, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	order := []int{0, 9, 4, 30, 1, 8, 2, 10, 13, 5}
	g := &Game{partyMembers: make(map[int]bool, len(order)), partyJoinOrder: append([]int(nil), order...)}
	for _, id := range order {
		g.partyMembers[id] = true
	}
	if err := g.loadMap("assets/maps/map9"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map9/map9_units.json", "assets/scenarios/ch10.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil || !g.sc.RuntimeAppendGroups || len(g.st.Units) != 52 {
		t.Fatalf("chapter10 setup err=%q units=%d runtime_append=%v", g.loadErr, len(g.st.Units), g.sc != nil && g.sc.RuntimeAppendGroups)
	}
	// Event 32 appends the eight group-1 allies at turn five. This test enters
	// the already-completed battle boundary, so materialize that proven event
	// result directly instead of replaying five unrelated combat rounds.
	g.st.AppendGroup(1)
	if len(g.st.Units) != 60 {
		t.Fatalf("chapter10 post frontier=%d, want 60 without Keli", len(g.st.Units))
	}
	if err := g.seedPersistentPartyFromLoadCH(order, g.st.Units[:len(order)]); err != nil {
		t.Fatal(err)
	}
	emptyX, emptyY, foundEmpty := 0, 0, false
	for y := 0; y < 8 && !foundEmpty; y++ {
		for x := 0; x < 13; x++ {
			if g.st.UnitAt(x, y) == nil {
				emptyX, emptyY, foundEmpty = x, y, true
				break
			}
		}
	}
	if !foundEmpty {
		t.Fatal("chapter10 opening viewport has no empty HUD cursor cell")
	}
	g.curX, g.curY = emptyX, emptyY
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CursorX: emptyX, CursorY: emptyY, VisibleCursorX: emptyX, VisibleCursorY: emptyY,
	}); err != nil ||
		!g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatalf("chapter10 native view setup err=%v", err)
	}
	if _, ok := g.nativeMapHUDInput(); !ok {
		t.Fatalf("chapter10 HUD input unavailable assets=%v view=%v hud=%v cycle=%v cache=%v cur=(%d,%d) map=%dx%d",
			nativeMapAssetsAvailable(g.nativeMapAssets), g.st.HasNativeMapViewState, g.st.HasNativeMapHUDState,
			g.st.HasNativeMapCycleState, g.st.NativeMapSelectorCache != nil, g.curX, g.curY, g.m.W, g.m.H)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	beats, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch09_post.json"))
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch09_post compile err=%v issues=%#v", err, issues)
	}
	c := &campaign.Campaign{
		Start: "postbattle_ch10_persist",
		Nodes: map[string]*campaign.Node{
			"postbattle_ch10_persist": {Type: "cutscene", Next: "town_ch11"},
			"town_ch11":               {Type: "town"},
		},
	}
	g.camp = campaign.NewRunner(c)
	g.beats, g.beatIdx, g.storyBG = beats, -1, true
	g.beatAdvance()
	patchObserved, dacRestoredObserved := false, false
	for frame := 0; frame < 10000 && g.camp.NodeID() != "town_ch11"; frame++ {
		if g.nativePaletteRamp != nil {
			g.nativePaletteRamp.drawn = true
		}
		if len(g.dialog) != 0 {
			g.dialog = nil
			g.beatAdvance()
		}
		g.tick(1)
		if g.st != nil && len(g.st.Units) > 52 && g.st.Units[0].X == 14 && g.st.Units[0].Y == 38 &&
			g.st.Units[50].NativeTransient[4] == 0 && g.st.Units[51].NativeTransient[4] == 0 &&
			g.st.Units[52].NativeRecordByte5 == 0 {
			patchObserved = true
		}
		if g.nativeMapAssets != nil && bytes.Equal(g.nativeMapDAC, g.nativeMapAssets.PaletteDAC) && patchObserved {
			dacRestoredObserved = true
		}
		if g.loadErr != "" {
			t.Fatalf("ch09_post stopped at %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
		}
	}
	if g.camp.NodeID() != "town_ch11" || !g.partyMembers[11] || !g.partyMembers[6] || g.handlerChapter != 10 {
		t.Fatalf("ch09_post node=%q members=%v chapter=%d", g.camp.NodeID(), g.partyMembers, g.handlerChapter)
	}
	if !patchObserved || !dacRestoredObserved {
		t.Fatalf("ch09_post patch/DAC observation patch=%v restored=%v", patchObserved, dacRestoredObserved)
	}
	for _, id := range []int{11, 6} {
		joined, ok := g.partyRoster[id]
		if !ok || !joined.HasNativeIdentity || joined.NativeIdentity != id || !joined.HasNativeRecordByte8 || int(joined.NativeRecordByte8) != id {
			t.Fatalf("joined %d persistent record=%#v", id, joined)
		}
	}
}

func TestChapter10PostRuntimeFrontiersTrackConditionalKailey(t *testing.T) {
	tests := []struct {
		name  string
		order []int
		want  int
	}{
		{name: "without_Kailey", order: []int{0, 9, 4, 30, 1, 8, 2, 10, 13, 5}, want: 60},
		{name: "with_Kailey", order: []int{0, 9, 4, 30, 1, 8, 2, 10, 13, 12, 5}, want: 61},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := &Game{
				partyMembers:   make(map[int]bool, len(test.order)),
				partyJoinOrder: append([]int(nil), test.order...),
			}
			for _, id := range test.order {
				g.partyMembers[id] = true
			}
			if err := g.loadMap("assets/maps/map9"); err != nil {
				t.Fatal(err)
			}
			g.resetBattle("assets/maps/map9/map9_units.json", "assets/scenarios/ch10.json")
			if g.loadErr != "" || g.st == nil || g.sc == nil || !g.sc.RuntimeAppendGroups {
				t.Fatalf("chapter10 setup err=%q runtime_append=%v", g.loadErr, g.sc != nil && g.sc.RuntimeAppendGroups)
			}
			g.st.AppendGroup(1)
			if len(g.st.Units) != test.want {
				t.Fatalf("chapter10 post frontier=%d, want %d", len(g.st.Units), test.want)
			}
		})
	}
}

func TestChapter20PreparationBuildsFixedLeaderPlusFifteenSlotFrontier(t *testing.T) {
	// Every ID below exists in the authored chapter-20 scenario. The seventeenth
	// persistent member forces 0x318AD; record zero remains fixed while the next
	// fifteen are selected and the final member stays in reserve.
	order := []int{0, 9, 4, 30, 1, 8, 2, 10, 13, 12, 5, 6, 11, 14, 17, 15, 18}
	g := &Game{
		partyMembers:   make(map[int]bool, len(order)),
		partyJoinOrder: append([]int(nil), order...),
		partyRoster:    make(map[int]battle.Unit, len(order)),
	}
	for _, id := range order {
		g.partyMembers[id] = true
		g.partyRoster[id] = battle.Unit{Fig: id}
	}
	g.setupPreparation(&campaign.Node{Type: "preparation", PartyLimit: 15})
	if len(g.prepIDs) != 16 || g.acceptTownDeparturePrompt() {
		t.Fatalf("chapter20 preparation selectable=%v selecting=%v", g.prepIDs, g.prepSelecting)
	}
	for _, id := range g.prepIDs[:15] {
		g.partyDeploy[id] = true
	}
	if got := g.battlePartyMembers(); len(got) != 16 || !got[0] || got[18] {
		t.Fatalf("chapter20 fixed+selected roster=%#v", got)
	}
	if err := g.loadMap("assets/maps/map19"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map19/map19_units.json", "assets/scenarios/ch20.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil || len(g.st.Units) != 83 || len(g.sc.Party) != 16 {
		units, party := 0, 0
		if g.st != nil {
			units = len(g.st.Units)
		}
		if g.sc != nil {
			party = len(g.sc.Party)
		}
		t.Fatalf("chapter20 post frontier err=%q units=%d party=%d", g.loadErr, units, party)
	}
	for i, id := range append([]int{0}, g.prepIDs[:15]...) {
		if g.st.Units[i] == nil || g.st.Units[i].Fig != id {
			t.Fatalf("chapter20 runtime slot%d fig=%v, want %d", i, g.st.Units[i], id)
		}
	}
}

func TestChapter20PostRoundGateControlsReinforcementAndJoinBeforeTown21(t *testing.T) {
	originalBase := "../../../org_game/炎龍騎士團/FLAME2"
	for _, archive := range []string{"FDFIELD.DAT", "FDSHAP.DAT", "FDOTHER.DAT", "FDICON.DAT", "FDTXT.DAT", "DATO.DAT"} {
		if _, err := os.Stat(filepath.Join(originalBase, archive)); err != nil {
			t.Skipf("player-provided original %s is absent: %v", archive, err)
		}
	}
	t.Setenv("FD2_ORIGINAL_FDFIELD", filepath.Join(originalBase, "FDFIELD.DAT"))
	t.Setenv("FD2_ORIGINAL_FDSHAP", filepath.Join(originalBase, "FDSHAP.DAT"))
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(originalBase, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDICON", filepath.Join(originalBase, "FDICON.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(originalBase, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(originalBase, "DATO.DAT"))
	t.Setenv("FD2_MUTE", "1")
	order := []int{0, 9, 4, 30, 1, 8, 2, 10, 13, 12, 5, 6, 11, 14, 17, 15, 18}
	for _, test := range []struct {
		name       string
		round      int
		wantJoin28 bool
		wantSlots  int
		wantDialog int
	}{
		{name: "round15_runs_optional_arm", round: 15, wantJoin28: true, wantSlots: 84, wantDialog: 29},
		{name: "round16_skips_optional_arm", round: 16, wantJoin28: false, wantSlots: 83, wantDialog: 15},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			if err := g.loadMap("assets/maps/map19"); err != nil {
				t.Fatal(err)
			}
			g.resetBattle("assets/maps/map19/map19_units.json", "assets/scenarios/ch20.json")
			if g.loadErr != "" || g.st == nil || g.sc == nil || len(g.st.Units) != 83 {
				t.Fatalf("chapter20 setup err=%q units=%d", g.loadErr, len(g.st.Units))
			}
			deployed := append([]int{order[0]}, order[1:16]...)
			if err := g.seedPersistentPartyFromLoadCH(deployed, g.st.Units[:len(deployed)]); err != nil {
				t.Fatal(err)
			}
			g.st.NativeRoundCounter = test.round
			emptyX, emptyY, foundEmpty := 0, 0, false
			for y := 0; y < g.st.H && !foundEmpty; y++ {
				for x := 0; x < g.st.W; x++ {
					if g.st.UnitAt(x, y) == nil {
						emptyX, emptyY, foundEmpty = x, y, true
						break
					}
				}
			}
			if !foundEmpty {
				t.Fatal("chapter20 map has no empty HUD cursor cell")
			}
			g.curX, g.curY = emptyX, emptyY
			if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
				CursorX: emptyX, CursorY: emptyY, VisibleCursorX: emptyX, VisibleCursorY: emptyY,
			}); err != nil || !g.st.MaterializeNativeMapHUDState(1, 1, 1) ||
				!g.st.MaterializeNativeMapRangeMode(1) {
				t.Fatalf("chapter20 native view setup err=%v", err)
			}
			if _, ok := g.nativeMapHUDInput(); !ok {
				t.Fatal("chapter20 native map input unavailable")
			}
			if err := g.composeNativeMapFrame(); err != nil {
				t.Fatal(err)
			}
			beats, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch19_post.json"))
			if err != nil || len(issues) != 0 {
				t.Fatalf("ch19_post compile err=%v issues=%#v", err, issues)
			}
			g.camp = campaign.NewRunner(&campaign.Campaign{
				Start: "postbattle_ch20_persist",
				Nodes: map[string]*campaign.Node{
					"postbattle_ch20_persist": {Type: "cutscene", Next: "town_ch21"},
					"town_ch21":               {Type: "town"},
				},
			})
			g.beats, g.beatIdx, g.storyBG = beats, -1, true
			g.beatAdvance()
			maxSlots := len(g.st.Units)
			type dialogueKey struct{ stringIndex, utterance int }
			seenDialogue := make(map[dialogueKey]bool, test.wantDialog)
			for frame := 0; frame < 80000 && g.camp.NodeID() != "town_ch21"; frame++ {
				if g.nativePaletteRamp != nil {
					g.nativePaletteRamp.drawn = true
				}
				if len(g.dialog) != 0 {
					current := g.dialog[len(g.dialog)-1]
					if current.NativeDialogue == nil || current.Upper == nil ||
						current.NativeDialogue.SourceDAT != "FDTXT_020" ||
						current.NativeDialogue.StringIndex < 11 || current.NativeDialogue.StringIndex > 16 ||
						len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) != 5 ||
						len(g.nativeDialogueProgressive) != len(current.NativeDialogue.Pages) {
						t.Fatalf("ch19_post dialog lost indexed lifecycle: %#v", current)
					}
					seenDialogue[dialogueKey{current.NativeDialogue.StringIndex, current.NativeDialogue.Utterance}] = true
					if g.nativeStoryDialogueAtInputWait() &&
						!g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
						t.Fatal("ch19_post formal story input was rejected")
					}
				}
				if err := g.Update(); err != nil {
					t.Fatalf("ch19_post Update: %v", err)
				}
				if g.st != nil && len(g.st.Units) > maxSlots {
					maxSlots = len(g.st.Units)
				}
				if g.loadErr != "" {
					t.Fatalf("ch19_post stopped at %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
				}
			}
			if g.camp.NodeID() != "town_ch21" || g.handlerChapter != 20 || !g.partyMembers[25] ||
				g.partyMembers[28] != test.wantJoin28 || maxSlots != test.wantSlots || len(seenDialogue) != test.wantDialog {
				t.Fatalf("round%d node=%q chapter=%d join25=%v join28=%v maxSlots=%d dialogues=%d",
					test.round, g.camp.NodeID(), g.handlerChapter, g.partyMembers[25], g.partyMembers[28], maxSlots, len(seenDialogue))
			}
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			userDataDirCached = ""
			g.saveGameToSlot(0)
			if g.msg != "已存檔(槽位1：town_ch21)" {
				t.Fatalf("round%d town21 save message=%q", test.round, g.msg)
			}
			g.camp.Cur = "postbattle_ch20_persist"
			g.partyMembers, g.partyJoinOrder = nil, nil
			g.partyDeploy, g.partyRoster = nil, nil
			g.loadGameFromSlot(0)
			if g.camp.NodeID() != "town_ch21" || !g.partyMembers[25] || g.partyMembers[28] != test.wantJoin28 {
				t.Fatalf("round%d town21 load node=%q members=%v order=%v", test.round, g.camp.NodeID(), g.partyMembers, g.partyJoinOrder)
			}
		})
	}
}

func TestChapter25PostMaterializesSlot70JoinsPartyAndReachesTown26SaveBoundary(t *testing.T) {
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	if _, err := os.Stat(filepath.Join(base, "FDOTHER.DAT")); err != nil {
		t.Skip("chapter25 native dialogue regression requires the read-only original asset bundle")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(base, "DATO.DAT"))
	order := []int{0, 4, 9, 30, 1, 8, 2, 10, 13, 12, 5, 6, 11, 14, 17, 15}
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
	if err := g.loadMap("assets/maps/map24"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map24/map24_units.json", "assets/scenarios/ch25.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil || !g.sc.RuntimeAppendGroups {
		t.Fatalf("chapter25 setup err=%q state=%v scenario=%v", g.loadErr, g.st != nil, g.sc != nil && g.sc.RuntimeAppendGroups)
	}
	if got := len(g.st.Units); got != 62 {
		t.Fatalf("chapter25 opening frontier=%d, want party16+group0(46)=62", got)
	}
	if err := g.seedPersistentPartyFromLoadCH(order, g.st.Units[:len(order)]); err != nil {
		t.Fatal(err)
	}

	// Original event 56 calls 0x10B4E(1) at turn 6.  Materialize that exact
	// pending group before victory so raw ch24_post enters with 70 records.
	g.st.Turn = 6
	actions := g.sc.TriggerActions(g.st, "on_turn_end", "")
	if len(actions) != 1 || actions[0].Type != "spawn_group" {
		t.Fatalf("chapter25 turn6 actions=%#v", actions)
	}
	if _, _, err := g.sc.ExecuteActionChecked(g.st, actions[0]); err != nil {
		t.Fatal(err)
	}
	if got := len(g.st.Units); got != 70 {
		t.Fatalf("chapter25 postbattle frontier=%d, want 62+group1(8)=70", got)
	}
	for _, unit := range g.st.Units {
		if unit != nil && unit.Group == 255 {
			t.Fatal("chapter25 reserved group255 was materialized into runtime")
		}
	}

	campaignData, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	runner := campaign.NewRunner(campaignData)
	runner.Cur = "battle_ch25"
	g.camp = runner
	// 此測試未繪製前一個戰鬥 frame 就直接進入勝利，因此注入正常 Draw loop 已持有的
	// indexed baseline。原版資產的像素合成另由
	// TestComposeNativeStoryDialoguePageUsesOriginalIndexedAssets 驗證；此橋接不構成
	// 未修改一般玩家路徑的 E2 證據。
	g.nativeMapVGA = make([]byte, 320*200)
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{}); err != nil {
		t.Fatal(err)
	}
	g.result = "win"
	if !g.confirmBattleResult() || g.camp.NodeID() != "postbattle_ch25_persist" || g.loadErr != "" {
		t.Fatalf("chapter25 result handoff node=%q err=%q", g.camp.NodeID(), g.loadErr)
	}
	maxSlots := len(g.st.Units)
	var dialogUpper []bool
	var nativePages []int
	for frame := 0; frame < 12000 && g.camp.NodeID() != "town_ch26"; frame++ {
		if len(g.dialog) != 0 {
			for i := len(g.dialog) - 1; i >= 0; i-- {
				if g.dialog[i].Upper == nil {
					t.Fatalf("ch24 runtime dialog %#v lost its explicit native placement", g.dialog[i])
				}
				dialogUpper = append(dialogUpper, *g.dialog[i].Upper)
				if g.dialog[i].NativeDialogue == nil || len(g.nativeDialogueFrames) != len(g.dialog[i].NativeDialogue.Pages) {
					t.Fatalf("ch24 runtime dialog lost native pages: line=%#v frames=%d", g.dialog[i], len(g.nativeDialogueFrames))
				}
				nativePages = append(nativePages, len(g.dialog[i].NativeDialogue.Pages))
			}
			current := g.dialog[len(g.dialog)-1]
			if !current.NativeDialogue.HasMotionTargetY || len(g.nativeDialogueClosing) < 5 {
				t.Fatalf("ch24 runtime dialog lacks proven closing: line=%#v frames=%d", current, len(g.nativeDialogueClosing))
			}
			g.dlgPage = len(g.nativeDialogueProgressive) - 1
			g.nativeDialogueProgress = len(g.nativeDialogueProgressive[g.dlgPage]) - 1
			if !g.beginNativeStoryDialogueClosing() {
				t.Fatal("ch24 runtime dialog refused its caller-owned closing")
			}
			for g.nativeDialogueClosingLive {
				g.stepNativeStoryDialogueProgress()
			}
		}
		g.tick(1)
		if g.st != nil && len(g.st.Units) > maxSlots {
			maxSlots = len(g.st.Units)
		}
		if g.loadErr != "" {
			t.Fatalf("ch24_post stopped at beat %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
		}
	}
	if g.camp.NodeID() != "town_ch26" || g.handlerChapter != 25 || g.st != nil || maxSlots != 71 {
		t.Fatalf("chapter25 post boundary node=%q chapter=%d state=%v maxSlots=%d", g.camp.NodeID(), g.handlerChapter, g.st != nil, maxSlots)
	}
	if !g.partyMembers[26] || !g.partyMembers[29] ||
		len(g.partyJoinOrder) != len(order)+2 ||
		g.partyJoinOrder[len(order)] != 26 || g.partyJoinOrder[len(order)+1] != 29 {
		t.Fatalf("chapter25 joins members=%v order=%v", g.partyMembers, g.partyJoinOrder)
	}
	wantUpper := []bool{
		true, false, true, false, false, true, false,
		true, false, true, false, true, false, true, false, true, true, true,
	}
	if len(dialogUpper) != len(wantUpper) {
		t.Fatalf("chapter25 post dialog placements=%v, want %v", dialogUpper, wantUpper)
	}
	for i := range wantUpper {
		if dialogUpper[i] != wantUpper[i] {
			t.Fatalf("chapter25 post dialog placement %d=%v, want %v", i, dialogUpper[i], wantUpper[i])
		}
	}
	wantPages := []int{2, 2, 2, 1, 1, 3, 1, 1, 1, 1, 1, 2, 1, 3, 1, 2, 1, 1}
	if !reflect.DeepEqual(nativePages, wantPages) {
		t.Fatalf("chapter25 post native pages=%v, want %v", nativePages, wantPages)
	}
	for _, id := range []int{26, 29} {
		joined, ok := g.partyRoster[id]
		if !ok || !joined.HasNativeIdentity || joined.NativeIdentity != id ||
			!joined.HasNativeRecordByte8 || int(joined.NativeRecordByte8) != id {
			t.Fatalf("JOIN%d persistent record=%#v", id, joined)
		}
	}

	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	g.saveGameToSlot(2)
	if g.msg != "已存檔(槽位3：town_ch26)" {
		t.Fatalf("town26 save message=%q", g.msg)
	}
	g.camp.Cur = "postbattle_ch25_persist"
	g.partyMembers, g.partyJoinOrder = nil, nil
	g.partyDeploy, g.partyRoster = nil, nil
	g.loadGameFromSlot(2)
	if g.camp.NodeID() != "town_ch26" || !g.partyMembers[26] || !g.partyMembers[29] ||
		len(g.partyJoinOrder) != len(order)+2 {
		t.Fatalf("town26 save/load node=%q members=%v order=%v roster=%v", g.camp.NodeID(), g.partyMembers, g.partyJoinOrder, g.partyRoster)
	}

	// Continue through the actual chapter-specific town/shop intermission. The
	// native table requires selection 4 plus Shift+F5 (BIOS scan 0x58); ch02's
	// Shift+F1 gate must not work here. The chord reveals selection 5 in place,
	// a separate confirmation enters variant 5, and the four-frame close returns
	// to town_ch26 with the hidden selection preserved.
	t.Run("town_ch26_secret_shop", func(t *testing.T) {
		base := filepath.Join("..", "..", "..", "org_game", "炎龍騎士團", "FLAME2")
		if _, err := os.Stat(filepath.Join(base, "FDOTHER.DAT")); err != nil {
			t.Skip("player-provided original facility resources are absent")
		}
		t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
		t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
		t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(base, "DATO.DAT"))
		shared, err := loadNativeClassUIAssets()
		if err != nil {
			t.Fatal(err)
		}
		townUI, err := loadNativeTownUIAssets()
		if err != nil {
			t.Fatal(err)
		}
		shopUI, err := loadNativeShopUIAssets(shared)
		if err != nil {
			t.Fatal(err)
		}
		g.nativeClassUI, g.nativeTownUI, g.nativeShopUI = shared, townUI, shopUI
		g.campSel = 4
		visible, ok := g.composeNativeTownFrame()
		if !ok || len(visible) != 320*200 {
			t.Fatalf("town26 visible frame=%d ok=%v", len(visible), ok)
		}
		ch02Scan, ok := nativeBIOSFunctionScan(nativeFunctionShift, 1)
		if !ok || ch02Scan != 0x54 || g.revealNativeTownSecret(ch02Scan) || g.campSel != 4 {
			t.Fatalf("town26 accepted ch02 secret gate: scan=%#x ok=%v selection=%d", ch02Scan, ok, g.campSel)
		}
		ch26Scan, ok := nativeBIOSFunctionScan(nativeFunctionShift, 5)
		if !ok || ch26Scan != 0x58 || !g.revealNativeTownSecret(ch26Scan) || g.campSel != 5 ||
			g.camp.NodeID() != "town_ch26" {
			t.Fatalf("town26 Shift+F5 reveal: scan=%#x ok=%v node=%q selection=%d", ch26Scan, ok, g.camp.NodeID(), g.campSel)
		}
		hidden, ok := g.composeNativeTownFrame()
		if !ok || bytes.Equal(visible, hidden) {
			t.Fatal("town26 hidden selection did not redraw the native town frame")
		}
		if !g.camp.ConfirmNativeTownSecret(g.campSel) {
			t.Fatal("town26 hidden selection confirmation was rejected")
		}
		g.enterNode()
		if g.camp.NodeID() != "shop_ch26_secret" || g.nativeShopVariant != 5 ||
			g.nativeShopMode != "menu" || g.nativeShopUIJob == nil ||
			len(g.nativeShopUIJob.frames) != 4 {
			t.Fatalf("town26 secret entry node=%q variant=%d mode=%q opening=%v", g.camp.NodeID(), g.nativeShopVariant, g.nativeShopMode, g.nativeShopUIJob != nil)
		}
		goods := g.camp.ShopGoods()
		if len(goods) != 3 || goods[0].ID != 195 || goods[1].ID != 207 || goods[2].ID != 40 {
			t.Fatalf("town26 secret goods=%v", goods)
		}
		for g.nativeShopUIJob != nil {
			g.nativeShopUIJob.drawn = true
			g.stepNativeShopUILifecycle(time.Time{})
		}
		if !g.beginNativeShopServiceClosing(g.leaveShop) || g.nativeShopUIJob == nil ||
			len(g.nativeShopUIJob.frames) != 4 {
			t.Fatal("town26 secret shop did not start the four-frame close lifecycle")
		}
		for g.nativeShopUIJob != nil {
			g.nativeShopUIJob.drawn = true
			g.stepNativeShopUILifecycle(time.Time{})
		}
		if g.camp.NodeID() != "town_ch26" || g.campSel != 5 ||
			!g.partyMembers[26] || !g.partyMembers[29] {
			t.Fatalf("town26 secret return node=%q selection=%d members=%v", g.camp.NodeID(), g.campSel, g.partyMembers)
		}
	})
}

func TestChapter15PostFourRawBranchesJoin18Town17AndSaveBoundary(t *testing.T) {
	const originalBase = "../../../org_game/炎龍騎士團/FLAME2"
	for _, archive := range []string{"FDFIELD.DAT", "FDSHAP.DAT", "FDOTHER.DAT", "FDICON.DAT", "FDTXT.DAT", "DATO.DAT"} {
		if _, err := os.Stat(filepath.Join(originalBase, archive)); err != nil {
			t.Skipf("player-provided original %s is absent: %v", archive, err)
		}
	}
	t.Setenv("FD2_ORIGINAL_FDFIELD", filepath.Join(originalBase, "FDFIELD.DAT"))
	t.Setenv("FD2_ORIGINAL_FDSHAP", filepath.Join(originalBase, "FDSHAP.DAT"))
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(originalBase, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDICON", filepath.Join(originalBase, "FDICON.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(originalBase, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(originalBase, "DATO.DAT"))
	t.Setenv("FD2_MUTE", "1")
	order := []int{0, 4, 9, 30, 1, 8, 2, 10, 13, 12, 5, 6, 11, 14, 17, 15}
	tests := []struct {
		name        string
		round       int
		inactive    int
		word42      uint16
		wantJoin18  bool
		wantDialog  int
		wantIndices map[int]bool
	}{
		{name: "round_gt_18", round: 19, inactive: 0, word42: 0x140, wantDialog: 8, wantIndices: map[int]bool{2: true, 3: true}},
		{name: "inactive_gt_4", round: 18, inactive: 5, word42: 0x140, wantDialog: 8, wantIndices: map[int]bool{2: true, 3: true}},
		{name: "word42_below_gate", round: 18, inactive: 4, word42: 0x13f},
		{name: "word42_join18", round: 18, inactive: 4, word42: 0x140, wantJoin18: true, wantDialog: 15, wantIndices: map[int]bool{4: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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
			pre, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch15_pre.json"))
			if err != nil || len(issues) != 0 || len(pre) == 0 || pre[0].LoadCH == nil {
				t.Fatalf("ch15_pre compile err=%v issues=%#v beats=%#v", err, issues, pre)
			}
			if err := g.applyLoadCH(pre[0].LoadCH); err != nil {
				t.Fatal(err)
			}
			g.resetBattle("assets/maps/map15/map15_units.json", "assets/scenarios/ch16.json")
			if g.loadErr != "" || g.st == nil || len(g.st.Units) != 76 {
				t.Fatalf("ch16 handoff err=%q units=%d", g.loadErr, len(g.st.Units))
			}
			for i, id := range order {
				if g.st.Units[i] == nil || g.st.Units[i].Fig != id {
					t.Fatalf("persistent-first slot%d=%#v, want fig%d", i, g.st.Units[i], id)
				}
			}
			joinMatches := 0
			for _, unit := range g.st.Units {
				if unit != nil && unit.HasNativeRecordByte8 && int(unit.NativeRecordByte8) == 18 {
					joinMatches++
				}
			}
			if joinMatches != 1 {
				t.Fatalf("JOIN18 raw identity matches=%d, want unique", joinMatches)
			}
			g.st.NativeRoundCounter = test.round
			if g.st.Units[0] == nil {
				t.Fatal("slot0 missing")
			}
			g.st.Units[0].NativeRecordWord42 = test.word42
			g.st.Units[0].HasNativeRecordWord42 = true
			for i := 66; i <= 73; i++ {
				if g.st.Units[i] == nil || !g.st.Units[i].HasNativeRecordByte5 {
					t.Fatalf("slot%d lacks raw +5 provenance", i)
				}
				if i-66 < test.inactive {
					g.st.Units[i].NativeRecordByte5 = 1
				} else {
					g.st.Units[i].NativeRecordByte5 = 0
				}
			}
			g.curX, g.curY = 0, 0
			if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{}); err != nil ||
				!g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
				t.Fatalf("chapter16 native view setup err=%v", err)
			}
			if err := g.composeNativeMapFrame(); err != nil {
				t.Fatal(err)
			}
			beats, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch15_post.json"))
			if err != nil || len(issues) != 0 {
				t.Fatalf("ch15_post compile err=%v issues=%#v", err, issues)
			}
			g.camp = campaign.NewRunner(&campaign.Campaign{
				Start: "postbattle_ch16_persist",
				Nodes: map[string]*campaign.Node{
					"postbattle_ch16_persist": {Type: "cutscene", Next: "town_ch17"},
					"town_ch17":               {Type: "town"},
				},
			})
			g.beats, g.beatIdx, g.storyBG = beats, -1, true
			g.beatAdvance()
			seen := make(map[int]bool, test.wantDialog)
			seenIndices := make(map[int]bool, len(test.wantIndices))
			for frame := 0; frame < 40000 && g.camp.NodeID() != "town_ch17"; frame++ {
				if len(g.dialog) != 0 {
					current := g.dialog[len(g.dialog)-1]
					if current.NativeDialogue == nil || current.Upper == nil || current.NativeDialogue.SourceDAT != "FDTXT_016" ||
						!test.wantIndices[current.NativeDialogue.StringIndex] || len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) != 5 {
						t.Fatalf("ch15_post dialog lost indexed lifecycle: %#v", current)
					}
					seen[current.NativeDialogue.StringIndex*100+current.NativeDialogue.Utterance] = true
					seenIndices[current.NativeDialogue.StringIndex] = true
					if g.nativeStoryDialogueAtInputWait() && !g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
						t.Fatal("ch15_post formal story input was rejected")
					}
				}
				if err := g.Update(); err != nil {
					t.Fatal(err)
				}
				if g.loadErr != "" {
					t.Fatalf("ch15_post stopped at beat %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
				}
			}
			if g.camp.NodeID() != "town_ch17" || g.handlerChapter != 16 || g.st != nil ||
				len(seen) != test.wantDialog || len(seenIndices) != len(test.wantIndices) {
				t.Fatalf("ch15_post boundary node=%q chapter=%d st=%v dialogs=%d/%d indices=%v/%v", g.camp.NodeID(), g.handlerChapter, g.st != nil, len(seen), test.wantDialog, seenIndices, test.wantIndices)
			}
			if got := g.partyMembers[18]; got != test.wantJoin18 {
				t.Fatalf("JOIN18 membership=%v, want %v", got, test.wantJoin18)
			}
			if test.wantJoin18 {
				if _, ok := g.partyRoster[18]; !ok {
					t.Fatal("JOIN18 did not materialize persistent roster record")
				}
			} else if _, ok := g.partyRoster[18]; ok {
				t.Fatal("non-join branch materialized JOIN18")
			}
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			userDataDirCached = ""
			g.saveGameToSlot(2)
			if g.msg != "已存檔(槽位3：town_ch17)" {
				t.Fatalf("town17 save message=%q", g.msg)
			}
			g.camp.Cur = "postbattle_ch16_persist"
			g.partyMembers, g.partyJoinOrder = nil, nil
			g.partyDeploy, g.partyRoster = nil, nil
			g.loadGameFromSlot(2)
			wantOrder := 16
			if test.wantJoin18 {
				wantOrder = 17
			}
			if g.camp.NodeID() != "town_ch17" || g.partyMembers[18] != test.wantJoin18 || len(g.partyJoinOrder) != wantOrder {
				t.Fatalf("town17 save/load node=%q members=%v order=%v roster=%v", g.camp.NodeID(), g.partyMembers, g.partyJoinOrder, g.partyRoster)
			}
		})
	}
}

func TestChapter18PostJoins21And7Town19SaveBoundary(t *testing.T) {
	const originalBase = "../../../org_game/炎龍騎士團/FLAME2"
	for _, archive := range []string{"FDFIELD.DAT", "FDSHAP.DAT", "FDOTHER.DAT", "FDICON.DAT", "FDTXT.DAT", "DATO.DAT"} {
		if _, err := os.Stat(filepath.Join(originalBase, archive)); err != nil {
			t.Skipf("player-provided original %s is absent: %v", archive, err)
		}
	}
	t.Setenv("FD2_ORIGINAL_FDFIELD", filepath.Join(originalBase, "FDFIELD.DAT"))
	t.Setenv("FD2_ORIGINAL_FDSHAP", filepath.Join(originalBase, "FDSHAP.DAT"))
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(originalBase, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDICON", filepath.Join(originalBase, "FDICON.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(originalBase, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(originalBase, "DATO.DAT"))
	t.Setenv("FD2_MUTE", "1")

	order := []int{0, 4, 9, 30, 1, 8, 2, 10, 13, 12, 5, 6, 11, 14, 17, 15, 18, 16}
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
	if err := g.loadMap("assets/maps/map17"); err != nil {
		t.Fatal(err)
	}
	pre, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch17_pre.json"))
	if err != nil || len(issues) != 0 || len(pre) == 0 || pre[0].LoadCH == nil {
		t.Fatalf("ch17_pre compile err=%v issues=%#v beats=%#v", err, issues, pre)
	}
	if err := g.applyLoadCH(pre[0].LoadCH); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map17/map17_units.json", "assets/scenarios/ch18.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil || !g.sc.RuntimeAppendGroups || len(g.st.Units) != 55 {
		t.Fatalf("ch18 handoff err=%q units=%d runtime_append=%v", g.loadErr, len(g.st.Units), g.sc != nil && g.sc.RuntimeAppendGroups)
	}
	for i, id := range order {
		if g.st.Units[i] == nil || g.st.Units[i].Fig != id {
			t.Fatalf("ch18 persistent-first slot%d=%#v, want fig%d", i, g.st.Units[i], id)
		}
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
		t.Fatal("chapter18 map has no empty cursor cell")
	}
	g.curX, g.curY = emptyX, emptyY
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CursorX: emptyX, CursorY: emptyY, VisibleCursorX: emptyX, VisibleCursorY: emptyY,
	}); err != nil || !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatalf("chapter18 native view setup err=%v", err)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	beats, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch17_post.json"))
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch17_post compile err=%v issues=%#v", err, issues)
	}
	g.camp = campaign.NewRunner(&campaign.Campaign{
		Start: "postbattle_ch18_persist",
		Nodes: map[string]*campaign.Node{
			"postbattle_ch18_persist": {Type: "cutscene", Next: "town_ch19"},
			"town_ch19":               {Type: "town"},
		},
	})
	g.beats, g.beatIdx, g.storyBG = beats, -1, true
	g.beatAdvance()
	type dialogueKey struct{ stringIndex, utterance int }
	seenDialogue := make(map[dialogueKey]bool, 21)
	for frame := 0; frame < 50000 && g.camp.NodeID() != "town_ch19"; frame++ {
		if len(g.dialog) != 0 {
			current := g.dialog[len(g.dialog)-1]
			if current.NativeDialogue == nil || current.Upper == nil ||
				current.NativeDialogue.SourceDAT != "FDTXT_018" ||
				current.NativeDialogue.StringIndex < 7 || current.NativeDialogue.StringIndex > 10 ||
				len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) != 5 ||
				len(g.nativeDialogueProgressive) != len(current.NativeDialogue.Pages) {
				t.Fatalf("ch17_post dialog lost indexed lifecycle: %#v", current)
			}
			seenDialogue[dialogueKey{current.NativeDialogue.StringIndex, current.NativeDialogue.Utterance}] = true
			if g.nativeStoryDialogueAtInputWait() &&
				!g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
				t.Fatal("ch17_post formal story input was rejected")
			}
		}
		if err := g.Update(); err != nil {
			t.Fatalf("ch17_post Update: %v", err)
		}
		if g.loadErr != "" {
			t.Fatalf("ch17_post stopped at beat %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
		}
	}
	if g.camp.NodeID() != "town_ch19" || g.handlerChapter != 18 || g.st != nil {
		t.Fatalf("ch17_post boundary node=%q chapter=%d st=%v", g.camp.NodeID(), g.handlerChapter, g.st != nil)
	}
	if len(seenDialogue) != 21 {
		t.Fatalf("ch17_post formal native dialogues=%d, want 21", len(seenDialogue))
	}
	for _, id := range []int{21, 7} {
		if !g.partyMembers[id] {
			t.Fatalf("JOIN%d membership missing: %v", id, g.partyMembers)
		}
		joined, ok := g.partyRoster[id]
		if !ok || !joined.HasNativeIdentity || joined.NativeIdentity != id ||
			!joined.HasNativeRecordByte8 || int(joined.NativeRecordByte8) != id {
			t.Fatalf("JOIN%d persistent record=%#v", id, joined)
		}
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	g.saveGameToSlot(2)
	if g.msg != "已存檔(槽位3：town_ch19)" {
		t.Fatalf("town19 save message=%q", g.msg)
	}
	g.camp.Cur = "postbattle_ch18_persist"
	g.partyMembers, g.partyJoinOrder = nil, nil
	g.partyDeploy, g.partyRoster = nil, nil
	g.loadGameFromSlot(2)
	if g.camp.NodeID() != "town_ch19" || !g.partyMembers[21] || !g.partyMembers[7] || len(g.partyJoinOrder) != len(order)+2 {
		t.Fatalf("town19 save/load node=%q members=%v order=%v roster=%v", g.camp.NodeID(), g.partyMembers, g.partyJoinOrder, g.partyRoster)
	}
}

func TestChapter13PostNativeDialogueJoins3Town14SaveBoundary(t *testing.T) {
	const originalBase = "../../../org_game/炎龍騎士團/FLAME2"
	for _, archive := range []string{"FDFIELD.DAT", "FDSHAP.DAT", "FDOTHER.DAT", "FDICON.DAT", "FDTXT.DAT", "DATO.DAT"} {
		if _, err := os.Stat(filepath.Join(originalBase, archive)); err != nil {
			t.Skipf("player-provided original %s is absent: %v", archive, err)
		}
	}
	t.Setenv("FD2_ORIGINAL_FDFIELD", filepath.Join(originalBase, "FDFIELD.DAT"))
	t.Setenv("FD2_ORIGINAL_FDSHAP", filepath.Join(originalBase, "FDSHAP.DAT"))
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(originalBase, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDICON", filepath.Join(originalBase, "FDICON.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(originalBase, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(originalBase, "DATO.DAT"))
	t.Setenv("FD2_MUTE", "1")

	order := []int{0, 4, 9, 30, 1, 8, 2, 10, 13, 12, 5, 6, 11, 14, 17}
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
	pre, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch12_pre.json"))
	if err != nil || len(issues) != 0 || len(pre) == 0 || pre[0].LoadCH == nil {
		t.Fatalf("ch12_pre compile err=%v issues=%#v beats=%#v", err, issues, pre)
	}
	if err := g.applyLoadCH(pre[0].LoadCH); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map12/map12_units.json", "assets/scenarios/ch13.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil {
		t.Fatalf("ch13 handoff err=%q state=%v scenario=%v", g.loadErr, g.st != nil, g.sc != nil)
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
		t.Fatal("chapter13 map has no empty cursor cell")
	}
	g.curX, g.curY = emptyX, emptyY
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CursorX: emptyX, CursorY: emptyY, VisibleCursorX: emptyX, VisibleCursorY: emptyY,
	}); err != nil || !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatalf("chapter13 native view setup err=%v", err)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	beats, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch12_post.json"))
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch12_post compile err=%v issues=%#v", err, issues)
	}
	g.camp = campaign.NewRunner(&campaign.Campaign{
		Start: "postbattle_ch13_persist",
		Nodes: map[string]*campaign.Node{
			"postbattle_ch13_persist": {Type: "cutscene", Next: "town_ch14"},
			"town_ch14":               {Type: "town"},
		},
	})
	g.beats, g.beatIdx, g.storyBG = beats, -1, true
	g.beatAdvance()
	seen := make(map[int]bool, 12)
	for frame := 0; frame < 40000 && g.camp.NodeID() != "town_ch14"; frame++ {
		if len(g.dialog) != 0 {
			current := g.dialog[len(g.dialog)-1]
			if current.NativeDialogue == nil || current.Upper == nil ||
				current.NativeDialogue.SourceDAT != "FDTXT_013" || current.NativeDialogue.StringIndex != 9 ||
				len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) != 5 ||
				len(g.nativeDialogueProgressive) != len(current.NativeDialogue.Pages) {
				t.Fatalf("ch12_post dialog lost indexed lifecycle: %#v", current)
			}
			seen[current.NativeDialogue.Utterance] = true
			if g.nativeStoryDialogueAtInputWait() &&
				!g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
				t.Fatal("ch12_post formal story input was rejected")
			}
		}
		if err := g.Update(); err != nil {
			t.Fatalf("ch12_post Update: %v", err)
		}
		if g.loadErr != "" {
			t.Fatalf("ch12_post stopped at beat %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
		}
	}
	if g.camp.NodeID() != "town_ch14" || g.handlerChapter != 13 || g.st != nil || len(seen) != 12 {
		t.Fatalf("ch12_post boundary node=%q chapter=%d state=%v dialogues=%d", g.camp.NodeID(), g.handlerChapter, g.st != nil, len(seen))
	}
	if !g.partyMembers[3] {
		t.Fatalf("JOIN3 membership missing: %v", g.partyMembers)
	}
	joined, ok := g.partyRoster[3]
	if !ok || !joined.HasNativeIdentity || joined.NativeIdentity != 3 {
		t.Fatalf("JOIN3 persistent record=%#v", joined)
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	g.saveGameToSlot(1)
	if g.msg != "已存檔(槽位2：town_ch14)" {
		t.Fatalf("town14 save message=%q", g.msg)
	}
	g.camp.Cur = "postbattle_ch13_persist"
	g.partyMembers, g.partyJoinOrder = nil, nil
	g.partyDeploy, g.partyRoster = nil, nil
	g.loadGameFromSlot(1)
	if g.camp.NodeID() != "town_ch14" || !g.partyMembers[3] || len(g.partyJoinOrder) != len(order)+1 {
		t.Fatalf("town14 save/load node=%q members=%v order=%v roster=%v", g.camp.NodeID(), g.partyMembers, g.partyJoinOrder, g.partyRoster)
	}
}

func TestChapter17PostBranchJoin16Town18SaveBoundary(t *testing.T) {
	const originalBase = "../../../org_game/炎龍騎士團/FLAME2"
	for _, archive := range []string{"FDFIELD.DAT", "FDSHAP.DAT", "FDOTHER.DAT", "FDICON.DAT", "FDTXT.DAT", "DATO.DAT"} {
		if _, err := os.Stat(filepath.Join(originalBase, archive)); err != nil {
			t.Skipf("player-provided original %s is absent: %v", archive, err)
		}
	}
	t.Setenv("FD2_ORIGINAL_FDFIELD", filepath.Join(originalBase, "FDFIELD.DAT"))
	t.Setenv("FD2_ORIGINAL_FDSHAP", filepath.Join(originalBase, "FDSHAP.DAT"))
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(originalBase, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDICON", filepath.Join(originalBase, "FDICON.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(originalBase, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(originalBase, "DATO.DAT"))
	t.Setenv("FD2_MUTE", "1")
	for _, tc := range []struct {
		name        string
		order       []int
		preSlots    int
		postSlots   int
		branchIndex int
		wantDialog  int
	}{
		{name: "roster_has_18", order: []int{0, 4, 9, 30, 1, 8, 2, 10, 13, 12, 5, 6, 11, 14, 17, 18}, preSlots: 60, postSlots: 61, branchIndex: 5, wantDialog: 23},
		{name: "roster_lacks_18", order: []int{0, 4, 9, 30, 1, 8, 2, 10, 13, 12, 5, 6, 11, 14, 17, 15}, preSlots: 61, postSlots: 62, branchIndex: 7, wantDialog: 22},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := &Game{
				partyMembers:   make(map[int]bool, len(tc.order)),
				partyJoinOrder: append([]int(nil), tc.order...),
				partyDeploy:    make(map[int]bool, len(tc.order)-1),
			}
			for _, id := range tc.order {
				g.partyMembers[id] = true
			}
			for _, id := range tc.order[1:] {
				g.partyDeploy[id] = true
			}
			pre, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch16_pre.json"))
			if err != nil || len(issues) != 0 || len(pre) == 0 || pre[0].LoadCH == nil {
				t.Fatalf("ch16_pre compile err=%v issues=%#v beats=%#v", err, issues, pre)
			}
			if err := g.applyLoadCH(pre[0].LoadCH); err != nil {
				t.Fatal(err)
			}
			if !g.partyMembers[18] {
				// Execute the proven ch16_pre else arm: native group1 is
				// present before battle17 only when character18 is absent.
				g.materializeStoryGroup(1)
			}
			g.resetBattle("assets/maps/map16/map16_units.json", "assets/scenarios/ch17.json")
			if g.loadErr != "" || g.st == nil || g.sc == nil || !g.sc.RuntimeAppendGroups {
				t.Fatalf("ch17 handoff err=%q state=%v scenario=%v", g.loadErr, g.st != nil, g.sc != nil && g.sc.RuntimeAppendGroups)
			}
			if _, err := g.st.AppendGroupWithNativePlacement(2, 0); err != nil {
				t.Fatalf("ch17 group2 frontier: %v", err)
			}
			if len(g.st.Units) != tc.preSlots {
				t.Fatalf("ch17 pre frontier=%d, want %d", len(g.st.Units), tc.preSlots)
			}
			g.curX, g.curY = 0, 0
			if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{}); err != nil ||
				!g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
				t.Fatalf("chapter17 native view setup err=%v", err)
			}
			if err := g.composeNativeMapFrame(); err != nil {
				t.Fatal(err)
			}
			beats, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch16_post.json"))
			if err != nil || len(issues) != 0 {
				t.Fatalf("ch16_post compile err=%v issues=%#v", err, issues)
			}
			g.camp = campaign.NewRunner(&campaign.Campaign{
				Start: "postbattle_ch17_persist",
				Nodes: map[string]*campaign.Node{
					"postbattle_ch17_persist": {Type: "cutscene", Next: "town_ch18"},
					"town_ch18":               {Type: "town"},
				},
			})
			g.beats, g.beatIdx, g.storyBG = beats, -1, true
			g.beatAdvance()
			maxSlots := len(g.st.Units)
			seen := make(map[int]bool, tc.wantDialog)
			seenIndices := make(map[int]bool, 3)
			for frame := 0; frame < 50000 && g.camp.NodeID() != "town_ch18"; frame++ {
				if len(g.dialog) != 0 {
					current := g.dialog[len(g.dialog)-1]
					if current.NativeDialogue == nil || current.Upper == nil || current.NativeDialogue.SourceDAT != "FDTXT_017" {
						t.Fatalf("ch16_post dialog lost indexed lifecycle: %#v", current)
					}
					index := current.NativeDialogue.StringIndex
					if (index != tc.branchIndex && index != 6 && index != 8) ||
						len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) != 5 {
						t.Fatalf("ch16_post dialog lost indexed lifecycle: %#v", current)
					}
					seen[index*100+current.NativeDialogue.Utterance] = true
					seenIndices[index] = true
					if g.nativeStoryDialogueAtInputWait() && !g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
						t.Fatal("ch16_post formal story input was rejected")
					}
				}
				if err := g.Update(); err != nil {
					t.Fatal(err)
				}
				if g.st != nil && len(g.st.Units) > maxSlots {
					maxSlots = len(g.st.Units)
				}
				if g.loadErr != "" {
					t.Fatalf("ch16_post stopped at beat %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
				}
			}
			if g.camp.NodeID() != "town_ch18" || g.handlerChapter != 17 || g.st != nil || maxSlots != tc.postSlots ||
				len(seen) != tc.wantDialog || len(seenIndices) != 3 {
				t.Fatalf("ch16_post boundary node=%q chapter=%d state=%v maxSlots=%d/%d dialogues=%d/%d indices=%v", g.camp.NodeID(), g.handlerChapter, g.st != nil, maxSlots, tc.postSlots, len(seen), tc.wantDialog, seenIndices)
			}
			if !g.partyMembers[16] {
				t.Fatal("JOIN16 membership missing")
			}
			if joined, ok := g.partyRoster[16]; !ok || !joined.HasNativeIdentity || joined.NativeIdentity != 16 ||
				!joined.HasNativeRecordByte8 || int(joined.NativeRecordByte8) != 16 {
				t.Fatalf("JOIN16 persistent record=%#v", joined)
			}
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			userDataDirCached = ""
			g.saveGameToSlot(2)
			if g.msg != "已存檔(槽位3：town_ch18)" {
				t.Fatalf("town18 save message=%q", g.msg)
			}
			g.camp.Cur = "postbattle_ch17_persist"
			g.partyMembers, g.partyJoinOrder = nil, nil
			g.partyDeploy, g.partyRoster = nil, nil
			g.loadGameFromSlot(2)
			if g.camp.NodeID() != "town_ch18" || !g.partyMembers[16] || len(g.partyJoinOrder) != len(tc.order)+1 {
				t.Fatalf("town18 save/load node=%q members=%v order=%v roster=%v", g.camp.NodeID(), g.partyMembers, g.partyJoinOrder, g.partyRoster)
			}
		})
	}
}
