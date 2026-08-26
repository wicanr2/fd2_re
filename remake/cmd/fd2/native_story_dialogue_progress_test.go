package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestNativeStoryDialogueProgressBlocksInputUntilPageComplete(t *testing.T) {
	upper := true
	g := &Game{
		dialog: []battle.DialogLine{{
			Speaker: 26,
			Upper:   &upper,
			NativeDialogue: &battle.NativeDialogueLayout{
				Pages: [][]string{{"甲乙"}},
			},
		}},
		dlgShown:               dlgNone,
		dlgPage:                0,
		nativeDialogueProgress: -1,
		nativeDialogueProgressive: [][][]byte{{
			{0}, {1}, {2},
		}},
	}

	if g.dlgAdvance() || len(g.dialog) != 1 {
		t.Fatal("native story input advanced before the first progressive frame")
	}
	g.stepNativeStoryDialogueProgress()
	if g.nativeDialogueProgress != 0 || g.dlgAdvance() {
		t.Fatalf("frame0 progression/input = %d/%v", g.nativeDialogueProgress, len(g.dialog) == 0)
	}
	g.stepNativeStoryDialogueProgress()
	if g.nativeDialogueProgress != 1 || g.dlgAdvance() {
		t.Fatalf("first glyph progression/input = %d/%v", g.nativeDialogueProgress, len(g.dialog) == 0)
	}
	g.stepNativeStoryDialogueProgress()
	if g.nativeDialogueProgress != 2 {
		t.Fatalf("final glyph progression=%d, want 2", g.nativeDialogueProgress)
	}
	if !g.dlgAdvance() || len(g.dialog) != 0 || g.nativeDialogueProgress != -1 {
		t.Fatalf("completed page did not advance/reset: progress=%d dialog=%d", g.nativeDialogueProgress, len(g.dialog))
	}
}

func TestNativeStoryDialogueMotionTargetReplays12C60ActiveRawLookup(t *testing.T) {
	layout := &campaign.NativeDialogueLayout{Control: "FFEE", Operand: 21}
	g := &Game{storyActors: []battle.Unit{
		{NativeRecordByte8: 21, HasNativeRecordByte8: true, NativeRecordByte5: 1, HasNativeRecordByte5: true},
		{NativeRecordByte8: 21, HasNativeRecordByte8: true, NativeRecordByte5: 0, HasNativeRecordByte5: true},
	}}
	if got, err := g.resolveNativeStoryDialogueMotionTarget(layout); err != nil || got != 112 {
		t.Fatalf("FFEE active lookup target=%d err=%v, want 112", got, err)
	}
	g.storyActors[1].NativeRecordByte5 = 1
	if got, err := g.resolveNativeStoryDialogueMotionTarget(layout); err != nil || got != 0 {
		t.Fatalf("FFEE inactive-only lookup target=%d err=%v, want 0", got, err)
	}
	g.storyActors[1].HasNativeRecordByte5 = false
	if _, err := g.resolveNativeStoryDialogueMotionTarget(layout); err == nil {
		t.Fatal("missing raw +5 provenance did not fail closed")
	}
	if got, err := g.resolveNativeStoryDialogueMotionTarget(&campaign.NativeDialogueLayout{Control: "FFED"}); err != nil || got != 2 {
		t.Fatalf("FFED direct target=%d err=%v, want 2", got, err)
	}
}

func TestNativeStoryDialogueSpeakerUsesRawControlOwner(t *testing.T) {
	g := &Game{storyActors: []battle.Unit{
		{BattleFig: 48, HasBattleFig: true, NativeRecordByte8: 7, HasNativeRecordByte8: true, HasNativeRecordByte5: true},
		{BattleFig: 30, HasBattleFig: true, NativeRecordByte8: 30, HasNativeRecordByte8: true, HasNativeRecordByte5: true},
	}}
	if got, err := g.resolveNativeStoryDialogueSpeaker(&campaign.NativeDialogueLayout{Control: "FFEF", Operand: 30}); err != nil || got != 30 {
		t.Fatalf("identity speaker=%d err=%v, want raw +7 30", got, err)
	}
	if got, err := g.resolveNativeStoryDialogueSpeaker(&campaign.NativeDialogueLayout{Control: "FFEC", Operand: 0}); err != nil || got != 48 {
		t.Fatalf("slot speaker=%d err=%v, want raw +7 48", got, err)
	}
	if got, err := g.resolveNativeStoryDialogueSpeaker(&campaign.NativeDialogueLayout{Control: "FFEE", Operand: 39}); err != nil || got != 39 {
		t.Fatalf("fallback speaker=%d err=%v, want direct DATO 39", got, err)
	}
	g.storyActors[1].NativeRecordByte5 = 1
	if got, err := g.resolveNativeStoryDialogueSpeaker(&campaign.NativeDialogueLayout{Control: "FFEF", Operand: 30}); err != nil || got != 30 {
		t.Fatalf("inactive fallback speaker=%d err=%v, want direct DATO 30", got, err)
	}
}

func TestNativeStoryDialogueClosingKeepsOldLineUntilAllFramesPublished(t *testing.T) {
	g := &Game{
		dialog:                []battle.DialogLine{{NativeDialogue: &battle.NativeDialogueLayout{Pages: [][]string{{"甲"}}}}},
		nativeDialogueClosing: [][]byte{{1}, {2}, {3}},
		dlgShown:              1,
	}
	if !g.beginNativeStoryDialogueClosing() || len(g.dialog) != 1 || g.nativeDialogueClosingT != 0 {
		t.Fatal("closing did not retain its caller-owned old line")
	}
	for want := 1; want < 3; want++ {
		g.stepNativeStoryDialogueProgress()
		if !g.nativeDialogueClosingLive || g.nativeDialogueClosingT != want || len(g.dialog) != 1 {
			t.Fatalf("closing stage%d state live=%v tick=%d dialog=%d", want, g.nativeDialogueClosingLive, g.nativeDialogueClosingT, len(g.dialog))
		}
	}
	g.stepNativeStoryDialogueProgress()
	if g.nativeDialogueClosingLive || len(g.dialog) != 0 || g.dlgShown != dlgNone {
		t.Fatalf("closing completion live=%v dialog=%d shown=%d", g.nativeDialogueClosingLive, len(g.dialog), g.dlgShown)
	}
}

func TestNativeStoryDialogueProgressWaitsForStableDialoguePhase(t *testing.T) {
	g := &Game{
		dialog:                    []battle.DialogLine{{NativeDialogue: &battle.NativeDialogueLayout{Pages: [][]string{{"甲"}}}}},
		dlgPhase:                  2,
		nativeDialogueProgress:    -1,
		nativeDialogueProgressive: [][][]byte{{{0}, {1}}},
	}
	g.stepNativeStoryDialogueProgress()
	if g.nativeDialogueProgress != -1 {
		t.Fatalf("opening phase leaked progressive glyph: %d", g.nativeDialogueProgress)
	}
	g.dlgPhase = 0
	g.dlgScrollT = 1
	g.stepNativeStoryDialogueProgress()
	if g.nativeDialogueProgress != -1 {
		t.Fatalf("page scroll leaked progressive glyph: %d", g.nativeDialogueProgress)
	}
}

func TestNativeStoryDialogueMouthRunsOnlyWhileCompletePageWaitsForInput(t *testing.T) {
	g := &Game{
		dialog:                    []battle.DialogLine{{NativeDialogue: &battle.NativeDialogueLayout{Pages: [][]string{{"甲"}}}}},
		dlgPhase:                  0,
		nativeDialogueProgress:    0,
		nativeDialogueProgressive: [][][]byte{{make([]byte, 1), make([]byte, 1)}},
		nativeDialogueMouthOpen:   [][]byte{make([]byte, 320*200)},
	}
	g.stepDialogueMouth()
	if g.nativeDialogueMouthReady || g.mouthOpen {
		t.Fatal("progressive glyph phase started the stable-page mouth owner")
	}
	g.nativeDialogueProgress = 1
	g.stepDialogueMouth()
	if !g.nativeDialogueMouthReady || g.mouthOpen || g.mouthState.Countdown < 2 || g.mouthState.Countdown > 31 {
		t.Fatalf("initial wait state = ready:%v open:%v countdown:%d", g.nativeDialogueMouthReady, g.mouthOpen, g.mouthState.Countdown)
	}
	g.mouthState.Countdown = 0
	g.frame = 2
	g.stepDialogueMouth()
	if !g.mouthOpen || g.mouthState.FrameIndex() != 3 {
		t.Fatalf("zero post-decrement did not select frame3: %+v", g.mouthState)
	}
	g.frame = 4
	g.stepDialogueMouth()
	if g.mouthOpen || g.mouthState.Countdown < 2 || g.mouthState.Countdown > 31 {
		t.Fatalf("one-tick mouth did not close/resample: %+v", g.mouthState)
	}
	g.dlgScrollT = 1
	g.stepDialogueMouth()
	if g.nativeDialogueMouthReady || g.mouthOpen || g.mouthState.Countdown != 0 {
		t.Fatalf("page scroll did not reset mouth owner: %+v", g.mouthState)
	}
}

func TestNativeStoryDialogueInputFailsClosedWithoutProgressiveFrames(t *testing.T) {
	g := &Game{
		dialog: []battle.DialogLine{{
			NativeDialogue: &battle.NativeDialogueLayout{Pages: [][]string{{"甲"}}},
		}},
	}
	if g.dlgAdvance() || len(g.dialog) != 1 {
		t.Fatal("native story dialogue advanced without its indexed progressive frames")
	}
}

func TestNativeStoryOpeningFrameIndexPreservesFiveStageOrder(t *testing.T) {
	want := []int{0, 0, 1, 2, 3, 4, 4, 4}
	for tick, expected := range want {
		if got := nativeStoryOpeningFrameIndex(tick, 5); got != expected {
			t.Fatalf("opening tick%d=%d, want %d", tick, got, expected)
		}
	}
	if got := nativeStoryOpeningFrameIndex(0, 0); got != -1 {
		t.Fatalf("missing opening frames returned index %d", got)
	}
}
