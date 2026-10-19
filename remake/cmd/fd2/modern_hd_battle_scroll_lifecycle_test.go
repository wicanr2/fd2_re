package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Each side is a separate bounded RunGame process. The existing fixture owns
// the native map return; only the battle bank is exchanged for real art.
func TestModernHDScrollRuntimeGPU(t *testing.T) {
	value := os.Getenv("FD2_HD_SCROLL_SIDE")
	if value != "0" && value != "2" {
		t.Skip("dedicated left/right scroll lifecycle required")
	}
	side, _ := strconv.Atoi(value)
	requirePhysicalScenePack(t)
	t.Setenv("FD2_MUTE", "1")
	g, scene, actor, target := modernScrollSceneFixture(t, byte(side))
	set := diagnosticModernBattleSet(t, g, scene)
	real := os.Getenv("FD2_HD_SCROLL_REAL") == "1"
	if real {
		full, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK"))
		if err != nil {
			t.Fatal(err)
		}
		set = &modernStoryPortraitSet{hasHD: true, battleAnimations: full.battleAnimations}
	}
	assets, field, state := completeNativeMapFrameFixture(t)
	dac, palette, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	assets.PaletteDAC, assets.Palette = dac, palette
	g.nativeMapAssets, g.m, g.st, g.sel = assets, field, state, nil
	g.nativeMapDAC = append([]byte(nil), dac...)
	g.baseMapTileset = image.NewRGBA(image.Rect(0, 0, 24, 24))
	probe := &modernHDBattleLifecycleProbe{g: g, scene: scene, actor: actor, target: target, seen: make([]bool, len(scene.preludeImages)+len(scene.body.jobs)), everyPhase: true, phaseDraws: make(map[string]int), phaseF2: make(map[string]int)}
	callbacks := 0
	g.atk = &atkAnim{nativeScene: scene, fpt: 1, after: func() { callbacks++; probe.finished = true }}
	if err := g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	bad := &modernStoryPortraitSet{hasHD: true, battleAnimations: map[int]*modernBattleHDResource{target.BattleFig * 3: {Frames: make([]image.Image, 1), paths: []string{"missing-scroll-frame.png"}, hashes: []string{""}, sizes: []image.Rectangle{image.Rect(0, 0, 6, 6)}}}}
	before := probe.stable()
	if err := g.applyTheme(bad); err == nil || g.modernStoryPortraits != set || probe.stable() != before {
		t.Fatal("failed F2 changed live scroll owner or state")
	}
	for i := 0; i < 3; i++ {
		if err := g.stepAttackPresentationTick(); err != nil {
			t.Fatal(err)
		}
	}
	if scene.preludeFrame != 0 || scene.body.index != 0 {
		t.Fatal("scroll advanced before Draw")
	}
	out := os.Getenv("FD2_TEST_HD_SCROLL_OUT")
	if out != "" {
		if err := os.Mkdir(out, 0755); err != nil {
			t.Fatal(err)
		}
	}
	shots := make(map[string]any)
	var readback *ebiten.Image
	pixels := make([]byte, 1920*1200*4)
	defer func() {
		if readback != nil {
			readback.Dispose()
		}
	}()
	probe.presentation = &modernHDPresentation{Game: g}
	probe.capture = func(screen *ebiten.Image, phase string) {
		if out == "" || g.modernStoryPortraits == nil || !g.modernHDBattleVisible || probe.phaseF2[phase] == 0 || shots[phase] != nil {
			return
		}
		var sources *modernHDBattleSources
		index := scene.body.index
		if phase == "prelude" {
			sources, _ = g.modernHDBattlePreludeSources(scene, scene.prelude[scene.preludeFrame].Stage)
		} else if index < len(scene.body.jobs) {
			sources = scene.body.jobs[index].hdSources
		}
		if sources == nil {
			return
		}
		visible := false
		layers := make([]map[string]any, 0, len(sources.Layers))
		for _, s := range sources.Layers {
			r := image.Rect(s.Frame.X+s.DX, s.Frame.Y+s.DY, s.Frame.X+s.DX+s.Frame.Width, s.Frame.Y+s.DY+s.Frame.Height)
			if s.Resource >= 0 && set.battleAnimations[s.Resource] != nil && s.Fill == -1 && r.In(image.Rect(0, 0, 320, 200)) {
				visible = true
			}
			layers = append(layers, map[string]any{"resource": s.Resource, "frame": s.Index, "rect": r, "clip": s.Clip, "fill": s.Fill})
		}
		if !visible {
			return
		}
		if readback == nil {
			readback = ebiten.NewImage(1920, 1200)
		}
		readback.Clear()
		var geometry ebiten.GeoM
		geometry.Scale(3, 3)
		probe.presentation.DrawFinalScreen(readback, screen, geometry)
		readback.ReadPixels(pixels)
		name := fmt.Sprintf("%s-job-%03d.png", phase, index)
		file, err := os.Create(filepath.Join(out, name))
		if err != nil {
			probe.err = err
			return
		}
		err = png.Encode(file, &image.RGBA{Pix: pixels, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			probe.err = err
			return
		}
		shots[phase] = map[string]any{"file": name, "layers": layers, "job": index}
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if g.modernHDBattleGPU != nil {
		defer g.modernHDBattleGPU.close()
	}
	if probe.err != nil || !probe.finished || callbacks != 1 || g.atk != nil || g.loadErr != "" {
		t.Fatalf("scroll lifecycle: %v callbacks%d error%s", probe.err, callbacks, g.loadErr)
	}
	for i, seen := range probe.seen {
		if i >= len(scene.preludeImages) {
			j := scene.body.jobs[i-len(scene.preludeImages)]
			if len(j.pixels) == 0 && j.present == nil {
				continue
			}
		}
		if !seen {
			t.Fatalf("scroll present%d skipped", i)
		}
	}
	for _, phase := range []string{"prelude", "departure", "scroll-out", "scroll-in", "body", "counter"} {
		if probe.phaseDraws[phase] == 0 || probe.phaseF2[phase] != 1 {
			t.Fatalf("phase%s missing HD or F2", phase)
		}
	}
	for i, seen := range probe.rampSeen {
		if !seen {
			t.Fatalf("battle DAC%d skipped", i)
		}
	}
	for i, seen := range probe.mapRampSeen {
		if !seen {
			t.Fatalf("map DAC%d skipped", i)
		}
	}
	if actor.HP != 80 || target.HP != 80 || g.nativeRNGState != 17791 || g.rng.Int63() != rand.New(rand.NewSource(73)).Int63() {
		t.Fatal("scroll changed HP/RNG")
	}
	if g.nativeMapVGA[4*320+4] != 3 {
		t.Fatal("continuation preceded native map return")
	}
	g.syncModernHDBattleOwner()
	if len(g.modernHDBattleGPU.sources) != 0 || g.modernHDBattleGPU.owner != nil {
		t.Fatal("scroll retained GPU textures")
	}
	report := map[string]any{"status": "PASS-SCOPED-RUNTIME-E1", "side": side, "real_registered_battle_bank": real, "phase_hd_draws": probe.phaseDraws, "phase_f2": probe.phaseF2, "presents": len(probe.seen), "hd_draws": probe.hdDraws, "faithful_draws": probe.faithfulDraws, "final_draws": probe.finals, "continuations": callbacks, "battle_dac_steps": len(probe.rampSeen), "map_dac_steps": len(probe.mapRampSeen), "source_shots": shots, "scope": "真實FIGANI25／288／289與BG55／5的修改fixture；兩次射擊及反擊的正式Game.Draw。非普通玩家或新原版oracle。real只換正式battle銀行，地圖返回沿既有native fixture；F2切換的測試tick停一次以要求同present兩主題Draw。"}
	if out != "" {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("side%d %d presents; %d HD / %d faithful draws, six phase F2, two shots/counter, all DAC and one continuation; fixture only", side, len(probe.seen), probe.hdDraws, probe.faithfulDraws)
}
