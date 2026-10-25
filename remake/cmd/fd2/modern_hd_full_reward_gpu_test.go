package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type fullRewardGPUProbe struct {
	t                          *testing.T
	set                        *modernStoryPortraitSet
	g, control                 *Game
	a, ca                      *battle.Unit
	index, calls, controlCalls int
	caseStarted                bool
	rows                       []map[string]any
	seen                       map[string]bool
	err                        error
	out                        string
	final, controlScreen       *ebiten.Image
	pixels, world              []byte
	checker                    unitPresentGPUProbe
	chromeSource               dialogueChromeSource
	chromePoints               int
	legacyChromeRejected       bool
	confirmationRows           []map[string]any
	confirmationSeen           map[string]bool
	confirmationF2             int
	itemChromePoints           int
}

// Derive coverage from the existing native job, independently of the new panel list.
func fullRewardChromeRegion(g *Game) (int, image.Rectangle) {
	if g.nativeDeathRewardUI == nil || g.itemOpen {
		return 0, image.Rectangle{}
	}
	shift := 0
	if job := g.nativeClassUIJob; job != nil {
		if job.frame == len(job.frames) && len(job.restore) != 0 {
			return 0, image.Rectangle{}
		}
		switch g.nativeDeathRewardUI.hd.kind {
		case fullRewardOpening:
			if job.frame < 6 {
				shift = (5 - job.frame) * 13
			}
		case fullRewardBlankClosing, fullRewardCancelClosing:
			shift = (job.frame + 1) * 13
		case fullRewardCancelOpening:
			shift = (5 - job.frame) * 13
		}
	}
	return shift, image.Rect(5, 112+shift, 315, 198+shift)
}

func (p *fullRewardGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *fullRewardGPUProbe) Update() error {
	if p.err != nil || p.index == 5 {
		return ebiten.Termination
	}
	return nil
}

func fullRewardGPUState(g *Game) any {
	choice, selected := -1, -1
	if g.nativeSystemEndTurnUI != nil {
		choice = g.nativeSystemEndTurnUI.choice
	}
	if g.nativeDeathRewardUI != nil {
		selected = g.nativeDeathRewardUI.selected
	}
	raw, err := json.Marshal([]any{rewardStageNativeState(g), g.itemOpen, g.itemClosing, g.itemAnimStep, g.itemSel, choice, selected, g.nativeSystemEndTurnDelay, g.nativeSystemEndTurnConfirm, g.st.Units})
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func (p *fullRewardGPUProbe) begin() {
	p.a = makeNativeDeathRewardKiller(p.t, p.g)
	p.ca = makeNativeDeathRewardKiller(p.t, p.control)
	for i, g := range []*Game{p.g, p.control} {
		game, isControl := g, i == 1
		a := p.a
		if i == 1 {
			a = p.ca
		}
		g.pendingNativeDeathRewards = []pendingNativeDeathReward{{killer: a, item: 211}}
		if p.index == 4 {
			g.pendingNativeDeathRewards = append(g.pendingNativeDeathRewards, pendingNativeDeathReward{killer: a, item: 212})
		}
		var after func()
		after = func() {
			if game.runPendingNativeDeathRewards(after) {
				return
			}
			if isControl {
				p.controlCalls++
			} else {
				p.calls++
			}
		}
		g.runPendingNativeDeathRewards(after)
		if p.index == 0 && g.nativeSystemEndTurnUI != nil {
			g.nativeSystemEndTurnUI.choice = 1
		}
		if g.loadErr != "" {
			p.err = fmt.Errorf("begin: %s", g.loadErr)
			return
		}
	}
	p.caseStarted = true
}

func fullRewardGPUPhase(g *Game) string {
	if g.nativeDeathRewardUI == nil {
		return "return"
	}
	if g.itemOpen {
		if g.itemClosing {
			return "item-close"
		}
		if g.itemAnimStep < 11 {
			return "item-open"
		}
		return "item-choice"
	}
	if j := g.nativeClassUIJob; j != nil {
		return fmt.Sprintf("job-%d", g.nativeDeathRewardUI.hd.kind)
	}
	if g.nativeSystemEndTurnConfirm {
		return "yes-no"
	}
	return "response-hold"
}

func (p *fullRewardGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == 5 {
		return
	}
	if !p.caseStarted {
		p.begin()
		if p.err != nil {
			return
		}
	}
	g, c := p.g, p.control
	phase := fullRewardGPUPhase(g)
	key := fmt.Sprintf("case%d-%s", p.index, phase)
	g.frame++
	c.frame++
	g.Draw(screen)
	c.Draw(p.controlScreen)
	if p.index == 0 && g.modernHDText != nil && len(g.modernHDText.choices) != 0 {
		p.checkConfirmationModes(screen)
		if p.err != nil {
			return
		}
		g.Draw(screen)
	}
	if g.loadErr != "" || c.loadErr != "" || !g.modernHDWorldVisible || !reflect.DeepEqual(fullRewardGPUState(g), fullRewardGPUState(c)) {
		p.err = fmt.Errorf("%s native control / world mismatch %s %s", key, g.loadErr, c.loadErr)
		return
	}
	frame := g.modernHDMapPresented
	if frame == nil {
		p.err = fmt.Errorf("missing HD world %s", key)
		return
	}
	g.modernHDMapGPU.World.ReadPixels(p.world)
	if !p.checker.compare(p.world, healHDCPUWorld(frame, p.set).Pix, key+" CPU/GPU") {
		p.err = p.checker.err
		return
	}
	var geom ebiten.GeoM
	geom.Scale(3, 3)
	presentation := &modernHDPresentation{Game: g}
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geom)
	p.final.ReadPixels(p.pixels)
	digest := sha256.Sum256(p.pixels)
	shift, chromeRect := fullRewardChromeRegion(g)
	chromePoints, err := checkDialogueChromePixels(p.pixels, p.chromeSource, chromeRect)
	wantPanels := 0
	if !chromeRect.Empty() {
		wantPanels = 1
	}
	dialoguePanels := 0
	for _, panel := range g.modernHDText.panels {
		if !panel.status {
			dialoguePanels++
		}
	}
	if err != nil || dialoguePanels != wantPanels {
		p.err = fmt.Errorf("%s dialogue chrome: %v", key, err)
		return
	}
	p.chromePoints += chromePoints
	itemPoints, err := checkItemPanelChromeGamePixels(g, p.pixels, p.chromeSource)
	if err != nil {
		p.err = fmt.Errorf("%s item chrome: %w", key, err)
		return
	}
	p.itemChromePoints += itemPoints
	// The modal never writes above native y=112; item regions have a separate
	// producer, checked by the existing panel tests and native state controller.
	outside := 0
	if !g.itemOpen && phase != "return" {
		for y := 0; y < 112*6; y++ {
			for x := 0; x < 1920; x++ {
				at := (y*1920 + x) * 4
				for ch := 0; ch < 4; ch++ {
					d := int(p.pixels[at+ch]) - int(p.world[at+ch])
					if d < -2 || d > 2 {
						p.err = fmt.Errorf("%s covered HD world at %d,%d", key, x, y)
						return
					}
				}
				outside++
			}
		}
	}
	if phase == "yes-no" || phase == "response-hold" {
		if len(g.modernHDText.runs) == 0 || g.modernHDOverlay == nil || g.modernHDOverlay.portrait != p.set.portraits[p.a.BattleFig] {
			p.err = fmt.Errorf("%s missing HD dialogue", key)
			return
		}
	}
	f2 := !p.seen[key]
	if f2 {
		before := fullRewardGPUState(g)
		if err := g.applyTheme(nil); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible || g.modernHDOverlay != nil {
			p.err = fmt.Errorf("%s retained HD in pixel mode", key)
			return
		}
		if !chromeRect.Empty() && !p.legacyChromeRejected && phase == "yes-no" {
			p.final.Clear()
			presentation.DrawFinalScreen(p.final, screen, geom)
			p.final.ReadPixels(p.pixels)
			if _, err := checkDialogueChromePixels(p.pixels, p.chromeSource, chromeRect); err == nil {
				p.err = fmt.Errorf("native enlarged border passed HD chrome check")
				return
			}
			p.legacyChromeRejected = true
		}
		if err := g.applyTheme(p.set); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		p.final.Clear()
		presentation.DrawFinalScreen(p.final, screen, geom)
		p.final.ReadPixels(p.pixels)
		if digest != sha256.Sum256(p.pixels) || !reflect.DeepEqual(before, fullRewardGPUState(g)) {
			p.err = fmt.Errorf("%s F2 changed output/state", key)
			return
		}
		p.seen[key] = true
	}
	row := map[string]any{"case": p.index, "phase": phase, "f2_roundtrip": f2, "native_control_unchanged": true, "world_sha256": fmt.Sprintf("%x", sha256.Sum256(p.world)), "final_sha256": fmt.Sprintf("%x", digest), "outside_hd_pixels": outside, "text_runs": len(g.modernHDText.runs), "dialogue_chrome_pixels": chromePoints, "dialogue_chrome_shift": shift, "dialogue_chrome_height": chromeRect.Intersect(image.Rect(0, 0, 320, 200)).Dy()}
	row["item_panel_chrome_pixels"], row["item_anim_step"], row["item_closing"] = itemPoints, g.itemAnimStep, g.itemClosing
	itemCapture := p.index == 2 && (phase == "item-open" && (g.itemAnimStep == 0 || g.itemAnimStep == 5 || g.itemAnimStep == 8) || phase == "item-close" && (g.itemAnimStep == 0 || g.itemAnimStep == 6 || g.itemAnimStep == 11))
	if itemCapture || f2 && (p.index == 0 || p.index == 2 || p.index == 3) && (phase == "yes-no" || phase == "response-hold" || phase == "item-choice" || phase == "return") {
		name := key + ".png"
		if itemCapture {
			name = fmt.Sprintf("%s-step%d.png", key, g.itemAnimStep)
		}
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, p.pixels)
		f, err := os.Create(filepath.Join(p.out, name))
		if err != nil {
			p.err = err
			return
		}
		err = png.Encode(f, pic)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			p.err = fmt.Errorf("PNG %v %v", err, closeErr)
			return
		}
		row["capture"] = name
	}
	p.rows = append(p.rows, row)
	if len(p.rows) > 800 {
		p.err = fmt.Errorf("unbounded modal")
		return
	}
	if phase == "return" {
		if p.calls != p.index+1 || p.controlCalls != p.calls {
			p.err = fmt.Errorf("continuation count HD%d native%d case%d", p.calls, p.controlCalls, p.index)
			return
		}
		want := []int{16, 17, 18, 19, 20, 21, 22, 23}
		if p.index == 2 {
			want = []int{16, 17, 18, 20, 21, 22, 23, 211}
		}
		if !reflect.DeepEqual(p.a.Inventory, want) {
			p.err = fmt.Errorf("wrong inventory %v", p.a.Inventory)
			return
		}
		p.index++
		p.caseStarted = false
		return
	}
	for _, game := range []*Game{g, c} {
		if job := game.nativeClassUIJob; job != nil {
			old := job.frame
			job.drawn = false
			game.stepNativeClassUILifecycle(time.Unix(1000, 0))
			if job.frame != old {
				p.err = fmt.Errorf("undrawn frame advanced")
				return
			}
			job.drawn = true
			game.stepNativeClassUILifecycle(time.Unix(1000, 0))
		} else if game.itemOpen {
			if game.itemClosing || game.itemAnimStep < 11 {
				game.stepNativeItemPanelAnimation()
			} else if p.index == 2 {
				for i := 0; i < 3; i++ {
					game.handleNativeDeathRewardItemInput(nativeDeathRewardInput{down: true})
				}
				game.handleNativeDeathRewardItemInput(nativeDeathRewardInput{confirm: true})
			} else {
				game.handleNativeDeathRewardItemInput(nativeDeathRewardInput{cancel: true})
			}
		} else if game.nativeSystemEndTurnConfirm {
			if p.index == 0 {
				game.cancelNativeSystemEndTurn()
			} else if p.index == 1 || p.index == 4 {
				game.cancelNativeSystemEndTurn()
			} else {
				game.confirmNativeSystemEndTurn()
			}
		} else {
			game.stepNativeSystemEndTurn()
		}
	}
}

func TestModernHDFullRewardGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_FULL_REWARD_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	set := itemHDSet(t)
	g, _ := rewardStageGame(t, set)
	control, _ := rewardStageGame(t, nil)
	out := os.Getenv("FD2_TEST_REWARD_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &fullRewardGPUProbe{t: t, set: set, g: g, control: control, out: out, seen: map[string]bool{}, final: ebiten.NewImage(1920, 1200), controlScreen: ebiten.NewImage(640, 400), pixels: make([]byte, 1920*1200*4), world: make([]byte, 1920*1200*4)}
	p.confirmationSeen = map[string]bool{}
	p.chromeSource = nativeDialogueChromeSource(g.nativePreparationUI.dialogue, g.nativeClassUI.palette)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 5 || !p.legacyChromeRejected || len(p.confirmationRows) != 48 || p.confirmationF2 != 16 {
		t.Fatal("incomplete full inventory flow", p.index, p.err)
	}
	d := map[string]any{"status": "PASS-FULL-REWARD-HD-RUNTIME-E1", "cases": 5, "map_index": 23, "native_seed": 7, "fixed_clock_unix": 1000, "rgba_tolerance": 2, "normal_player_verified": false, "rows": p.rows, "game_frames": len(p.rows), "f2_boundaries": len(p.seen), "continuations": p.calls, "dialogue_chrome": true, "chrome_source": p.chromeSource, "chrome_pixels": p.chromePoints, "legacy_chrome_rejected": p.legacyChromeRejected}
	d["hd_confirmation"], d["confirmation_rows"], d["confirmation_f2"] = true, p.confirmationRows, p.confirmationF2
	d["item_panel_chrome"], d["item_panel_chrome_pixels"] = true, p.itemChromePoints
	raw, _ := json.MarshalIndent(d, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
