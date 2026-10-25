package main

import (
	"bytes"
	"image"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func TestModernHDFullRewardQuestion(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g, _ := rewardStageGame(t, itemHDSet(t))
	a := makeNativeDeathRewardKiller(t, g)
	g.pendingNativeDeathRewards = []pendingNativeDeathReward{{killer: a, item: 211}}
	if !g.runPendingNativeDeathRewards(nil) || g.loadErr != "" {
		t.Fatal("full inventory prompt unavailable", g.loadErr)
	}
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	for i := 0; i < 10; i++ {
		done := g.beginModernHDText(screen)
		if !g.drawNativeSystemEndTurn(screen) {
			t.Fatal("prompt not drawn", i)
		}
		if i == 9 && (len(g.modernHDText.runs) == 0 || g.modernHDOverlay == nil) {
			t.Fatal("full inventory question still has native text/portrait")
		}
		done()
		g.stepNativeClassUILifecycle(time.Time{})
	}
	p := g.nativeDeathRewardUI
	questionWrites := 0
	counting := p.hd.ui.status.Font.WithNativePresenter(func(_, _, _ int, _ fdtxt.NativeGlyphStyle) bool { questionWrites++; return false })
	if _, err := campaign.ComposeNativeDeathRewardDiscardQuestion(p.dialogue, p.portrait, p.hd.ui.status.Strings, counting); err != nil {
		t.Fatal(err)
	}
	for i := range p.hd.canceled {
		done := g.beginModernHDText(screen)
		writes := []modernHDGlyphWrite{}
		capture := &modernHDIndexedText{palette: g.nativeClassUI.palette, writes: &writes}
		font := g.modernHDFacilityFont(p.hd.ui.status.Font, capture)
		if _, _, err := p.modernHDFrame(p.source, font, fullRewardResponse, i, 0, 0, capture); err != nil || len(writes) != questionWrites+i+1 {
			t.Fatal("response revealed future glyphs", i, len(writes), questionWrites, err)
		}
		done()
	}
}

func TestModernHDFullRewardGlyphClears(t *testing.T) {
	writes := []modernHDGlyphWrite{{start: 0}, {start: 1, clear: image.Rect(20, 20, 52, 52), background: true}}
	c := &modernHDIndexedText{writes: &writes, runs: []modernHDTextRun{
		{text: "old", clip: image.Rect(18, 20, 52, 54)},
		{text: "new", clip: image.Rect(18, 20, 52, 54)},
	}}
	runs := modernHDWrittenRuns(c)
	for _, r := range runs {
		if r.text == "old" && !r.clip.Intersect(writes[1].clear).Empty() {
			t.Fatal("old glyph survived native background clear")
		}
	}
	if len(runs) != 3 || runs[0].clip != image.Rect(18, 52, 52, 54) || runs[1].clip != image.Rect(18, 20, 20, 52) || runs[2].text != "new" {
		t.Fatal("shadow or new ink clipped incorrectly", runs)
	}
	writes[1].background = false
	if len(modernHDWrittenRuns(c)) != 2 {
		t.Fatal("transparent glyph erased older ink")
	}
}

func TestModernHDFullRewardChromeRejectsBeforeQueue(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	for _, hd := range []bool{false, true} {
		g, _ := rewardStageGame(t, nil)
		a := makeNativeDeathRewardKiller(t, g)
		ui := *g.nativePreparationUI
		ui.dialogue = append([]fdother.RawCell(nil), ui.dialogue...)
		ui.dialogue[13].Pixels = append([]byte(nil), ui.dialogue[13].Pixels...)
		ui.dialogue[13].Pixels[0] ^= 1 // Native compositor accepts this nonuniform fill.
		g.nativePreparationUI = &ui
		g.pendingNativeDeathRewards = []pendingNativeDeathReward{{killer: a, item: 211}}
		before := *a
		seed := g.nativeRNGState
		if hd {
			g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true}
		}
		g.runPendingNativeDeathRewards(nil)
		if !reflect.DeepEqual(before, *a) || g.nativeRNGState != seed {
			t.Fatal("chrome preflight changed inventory or RNG")
		}
		if hd {
			if !strings.Contains(g.loadErr, "background is not uniform") || len(g.pendingNativeDeathRewards) != 1 || g.nativeDeathRewardUI != nil || g.nativeSystemEndTurnUI != nil || g.nativeClassUIJob != nil || g.nativeItemPanel != nil {
				t.Fatal("bad HD chrome published modal or consumed queue", g.loadErr)
			}
		} else if g.loadErr != "" || len(g.pendingNativeDeathRewards) != 0 || g.nativeDeathRewardUI == nil {
			t.Fatal("HD-only source check blocked native mode", g.loadErr)
		}
	}
}

func TestModernHDFullRewardSource(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	set := itemHDSet(t)
	g, _ := rewardStageGame(t, set)
	a := makeNativeDeathRewardKiller(t, g)
	g.pendingNativeDeathRewards = []pendingNativeDeathReward{{killer: a, item: 211}}
	g.runPendingNativeDeathRewards(nil)
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	p := g.nativeDeathRewardUI
	want := healHDCPUWorld(p.hd.world, set)
	g.nativeCh23State.staging[0] ^= 1
	world, err := g.prepareModernHDFullRewardWorld(p, set)
	g.nativeCh23State.staging[0] ^= 1
	if err != nil || !bytes.Equal(want.Pix, healHDCPUWorld(world, set).Pix) {
		t.Fatal("live stage leaked", err)
	}
	active := p.hd.world
	check := func() {
		t.Helper()
		before := rewardStageNativeState(g)
		if err := g.applyTheme(set); err == nil || p.hd.world != active || g.modernStoryPortraits != set || !reflect.DeepEqual(before, rewardStageNativeState(g)) {
			t.Fatal("invalid source partly published", err)
		}
	}
	p.source[0] ^= 1
	check()
	p.source[0] ^= 1
	p.item++
	check()
	p.item--
	p.hd.scene.stage.staging[0] ^= 1
	check()
	p.hd.scene.stage.staging[0] ^= 1
	g.nativeClassUIJob.frames[0][0] ^= 1
	check()
	g.nativeClassUIJob.frames[0][0] ^= 1
	job := p.hd.job
	p.hd.job = nil
	check()
	p.hd.job = job
	frames := job.frames
	job.frames = frames[:1]
	check()
	job.frames = frames
	// Stable choice cells are not in the opening job, but they still belong
	// to the same frozen modal and must be checked before a later F2 publish.
	choices := append([]fdother.RawCell(nil), p.hd.ui.choices...)
	originalChoices := p.hd.ui.choices
	choices[49].Pixels = append([]byte(nil), choices[49].Pixels...)
	choices[49].Pixels[5*24+3] ^= 1
	p.hd.ui.choices = choices
	check()
	p.hd.ui.choices = originalChoices
	bad := *set
	bad.portraits = nil
	if err := g.applyTheme(&bad); err == nil || p.hd.world != active {
		t.Fatal("missing portrait accepted", err)
	}
	if err := g.applyTheme(nil); err != nil || p.hd.world != nil {
		t.Fatal("pixel switch", err)
	}
	if err := g.applyTheme(set); err != nil || !bytes.Equal(want.Pix, healHDCPUWorld(p.hd.world, set).Pix) {
		t.Fatal("HD restoration", err)
	}
	normal, actor := rewardHDTestGame(t)
	actor.BattleFig = 0
	makeNativeDeathRewardKiller(t, normal)
	normal.modernStoryPortraits = set
	normal.pendingNativeDeathRewards = []pendingNativeDeathReward{{killer: actor, item: 211}}
	normal.runPendingNativeDeathRewards(nil)
	if normal.loadErr != "" || normal.nativeDeathRewardUI == nil || normal.nativeDeathRewardUI.hd.world == nil || normal.nativeDeathRewardUI.hd.scene.stage != nil {
		t.Fatal("ordinary map prompt source", normal.loadErr)
	}
}
