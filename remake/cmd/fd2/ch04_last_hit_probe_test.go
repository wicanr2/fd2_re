package main

import (
	"image"
	"os"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestChapterFourLastPlayerAttackProbe(t *testing.T) {
	if os.Getenv("FD2_CH04_PROBE_SAVE") == "" {
		t.Skip("FD2_CH04_PROBE_SAVE: 第四章建構槽")
	}
	t.Run("before-reinforcements", func(t *testing.T) { probeChapterFourLastAttack(t, false) })
	t.Run("after-reinforcements", func(t *testing.T) { probeChapterFourLastAttack(t, true) })
}

func probeChapterFourLastAttack(t *testing.T, afterReinforcements bool) {
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_SEED", "4")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if err := g.loadNativeGameFromSlot(os.Getenv("FD2_CH04_PROBE_SAVE"), 0); err != nil {
		t.Fatal(err)
	}
	t.Logf("loaded native fixture into %s", g.camp.NodeID())
	g.camp.Cur = "battle_ch04"
	g.enterNode()
	if !pump(t, g, 18000, func() bool {
		if len(g.dialog) > 0 && g.battleEvent == nil {
			g.dlgAdvance()
		}
		return deathBattleIdle(g)
	}) {
		t.Fatal(ch01Blockers(g))
	}
	if afterReinforcements {
		g.st.Turn = 4
		g.st.NativeRoundCounter = 4
		before := len(g.st.Units)
		done := false
		g.runEditableTurnEvents(1, func() { done = true })
		if !pump(t, g, 16000, func() bool { return done && deathBattleIdle(g) }) {
			t.Fatal("fourth turn event did not settle", ch01Blockers(g))
		}
		if len(g.st.Units) != before+4 {
			t.Fatalf("reinforcement count=%d", len(g.st.Units)-before)
		}
		for _, u := range g.st.Units[before:] {
			if u.Camp != battle.Enemy || u.NativeRecordByte6 != 0 || u.Group != 2 || !u.OnField {
				t.Fatalf("unexpected reinforcement: %+v", u)
			}
		}
		t.Log("turn-four event spawned four counted enemy units in group 2")
	}
	actor := g.st.Units[0]
	enemies := livingEnemies(g)
	if len(enemies) == 0 || !actor.Alive() || !actor.OnField {
		t.Fatalf("invalid setup: enemies=%d actor=%+v", len(enemies), actor)
	}
	target := enemies[0]
	if afterReinforcements {
		target = g.st.Units[len(g.st.Units)-1]
	}
	var cell battle.Cell
	found := false
	for _, p := range []battle.Cell{{X: actor.X + 1, Y: actor.Y}, {X: actor.X - 1, Y: actor.Y}, {X: actor.X, Y: actor.Y + 1}, {X: actor.X, Y: actor.Y - 1}} {
		if p.X >= 0 && p.X < g.st.W && p.Y >= 0 && p.Y < g.st.H && g.st.UnitAt(p.X, p.Y) == nil {
			cell, found = p, true
			break
		}
	}
	if !found {
		t.Fatal("no adjacent target square")
	}
	presentation := &modernHDPresentation{Game: g}
	presentation.queueToolbarPoint(playerChapterButton.Min)
	if !presentation.chapterGuideOpen || strings.Contains(strings.Join(presentation.chapterGuideLines(), "\n"), "第 4 回合") {
		t.Fatal("toolbar failed to open the spoiler-free chapter guide")
	}
	presentation.chapterGuideDetail = true
	if !strings.Contains(strings.Join(presentation.chapterGuideLines(), "\n"), "第 4 回合") {
		t.Fatal("details did not reveal the reinforcement turn")
	}
	presentation.chapterGuideOpen = false
	for _, u := range enemies {
		if u != target {
			u.HP = 0
		}
	}
	g.st.MarkNativeDeadRecords()
	target.ApplyHPDamage(target.HP - 1)
	target.SetMapPlacement(cell.X, cell.Y, target.Dir)
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	if !g.positionScreenshotCursor(actor.X, actor.Y) {
		t.Fatal("cursor did not reach actor")
	}
	g.confirm()
	if g.sel != actor {
		t.Fatal("selection rejected")
	}
	g.confirm()
	if !pump(t, g, 240, func() bool { return g.ring }) || !nativeActionSelectable(g.actionOverlayAvailability(), 0) {
		t.Fatal("attack not available")
	}
	g.ringSel = 0
	if !closeRing(t, g, g.beginPlayerAttackTargeting) || !g.positionScreenshotCursor(target.X, target.Y) {
		t.Fatal("attack targeting failed")
	}
	t.Logf("before player attack: node=%s turn=%d living enemies=%d groups waiting=%v target class=%s group=%d raw camp=%d at=(%d,%d) hp=%d seed=%d", g.camp.NodeID(), g.st.Turn, len(livingEnemies(g)), g.st.PendingGroups, target.ClsName, target.Group, target.NativeRecordByte6, target.X, target.Y, target.HP, g.nativeRNGState)
	g.confirm()
	if g.atk == nil || target.HP != 0 {
		t.Fatalf("normal player attack did not kill: animation=%v hp=%d error=%s", g.atk != nil, target.HP, g.loadErr)
	}
	if !pump(t, g, 18000, func() bool {
		if g.nativeDeathRewardMessageAwaitingKey() {
			g.acknowledgeNativeDeathRewardMessage()
		}
		if g.nativeLevelUpDialogue != nil {
			g.stepNativeLevelUpDialogue(true)
		}
		return g.result != "" && g.atk == nil && !g.nativeActionDeathPresentationPending() && !g.nativeDeathProgramsPending() && g.nativeCh28PostPresent == nil
	}) || g.result != "win" {
		t.Fatalf("victory did not emerge: result=%q enemies=%d pendingActionDeath=%t pendingDeath=%t %s", g.result, len(livingEnemies(g)), g.nativeActionDeathPresentationPending(), g.nativeDeathProgramsPending(), ch01Blockers(g))
	}
	t.Logf("after normal player attack: hp=%d living enemies=%d result=%s", target.HP, len(livingEnemies(g)), g.result)
	if !pump(t, g, 16000, func() bool { return resultInputReachable(g) }) {
		t.Fatalf("victory confirmation not reachable: %s", ch01Blockers(g))
	}
	if !g.confirmBattleResult() || g.camp.NodeID() != "postbattle_ch04_persist" {
		t.Fatalf("win handoff: %s %s", g.camp.NodeID(), g.loadErr)
	}
	t.Logf("after victory confirmation: %s", g.camp.NodeID())
	driveStory(t, g, newJourneyTrace(), func() bool { return g.camp.NodeID() == "town_ch05" })
	t.Logf("next node: %s", g.camp.NodeID())
	if !pump(t, g, 480, func() bool { return g.fade == nil }) {
		t.Fatal("town fade did not settle")
	}
	presentation.queueToolbarPoint(image.Pt(playerChapterButton.Min.X+1, 12))
	if g.camp.NodeID() != "shop_ch05_secret" || g.loadErr != "" {
		t.Fatalf("direct secret shop: %s %s", g.camp.NodeID(), g.loadErr)
	}
	g.leaveShop()
	if !pump(t, g, 480, func() bool { return g.camp.NodeID() == "town_ch05" }) {
		t.Fatal("secret shop: missing return to town")
	}
}
