package main

import (
	"image"
	"testing"
	"time"
)

func TestBattleSpeedClockEntersLeavesWithoutJump(t *testing.T) {
	start := time.Unix(100, 0)
	g := &Game{playerBattleSpeed: 2}
	g.setPlayerSpeedAt(3, start)
	g.atk = &atkAnim{}
	g.syncPlayerClockSpeedAt(start.Add(time.Second))
	if !g.playerClock.game.Equal(start.Add(3*time.Second)) || g.effectivePlayerSpeed() != 2 {
		t.Fatal("battle entry jumped clock or multiplied speeds")
	}
	g.setPlayerSpeedAt(4, start.Add(2*time.Second))
	if !g.playerClock.game.Equal(start.Add(5*time.Second)) || g.playerClock.speed != 2 {
		t.Fatal("F8 overrode independent battle speed")
	}
	g.atk = nil
	g.syncPlayerClockSpeedAt(start.Add(3 * time.Second))
	if !g.playerClock.game.Equal(start.Add(7*time.Second)) || g.playerClock.speed != 4 {
		t.Fatal("battle exit jumped or failed to restore")
	}
	if got := g.playerClock.sample(start.Add(4 * time.Second)); !got.Equal(start.Add(11 * time.Second)) {
		t.Fatal(got)
	}
}

func TestBattleSpeedSettingsPreserveOtherPreferencesAndOldDefaults(t *testing.T) {
	old := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = old })
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	saveSettings(settings{BGMSource: "mt32", LocaleID: "en", GameSpeed: 3, FontScale: 1.25, ThemeID: "faithful"})
	if loadSettings().BattleAnimationSpeed != 0 {
		t.Fatal("old settings do not follow F8")
	}
	g := &Game{playerSpeed: 3}
	g.setBattleAnimationSpeed(2)
	g.chooseGeneralSpeed(4)
	s := loadSettings()
	if s.BattleAnimationSpeed != 2 || s.GameSpeed != 4 || s.BGMSource != "mt32" || s.LocaleID != "en" || s.FontScale != 1.25 || s.ThemeID != "faithful" || g.st != nil || g.camp != nil {
		t.Fatal("display setting affected unrelated state", s)
	}
	for _, bad := range []int{-1, 5, 100} {
		s.BattleAnimationSpeed = bad
		saveSettings(s)
		if loadSettings().BattleAnimationSpeed != 0 {
			t.Fatal("invalid setting accepted")
		}
	}
}

func TestBattleSpeedFollowsAndCoversAllPresentationOwners(t *testing.T) {
	owners := []func(*Game){
		func(g *Game) { g.atk = &atkAnim{} }, func(g *Game) { g.nativeHealPresentation = &nativeCommandHealPresentationJob{} },
		func(g *Game) { g.nativeModifierPresentation = &nativeCommandModifierPresentationJob{} }, func(g *Game) { g.nativeAICommandModifier = &nativeAICommandModifierPresentationJob{} },
		func(g *Game) { g.nativeAIItemPresentation = &nativeAIItemPresentationJob{} },
		func(g *Game) { g.nativeCmd0Presentation = &nativeCommand0PresentationJob{} }, func(g *Game) { g.nativeCmd1Presentation = &nativeCommand1PresentationJob{} },
		func(g *Game) { g.nativeCmd2Presentation = &nativeCommand2PresentationJob{} }, func(g *Game) { g.nativeCmd3Presentation = &nativeCommand3PresentationJob{} },
		func(g *Game) { g.nativeCmd5Presentation = &nativeCommand5PresentationJob{} }, func(g *Game) { g.nativeCmd6Presentation = &nativeCommand6PresentationJob{} },
		func(g *Game) { g.nativeCmd7Presentation = &nativeCommand7PresentationJob{} }, func(g *Game) { g.nativeCmd8Presentation = &nativeCommand8PresentationJob{} },
		func(g *Game) { g.nativeCmd9Player = &nativeCommand9PlayerJob{} }, func(g *Game) { g.nativeCmd9AIPresentation = &nativeCommand9AIPresentationJob{} },
		func(g *Game) { g.nativeCmd1012 = &nativeCommand1012Job{} }, func(g *Game) { g.nativeCmd24Presentation = &nativeCommand24PresentationJob{} },
		func(g *Game) { g.nativeCmd29Presentation = &nativeCommand29PresentationJob{} }, func(g *Game) { g.nativeCmd32Presentation = &nativeCommand32PresentationJob{} },
		func(g *Game) { g.nativeCmd33Presentation = &nativeCommand33PresentationJob{} }, func(g *Game) { g.nativeCmd34Presentation = &nativeCommand34PresentationJob{} },
		func(g *Game) { g.nativeCmd35Presentation = &nativeCommand35PresentationJob{} },
		func(g *Game) { g.nativeCh28PostPresent = &nativeCh28PostPresentJob{kind: "action"} },
		func(g *Game) { g.nativeCh28PostPresent = &nativeCh28PostPresentJob{kind: "status"} },
	}
	for general := 1; general <= 4; general++ {
		for independent := 0; independent <= 4; independent++ {
			for i, owner := range owners {
				g := &Game{playerSpeed: general, playerBattleSpeed: independent}
				if g.effectivePlayerSpeed() != general {
					t.Fatal("map speed changed")
				}
				owner(g)
				want := independent
				if want == 0 {
					want = general
				}
				if g.effectivePlayerSpeed() != want {
					t.Fatalf("owner%d speed mismatch", i)
				}
			}
		}
	}
}

func TestBattleSpeedLeavesScriptPresentationAtGeneralRate(t *testing.T) {
	g := &Game{playerSpeed: 1, playerBattleSpeed: 4, nativeCh28PostPresent: &nativeCh28PostPresentJob{kind: "script"}}
	if g.effectivePlayerSpeed() != 1 {
		t.Fatal("battle preference changed story presentation speed")
	}
}

func TestBattleSpeedToolbarConsumesClicksAndKeepsSingleRow(t *testing.T) {
	old := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = old })
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	g := &Game{playerSpeed: 3}
	p := &modernHDPresentation{Game: g, toolbar: true, floatingToolbar: true}
	point := func(r image.Rectangle) image.Point { return r.Min.Add(image.Pt(r.Dx()/2, r.Dy()/2)) }
	p.queueToolbarPoint(point(playerSpeedButton))
	if !p.speedMenuOpen || g.playerSpeedRequested || !p.poster.consumeInput {
		t.Fatal("speed button leaked into game")
	}
	p.queueToolbarPoint(point(playerBattleSpeedChoices[2]))
	if g.playerBattleSpeed != 2 || g.playerSpeed != 3 {
		t.Fatal("battle choice changed general speed")
	}
	p.queueToolbarPoint(point(playerGeneralSpeedChoices[3]))
	if g.playerBattleSpeed != 2 || g.playerSpeed != 4 {
		t.Fatal("general choice erased battle setting")
	}
	p.stepToolbarVisibility(point(playerBattleSpeedChoices[2]), nil)
	if !p.toolbarVisible() {
		t.Fatal("menu hover hid toolbar")
	}
	p.poster.consumeInput = false
	p.queueToolbarPoint(image.Pt(10, 300))
	if p.speedMenuOpen || !p.poster.consumeInput || g.st != nil {
		t.Fatal("dismiss leaked into map")
	}
}
