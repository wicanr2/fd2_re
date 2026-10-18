package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestModernHDPreparationTransitionRecipesMatchNativeFrames(t *testing.T) {
	for _, kind := range []string{"record", "departure", "confirmation", "required"} {
		for _, closing := range []bool{false, true} {
			if kind == "required" && closing {
				continue // Its closing frames intentionally contain no question.
			}
			t.Run(kind+map[bool]string{false: "_opening", true: "_closing"}[closing], func(t *testing.T) {
				g := newPreparationRecordTestGame(t)
				class, err := loadNativeClassUIAssets()
				if err != nil {
					t.Fatal(err)
				}
				g.nativeClassUI, g.font = class, loadFont()
				_, g.nativeBattleGlyphs, err = loadNativeBattleNameAssets()
				if err != nil {
					t.Fatal(err)
				}
				identity := 1
				if kind == "departure" {
					g.camp.Node().Cancel = "town"
				}
				if kind == "required" {
					g.prepRequiredMissing = &identity
				}
				called := 0
				after := func() { called++ }
				var ok bool
				switch {
				case kind == "required":
					ok = g.beginNativePreparationRequiredOpening()
				case kind == "confirmation" && closing:
					ok = g.beginNativePreparationConfirmationClosing(after)
				case kind == "confirmation":
					ok = g.beginNativePreparationConfirmationOpening()
				case closing:
					ok = g.beginNativePreparationPromptClosing(after)
				default:
					ok = g.beginNativePreparationPromptOpening()
				}
				if !ok || g.nativeClassUIJob.preparationText == nil {
					t.Fatal("transition recipe unavailable")
				}
				job := g.nativeClassUIJob
				d := job.preparationText
				before := make([][]byte, len(job.frames))
				for i, frame := range job.frames {
					before[i] = append([]byte(nil), frame...)
					composed, active := d.compose(i, d.assets.status.Font)
					expected := i >= d.start && i < d.start+4 && (kind != "required" || i == d.start)
					if active != expected || (active && !bytes.Equal(composed, frame)) {
						t.Fatalf("native frame %d differs from its recipe", i)
					}
				}
				// Mutating the live owner must not retarget an existing job.
				g.prepPromptSource[0] ^= 255
				g.prepRequiredMissing = nil
				g.nativePreparationUI.status.Font = nil
				g.camp.Node().Cancel = "other"
				identity = 31
				for i := range job.frames {
					composed, active := d.compose(i, d.assets.status.Font)
					if active && !bytes.Equal(composed, before[i]) {
						t.Fatal("recipe changed with live owner")
					}
				}
				screen := ebiten.NewImage(logicalW, logicalH)
				job.frame = d.start
				for _, hd := range []bool{true, false, true} {
					g.modernHDText = nil
					if hd {
						g.modernHDText = &modernHDTextFrame{}
					}
					if !g.drawNativeClassUIJob(screen) || job.frame != d.start || !job.drawn {
						t.Fatal("draw/theme switch advanced transition")
					}
					if hd && len(g.modernHDText.runs) == 0 {
						t.Fatal("question stage has no HD text")
					}
					if g.font.indexedHD != nil {
						t.Fatal("transition adapter leaked")
					}
				}
				job.frame, job.drawn = 0, false
				count := 0
				for g.nativeClassUIJob != nil && count < 16 {
					g.modernHDText = &modernHDTextFrame{}
					if count%2 == 1 {
						g.modernHDText = nil
					}
					if !g.drawNativeClassUIJob(screen) {
						t.Fatal("native transition stopped drawing")
					}
					g.stepNativeClassUILifecycle(time.Time{})
					count++
				}
				wantCount := len(job.frames)
				if closing {
					wantCount++ // The caller-owned restore is acknowledged too.
				}
				if g.nativeClassUIJob != nil || count != wantCount || called != map[bool]int{false: 0, true: 1}[closing] {
					t.Fatalf("changed schedule/continuation: frames=%d callbacks=%d", count, called)
				}
				for i := range before {
					if !bytes.Equal(before[i], job.frames[i]) {
						t.Fatal("HD modified indexed cache")
					}
				}
			})
		}
	}
}

func TestModernHDPreparationTransitionInvalidRecipeKeepsNativeJob(t *testing.T) {
	g := newPreparationRecordTestGame(t)
	class, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeClassUI = class
	g.font, g.modernHDText = loadFont(), &modernHDTextFrame{}
	if !g.beginNativePreparationPromptOpening() {
		t.Fatal("native opening unavailable")
	}
	job := g.nativeClassUIJob
	job.frame = 6
	job.preparationText.kind = 255
	screen := ebiten.NewImage(logicalW, logicalH)
	if g.drawModernHDPreparationTransition(screen, job) || !g.drawNativeClassUIJob(screen) || !job.drawn || job.frame != 6 || len(g.modernHDText.runs) != 0 {
		t.Fatal("invalid recipe did not retain native acknowledgement path")
	}
}
