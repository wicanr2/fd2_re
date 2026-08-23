package main

import "testing"

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
