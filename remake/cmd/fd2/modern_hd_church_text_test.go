package main

import (
	"bytes"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func prepareHDChurchText(t *testing.T, g *Game, screen *ebiten.Image) func() {
	t.Helper()
	g.font = loadFont()
	if g.font == nil {
		t.Fatal("maintained image requires Noto CJK")
	}
	_, glyphs, err := loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeBattleGlyphs = glyphs
	g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true}
	return g.beginModernHDText(screen)
}

func checkHDChurchFrame(t *testing.T, g *Game, screen *ebiten.Image, compose func() ([]byte, bool), draw func(*ebiten.Image) bool) {
	t.Helper()
	legacy, ok := compose()
	if !ok {
		t.Fatalf("%s legacy composition failed", g.churchMode)
	}
	shared, originalFont := g.nativeClassUI, g.nativeClassUI.font
	if !draw(screen) || len(g.modernHDText.runs) == 0 {
		t.Fatalf("%s has no HD text", g.churchMode)
	}
	if g.nativeClassUI != shared || shared.font != originalFont || g.font.indexedHD != nil {
		t.Fatal("HD adapter leaked outside Draw")
	}
	positions := map[[2]float64]bool{}
	gold := 0
	for _, run := range g.modernHDText.runs {
		if run.inkTop {
			if run.y < 220 {
				gold++
			}
			continue
		}
		if run.color == modernHDNativeTextColor(shared.palette, 0x4c) {
			continue
		}
		pos := [2]float64{run.x, run.y}
		if positions[pos] {
			t.Fatalf("%s retained covered text at %v", g.churchMode, pos)
		}
		positions[pos] = true
	}
	if gold != 8 {
		t.Fatalf("%s has %d gold digits, want 8", g.churchMode, gold)
	}
	after, ok := compose()
	if !ok || !bytes.Equal(legacy, after) {
		t.Fatalf("%s changed faithful indexed output", g.churchMode)
	}
}

func TestModernHDChurchClassListAndConfirmationKeepGoldAndDropPrompt(t *testing.T) {
	g, screen := newNativeChurchClassPathGame(t)
	done := prepareHDChurchText(t, g, screen)
	defer done()
	checkHDChurchFrame(t, g, screen, g.composeNativeClassListFrame, g.drawNativeClassList)
	enterNativeChurchClassConfirmation(t, g, screen)
	checkHDChurchFrame(t, g, screen, g.composeNativeClassConfirmationFrame, g.drawNativeClassConfirmation)
}

func TestModernHDChurchRevivePanelsKeepGoldAndDropPrompt(t *testing.T) {
	g, screen := newNativeChurchRevivePathGame(t, 20)
	done := prepareHDChurchText(t, g, screen)
	defer done()
	checkHDChurchFrame(t, g, screen, g.composeNativeChurchReviveListFrame, g.drawNativeChurchReviveList)
	enterNativeChurchReviveConfirmation(t, g, screen)
	checkHDChurchFrame(t, g, screen, g.composeNativeChurchReviveConfirmation, g.drawNativeChurchReviveConfirmation)
	g.handleNativeChurchReviveInput(nativeChurchReviveInput{enter: true})
	drainNativeChurchReviveJobs(t, g, screen)
	if g.churchMode != "revive_insufficient" {
		t.Fatal("did not reach normal insufficient branch")
	}
	checkHDChurchFrame(t, g, screen, g.composeNativeChurchReviveMessage, g.drawNativeChurchReviveMessage)
	g.churchMode = "revive_empty"
	checkHDChurchFrame(t, g, screen, g.composeNativeChurchReviveMessage, g.drawNativeChurchReviveMessage)
	g.churchMode = "menu"
	checkHDChurchFrame(t, g, screen, g.composeNativeChurchMenuFrame, g.drawNativeChurchMenu)
}

func TestModernHDChurchTransferPanelsKeepGoldAndDropPrompt(t *testing.T) {
	g := newNativeChurchTransferPathGame(t)
	screen := ebiten.NewImage(logicalW, logicalH)
	done := prepareHDChurchText(t, g, screen)
	defer done()
	checkHDChurchFrame(t, g, screen, g.composeNativeChurchRosterFrame, g.drawNativeChurchRoster)
	g.handleNativeChurchTransferInput(nativeChurchTransferInput{enter: true})
	drainNativeChurchTransferJobs(t, g)
	checkHDChurchFrame(t, g, screen, g.composeNativeChurchTransferItemFrame, g.drawNativeChurchTransferItem)
	g.handleNativeChurchTransferInput(nativeChurchTransferInput{enter: true})
	drainNativeChurchTransferJobs(t, g)
	checkHDChurchFrame(t, g, screen, g.composeNativeChurchRosterFrame, g.drawNativeChurchRoster)
	g.partyRoster[9] = setNativeTransferInventory(g.partyRoster[9], 0, 0, 0, 0, 0, 0, 0, 0)
	g.handleNativeChurchTransferInput(nativeChurchTransferInput{delta: 1})
	g.handleNativeChurchTransferInput(nativeChurchTransferInput{enter: true})
	drainNativeChurchTransferJobs(t, g)
	if g.churchMode != "transfer_full" {
		t.Fatal("did not reach normal full branch")
	}
	checkHDChurchFrame(t, g, screen, g.composeNativeChurchTransferFullMessage, g.drawNativeChurchTransferFull)
}

func TestModernHDChurchRosterLocalesUseExistingNamesAndBoundedRuns(t *testing.T) {
	g := newNativeChurchTransferPathGame(t)
	screen := ebiten.NewImage(logicalW, logicalH)
	done := prepareHDChurchText(t, g, screen)
	defer done()
	for _, locale := range []string{"zh-Hant", "zh-Hans", "ja", "en"} {
		entities, err := loadOfficialLocaleEntities(locale)
		if err != nil {
			t.Fatal(err)
		}
		g.localeID, g.localeEntities = locale, entities
		checkHDChurchFrame(t, g, screen, g.composeNativeChurchRosterFrame, g.drawNativeChurchRoster)
		for _, run := range g.modernHDText.runs {
			if run.clip.Empty() || run.clip.Min.X < 0 || run.clip.Min.Y < 0 || run.clip.Max.X > 640 || run.clip.Max.Y > 400 {
				t.Fatalf("%s unbounded text %+v", locale, run)
			}
		}
	}
}
