package main

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func systemHDTestState(t *testing.T, g *Game, kind int) *nativeSystemEndTurnUIState {
	t.Helper()
	s := &nativeSystemEndTurnUIState{source: append([]byte(nil), g.nativeMapVGA...), portraits: g.nativePreparationUI.portraits}
	s.exitProgram, s.saveCurrent, s.loadCurrent, s.groupMarch = kind == 1, kind == 2, kind == 3, kind == 4
	u := g.nativePreparationUI
	var err error
	s.dialogue, err = campaign.ComposeNativePreparationConfirmationDialogue(s.source, u.dialogue, u.portrait)
	if err != nil {
		t.Fatal(err)
	}
	q, r := systemConfirmationProducer(s)
	s.question, err = q(s.dialogue, u.portrait, u.status.Strings, u.status.Font)
	if err != nil {
		t.Fatal(err)
	}
	s.accepted, err = r(s.question, u.status.Strings, u.status.Font, true)
	if err != nil {
		t.Fatal(err)
	}
	s.canceled, err = r(s.question, u.status.Strings, u.status.Font, false)
	if err != nil {
		t.Fatal(err)
	}
	if !g.freezeModernHDSystemConfirmation(s) {
		t.Fatal(g.loadErr)
	}
	return s
}

func systemHDTestGame(t *testing.T) *Game {
	t.Helper()
	g := newPreparationRecordTestGame(t)
	g.camp = nil
	g.st = &battle.State{W: 24, H: 24}
	var err error
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.font = loadFont()
	_, g.nativeBattleGlyphs, err = loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.interfaceMode = playerInterfaceHD
	g.nativeMapVGA = make([]byte, 64000)
	return g
}

func TestModernHDSystemConfirmationNativeProducers(t *testing.T) {
	g := systemHDTestGame(t)
	for kind := 0; kind < 5; kind++ {
		s := systemHDTestState(t, g, kind)
		h := s.hd
		u := g.nativePreparationUI
		opening, err := campaign.NativePreparationConfirmationOpeningFrames(s.source, s.dialogue, s.question, u.choices)
		if err != nil {
			t.Fatal(err)
		}
		closing, err := campaign.NativeClassConfirmationClosingFrames(s.question, u.choices)
		if err != nil {
			t.Fatal(err)
		}
		blank, err := campaign.NativeClassListClosingFrames(s.source, s.dialogue)
		if err != nil {
			t.Fatal(err)
		}
		for phase, frames := range map[fullRewardPhase][][]byte{fullRewardOpening: opening, fullRewardChoiceClosing: closing, fullRewardBlankClosing: blank} {
			for i, want := range frames {
				got, _, err := h.frame(s, s.source, u.status.Font, phase, i, 0, 0, nil)
				if err != nil || !bytes.Equal(got, want) {
					t.Fatal(kind, phase, i, err)
				}
			}
		}
		for _, accepted := range []bool{false, true} {
			s.acceptedOutcome = accepted
			frames := s.canceled
			if accepted {
				frames = s.accepted
			}
			for i, want := range frames {
				got, _, err := h.frame(s, s.source, u.status.Font, fullRewardResponse, i, 0, 0, nil)
				if err != nil || !bytes.Equal(got, want) {
					t.Fatal(kind, "response", i, err)
				}
			}
		}
		g.nativeSystemEndTurnUI = s
		g.nativeSystemEndTurnConfirm = true
		g.nativeClassUIJob = nil
		screen := ebiten.NewImage(640, 400)
		finish := g.beginModernHDText(screen)
		if !g.drawModernHDSystemConfirmation(screen) || len(g.modernHDText.runs) == 0 || len(g.modernHDText.choices) != 2 || len(g.modernHDText.panels) != 1 {
			t.Fatal("missing HD question", kind, g.loadErr)
		}
		finish()
		screen.Dispose()
	}
}

func TestModernHDSystemConfirmationAtomicRejection(t *testing.T) {
	for _, fault := range []string{"question", "response", "source", "choices", "dialogue", "portrait", "timeline", "restore", "font", "glyphs", "palette"} {
		t.Run(fault, func(t *testing.T) {
			g := systemHDTestGame(t)
			s := systemHDTestState(t, g, 0)
			g.nativeSystemEndTurnUI = s
			g.nativeSystemEndTurnConfirm = true
			g.interfaceMode = playerInterfaceOriginal
			g.modernThemeCache = &modernStoryPortraitSet{}
			if err := g.setInterfaceMode(playerInterfaceHD); err != nil {
				t.Fatal("positive interface switch", err)
			}
			if err := g.setInterfaceMode(playerInterfaceOriginal); err != nil {
				t.Fatal(err)
			}
			old := s.hd
			switch fault {
			case "question":
				s.question[0] ^= 1
			case "response":
				s.accepted[0][0] ^= 1
			case "source":
				s.source[0] ^= 1
			case "choices":
				g.nativePreparationUI.choices[48].Pixels[0] ^= 1
			case "dialogue":
				g.nativePreparationUI.dialogue[1].Pixels[0] ^= 1
			case "portrait":
				s.portraits[0].Pixels[0] ^= 1
			case "timeline":
				g.nativeClassUIJob = &nativeClassUIJob{frames: [][]byte{s.source}}
				s.hd.job = g.nativeClassUIJob
			case "restore":
				frames, _ := campaign.NativeClassListClosingFrames(s.source, s.dialogue)
				g.nativeClassUIJob = &nativeClassUIJob{frames: frames, restore: make([]byte, 1)}
				s.hd.job, s.hd.kind = g.nativeClassUIJob, fullRewardBlankClosing
			case "font":
				g.font = nil
			case "glyphs":
				g.nativeBattleGlyphs = nil
			case "palette":
				g.nativeClassUI.palette[0] = nil
			}
			if err := g.setInterfaceMode(playerInterfaceHD); err == nil || g.interfaceMode != playerInterfaceOriginal || s.hd != old || g.modernStoryPortraits != nil {
				t.Fatal("bad switch published", fault, err)
			}
		})
	}
}

func TestModernHDSystemConfirmationWorldAndSwitches(t *testing.T) {
	if os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG") == "" {
		t.Skip("explicit real theme required")
	}
	set := modifierHDTestTheme(t)
	checkSystemConfirmationWorldAndSwitches(t, set)
}

func checkSystemConfirmationWorldAndSwitches(t *testing.T, set *modernStoryPortraitSet) {
	t.Helper()
	g, _ := rewardStageGame(t, set)
	g.interfaceMode = playerInterfaceHD
	g.modernThemeCache = set
	s := systemHDTestState(t, g, 0)
	g.nativeSystemEndTurnUI = s
	g.nativeSystemEndTurnConfirm = true
	if s.hd.world == nil || s.hd.scene == nil || !s.hd.scene.sealed {
		t.Fatal("world missing")
	}
	before := rewardStageNativeState(g)
	for _, art := range []*modernStoryPortraitSet{nil, set} {
		for _, mode := range []string{playerInterfaceOriginal, playerInterfaceHD} {
			if err := g.applyTheme(art); err != nil {
				t.Fatal(err)
			}
			if err := g.setInterfaceMode(mode); err != nil {
				t.Fatal(err)
			}
			if (s.hd.world != nil) != (art != nil) || g.modernStoryPortraits != art || !reflect.DeepEqual(before, rewardStageNativeState(g)) {
				t.Fatal("switch changed world or native state", mode)
			}
		}
	}
	for _, fault := range []string{"world", "work", "stage", "portrait"} {
		old := s.hd
		var restore func()
		switch fault {
		case "world":
			s.hd.scene.vga[0] ^= 1
			restore = func() { s.hd.scene.vga[0] ^= 1 }
		case "work":
			s.hd.scene.work[0] ^= 1
			restore = func() { s.hd.scene.work[0] ^= 1 }
		case "stage":
			s.hd.scene.stage.staging[0] ^= 1
			restore = func() { s.hd.scene.stage.staging[0] ^= 1 }
		case "portrait":
			old := set.portraits[75]
			delete(set.portraits, 75)
			restore = func() { set.portraits[75] = old }
		}
		if err := g.applyTheme(set); err == nil || s.hd != old || !reflect.DeepEqual(before, rewardStageNativeState(g)) {
			t.Fatal("invalid world published", fault, err)
		}
		restore()
	}
}
