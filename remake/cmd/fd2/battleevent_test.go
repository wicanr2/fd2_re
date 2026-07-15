package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

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
