package main

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// Controlled BIOS samples isolate the waiting owner. No countdown or mouth
// frame is assigned by this test; the normal lifecycle must produce both.
func TestNativePreparationConfirmationMouthWaitShowsBothFrames(t *testing.T) {
	g := newPreparationRecordTestGame(t)
	g.nativeConfirmationMouthRNG = rand.New(rand.NewSource(177))
	g.nativeConfirmationMouthOwner = nil
	portraits, err := loadNativeSeparatedPortrait(0x4b)
	if err != nil || len(portraits) < 4 {
		t.Fatal("confirmation DATO0/3 unavailable", err)
	}
	seen := map[int]bool{}
	start := time.Unix(100, 0)
	for tick := 0; tick < 80; tick++ {
		g.stepNativeClassUILifecycle(start.Add(time.Duration(tick) * 2 * nativeBIOSTickPeriod))
		got, ok := g.composeNativePreparationPromptFrame()
		if !ok {
			t.Fatal("waiting frame unavailable")
		}
		matched := false
		for _, index := range []int{0, 3} {
			question, err := campaign.ComposeNativePreparationRecordQuestion(
				g.prepPromptSource, g.nativePreparationUI.dialogue, portraits[index],
				g.nativePreparationUI.status.Strings, g.nativePreparationUI.status.Font,
			)
			if err != nil {
				t.Fatal(err)
			}
			want, err := campaign.ComposeNativeConfirmationChoices(
				question, g.nativePreparationUI.choices, g.prepConfirmSel, g.nativeClassUIPulse/2,
			)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(got, want) {
				seen[index], matched = true, true
			}
		}
		if !matched {
			t.Fatal("waiting frame is neither complete DATO0 nor DATO3", tick)
		}
	}
	if !seen[0] || !seen[3] {
		t.Fatal("formal confirmation wait does not show both mouth frames", seen)
	}
}

func TestNativeConfirmationMouthENDWaitAndCallerPortrait(t *testing.T) {
	g := newPreparationRecordTestGame(t)
	g.camp = nil
	g.st = &battle.State{W: 24, H: 24}
	g.ring, g.ringSel, g.nativeSystemCursorOverlay = true, 3, true
	g.nativeMapVGA = make([]byte, 320*200)
	g.nativeConfirmationMouthRNG = rand.New(rand.NewSource(177))
	if !g.beginNativeSystemEndTurn() {
		t.Fatal("normal END refused")
	}
	for present := 0; present < 4; present++ {
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	for tick := 0; tick < 80; tick++ {
		g.stepNativeClassUILifecycle(time.Unix(100, 0).Add(time.Duration(tick) * 2 * nativeBIOSTickPeriod))
	}
	if g.nativeClassUIJob == nil || g.nativeClassUIJob.frame != 0 || g.nativeConfirmationMouthOwner != nil {
		t.Fatal("unacknowledged opening advanced or acquired mouth owner")
	}
	finishPreparationRecordTransition(t, g)
	state := g.nativeSystemEndTurnUI
	if !g.nativeSystemEndTurnConfirm || len(state.portraits) < 4 {
		t.Fatal("END lost caller portrait frames")
	}
	question := append([]byte(nil), state.question...)
	// A shared YESNO state can belong to another actual portrait caller.
	// Use real DATO26 to catch accidentally reading preparation's DATO4B.
	other, err := loadNativeSeparatedPortrait(26)
	if err != nil || len(other) < 4 {
		t.Fatal(err)
	}
	state.portraits = other
	opened := false
	start := time.Unix(200, 0)
	for tick := 0; tick < 80; tick++ {
		g.stepNativeClassUILifecycle(start.Add(time.Duration(tick) * 2 * nativeBIOSTickPeriod))
		if !g.nativeConfirmationMouth.Open {
			continue
		}
		opened = true
		got, ok := g.composeNativeConfirmationMouth(state.question, state.portraits)
		want, err := campaign.ComposeNativeConfirmationPortrait(state.question, other[3])
		if !ok || err != nil || !bytes.Equal(got, want) || !bytes.Equal(state.question, question) || g.st.Turn != 0 || g.aiBusy {
			t.Fatal("shared confirmation changed caller source, portrait or battle state")
		}
		before := g.nativeConfirmationMouth
		g.composeNativeConfirmationMouth(state.question, state.portraits)
		if g.nativeConfirmationMouth != before {
			t.Fatal("render advanced mouth state")
		}
		g.cancelNativeSystemEndTurn()
		g.stepNativeClassUILifecycle(start.Add(time.Duration(tick+10) * 2 * nativeBIOSTickPeriod))
		if g.nativeSystemEndTurnConfirm || g.nativeConfirmationMouthOwner != nil || g.nativeConfirmationMouth != before || g.st.Turn != 0 {
			t.Fatal("closing kept mouth owner or committed turn")
		}
		break
	}
	if !opened {
		t.Fatal("END confirmation never opened its mouth")
	}
}

func TestNativeConfirmationMouthMissingFrameFailsBeforeENDMutation(t *testing.T) {
	g := newPreparationRecordTestGame(t)
	g.camp = nil
	g.st = &battle.State{W: 24, H: 24}
	g.ring, g.ringSel, g.nativeSystemCursorOverlay = true, 3, true
	g.nativeMapVGA = make([]byte, 320*200)
	g.nativePreparationUI.portraits = g.nativePreparationUI.portraits[:1]
	if g.beginNativeSystemEndTurn() || !g.ring || !g.nativeSystemCursorOverlay || g.nativeSystemEndTurnUI != nil || g.actionOverlayPhase != "" {
		t.Fatal("missing DATO3 changed END owner")
	}
}
