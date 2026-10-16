package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestEvent76PulseBlocksNormalUpdateAndRepeatedEND(t *testing.T) {
	if os.Getenv("FD2_ORIGINAL_FDOTHER") == "" {
		t.Skip("需要唯讀原版素材")
	}
	t.Setenv("FD2_ASSET_PACK", filepath.Clean("../../generated-assets/fd2-original-b97caf22"))
	t.Setenv("FD2_MUTE", "1")
	g := newChapter29RuntimeBattle(t)
	if err := g.startNativePalettePulse(nativeEvent76PulseSpec(), nil); err != nil {
		t.Fatal(err)
	}
	job, before := g.nativePalettePulse, g.st.Turn
	r := &parityReplay{g: g, battle: "battle_ch29"}
	if r.playerHasControl() {
		t.Fatal("pulse期間重播提前取得玩家控制")
	}
	g.endTurn()
	if g.nativeEndTurnRecovery != nil || g.st.Turn != before || g.loadErr != "" {
		t.Fatal("pulse期間END建立了另一個演出owner")
	}
	job.drawn = true
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if g.nativePalettePulse != job || job.step != 1 || g.nativeEndTurnRecovery != nil || g.st.Turn != before {
		t.Fatal("正常Update未推進pulse或誤啟動END")
	}
}

func TestEvent76PresentationRunsSixPulsesTwoDelaysAndEditableTail(t *testing.T) {
	g := completeNativeUnitPresentGame(t)
	sc, err := battle.LoadScenario(assetPath("assets/scenarios/ch29.json"))
	if err != nil {
		t.Fatal(err)
	}
	g.sc = sc
	var event76 battle.NativeTurnEvent
	for _, event := range sc.NativeTurnEvents {
		if event.EventID == 76 {
			event76 = event
		}
	}
	done := 0
	g.runNativeEvent76Presentation(event76, 0, func() { done++ })
	wantDialogueCounts := []int{0, 0, 1, 1, 1, 5}
	for pulse, wantDialogue := range wantDialogueCounts {
		if g.nativePalettePulse == nil {
			t.Fatalf("pulse%d did not start: err=%q", pulse, g.loadErr)
		}
		for steps := 0; g.nativePalettePulse != nil && steps < 200; steps++ {
			g.nativePalettePulse.drawn = true
			g.nativePalettePulse.wait = 0
			g.stepNativePalettePulse()
		}
		if pulse < 2 {
			if g.battleEvent == nil || g.battleEventDelay <= 0 {
				t.Fatalf("pulse%d missing 400ms delay", pulse)
			}
			g.battleEventDelay = 1
			g.stepBattleEventDelay()
			continue
		}
		for page := 0; page < wantDialogue; page++ {
			if len(g.dialog) != 1 || g.battleEvent == nil {
				t.Fatalf("pulse%d dialogue%d unavailable", pulse, page)
			}
			g.dialog = nil
			g.advanceBattleEvent()
		}
	}
	if done != 1 || g.nativePalettePulse != nil || g.battleEvent != nil || len(g.dialog) != 0 || g.loadErr != "" {
		t.Fatalf("event76 presentation done=%d pulse=%v event=%v dialogue=%d err=%q", done, g.nativePalettePulse != nil, g.battleEvent != nil, len(g.dialog), g.loadErr)
	}
}
