package main

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestModernHDPreparationStableFramesRestoreNativeAssets(t *testing.T) {
	g := newPreparationRecordTestGame(t)
	class, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeClassUI = class
	g.font = loadFont()
	_, g.nativeBattleGlyphs, err = loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := battle.LoadScenario("../../assets/scenarios/ch01.json")
	if err != nil {
		t.Fatal(err)
	}
	g.prepIDs = nil
	g.partyRoster = make(map[int]battle.Unit)
	for _, u := range scenario.PartyUnits(nil) {
		g.prepIDs = append(g.prepIDs, u.Fig)
		g.partyRoster[u.Fig] = *u
	}
	g.partyDeploy = map[int]bool{g.prepIDs[0]: true}
	original := g.nativePreparationUI
	statusFont := original.status.Font
	statusFrames := make(map[int]fdother.Frame, len(original.status.Frames))
	for id, frame := range original.status.Frames {
		frame.Pixels = append([]byte(nil), frame.Pixels...)
		statusFrames[id] = frame
	}
	rosterBefore := make(map[int]battle.Unit)
	for id, u := range g.partyRoster {
		rosterBefore[id] = u
	}
	screen := ebiten.NewImage(logicalW, logicalH)
	g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true}
	done := g.beginModernHDText(screen)
	defer done()
	required := 1
	for _, mode := range []string{"record_prompt", "departure_prompt", "required", "roster", "confirm", "slots"} {
		g.camp.Node().Cancel = ""
		g.prepSelecting, g.prepConfirm, g.prepRecordSlots = false, false, false
		g.prepRequiredMissing = nil
		switch mode {
		case "departure_prompt":
			g.camp.Node().Cancel = "town"
		case "required":
			g.prepRequiredMissing = &required
		case "roster":
			g.prepSelecting = true
		case "confirm":
			g.prepConfirm = true
		case "slots":
			g.prepRecordSlots, g.prepRecordSlotSel = true, 3
		}
		before, ok := g.composeNativePreparationStableFrame()
		if !ok {
			t.Fatalf("%s native composition failed", mode)
		}
		if !g.drawNativePreparation(screen) || len(g.modernHDText.runs) == 0 {
			t.Fatalf("%s HD text missing", mode)
		}
		if g.nativePreparationUI != original || g.nativeClassUI != class || g.font.indexedHD != nil || original.status.Font != statusFont || !reflect.DeepEqual(original.status.Frames, statusFrames) {
			t.Fatalf("%s leaked adapted assets", mode)
		}
		after, ok := g.composeNativePreparationStableFrame()
		if !ok || !bytes.Equal(before, after) || !reflect.DeepEqual(rosterBefore, g.partyRoster) {
			t.Fatalf("%s changed native frame or roster", mode)
		}
	}
	// A failed stable composition must restore the adapters as well.
	g.prepRecordSlotSel = 4
	if g.drawNativePreparation(screen) || g.nativePreparationUI != original || g.nativeClassUI != class || g.font.indexedHD != nil {
		t.Fatal("failed composition leaked adapters")
	}
	g.modernHDText = nil
	capture, restore := g.beginModernHDPreparationText()
	restore()
	if capture != nil || g.nativePreparationUI != original || g.nativeClassUI != class {
		t.Fatal("faithful preparation was adapted")
	}
}

func TestModernHDTitleLoadSlotsClearCoveredTextAndRejectInvalidSource(t *testing.T) {
	t.Setenv("FD2_NATIVE_SAVE", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	assets, err := loadNativeLoadSlotsUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	_, glyphs, err := loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{font: loadFont(), nativeLoadSlotsUI: assets, nativeBattleGlyphs: glyphs,
		modernStoryPortraits: &modernStoryPortraitSet{hasHD: true}}
	screen := ebiten.NewImage(logicalW, logicalH)
	done := g.beginModernHDText(screen)
	defer done()
	for slot := 0; slot < 4; slot++ {
		g.titleSlotSel = slot
		g.modernHDText.runs = []modernHDTextRun{{text: "covered"}}
		if !g.drawNativeLoadSlots(screen) || len(g.modernHDText.runs) == 0 {
			t.Fatal("empty native slots missing HD text", slot)
		}
		for _, run := range g.modernHDText.runs {
			if run.text == "covered" || run.clip.Empty() {
				t.Fatal("full frame retained covered text or missing clip")
			}
		}
		if g.nativeLoadSlotsUI != assets || g.font.indexedHD != nil {
			t.Fatal("title mutated shared font")
		}
	}
	before := len(g.modernHDText.runs)
	g.titleSlotSel = 4
	if g.drawNativeLoadSlots(screen) || len(g.modernHDText.runs) != before {
		t.Fatal("invalid selector published a partial frame")
	}
	g.titleSlotSel = 0
	if err := os.WriteFile(saveSlotPath(0), []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if g.drawNativeLoadSlots(screen) || len(g.modernHDText.runs) != before {
		t.Fatal("invalid metadata published HD text")
	}
}
