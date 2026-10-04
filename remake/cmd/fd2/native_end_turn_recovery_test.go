package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func endRecoveryTestGame(t *testing.T) *Game {
	t.Helper()
	t.Setenv("FD2_MUTE", "1")
	a, m, st := completeNativeMapFrameFixture(t)
	u := st.Units[0]
	u.OnField, u.HP, u.MaxHP, u.NativeRecordByte6 = true, 5, 10, 2
	u.SetMapPlacement(4, 4, 0)
	g := &Game{nativeMapAssets: a, m: m, st: st, curX: 4, curY: 4}
	if err := g.composeNativeMapFrameAt(time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	return g
}

func finishNativeEndTurnRecoveryTest(t *testing.T, g *Game) {
	t.Helper()
	if g.nativeEndTurnRecovery == nil {
		t.Fatalf("END owner 尚未開始：%s", g.loadErr)
	}
	screen := ebiten.NewImage(640, 400)
	for stage := 0; stage < 3; stage++ {
		job := g.nativeEndTurnRecovery
		if job == nil {
			t.Fatal("尚未發布兩張 END 畫面便離開 owner")
		}
		g.Draw(screen)
		g.stepNativeEndTurnRecovery(job.readyAt)
	}
	if g.nativeEndTurnRecovery != nil {
		t.Fatal("END發布未收尾")
	}
}

func TestNativeEndTurnRecoveryAutoENDUsesSameOwner(t *testing.T) {
	g := endRecoveryTestGame(t)
	u := g.st.Units[0]
	u.Camp, u.Acted, u.NativeRecordByte5 = battle.Own, true, 0x80
	if !g.autoEndPlayerPhase(u) || g.nativeEndTurnRecovery == nil || g.loadErr != "" {
		t.Fatalf("自動 END 沒有進同一 owner：%s", g.loadErr)
	}
	job := g.nativeEndTurnRecovery
	g.endTurn()
	if g.nativeEndTurnRecovery != job || len(job.plan) != 0 || u.HP != 5 || g.aiBusy {
		t.Fatal("重入或全員已行動的 END 提前寫入／啟動 AI")
	}
}

func TestNativeEndTurnRecoveryCommittedFieldsRoundTripCurrentSave(t *testing.T) {
	g, _, _ := nativeCurrentSaveTestGame(t)
	plan := g.st.PlanNativeEndTurnRecovery()
	if len(plan) != 1 || plan[0].Before != 20 || plan[0].After != 26 {
		t.Fatal("存檔夾具的回復計畫")
	}
	if err := g.st.CommitNativeEndTurnRecovery(plan); err != nil {
		t.Fatal(err)
	}
	_, stored, err := g.buildNativeCurrentSaveStored()
	if err != nil {
		t.Fatal(err)
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil {
		t.Fatal(err)
	}
	view := snapshot.RuntimeRecords[0].View()
	if view.HP != 26 || view.MaxHP != 30 || view.RawByte5 != 0x80 || !g.st.Units[0].Acted {
		t.Fatalf("回復的 HP／已行動旗標存檔不一致：%+v", view)
	}
}

func TestNativeEndTurnRecoveryDrawCommitAndCallbackOrder(t *testing.T) {
	g := endRecoveryTestGame(t)
	u := g.st.Units[0]
	finished := false
	if err := g.beginNativeEndTurnRecovery(func() { finished = true }); err != nil {
		t.Fatal(err)
	}
	job := g.nativeEndTurnRecovery
	screen := ebiten.NewImage(640, 400)
	g.stepNativeEndTurnRecovery(job.readyAt)
	if job.frame != 0 || u.HP != 5 {
		t.Fatal("未 Draw 已推進或提交")
	}
	g.Draw(screen)
	g.stepNativeEndTurnRecovery(job.readyAt.Add(-time.Nanosecond))
	if job.frame != 0 {
		t.Fatal("initial BIOS tick 尚未到期")
	}
	g.stepNativeEndTurnRecovery(job.readyAt)
	if job.frame != 1 || u.HP != 5 || job.cueCount != 0 || finished {
		t.Fatal("遮罩尚未發布便提交")
	}
	g.Draw(screen)
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if job.frame != 2 || u.HP != 7 || u.NativeRecordByte5 != 0x80 || job.cueCount != 1 || finished {
		t.Fatal("第二輪交易或單次 sample4 順序不正確")
	}
	if g.saveGameToSlot(0) == nil {
		t.Fatal("發布期间 JSON 存檔未阻擋")
	}
	if _, _, err := g.buildNativeCurrentSaveStored(); err == nil {
		t.Fatal("發布期间原版存檔未阻擋")
	}
	g.Update()
	if finished || job.frame != 2 {
		t.Fatal("第二張未 Draw 已跑回合事件")
	}
	g.Draw(screen)
	g.Update()
	if !finished || g.nativeEndTurnRecovery != nil || job.cueCount != 1 || g.loadErr != "" {
		t.Fatalf("收尾=%v owner=%v cue=%d err=%s", finished, g.nativeEndTurnRecovery != nil, job.cueCount, g.loadErr)
	}
}

func TestNativeEndTurnRecoveryUsesOpaqueMaskAndRawRestore(t *testing.T) {
	g := endRecoveryTestGame(t)
	// Every frame of the fixture has opaque pixels. Make source0 opaque and
	// the neighboring pixel transparent, then compose the formal source.
	for i := range g.nativeMapAssets.Units.Sprites {
		g.nativeMapAssets.Units.Sprites[i].Pixels[0] = 0
		g.nativeMapAssets.Units.Sprites[i].Mask[1] = 0
	}
	if err := g.composeNativeMapFrameAt(time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), g.nativeMapVGA...)
	if err := g.beginNativeEndTurnRecovery(nil); err != nil {
		t.Fatal(err)
	}
	job := g.nativeEndTurnRecovery
	const pixel = (4*24-2)*320 + 4 + 4*24 // VGA y4 plus native -6 row bias
	if job.frames[1][pixel] != 0xc8 || job.frames[1][pixel+1] != before[pixel+1] ||
		job.frames[2][pixel] != 0 || !bytes.Equal(job.frames[2], before) {
		t.Fatalf("mode2/source0/透明/raw restore：%x %x %x", job.frames[1][pixel], job.frames[1][pixel+1], job.frames[2][pixel])
	}
}

func TestNativeEndTurnRecoveryRejectsAssetsAndStalePlanWithoutWrites(t *testing.T) {
	for _, kind := range []string{"asset", "hp", "selector", "gate"} {
		t.Run(kind, func(t *testing.T) {
			g := endRecoveryTestGame(t)
			u := g.st.Units[0]
			beforeGate := g.st.NativeMapHUDState.DisplayGateB
			if kind == "asset" {
				g.nativeMapAssets.Units.Sprites = nil
				if err := g.beginNativeEndTurnRecovery(nil); err == nil {
					t.Fatal("缺資產未拒絕")
				}
			} else {
				if err := g.beginNativeEndTurnRecovery(nil); err != nil {
					t.Fatal(err)
				}
				job := g.nativeEndTurnRecovery
				job.frame, job.drawn = 1, true // isolates stale transaction preflight
				switch kind {
				case "hp":
					u.MaxHP = 11 // same floor(/5); must still reject
				case "selector":
					u.NativeMapPresentation.Pose = 1
				case "gate":
					u.NativeRecordByte6 = 0
				}
				g.stepNativeEndTurnRecovery(job.readyAt)
				if g.loadErr == "" || job.cueCount != 0 {
					t.Fatal("過期交易或sample未拒絕")
				}
			}
			if u.HP != 5 || u.NativeRecordByte5 != 0 || g.st.NativeMapHUDState.DisplayGateB != beforeGate || g.nativeEndTurnRecovery != nil {
				t.Fatal("失敗交易留下HP/flags/HUD寫入")
			}
		})
	}
}

func TestNativeEndTurnRecoveryOffscreenAndNoCandidate(t *testing.T) {
	for _, acted := range []bool{false, true} {
		g := endRecoveryTestGame(t)
		u := g.st.Units[0]
		u.SetMapPlacement(30, 30, 0)
		if acted {
			u.NativeRecordByte5 = 0x80
		}
		before := append([]byte(nil), g.nativeMapVGA...)
		if err := g.beginNativeEndTurnRecovery(nil); err != nil {
			t.Fatal(err)
		}
		job := g.nativeEndTurnRecovery
		if !bytes.Equal(job.frames[1], before) || !bytes.Equal(job.frames[2], before) {
			t.Fatal("畫面外候選仍寫pixel")
		}
		job.frame, job.drawn = 1, true
		g.stepNativeEndTurnRecovery(job.readyAt)
		if (!acted && (u.HP != 7 || job.cueCount != 1)) || (acted && (u.HP != 5 || job.cueCount != 0)) {
			t.Fatal("畫面外／無候選的數值或cue錯誤")
		}
	}
}
