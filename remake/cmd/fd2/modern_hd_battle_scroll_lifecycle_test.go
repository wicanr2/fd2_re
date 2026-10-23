package main

import (
	"crypto/sha256"
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
	targetFigure := 96
	if value := os.Getenv("FD2_HD_SCROLL_TARGET"); value != "" {
		if value != "76" && value != "96" {
			t.Fatal("dedicated scroll target must be 76 or 96")
		}
		targetFigure, _ = strconv.Atoi(value)
	}
	g, scene, actor, target := modernScrollTargetSceneFixture(t, byte(side), targetFigure)
	set := diagnosticModernBattleSet(t, g, scene)
	real := os.Getenv("FD2_HD_SCROLL_REAL") == "1"
	if real {
		if os.Getenv("FD2_HD_SCROLL_SCENERY") != "" {
			set = modernHDBattleFixtureSet(t, os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK"), scene.modernResources())
		} else {
			full, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK"))
			if err != nil {
				t.Fatal(err)
			}
			set = &modernStoryPortraitSet{hasHD: true, battleAnimations: full.battleAnimations}
		}
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
	if os.Getenv("FD2_HD_SCROLL_SCENERY") != "" {
		wanted := []struct {
			archive string
			index   byte
		}{{"BG.DAT", scene.selection.BaseBG}, {"BG.DAT", modernPhysicalTargetBGIndex(scene)}, {"TAI.DAT", scene.selection.TAI}}
		if os.Getenv("FD2_HD_TRANSITION_REQUIRED") == "1" {
			for index := 0; index < 3; index++ {
				if set.battleSurfaces["BG.DAT"][index] == nil {
					t.Fatal("required HD transition absent", index)
				}
				wanted = append(wanted, struct {
					archive string
					index   byte
				}{"BG.DAT", byte(index)})
			}
		}
		for _, source := range wanted {
			entry := *set.battleSurfaces[source.archive][int(source.index)]
			entry.picture = nil
			bad := *set
			bad.battleSurfaces = map[string]map[int]*modernBattleSurfaceHD{source.archive: {int(source.index): &entry}}
			before := probe.stable()
			if err := g.applyTheme(&bad); err == nil || g.modernStoryPortraits != set || probe.stable() != before {
				t.Fatal("bad scroll scenery F2 changed live state", source)
			}
		}
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
	sceneryDraws := make(map[string]int)
	panelDraws := make(map[string]int)
	var readback *ebiten.Image
	pixels := make([]byte, 1920*1200*4)
	defer func() {
		if readback != nil {
			readback.Dispose()
		}
	}()
	probe.presentation = &modernHDPresentation{Game: g}
	probe.capture = func(screen *ebiten.Image, phase string) {
		if out == "" || g.modernStoryPortraits == nil || !g.modernHDBattleVisible || probe.phaseF2[phase] == 0 {
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
		if os.Getenv("FD2_HD_PANEL_TEXT") == "1" {
			for _, source := range append(append([]modernHDBattleSource(nil), sources.BaseScenery...), sources.Layers...) {
				if p := source.PanelText; p != nil {
					r := image.Rect(source.Frame.X+source.DX, source.Frame.Y+source.DY, source.Frame.X+source.DX+149, source.Frame.Y+source.DY+42)
					if source.Clip != nil {
						r = r.Intersect(*source.Clip)
					}
					if r.Intersect(image.Rect(0, 0, 320, 200)).Empty() {
						continue
					}
					if g.modernHDBattleGPU.sources["hd/"+p.key] == nil {
						probe.err = fmt.Errorf("visible scroll panel missing GPU texture")
						return
					}
					panelDraws[phase]++
				}
			}
		}
		shotKey := phase
		if os.Getenv("FD2_HD_TRANSITION_REQUIRED") == "1" && (phase == "scroll-in" || phase == "scroll-out") {
			for _, source := range sources.Layers {
				if source.Archive == "BG.DAT" && source.Resource >= 0 && source.Resource < 3 {
					shotKey = fmt.Sprintf("%s-bg%d", phase, source.Resource)
				}
			}
		}
		if os.Getenv("FD2_HD_SCROLL_PROJECTILES") == "1" {
			for _, s := range sources.Layers {
				if s.Resource == 25 && s.Index >= 15 && s.Index <= 18 && s.Fill == -1 {
					shotKey = fmt.Sprintf("%s-bolt-%02d", phase, s.Index)
				}
			}
		}
		if os.Getenv("FD2_HD_SCROLL_TARGET_FRAMES") == "1" {
			for _, s := range sources.Layers {
				if s.Resource == targetFigure*3+1 && s.Index > 0 && s.Fill == -1 {
					shotKey = fmt.Sprintf("%s-target-%02d", phase, s.Index)
				}
			}
		}
		if shots[shotKey] != nil {
			return
		}
		visible := false
		layers := make([]map[string]any, 0, len(sources.Layers))
		for _, s := range append(append([]modernHDBattleSource(nil), sources.BaseScenery...), sources.Layers...) {
			r := image.Rect(s.Frame.X+s.DX, s.Frame.Y+s.DY, s.Frame.X+s.DX+s.Frame.Width, s.Frame.Y+s.DY+s.Frame.Height)
			if s.Resource >= 0 && set.battleAnimations[s.Resource] != nil && s.Fill == -1 && r.In(image.Rect(0, 0, 320, 200)) {
				visible = true
			}
			row := map[string]any{"archive": s.Archive, "resource": s.Resource, "frame": s.Index, "rect": r, "clip": s.Clip, "fill": s.Fill}
			if os.Getenv("FD2_HD_SCROLL_SCENERY") != "" && (s.Archive == "BG.DAT" || s.Archive == "TAI.DAT") {
				entry := set.battleSurfaces[s.Archive][s.Resource]
				if entry == nil && s.Archive == "BG.DAT" && s.Resource >= 0 && s.Resource < 3 && os.Getenv("FD2_HD_TRANSITION_REQUIRED") != "1" {
					row["native_fallback"] = true
				} else if entry == nil || g.modernHDBattleGPU.sources["hd/"+entry.hash] == nil {
					probe.err = fmt.Errorf("Game.Draw lacks actual HD scenery texture %s/%d", s.Archive, s.Resource)
					return
				} else {
					row["sha256"] = entry.hash
					sceneryDraws[entry.hash]++
				}
			}
			layers = append(layers, row)
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
		name := fmt.Sprintf("%s-job-%03d.png", shotKey, index)
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
		shots[shotKey] = map[string]any{"file": name, "layers": layers, "job": index}
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
	if os.Getenv("FD2_HD_SCROLL_PROJECTILES") == "1" {
		for frame := 15; frame <= 18; frame++ {
			if shots[fmt.Sprintf("body-bolt-%02d", frame)] == nil {
				t.Fatalf("projectile frame%d missing GPU capture", frame)
			}
		}
	}
	if os.Getenv("FD2_HD_SCROLL_TARGET_FRAMES") == "1" {
		for frame := 1; frame < len(scene.bodyResources.counter.Frames); frame++ {
			if shots[fmt.Sprintf("counter-target-%02d", frame)] == nil {
				t.Fatalf("target attack frame%d missing GPU capture", frame)
			}
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
	if os.Getenv("FD2_HD_PANEL_TEXT") == "1" {
		for _, phase := range []string{"prelude", "departure", "scroll-out", "scroll-in", "body", "counter"} {
			if panelDraws[phase] == 0 {
				t.Fatal("missing visible HD panel phase", phase)
			}
		}
		report["panel_text"] = map[string]any{"locale": g.localeID, "frozen_snapshots": len(scene.hdPanelText), "actual_gpu_phase_draws": panelDraws}
	}
	report["target_figure"] = targetFigure
	report["target_attack_frames_captured"] = os.Getenv("FD2_HD_SCROLL_TARGET_FRAMES") == "1"
	if pair := os.Getenv("FD2_HD_SCROLL_SCENERY"); pair != "" {
		catalog, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"))
		if err != nil {
			t.Fatal(err)
		}
		report["catalog_sha256"] = fmt.Sprintf("%x", sha256.Sum256(catalog))
		report["scenery_pair"], report["scene_selection"] = pair, scene.selection
		var surfaces []map[string]any
		for _, source := range []struct {
			archive string
			index   byte
		}{{"BG.DAT", scene.selection.BaseBG}, {"BG.DAT", modernPhysicalTargetBGIndex(scene)}, {"TAI.DAT", scene.selection.TAI}} {
			entry := set.battleSurfaces[source.archive][int(source.index)]
			if sceneryDraws[entry.hash] == 0 {
				t.Fatal("no actual Game.Draw scenery source", source)
			}
			if g.modernHDBattleGPU.sources["hd/"+entry.hash] != nil {
				t.Fatal("returned owner retained scenery textures")
			}
			surfaces = append(surfaces, map[string]any{"archive": source.archive, "resource": source.index, "sha256": entry.hash})
		}
		report["scenery_sources"] = surfaces
		report["scenery_actual_capture_draws"] = sceneryDraws
		if os.Getenv("FD2_HD_TRANSITION_REQUIRED") == "1" {
			var transitions []map[string]any
			for index := 0; index < 3; index++ {
				entry := set.battleSurfaces["BG.DAT"][index]
				if entry == nil || sceneryDraws[entry.hash] == 0 {
					t.Fatal("no actual transition draw", index)
				}
				transitions = append(transitions, map[string]any{"archive": "BG.DAT", "resource": index, "sha256": entry.hash})
			}
			report["transition_sources"] = transitions
		}
		report["scope"] = "明示三組地形的分離背景物理fixture，兩次射擊與強制反擊；原生過渡BG0..2保持，六階段F2與正式Game.Draw；不是正常玩家或原版RGB對拍。"
	}
	report["scope"] = fmt.Sprintf("真實FIGANI25／%d／%d與BG55／5的修改fixture；兩次射擊及強制反擊計畫的正式Game.Draw。非普通玩家或新原版oracle；地圖返回沿native fixture，六階段F2測試tick停一次以要求同present兩主題Draw。", targetFigure*3, targetFigure*3+1)
	if out != "" {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("side%d %d presents; %d HD / %d faithful draws, six phase F2, two shots/counter, all DAC and one continuation; fixture only", side, len(probe.seen), probe.hdDraws, probe.faithfulDraws)
}
