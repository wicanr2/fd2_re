package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestRustGoldenR6TimedDialogue(t *testing.T) {
	out := os.Getenv("FD2_R6_OPENING_OUT")
	if out == "" {
		t.Skip("set FD2_R6_OPENING_OUT to export the Rust golden")
	}
	g := storyMaskTestGame(t, "zh-Hant")
	g.interfaceMode = playerInterfaceOriginal
	g.nativeMapAssets, g.m, g.st = completeNativeMapFrameFixture(t)
	g.nativeMapFrozenNow = func() time.Time { return time.Unix(1, 0) }
	unit := g.st.Units[0]
	unit.HasNativeMapPresentation = true
	pages, lastWait, err := nativeLevelUpDialogueLines(g.nativeClassUI.strings, pendingNativeLevelUp{
		unit: unit, exp: 12, ups: []battle.LevelUpEvent{{NewLv: 2, ApGain: 2, HpGain: 4}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.beginNativeTimedMessageDialogue(unit, nativeLevelUpExpTextIndex, pages, lastWait, nativeLevelUpCloseFrames, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := g.freezeModernHDTimedDialogue(&modernHDMapFrame{}); err != nil {
		t.Fatal(err)
	}
	var report strings.Builder
	for _, phase := range []struct {
		name   string
		kind   campaign.NativeDialogueFramePhase
		frames [][]byte
	}{
		{"opening", campaign.NativeDialogueOpening, g.nativeDialogueOpening},
		{"page", campaign.NativeDialogueSteady, g.nativeDialogueFrames},
		{"closing", campaign.NativeDialogueClosing, g.nativeDialogueClosing},
	} {
		for i, frame := range phase.frames {
			index := i
			if phase.kind == campaign.NativeDialogueSteady {
				index = 0
			}
			mask := g.nativeLevelUpDialogue.hdCoverage[phase.kind][index]
			picture, err := modernHDTimedDialoguePixels(frame, mask, g.nativeClassUI.palette)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(&report, "%s %d %x\n", phase.name, i, sha256.Sum256(picture.Pix))
		}
	}
	if err := os.WriteFile(filepath.Join(out, "go-timed-dialogue.txt"), []byte(report.String()), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestModernHDTimedDialogueCoverageMatchesNativeSnapshots(t *testing.T) {
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	for _, control := range []string{"FFEC", "FFEE", "FFED", "FFEF"} {
		layout := &campaign.NativeDialogueLayout{SourceDAT: "FDTXT_000", Control: control, Pages: [][]string{{"A"}}}
		for _, phase := range []campaign.NativeDialogueFramePhase{campaign.NativeDialogueOpening, campaign.NativeDialogueClosing} {
			var frames [2][][]byte
			for i, fill := range []byte{0, 255} {
				background := bytes.Repeat([]byte{fill}, 320*200)
				if phase == campaign.NativeDialogueOpening {
					frames[i], err = campaign.ComposeNativeStoryDialogueOpeningFrames(background, assets.dialogue, layout)
				} else {
					frames[i], err = campaign.ComposeNativeStoryDialogueClosingFrames(background, assets.dialogue, layout, 0, 0, 0)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			for index := range frames[0] {
				mask, e := campaign.NativeStoryDialogueCoverage(assets.dialogue, control, phase, index)
				if e != nil {
					t.Fatal(e)
				}
				for p, covered := range mask {
					written := frames[0][index][p] == frames[1][index][p]
					if covered != written {
						t.Fatalf("%s phase=%d index=%d pixel=%d coverage=%v written=%v", control, phase, index, p, covered, written)
					}
				}
			}
		}
		mask, e := campaign.NativeStoryDialogueCoverage(assets.dialogue, control, campaign.NativeDialogueSteady, 0)
		if e != nil {
			t.Fatal(e)
		}
		r, _ := campaign.NativeStoryDialoguePortraitRect(control)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if !mask[y*320+x] {
					t.Fatal("portrait exceeds opaque grid")
				}
			}
		}
		x, y, w, _, _, _ := campaign.NativeStoryDialogueTextGeometry(control)
		// Runtime admits FFEC only. FFEE's wider text window extends beyond
		// the grid; this grid mask cannot authorize that other caller.
		for row := y; control == "FFEC" && row < y+72; row++ {
			for col := x - 1; col < x+w; col++ {
				if !mask[row*320+col] {
					t.Fatalf("text exceeds opaque grid: %s x=%d y=%d width=%d pixel=%d,%d", control, x, y, w, col, row)
				}
			}
		}
	}
	if _, err := campaign.NativeStoryDialogueCoverage(assets.dialogue, "FFEC", campaign.NativeDialogueClosing, 5); err == nil {
		t.Fatal("cursor tail admitted without provenance")
	}
}

func TestModernHDTimedDialogueFreezeAndAtomicRejection(t *testing.T) {
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{nativeClassUI: assets, nativeDialogueLayout: &campaign.NativeDialogueLayout{Control: "FFEC"}, nativeDialogueClosing: make([][]byte, 5),
		dialog: []battle.DialogLine{{NativeDialogue: &battle.NativeDialogueLayout{Control: "FFEC", HasMotionTargetY: true}}}, nativeLevelUpDialogue: &nativeLevelUpDialogueState{}}
	first := &modernHDMapFrame{Background: image.NewRGBA(image.Rect(0, 0, 320, 200))}
	second := &modernHDMapFrame{Background: image.NewRGBA(image.Rect(0, 0, 320, 200))}
	if got, e := g.freezeModernHDTimedDialogue(first); e != nil || got != first {
		t.Fatalf("first freeze %p %v", got, e)
	}
	second.Background.SetRGBA(10, 10, color.RGBA{R: 255, A: 255})
	if got, e := g.freezeModernHDTimedDialogue(second); e != nil || got != first || first.Background.RGBAAt(10, 10).R != 0 {
		t.Fatal("dialogue background was replaced")
	}
	g.nativeLevelUpDialogue = nil
	if got, e := g.freezeModernHDTimedDialogue(second); e != nil || got != second {
		t.Fatal("discarded owner leaked world")
	}
	g.nativeLevelUpDialogue = &nativeLevelUpDialogueState{}
	bad := *assets
	bad.dialogue = append(bad.dialogue[:0:0], assets.dialogue...)
	bad.dialogue[13].Width = 320
	bad.dialogue[13].Height = 200
	bad.dialogue[13].Pixels = make([]byte, 320*200)
	g.nativeClassUI = &bad
	if _, e := g.freezeModernHDTimedDialogue(second); e == nil || g.nativeLevelUpDialogue.hdWorld != nil || g.nativeLevelUpDialogue.hdCoverage != nil {
		t.Fatal("invalid coverage published snapshot")
	}
}

func TestModernHDTimedDialoguePixelsKeepsOpaqueZero(t *testing.T) {
	frame := make([]byte, 320*200)
	mask := make([]bool, len(frame))
	mask[0] = true
	pal := make(color.Palette, 256)
	pal[0] = color.RGBA{A: 255}
	got, err := modernHDTimedDialoguePixels(frame, mask, pal)
	if err != nil || got.RGBAAt(0, 0).A != 255 || got.RGBAAt(1, 0).A != 0 {
		t.Fatal("zero-color write lost coverage", err)
	}
	mask[1] = true
	frame[1] = 1
	if _, err := modernHDTimedDialoguePixels(frame, mask, pal); err == nil {
		t.Fatal("missing palette admitted")
	}
}
