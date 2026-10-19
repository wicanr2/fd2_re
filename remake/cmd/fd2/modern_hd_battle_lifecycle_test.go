package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"math/rand"
	"os"
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
}

func (p *modernHDBattleLifecycleProbe) stable() [5]int {
	return [5]int{p.scene.preludeFrame, p.scene.body.index, int(p.g.nativeRNGState), p.actor.HP, p.target.HP}
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
	if (!p.offDone && p.draws >= 3) || (p.offDone && !p.onDone && p.faithfulDraws > 0) {
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
	} else {
		if p.g.modernHDBattleVisible {
			p.err = fmt.Errorf("faithful theme retained HD layer")
			return
		}
		p.faithfulDraws++
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
	scene, err := g.prepareNativePhysicalScene(actor, target)
	if err != nil {
		t.Fatal(err)
	}
	scene.bodyResources.counter, err = figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3+1)
	if err != nil {
		t.Fatal(err)
	}
	scene.bodyResources.counterSounds, err = g.nativePhysicalSoundBank(scene.bodyResources.counter)
	if err != nil {
		t.Fatal(err)
	}
	result := battle.NativePhysicalAttackResult{AttackStrikes: []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 7, Crit: true, Status: true}, DefenderHP: 73}},
		Counter: &battle.AttackResult{}, CounterStrikes: []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 9}, DefenderHP: 71}}}
	if err := g.attachNativePhysicalBody(scene, result); err != nil {
		t.Fatal(err)
	}
	set := diagnosticModernBattleSet(t, g, scene)
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
	g.atk = &atkAnim{nativeScene: scene, fpt: 1, after: func() { probe.finished = true }}
	if err := g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	// 主題切換的壞圖源不能發布，也不能推進演出。
	bad := &modernStoryPortraitSet{hasHD: true, battleAnimations: map[int]*modernBattleHDResource{0: {Frames: make([]image.Image, 1), paths: []string{"missing-frame.png"}, hashes: []string{""}, sizes: []image.Rectangle{image.Rect(0, 0, 6, 6)}}}}
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
	ebiten.SetWindowSize(1920, 1200)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if g.modernHDBattleGPU != nil {
		defer g.modernHDBattleGPU.close()
	}
	if probe.err != nil || !probe.finished || g.atk != nil || g.loadErr != "" {
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
	t.Logf("Game.Draw + final1920 layer: %d HD / %d faithful / %d final draws; all presents, 64 battle fade and 65 map return DAC stages; F2 and failed preflight preserve HP/RNG; source-exact projection fixture only", probe.hdDraws, probe.faithfulDraws, probe.finals)
}
