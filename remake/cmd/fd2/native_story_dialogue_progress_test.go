package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
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
