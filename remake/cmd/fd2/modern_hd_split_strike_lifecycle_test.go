package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type splitStrikeCursor struct {
	Prelude, Transition, Target, Effect, Idle, Repeat, IdleRepeat, Shake int
	Drawn, MP, HP                                                        bool
}

type modernHDSplitStrikeProbe struct {
	g                            *Game
	command                      int
	actor                        *battle.Unit
	targets                      []*battle.Unit
	j24                          *nativeCommand24PresentationJob
	j29                          *nativeCommand29PresentationJob
	presentation                 *modernHDPresentation
	rng                          *splitStrikeCountedSource
	planningCalls                int
	seed                         int64
	out                          string
	hashesOnly                   bool
	beforeMP                     int
	beforeHP, afterHP, hpChanges []int
	mpChanges, callbacks         int
	phases, switches             map[string]int
	pending                      string
	faithful, hd, finals, ticks  int
	finished                     bool
	err                          error
	shots                        map[string]map[string]any
	readback                     *ebiten.Image
	pixels                       []byte
}

func (p *modernHDSplitStrikeProbe) cursor() splitStrikeCursor {
	if p.command == 24 {
		j := p.j24
		return splitStrikeCursor{j.preludeFrame, j.transitionFrame, 0, j.frame, j.targetFrame, j.repeat, j.targetRepeat, j.shakeCounter, j.drawn, j.mpPublished, j.damagePublished}
	}
	j := p.j29
	return splitStrikeCursor{j.preludeFrame, j.transitionFrame, j.targetIndex, j.frame, j.idleFrame, j.repeat, j.idleRepeat, j.shakeCounter, j.drawn, j.mpPublished, j.targets[j.targetIndex].damagePublished}
}

func (p *modernHDSplitStrikeProbe) frame() *nativeCompoundPresentedFrame {
	if p.command == 24 {
		return p.j24.modernFrame()
	}
	return p.j29.modernFrame()
}

func (p *modernHDSplitStrikeProbe) stable() []int {
	c := p.cursor()
	v := []int{c.Prelude, c.Transition, c.Target, c.Effect, c.Idle, c.Repeat, c.IdleRepeat, c.Shake,
		p.actor.HP, p.actor.MP, int(p.g.nativeRNGState), p.rng.calls}
	for _, target := range p.targets {
		v = append(v, target.HP, int(target.NativeRecordByte5))
	}
	return v
}

func (p *modernHDSplitStrikeProbe) Update() error {
	p.ticks++
	if p.err != nil || p.finished {
		return ebiten.Termination
	}
	if p.ticks > 2400 {
		p.err = fmt.Errorf("split strike lifecycle exceeded bound")
		return ebiten.Termination
	}
	f := p.frame()
	if f == nil {
		p.err = fmt.Errorf("split strike source frame missing")
		return ebiten.Termination
	}
	if p.pending != "" {
		if p.faithful == 0 {
			return nil
		}
		before := p.stable()
		p.g.cycleTheme()
		if p.g.modernStoryPortraits == nil || !reflect.DeepEqual(before, p.stable()) {
			p.err = fmt.Errorf("F2 restore changed controller/state/RNG")
			return ebiten.Termination
		}
		p.switches[p.pending]++
		p.pending, p.faithful = "", 0
		return nil
	}
	if p.phases[f.hdPhase] > 0 && p.switches[f.hdPhase] == 0 {
		before := p.stable()
		p.g.cycleTheme()
		if p.g.modernStoryPortraits != nil || !reflect.DeepEqual(before, p.stable()) {
			p.err = fmt.Errorf("F2 pixel mode changed controller/state/RNG")
			return ebiten.Termination
		}
		p.pending = f.hdPhase
		return nil
	}
	c := p.cursor()
	mp := p.actor.MP
	hp := make([]int, len(p.targets))
	for i, target := range p.targets {
		hp[i] = target.HP
	}
	if p.command == 24 {
		p.g.stepNativeCommand24Presentation()
	} else {
		p.g.stepNativeCommand29Presentation()
	}
	if mp != p.actor.MP {
		p.mpChanges++
		impact := 4
		if p.command == 29 {
			impact = 6
		}
		if c.Prelude < 9 || c.Transition >= 0 || c.Effect != impact || p.mpChanges != 1 {
			p.err = fmt.Errorf("MP publication outside original actor marker")
		}
	}
	for i, target := range p.targets {
		if hp[i] != target.HP {
			p.hpChanges[i]++
			impact := 10
			if p.command == 29 {
				impact = 14
			}
			if c.Target != i || c.Transition >= 0 || c.Effect != impact || p.hpChanges[i] != 1 {
				p.err = fmt.Errorf("HP publication outside original target marker%d", i)
			}
		}
	}
	if p.g.loadErr != "" || p.rng.calls != p.planningCalls {
		p.err = fmt.Errorf("presentation changed RNG or failed: %s", p.g.loadErr)
	}
	return nil
}

func (p *modernHDSplitStrikeProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *modernHDSplitStrikeProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	p.presentation.DrawFinalScreen(screen, offscreen, geometry)
	p.finals++
}

func (p *modernHDSplitStrikeProbe) saveImage(prefix string, im image.Image) (string, string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, im); err != nil {
		return "", "", err
	}
	sha := fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))
	name := prefix + "-" + sha + ".png"
	if !p.hashesOnly {
		if err := os.WriteFile(filepath.Join(p.out, name), buf.Bytes(), 0644); err != nil {
			return "", "", err
		}
	}
	return name, sha, nil
}

func (p *modernHDSplitStrikeProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.finished {
		return
	}
	f, c := p.frame(), p.cursor()
	if f == nil {
		p.err = fmt.Errorf("missing source frame")
		return
	}
	p.g.Draw(screen)
	if p.g.loadErr != "" || !p.cursor().Drawn {
		p.err = fmt.Errorf("Game.Draw failed: %s", p.g.loadErr)
		return
	}
	if p.g.modernStoryPortraits == nil {
		if p.g.modernHDBattleVisible {
			p.err = fmt.Errorf("pixel mode leaked HD")
		}
		p.faithful++
		return
	}
	if !p.g.modernHDBattleVisible {
		p.err = fmt.Errorf("HD layer absent in %s", f.hdPhase)
		return
	}
	p.hd++
	p.phases[f.hdPhase]++
	index := c.Effect
	if c.Prelude < 9 {
		index = c.Prelude
	} else if c.Transition >= 0 {
		index = c.Transition
	}
	label := fmt.Sprintf("%s-%03d-idle%d-shake%d", f.hdPhase, index, c.Idle, c.Shake)
	if p.shots[label] != nil {
		return
	}
	if p.readback == nil {
		p.readback = ebiten.NewImage(1920, 1200)
		p.pixels = make([]byte, 1920*1200*4)
	}
	p.readback.Clear()
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	p.presentation.DrawFinalScreen(p.readback, screen, geometry)
	p.readback.ReadPixels(p.pixels)
	name, sha, err := p.saveImage("screen", &image.RGBA{Pix: p.pixels, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
	if err != nil {
		p.err = err
		return
	}
	base := image.NewPaletted(image.Rect(0, 0, 320, 200), f.hdPalette)
	copy(base.Pix, f.hdSources.Base)
	baseFile, baseSHA, err := p.saveImage("base", base)
	if err != nil {
		p.err = err
		return
	}
	plan, err := p.g.modernHDBattlePlan(f.hdSources, f.hdPalette, f.hdSubtract)
	if err != nil {
		p.err = err
		return
	}
	layers := make([]map[string]any, 0, len(plan.Layers))
	ordered := append(append([]modernHDBattleSource(nil), f.hdSources.BaseScenery...), f.hdSources.Layers...)
	for i, layer := range plan.Layers {
		source := ordered[i]
		v := map[string]any{"archive": source.Archive, "resource": source.Resource, "frame": source.Index, "hd_sha256": layer.HDKey,
			"x": source.Frame.X, "y": source.Frame.Y, "width": source.Frame.Width, "height": source.Frame.Height,
			"dx": source.DX, "dy": source.DY, "fill": source.Fill,
			"rect": []int{layer.Rect.Min.X, layer.Rect.Min.Y, layer.Rect.Max.X, layer.Rect.Max.Y}}
		if layer.Clip != nil {
			v["clip"] = []int{layer.Clip.Min.X, layer.Clip.Min.Y, layer.Clip.Max.X, layer.Clip.Max.Y}
		}
		if layer.HD == nil {
			im := image.NewRGBA(image.Rect(0, 0, source.Frame.Width, source.Frame.Height))
			for at, index := range source.Frame.Pixels {
				if source.Frame.Mask[at] != 0 {
					value := color.RGBAModel.Convert(f.hdPalette[index]).(color.RGBA)
					im.SetRGBA(at%im.Bounds().Dx(), at/im.Bounds().Dx(), value)
				}
			}
			file, digest, err := p.saveImage("native", im)
			if err != nil {
				p.err = err
				return
			}
			v["native_file"], v["native_sha256"] = file, digest
		} else if source.Resource < 0 {
			p.err = fmt.Errorf("native layer unexpectedly has registered HD")
			return
		}
		layers = append(layers, v)
	}
	p.shots[label] = map[string]any{"phase": f.hdPhase, "cursor": c, "file": name, "sha256": sha,
		"base_file": baseFile, "base_sha256": baseSHA, "subtract": f.hdSubtract, "layers": layers}
}

func TestModernHDSplitStrikeRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SPLIT_STRIKE_GPU") != "1" {
		t.Skip("dedicated split strike lifecycle process required")
	}
	command, err := strconv.Atoi(os.Getenv("FD2_TEST_SPLIT_STRIKE_COMMAND"))
	if err != nil || command != 24 && command != 29 {
		t.Fatal("explicit command24 or29 required")
	}
	g, actor, targets := splitStrikeHDTestGame(t, command)
	var set *modernStoryPortraitSet
	if os.Getenv("FD2_HD_SPLIT_SCENERY") != "" {
		resources := []int{actor.BattleFig * 3, actor.BattleFig*3 + 2}
		for _, target := range targets {
			resources = append(resources, target.BattleFig*3)
		}
		set = modernHDBattleFixtureSet(t, os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"), resources)
	} else {
		set, err = loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"))
		if err != nil {
			t.Fatal(err)
		}
	}
	g.modernStoryPortraits, g.modernThemeCache = set, set
	seed := int64(2)
	if command == 29 {
		seed = 29
	}
	rng := &splitStrikeCountedSource{Source: rand.NewSource(seed)}
	g.rng = rand.New(rng)
	p := &modernHDSplitStrikeProbe{g: g, command: command, actor: actor, targets: targets, rng: rng, seed: seed,
		beforeMP: actor.MP, beforeHP: make([]int, len(targets)), hpChanges: make([]int, len(targets)),
		out: os.Getenv("FD2_TEST_MODERN_EFFECT_OUT"), hashesOnly: os.Getenv("FD2_SPLIT_HASHES_ONLY") == "1",
		phases: make(map[string]int), switches: make(map[string]int), shots: make(map[string]map[string]any)}
	if p.out == "" {
		t.Fatal("fresh output directory required")
	}
	if err := os.Mkdir(p.out, 0755); err != nil {
		t.Fatal(err)
	}
	for i, target := range targets {
		p.beforeHP[i] = target.HP
	}
	if err := startSplitStrikeHD(g, command, actor, targets[0], func(results []battle.NativeCommand24Damage) {
		p.callbacks++
		p.finished = true
		if len(results) != len(targets) {
			p.err = fmt.Errorf("continuation target list differs")
		}
	}); err != nil {
		t.Fatal(err)
	}
	p.planningCalls = rng.calls
	p.j24, p.j29 = g.nativeCmd24Presentation, g.nativeCmd29Presentation
	p.presentation = &modernHDPresentation{Game: g}
	before := p.stable()
	for i := 0; i < 3; i++ {
		if command == 24 {
			g.stepNativeCommand24Presentation()
		} else {
			g.stepNativeCommand29Presentation()
		}
	}
	if !reflect.DeepEqual(before, p.stable()) {
		t.Fatal("owner advanced without Draw")
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.readback != nil {
		defer p.readback.Dispose()
	}
	if g.modernHDBattleGPU != nil {
		defer g.modernHDBattleGPU.close()
	}
	if p.err != nil || !p.finished || p.callbacks != 1 || g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil || !actor.Acted || p.mpChanges != 1 || rng.calls != p.planningCalls {
		t.Fatalf("split strike lifecycle: %v finished%v callbacks%d MP%d", p.err, p.finished, p.callbacks, p.mpChanges)
	}
	for phase, count := range p.phases {
		if count <= 0 || p.switches[phase] != 1 {
			t.Fatalf("missing F2 for%s", phase)
		}
	}
	if len(p.phases) != 2+len(targets)*3 {
		t.Fatalf("phase count%d", len(p.phases))
	}
	for _, count := range p.hpChanges {
		if count != 1 {
			t.Fatal("target HP did not publish once")
		}
	}
	expected := rand.New(rand.NewSource(seed))
	for range targets {
		expected.Intn(100)
	}
	if g.rng.Int63() != expected.Int63() {
		t.Fatal("Go RNG differs from original per-target planning rolls")
	}
	g.syncModernHDBattleOwner()
	if g.modernHDBattleGPU.owner != nil || len(g.modernHDBattleGPU.sources) != 0 {
		t.Fatal("completed owner retained GPU sources")
	}
	after := make([]int, len(targets))
	for i, target := range targets {
		after[i] = target.HP
	}
	catalog, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"status": "PASS-SPLIT-STRIKE-RUNTIME-E1", "command_id": command, "actor_figure": actor.BattleFig,
		"catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(catalog)), "hashes_only": p.hashesOnly,
		"go_seed": seed, "go_rng_planning_calls": p.planningCalls, "go_rng_next_matches_native_plan": true, "native_rng": g.nativeRNGState,
		"mp_before": p.beforeMP, "mp_after": actor.MP, "mp_changes": p.mpChanges,
		"hp_before": p.beforeHP, "hp_after": after, "hp_changes": p.hpChanges,
		"continuations": p.callbacks, "owner_released": true, "gpu_sources_cleared": true,
		"phase_draws": p.phases, "phase_f2": p.switches, "hd_draws": p.hd, "final_draws": p.finals,
		"source_shots": p.shots, "scope": "Existing indexed command24/29 fixtures with real Game.Draw and1920 final GPU; no normal player/original oracle/PLAYER-E2 or manual-art claim."}
	if scenery := os.Getenv("FD2_HD_SPLIT_SCENERY"); scenery != "" {
		report["scenery_triple"], report["scenery_side"] = scenery, os.Getenv("FD2_HD_SPLIT_SCENERY_SIDE")
		used := map[string]int{}
		for _, shot := range p.shots {
			for _, layer := range shot["layers"].([]map[string]any) {
				if layer["archive"] == "BG.DAT" || layer["archive"] == "TAI.DAT" {
					key := layer["hd_sha256"].(string)
					if key == "" {
						t.Fatal("registered scenery missing in actual draw")
					}
					used[key]++
				}
			}
		}
		if len(used) < len(targets)+1 {
			t.Fatal("missing actual scenery sources")
		}
		report["scenery_actual_sources"] = used
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.out, "receipt.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("command%d: %d HD draws, %d complete snapshots, %d F2 phases; MP1 HP%v one continuation", command, p.hd, len(p.shots), len(p.phases), p.hpChanges)
}
