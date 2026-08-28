package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestTitleCutScriptMatchesNativeInterleavedSchedule(t *testing.T) {
	wantKinds := []string{
		"publisher", "afm", "scroll", "static", "scroll", "afm", "afm", "scroll",
		"afm", "afm", "scroll", "afm", "scroll", "afm", "scroll",
		"static", "scroll", "hold", "redfade", "redhold", "afm", "titlefade",
	}
	if len(cutScript) != len(wantKinds) {
		t.Fatalf("cut step count = %d, want %d", len(cutScript), len(wantKinds))
	}
	for i, kind := range wantKinds {
		if cutScript[i].kind != kind {
			t.Fatalf("cut step %d kind = %q, want %q", i, cutScript[i].kind, kind)
		}
	}

	wantScroll := [][3]int{
		{535, 450, 153}, {450, 330, 216}, {330, 210, 216},
		{210, 110, 180}, {110, 25, 153}, {25, 10, 27}, {10, 0, 18},
	}
	var gotScroll [][3]int
	for _, step := range cutScript {
		if step.kind == "scroll" {
			gotScroll = append(gotScroll, [3]int{step.scrollFrom, step.scrollTo, step.tick})
		}
	}
	if len(gotScroll) != len(wantScroll) {
		t.Fatalf("scroll segment count = %d, want %d", len(gotScroll), len(wantScroll))
	}
	for i := range wantScroll {
		if gotScroll[i] != wantScroll[i] {
			t.Fatalf("scroll segment %d = %v, want %v", i, gotScroll[i], wantScroll[i])
		}
	}

	wantAFM := [][3]int{
		{3, 5, 1}, {4, 5, 0}, {5, 3, 0}, {6, 5, 0},
		{7, 3, 0}, {8, 5, 0}, {0, 1, 0}, {1, 1, 1},
	}
	var gotAFM [][3]int
	for _, step := range cutScript {
		if step.kind == "afm" {
			skip := 0
			if step.skip {
				skip = 1
			}
			gotAFM = append(gotAFM, [3]int{step.res, step.tick, skip})
		}
	}
	if len(gotAFM) != len(wantAFM) {
		t.Fatalf("AFM step count = %d, want %d", len(gotAFM), len(wantAFM))
	}
	for i := range wantAFM {
		if gotAFM[i] != wantAFM[i] {
			t.Fatalf("AFM step %d = %v, want %v", i, gotAFM[i], wantAFM[i])
		}
	}
}

func TestTitleCutAdvanceRestoresNativeScrollBoundary(t *testing.T) {
	g := &Game{titlePhase: "cutscene", cutIdx: 0, cutFrame: 7, cutTick: 3}
	g.cutAdvance()
	if g.cutIdx != 1 || g.cutFrame != 0 || g.cutTick != 0 {
		t.Fatalf("publisher boundary idx=%d frame=%d tick=%d", g.cutIdx, g.cutFrame, g.cutTick)
	}
	g.cutAdvance()
	if g.cutIdx != 2 || g.scrollY != 535 {
		t.Fatalf("first scroll boundary idx=%d y=%v", g.cutIdx, g.scrollY)
	}
	g.cutIdx = 3 // guardian static → native resumes at the same esi=450
	g.scrollY = 450
	g.cutAdvance()
	if g.cutIdx != 4 || g.scrollY != 450 {
		t.Fatalf("guardian resume idx=%d y=%v", g.cutIdx, g.scrollY)
	}
}

func TestTitlePublisherScheduleAndBrightness(t *testing.T) {
	if got := cutScript[0]; got.kind != "publisher" || got.tick != 119 || got.skip {
		t.Fatalf("publisher step=%+v", got)
	}
	for tick, want := range map[int]float32{-1: 0, 0: 0.125, 7: 1, 8: 1, 110: 1, 111: 1, 118: 0.125, 119: 0} {
		if got := titlePublisherBrightness(tick); got != want {
			t.Fatalf("publisher brightness tick %d=%v, want %v", tick, got, want)
		}
	}
}

func TestTitleLogoPaletteTransitionSchedule(t *testing.T) {
	start := titleLogoTransitionIndex()
	if start < 0 || start+3 >= len(cutScript) {
		t.Fatalf("logo transition start=%d len=%d", start, len(cutScript))
	}
	want := []cutStep{
		{kind: "redfade", tick: 20},
		{kind: "redhold", tick: 6},
		{kind: "afm", res: 1, tick: 1, skip: true},
		{kind: "titlefade", tick: 20},
	}
	for i, expected := range want {
		got := cutScript[start+i]
		if got.kind != expected.kind || got.tick != expected.tick || got.res != expected.res || got.skip != expected.skip {
			t.Fatalf("logo transition step %d=%+v, want %+v", i, got, expected)
		}
	}
	wantPhases := map[int]int{-1: 0, 0: 0, 1: 2, 9: 19, 10: 21, 18: 38, 19: 40, 20: 40}
	for tick, expected := range wantPhases {
		if got := titlePalettePhase(tick); got != expected {
			t.Fatalf("palette phase tick %d=%d, want %d", tick, got, expected)
		}
	}
}

func TestNativeTitlePaletteTransitionsUseIndexedFDOTHERFrames(t *testing.T) {
	path := filepath.Join("../../../org_game/炎龍騎士團/FLAME2", "FDOTHER.DAT")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("玩家未提供 FDOTHER.DAT：%v", err)
	}
	red, title, err := decodeNativeTitlePaletteTransitions(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range red {
		for _, item := range []struct {
			name  string
			frame *image.Paletted
		}{
			{name: "red", frame: red[i]}, {name: "title", frame: title[i]},
		} {
			if item.frame == nil || item.frame.Bounds() != image.Rect(0, 0, 320, 200) {
				t.Fatalf("%s transition frame %d=%v", item.name, i, item.frame)
			}
		}
	}
	if got := red[0].Palette[0]; got == red[19].Palette[0] {
		t.Fatalf("red transition endpoints unexpectedly equal: %v", got)
	}
	if got := title[0].Palette[0]; got == title[19].Palette[0] {
		t.Fatalf("title transition endpoints unexpectedly equal: %v", got)
	}
}

func TestNativeTitlePublisherUsesFixedFDOTHERResources(t *testing.T) {
	path := filepath.Join("../../../org_game/炎龍騎士團/FLAME2", "FDOTHER.DAT")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("玩家自備 FDOTHER.DAT 不存在：%v", err)
	}
	indexed, err := decodeNativeTitlePublisher(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := indexed.Bounds().Dx(); got != 320 {
		t.Fatalf("publisher width=%d", got)
	}
	if got := indexed.Bounds().Dy(); got != 200 {
		t.Fatalf("publisher height=%d", got)
	}
	got := fmt.Sprintf("%x", sha256.Sum256(indexed.Pix))
	const want = "9ffe75b509e191db498528a584e63f4b048075257c0ab4e2ffe6a0140182f7bf"
	if got != want {
		t.Fatalf("publisher indexed sha256=%s, want %s", got, want)
	}
}

func TestSeparatedTitlePublisherMatchesFixedArchive(t *testing.T) {
	archive := filepath.Join("../../../org_game/炎龍騎士團/FLAME2", "FDOTHER.DAT")
	pack := filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22")
	if _, err := os.Stat(archive); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	want, err := decodeNativeTitlePublisher(archive)
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadSeparatedTitlePublisher(pack)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("separated publisher indexed pixels differ")
	}
	for i := 0; i < 256; i++ {
		wr, wg, wb, wa := want.Palette[i].RGBA()
		gr, gg, gb, ga := got.Palette[i].RGBA()
		if wr != gr || wg != gg || wb != gb || wa != ga {
			t.Fatalf("publisher palette index %d differs", i)
		}
	}
}

func TestSeparatedTitlePublisherFailsClosedWithoutPack(t *testing.T) {
	if _, err := loadSeparatedTitlePublisher(t.TempDir()); err == nil {
		t.Fatal("missing publisher pack was accepted")
	}
}

func TestTitleMenuRowsMatchStableDOSBoxOracle(t *testing.T) {
	want := [...]int{164, 173, 182}
	for item, y := range want {
		if got := titleMenuY(item); got != y {
			t.Fatalf("title menu row %d = %d, want %d", item, got, y)
		}
	}
}

func TestTitleCutSkipIsPerStepAndDoesNotGrantEscapeOverride(t *testing.T) {
	g := &Game{titlePhase: "cutscene", cutIdx: 2, cutFrame: 7, cutTick: 3}
	if !g.trySkipTitleCutStep(cutStep{skip: true}, true) {
		t.Fatal("skippable AFM step rejected a pressed key")
	}
	if g.cutIdx != 3 || g.titlePhase != "cutscene" || g.cutFrame != 0 || g.cutTick != 0 {
		t.Fatalf("skipped step state idx=%d phase=%q frame=%d tick=%d", g.cutIdx, g.titlePhase, g.cutFrame, g.cutTick)
	}
	if g.trySkipTitleCutStep(cutStep{skip: false}, true) {
		t.Fatal("non-skippable AFM step accepted a pressed key")
	}
	if g.cutIdx != 3 || g.titlePhase != "cutscene" {
		t.Fatal("rejected skip changed the opening state")
	}
	if g.trySkipTitleCutStep(cutStep{skip: true}, false) {
		t.Fatal("skippable AFM step advanced without input")
	}
}

func TestTitleScrollKeyJumpsToNativeLogoBoundary(t *testing.T) {
	g := &Game{titlePhase: "cutscene", cutIdx: 4, cutFrame: 4, cutTick: 17, scrollY: 441}
	if !g.trySkipTitleCutStep(cutStep{kind: "scroll"}, true) {
		t.Fatal("native scroll key did not enter the logo boundary")
	}
	if g.cutIdx != titleLogoTransitionIndex() || g.cutCur != nil || g.cutFrame != 0 || g.cutTick != 0 {
		t.Fatalf("scroll skip state idx=%d frame=%d tick=%d", g.cutIdx, g.cutFrame, g.cutTick)
	}
}

func TestTitleMenuShotOracleRequiresExplicitOutput(t *testing.T) {
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_SHOT_TITLE_MENU", "1")
	t.Setenv("FD2_SHOT", "")
	t.Setenv("FD2_MUTE", "1")
	g := loadGame()
	if g.loadErr != "FD2_SHOT_TITLE_MENU requires FD2_SHOT and loaded title assets" {
		t.Fatalf("title menu shot guard error=%q", g.loadErr)
	}
}

func TestTitleMenuShotOracleEntersBoundedMenuState(t *testing.T) {
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_SHOT_TITLE_MENU", "1")
	t.Setenv("FD2_SHOT", t.TempDir()+"/title.png")
	t.Setenv("FD2_MUTE", "1")
	g := loadGame()
	if g.loadErr != "" || g.titleAssets == nil || g.titlePhase != "menu" || g.titleSel != 0 || g.titleFlash != 0 {
		t.Fatalf("title menu shot state phase=%q sel=%d flash=%d assets=%v err=%q", g.titlePhase, g.titleSel, g.titleFlash, g.titleAssets != nil, g.loadErr)
	}
}

func TestTitleMenuTraceWrapsAndConfirmsAfterFlash(t *testing.T) {
	var s TitleMenuState
	if s.Step(TitleMenuUp) != TitleMenuNoAction || s.Selection != 2 {
		t.Fatalf("up from zero = selection %d", s.Selection)
	}
	if s.Step(TitleMenuDown) != TitleMenuNoAction || s.Selection != 0 {
		t.Fatalf("down wrap = selection %d", s.Selection)
	}
	s.Step(TitleMenuDown)
	if s.Selection != 1 {
		t.Fatalf("load selection = %d, want 1", s.Selection)
	}
	s.Step(TitleMenuConfirm)
	if s.FlashTicks != 24 {
		t.Fatalf("confirm flash = %d, want 24", s.FlashTicks)
	}
	for i := 0; i < 23; i++ {
		if got := s.Step(TitleMenuTick); got != TitleMenuNoAction {
			t.Fatalf("action fired early at tick %d: %d", i, got)
		}
	}
	if got := s.Step(TitleMenuTick); got != TitleMenuLoadSlots {
		t.Fatalf("final flash action = %d, want load slots", got)
	}
}

func TestTitleMenuThirdSelectionIsNativeContinue(t *testing.T) {
	s := TitleMenuState{Selection: 2}
	s.Step(TitleMenuConfirm)
	for i := 0; i < 23; i++ {
		if got := s.Step(TitleMenuTick); got != TitleMenuNoAction {
			t.Fatalf("action fired early at tick %d: %d", i, got)
		}
	}
	if got := s.Step(TitleMenuTick); got != TitleMenuContinue {
		t.Fatalf("third selection action=%d, want native CONTINUE", got)
	}
}

func TestTitleSlotTraceIsBoundedAndCancelable(t *testing.T) {
	s := TitleSlotState{Selection: 3}
	if got, _, _ := s.Step(TitleSlotDown); got != 3 {
		t.Fatalf("down at last slot wrapped to %d", got)
	}
	for i := 0; i < 3; i++ {
		s.Step(TitleSlotUp)
	}
	if s.Selection != 0 {
		t.Fatalf("up trace selection = %d, want 0", s.Selection)
	}
	if got, confirm, cancel := s.Step(TitleSlotConfirm); got != 0 || !confirm || cancel {
		t.Fatalf("confirm result=(%d,%v,%v)", got, confirm, cancel)
	}
	if got, confirm, cancel := s.Step(TitleSlotCancel); got != 0 || confirm || !cancel {
		t.Fatalf("cancel result=(%d,%v,%v)", got, confirm, cancel)
	}
}
