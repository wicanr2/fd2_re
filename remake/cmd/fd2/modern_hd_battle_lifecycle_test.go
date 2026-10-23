package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// 只把原始索引圖最近鄰投影為HD診斷圖；不是新美術、存檔槽或一般玩家收據。
func diagnosticModernBattleSet(t *testing.T, g *Game, scene *nativePhysicalScene) *modernStoryPortraitSet {
	t.Helper()
	r := scene.bodyResources
	animations := map[int]*figani.Animation{scene.actorBattleFig * 3: r.actorIdle, scene.actorBattleFig*3 + 1: r.attack,
		scene.targetBattleFig * 3: r.targetIdle, scene.targetBattleFig*3 + 1: r.counter}
	set := &modernStoryPortraitSet{hasHD: true, battleAnimations: make(map[int]*modernBattleHDResource)}
	for resource, animation := range animations {
		if animation == nil {
			continue
		}
		entry := &modernBattleHDResource{Frames: make([]image.Image, len(animation.Frames)), hashes: make([]string, len(animation.Frames)), paths: make([]string, len(animation.Frames))}
		for index, frame := range animation.Frames {
			picture := modernHDBattleCandidateIndexed(frame.Width, frame.Height, frame.Pixels, frame.Mask, g.nativeUIPalette)
			entry.Frames[index] = picture
			entry.hashes[index] = fmt.Sprintf("%x", sha256.Sum256(picture.Pix))
		}
		set.battleAnimations[resource] = entry
	}
	// 每一份正式job綁定都要對回真正原始資源與frame，避免主動／反擊選錯圖源。
	for _, job := range scene.body.jobs {
		if job.hdSources == nil {
			continue
		}
		for _, source := range job.hdSources.Layers {
			if source.Resource < 0 {
				continue
			}
			animation := animations[source.Resource]
			if animation == nil || source.Index < 0 || source.Index >= len(animation.Frames) || !reflect.DeepEqual(source.Frame, animation.Frames[source.Index]) {
				t.Fatalf("job source binding differs: %d/%d", source.Resource, source.Index)
			}
		}
	}
	return set
}

type modernHDBattleLifecycleProbe struct {
	g                                                         *Game
	presentation                                              *modernHDPresentation
	scene                                                     *nativePhysicalScene
	actor, target                                             *battle.Unit
	seen                                                      []bool
	rampSeen                                                  [64]bool
	mapRampSeen                                               [65]bool
	offDone, onDone                                           bool
	steps, draws, finals, hdDraws, faithfulDraws, returnDraws int
	finished                                                  bool
	err                                                       error
	everyPhase                                                bool
	phaseDraws, phaseF2                                       map[string]int
	pendingPhase                                              string
	pendingFaithful                                           int
	capture                                                   func(*ebiten.Image, string)
	requiredPedestalHash                                      string
}

type modernHDZeroDelaySource struct {
	Resource int `json:"resource"`
	Frame    int `json:"frame"`
	Delay    int `json:"delay_native"`
}

// 共背景攻擊尾段的delay0沒有inner present；待機與非共背景departure不適用。
// 已閉合契約：physical_tail_evidence.claims[3]，原0x297D0..0x2987F。
func checkModernHDPhysicalCaptures(figure int, animations map[int]*figani.Animation, shots map[string]any, bound map[string]bool) ([]modernHDZeroDelaySource, error) {
	var zero []modernHDZeroDelaySource
	for resource, animation := range animations {
		for index, frame := range animation.Frames {
			key := fmt.Sprintf("%03d-%02d", resource, index)
			if resource == figure*3+1 && animation.HeaderByte1 == 0 && frame.Delay == 0 {
				if bound[key] || shots[key] != nil {
					return nil, fmt.Errorf("zero-delay source%d/%d unexpectedly presented", resource, index)
				}
				zero = append(zero, modernHDZeroDelaySource{Resource: resource, Frame: index})
				continue
			}
			if shots[key] == nil {
				return nil, fmt.Errorf("selected source%d/%d missing GPU capture", resource, index)
			}
		}
	}
	return zero, nil
}

func (p *modernHDBattleLifecycleProbe) stable() [7]int {
	return [7]int{p.scene.preludeFrame, p.scene.body.index, int(p.g.nativeRNGState), p.actor.HP, p.target.HP, p.actor.MP, p.target.MP}
}

func (p *modernHDBattleLifecycleProbe) phase() string {
	if ramp := p.g.nativePaletteRamp; ramp != nil {
		if ramp.hdBattleSources != nil {
			return "fade"
		}
		return "map-fade"
	}
	if p.scene.preludeFrame < len(p.scene.preludeImages) {
		return "prelude"
	}
	i := p.scene.body.index
	if i == len(p.scene.body.jobs) {
		i = p.scene.body.lastPresent
	}
	if i >= 0 && i < len(p.scene.body.jobs) {
		return p.scene.body.jobs[i].hdPhase
	}
	return ""
}

func (p *modernHDBattleLifecycleProbe) Update() error {
	p.steps++
	if p.err != nil || p.finished {
		return ebiten.Termination
	}
	if p.steps > 1800 {
		p.err = fmt.Errorf("lifecycle did not finish")
		return ebiten.Termination
	}
	// F2同一路徑切出及切回；每次切換不改演出游標或已結算狀態。
	if p.everyPhase {
		phase := p.phase()
		restore := p.pendingPhase != "" && p.faithfulDraws > p.pendingFaithful
		eligible := phase == "prelude" || phase == "departure" || phase == "scroll-out" || phase == "scroll-in" || phase == "body" || phase == "counter"
		start := p.pendingPhase == "" && eligible && p.phaseDraws[phase] > 0 && p.phaseF2[phase] == 0
		if restore || start {
			before, elapsed, timer, accum := p.stable(), p.scene.body.elapsedMillis, p.g.atk.timer, p.g.atk.accumMillis
			p.g.cycleTheme()
			if p.stable() != before || p.scene.body.elapsedMillis != elapsed || p.g.atk.timer != timer || p.g.atk.accumMillis != accum {
				p.err = fmt.Errorf("phase F2 changed state")
				return ebiten.Termination
			}
			if restore {
				p.phaseF2[p.pendingPhase]++
				p.pendingPhase = ""
			} else {
				p.pendingPhase = phase
				p.pendingFaithful = p.faithfulDraws
			}
			return nil // explicit test clock pause; allow Draw of this same present.
		}
		if p.pendingPhase != "" {
			return nil
		}
	} else if (!p.offDone && p.draws >= 3) || (p.offDone && !p.onDone && p.faithfulDraws > 0) {
		before := p.stable()
		p.g.cycleTheme()
		if !p.offDone {
			p.offDone = true
		} else {
			p.onDone = true
		}
		if p.stable() != before {
			p.err = fmt.Errorf("F2 changed physical state")
			return ebiten.Termination
		}
	}
	if p.g.nativePaletteRamp != nil {
		p.g.stepNativePaletteRamp()
		return nil
	}
	if p.scene.preludeFrame < len(p.scene.preludeImages) {
		return p.g.stepAttackPresentationTick()
	}
	return p.g.stepNativePhysicalBodyMillis(p.scene, 1000.0/60)
}
func (p *modernHDBattleLifecycleProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *modernHDBattleLifecycleProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	p.presentation.DrawFinalScreen(screen, offscreen, geometry)
	p.finals++
}
func (p *modernHDBattleLifecycleProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.finished {
		return
	}
	p.draws++
	p.g.Draw(screen)
	if p.g.loadErr != "" {
		p.err = fmt.Errorf("Game.Draw: %s", p.g.loadErr)
		return
	}
	mapRamp := p.g.nativePaletteRamp != nil && p.g.nativePaletteRamp.hdBattleSources == nil
	if mapRamp {
		if p.g.modernHDBattleVisible {
			p.err = fmt.Errorf("map return retained battle layer")
			return
		}
	} else if p.g.modernStoryPortraits != nil {
		if !p.g.modernHDBattleVisible {
			rampSources := p.g.nativePaletteRamp != nil && p.g.nativePaletteRamp.hdBattleSources != nil
			p.err = fmt.Errorf("HD layer missing at prelude%d job%d ramp%v rampSources%v wanted%v title%q defeat%v", p.scene.preludeFrame, p.scene.body.index, p.g.nativePaletteRamp != nil, rampSources, p.g.modernHDBattleWanted(p.scene), p.g.titlePhase, p.g.nativeDefeat != nil)
			return
		}
		p.hdDraws++
		if p.requiredPedestalHash != "" && p.g.modernHDBattleGPU.sources["hd/"+p.requiredPedestalHash] == nil {
			p.err = fmt.Errorf("HD pedestal source missing in actual Game.Draw")
			return
		}
		if p.everyPhase {
			p.phaseDraws[p.phase()]++
		}
	} else {
		if p.g.modernHDBattleVisible {
			p.err = fmt.Errorf("faithful theme retained HD layer")
			return
		}
		p.faithfulDraws++
	}
	if p.capture != nil {
		p.capture(screen, p.phase())
	}
	if ramp := p.g.nativePaletteRamp; ramp != nil {
		limit := len(p.rampSeen)
		if mapRamp {
			limit = len(p.mapRampSeen)
		}
		if ramp.step < 0 || ramp.step >= limit || !ramp.drawn {
			p.err = fmt.Errorf("ramp acknowledgment missing: stage%d drawn%v map%v", ramp.step, ramp.drawn, mapRamp)
			return
		}
		if mapRamp {
			if !bytes.Equal(ramp.vga, p.g.nativeMapVGA) {
				p.err = fmt.Errorf("map fade uses different composed frame")
				return
			}
			p.mapRampSeen[ramp.step] = true
		} else {
			p.rampSeen[ramp.step] = true
		}
		return
	}
	index := p.scene.preludeFrame
	if index < len(p.scene.preludeImages) {
		if !p.scene.preludeDrawn {
			p.err = fmt.Errorf("prelude acknowledgment missing")
			return
		}
	} else {
		job := p.scene.body.index
		if job == len(p.scene.body.jobs) {
			job = p.scene.body.lastPresent
			p.returnDraws++
		}
		if job < 0 || job >= len(p.scene.body.jobs) || !p.scene.body.drawn {
			p.err = fmt.Errorf("body acknowledgment missing")
			return
		}
		index = len(p.scene.preludeImages) + job
	}
	p.seen[index] = true
}

func TestModernHDBattleRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_BATTLE_LIFECYCLE_GPU") != "1" {
		t.Skip("dedicated GPU lifecycle required")
	}
	requirePhysicalScenePack(t)
	t.Setenv("FD2_MUTE", "1")
	g, actor, target := physicalSceneTestGame(t)
	real := os.Getenv("FD2_HD_BATTLE_REAL") == "1"
	background := os.Getenv("FD2_HD_BATTLE_BACKGROUND")
	surfaceIndex := surfaceTestIndex(t)
	if background != "" {
		if background != fmt.Sprint(surfaceIndex) || !real {
			t.Fatal("background fixture requires real selected BG")
		}
		g.m.NativeTerrainControl[2] = byte(surfaceIndex)
	}
	idleTarget := os.Getenv("FD2_HD_BATTLE_IDLE_TARGET") == "1"
	enemyActor := os.Getenv("FD2_HD_BATTLE_ENEMY_ACTOR") == "1"
	if idleTarget && !real {
		t.Fatal("idle-target requires explicit real candidate")
	}
	if enemyActor && (!real || idleTarget) {
		t.Fatal("enemy-actor requires a real armed candidate")
	}
	figure := 0
	if real {
		figure = modernBattleCandidateFigure(t)
		if enemyActor {
			// 前向原圖在左側。交換既有敵我單位，不改Camp或原生side；
			// 右側使用已登錄的0，避免和固定左側76重疊。
			actor, target = target, actor
			actor.BattleFig, target.BattleFig = figure, 0
			g.sel, g.curX, g.curY = actor, target.X, target.Y
			if actor.Camp != battle.Enemy || target.Camp != battle.Own || actor.NativeRecordByte6 != 0 || target.NativeRecordByte6 != 2 {
				t.Fatal("enemy-actor fixture changed native camp or side")
			}
		} else if idleTarget {
			// 既有前向劍盾兵在x0..132；後向村民在x200之後，避免兩個右側描述子重疊。
			actor.BattleFig, target.BattleFig = 76, figure
			if g.st.NativePhysicalCounterattackEligible(actor, target) {
				t.Fatal("unarmed idle target unexpectedly has counterattack")
			}
		} else {
			actor.BattleFig, target.BattleFig = figure, 76
		}
	}
	scene, err := g.prepareNativePhysicalScene(actor, target)
	if err != nil {
		t.Fatal(err)
	}
	if background != "" && (scene.selection.HasSeparateBackgrounds || int(scene.selection.BaseBG) != surfaceIndex) {
		t.Fatal("selected BG fixture did not select common background")
	}
	result := battle.NativePhysicalAttackResult{AttackStrikes: []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 7, Crit: true, Status: true}, DefenderHP: 73}}}
	if !idleTarget {
		scene.bodyResources.counter, err = figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3+1)
		if err != nil {
			t.Fatal(err)
		}
		scene.bodyResources.counterSounds, err = g.nativePhysicalSoundBank(scene.bodyResources.counter)
		if err != nil {
			t.Fatal(err)
		}
		result.Counter = &battle.AttackResult{}
		result.CounterStrikes = []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 9}, DefenderHP: 71}}
	} else if scene.bodyResources.counter != nil {
		t.Fatal("idle target loaded an attack bank")
	}
	if err := g.attachNativePhysicalBody(scene, result); err != nil {
		t.Fatal(err)
	}
	set := diagnosticModernBattleSet(t, g, scene)
	if real {
		set = modernHDBattleFixtureSet(t, os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"),
			os.Getenv("FD2_TEST_MODERN_BATTLE_PACK"), scene.modernResources())
	}
	if background != "" {
		if set.battleSurfaces["BG.DAT"][surfaceIndex] == nil {
			t.Fatal("selected BG HD source missing")
		}
		for _, job := range scene.body.jobs {
			if job.hdSources != nil && (len(job.hdSources.BaseScenery) != 2 || job.hdSources.BaseScenery[0].Resource != surfaceIndex || job.hdSources.BaseScenery[1].Archive != "TAI.DAT" || job.hdSources.BaseScenery[1].Resource != surfaceIndex) {
				t.Fatal("body lost selected BG provenance")
			}
		}
	}
	selectedResources := map[int]*figani.Animation{figure * 3: scene.bodyResources.actorIdle, figure*3 + 1: scene.bodyResources.attack}
	phases := []string{"prelude", "body", "counter"}
	if !idleTarget && scene.bodyResources.attack.HeaderByte1 != 0 && scene.bodyResources.attack.HeaderByte2 != 0 {
		phases = append(phases, "departure")
	}
	if idleTarget {
		selectedResources = map[int]*figani.Animation{figure * 3: scene.bodyResources.targetIdle}
		phases = []string{"prelude", "body"}
		for _, job := range scene.body.jobs {
			if job.hdPhase == "counter" {
				t.Fatal("idle-target fixture fabricated counter phase")
			}
			if job.hdSources != nil {
				for _, source := range job.hdSources.Layers {
					if source.Resource == figure*3+1 || source.Resource == figure*3+2 {
						t.Fatal("idle-target fixture references empty attack/cast bank")
					}
				}
			}
		}
	}
	// 沿既有完整地圖fixture測返回composer，戰鬥圖源仍為上面的真正分離資源。
	assets, field, state := completeNativeMapFrameFixture(t)
	dac, palette, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	assets.PaletteDAC, assets.Palette = dac, palette
	g.nativeMapAssets, g.m, g.st, g.sel = assets, field, state, nil
	g.nativeMapDAC = append([]byte(nil), dac...)
	g.baseMapTileset = image.NewRGBA(image.Rect(0, 0, 24, 24))
	probe := &modernHDBattleLifecycleProbe{g: g, scene: scene, actor: actor, target: target, seen: make([]bool, len(scene.preludeImages)+len(scene.body.jobs))}
	if os.Getenv("FD2_HD_BATTLE_PEDESTAL") != "" {
		entry := set.battleSurfaces["TAI.DAT"][surfaceIndex]
		if entry == nil {
			t.Fatal("required selected TAI is absent")
		}
		probe.requiredPedestalHash = entry.hash
	}
	if real {
		probe.everyPhase, probe.phaseDraws, probe.phaseF2 = true, make(map[string]int), make(map[string]int)
	}
	callbacks := 0
	g.atk = &atkAnim{nativeScene: scene, fpt: 1, after: func() { callbacks++; probe.finished = true }}
	if err := g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	if background != "" {
		badSurface := *set
		entry := *set.battleSurfaces["BG.DAT"][surfaceIndex]
		entry.picture = nil
		badSurface.battleSurfaces = map[string]map[int]*modernBattleSurfaceHD{"BG.DAT": {surfaceIndex: &entry}}
		before := probe.stable()
		if err := g.applyTheme(&badSurface); err == nil || g.modernStoryPortraits != set || probe.stable() != before {
			t.Fatal("failed selected BG F2 preflight changed theme or state")
		}
		if probe.requiredPedestalHash != "" {
			entry := *set.battleSurfaces["TAI.DAT"][surfaceIndex]
			entry.picture = nil
			badTAI := *set
			badTAI.battleSurfaces = map[string]map[int]*modernBattleSurfaceHD{"BG.DAT": set.battleSurfaces["BG.DAT"], "TAI.DAT": {surfaceIndex: &entry}}
			if err := g.applyTheme(&badTAI); err == nil || g.modernStoryPortraits != set || probe.stable() != before {
				t.Fatal("failed selected TAI F2 preflight changed state")
			}
		}
	}
	// 主題切換的壞圖源不能發布，也不能推進演出。
	bad := &modernStoryPortraitSet{hasHD: true, battleAnimations: map[int]*modernBattleHDResource{actor.BattleFig * 3: {Frames: make([]image.Image, 1), paths: []string{"missing-frame.png"}, hashes: []string{""}, sizes: []image.Rectangle{image.Rect(0, 0, 6, 6)}}}}
	before := probe.stable()
	if err := g.applyTheme(bad); err == nil || g.modernStoryPortraits != set || probe.stable() != before {
		t.Fatal("failed theme preflight changed live state")
	}
	for i := 0; i < 3; i++ {
		if err := g.stepAttackPresentationTick(); err != nil {
			t.Fatal(err)
		}
	}
	if scene.preludeFrame != 0 || scene.body.index != 0 {
		t.Fatal("owner advanced without Draw")
	}
	probe.presentation = &modernHDPresentation{Game: g}
	out := os.Getenv("FD2_TEST_HD_BATTLE_OUT")
	shots := make(map[string]any)
	if real && out != "" {
		if err := os.Mkdir(out, 0755); err != nil {
			t.Fatal(err)
		}
		readback := ebiten.NewImage(1920, 1200)
		defer readback.Dispose()
		pixels := make([]byte, 1920*1200*4)
		probe.capture = func(screen *ebiten.Image, phase string) {
			if g.modernStoryPortraits == nil || !g.modernHDBattleVisible || (phase != "body" && phase != "counter" && phase != "departure") {
				return
			}
			index := scene.body.index
			if index < 0 || index >= len(scene.body.jobs) || scene.body.jobs[index].hdSources == nil {
				return
			}
			for _, source := range scene.body.jobs[index].hdSources.Layers {
				if _, selected := selectedResources[source.Resource]; !selected || source.Fill != -1 {
					continue
				}
				key := fmt.Sprintf("%03d-%02d", source.Resource, source.Index)
				if shots[key] != nil {
					continue
				}
				_, _, layers := physicalBattleAuditSources(scene)
				readback.Clear()
				var geometry ebiten.GeoM
				geometry.Scale(3, 3)
				probe.presentation.DrawFinalScreen(readback, screen, geometry)
				readback.ReadPixels(pixels)
				name := key + ".png"
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
				raw, err := os.ReadFile(filepath.Join(out, name))
				if err != nil {
					probe.err = err
					return
				}
				state := map[string]any{"battle_scene": true, "hd_battle": true, "battle_source_layers": layers}
				shots[key] = map[string]any{"scene": "fixture", "file": name, "sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "before": state, "after": state, "job": index, "phase": phase}
			}
		}
	}
	ebiten.SetWindowSize(1920, 1200)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if g.modernHDBattleGPU != nil {
		defer g.modernHDBattleGPU.close()
	}
	if probe.err != nil || !probe.finished || callbacks != 1 || g.atk != nil || g.loadErr != "" {
		t.Fatalf("runtime lifecycle: %v finished%v error%s", probe.err, probe.finished, g.loadErr)
	}
	for i, seen := range probe.seen {
		if i >= len(scene.preludeImages) {
			job := scene.body.jobs[i-len(scene.preludeImages)]
			if len(job.pixels) == 0 && job.present == nil {
				continue
			}
		}
		if !seen {
			t.Fatalf("present %d skipped", i)
		}
	}
	for step, seen := range probe.rampSeen {
		if !seen {
			t.Fatalf("return DAC stage %d skipped", step)
		}
	}
	for step, seen := range probe.mapRampSeen {
		if !seen {
			t.Fatalf("map return DAC stage %d skipped", step)
		}
	}
	if probe.hdDraws == 0 || probe.faithfulDraws == 0 || probe.finals == 0 || probe.returnDraws == 0 {
		t.Fatal("missing HD/faithful/final/caller-wait path")
	}
	if actor.HP != 80 || target.HP != 80 || g.nativeRNGState != 17791 || g.rng.Int63() != rand.New(rand.NewSource(73)).Int63() {
		t.Fatal("presentation changed HP or RNG")
	}
	if g.nativeMapVGA[4*320+4] != 3 {
		t.Fatal("continuation preceded native map composition")
	}
	g.syncModernHDBattleOwner()
	if len(g.modernHDBattleGPU.sources) != 0 || g.modernHDBattleGPU.owner != nil {
		t.Fatal("completed scene retained old GPU sources")
	}
	if real && out != "" {
		for _, phase := range phases {
			if probe.phaseDraws[phase] == 0 || probe.phaseF2[phase] != 1 {
				t.Fatalf("phase%s missing HD/F2", phase)
			}
		}
		bound := make(map[string]bool)
		for _, job := range scene.body.jobs {
			if job.hdSources == nil {
				continue
			}
			for _, source := range job.hdSources.Layers {
				bound[fmt.Sprintf("%03d-%02d", source.Resource, source.Index)] = true
			}
		}
		zeroDelaySources, err := checkModernHDPhysicalCaptures(figure, selectedResources, shots, bound)
		if err != nil {
			t.Fatal(err)
		}
		catalog, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"))
		if err != nil {
			t.Fatal(err)
		}
		report := map[string]any{"status": "PASS-CHARACTER-FIXTURE-RUNTIME-E1", "figure": figure, "catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(catalog)), "phase_f2": probe.phaseF2, "phase_hd_draws": probe.phaseDraws, "continuations": callbacks, "battle_dac_steps": len(probe.rampSeen), "map_dac_steps": len(probe.mapRampSeen), "source_shots": shots, "scope": "真實FIGANI、強制一次攻擊與反擊計畫及native地圖fixture；Game.Draw最終1920回讀。非普通玩家或原版oracle。"}
		if background != "" {
			report["background_selection"], report["background_hd_sha256"] = scene.selection, set.battleSurfaces["BG.DAT"][surfaceIndex].hash
			report["background_scope"] = "selected BG shared physical scene; native TAI and UI retained; explicit controlled fixture, not normal player or original RGB parity"
			if probe.requiredPedestalHash != "" {
				report["pedestal_hd_sha256"] = probe.requiredPedestalHash
				report["background_scope"] = "Selected BG and TAI shared physical scene; original UI retained; controlled fixture only"
			}
		}
		report["fixture_asset_load_scope"], report["fixture_requested_resources"] = "selected-battle-banks", scene.modernResources()
		if len(zeroDelaySources) != 0 {
			report["zero_delay_sources"] = zeroDelaySources
		}
		if enemyActor {
			report["role"], report["counterpart_figure"] = "enemy-actor", target.BattleFig
			report["actor_camp"], report["target_camp"] = actor.Camp, target.Camp
			report["actor_native_side"], report["target_native_side"] = actor.NativeRecordByte6, target.NativeRecordByte6
			report["scope"] = "交換既有敵我測試單位後明示換圖源，保持Camp與原生side；左側敵方主攻、右側我方0反擊的Game.Draw最終1920回讀。非普通玩家、自然AI或新原版oracle。"
		}
		if idleTarget {
			report["status"], report["role"] = "PASS-IDLE-TARGET-FIXTURE-RUNTIME-E1", "idle-target"
			report["selected_resources"], report["expected_phases"], report["counter_present"] = []int{figure * 3}, phases, false
			report["scope"] = "真實FIGANI與無武器修改目標，只有主動呈現計畫；四待機幀Game.Draw最終1920回讀、兩階段F2與完整返回。非原版村民規則、普通玩家或原版oracle。"
		}
		raw, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("Game.Draw + final1920 layer: %d HD / %d faithful / %d final draws; all presents, 64 battle fade and 65 map return DAC stages; F2 and failed preflight preserve HP/RNG; realArt%v figure%d; fixture only", probe.hdDraws, probe.faithfulDraws, probe.finals, real, figure)
}
