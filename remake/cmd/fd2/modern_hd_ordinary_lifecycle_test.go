package main

import (
	"encoding/json"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"image"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type modernHDOrdinaryProbe struct {
	g                                                                 *Game
	actor, target                                                     *battle.Unit
	job                                                               ordinaryRemainingTestOwner
	presentation                                                      *modernHDPresentation
	command                                                           int
	out                                                               string
	seen                                                              map[string]bool
	phases, switches                                                  map[string]int
	sourceShots                                                       map[string]any
	packedShaders                                                     map[int]bool
	pending                                                           string
	draws, updates, faithful, finals, callbacks, mpChanges, hpChanges int
	lastMP, lastHP                                                    int
	finished                                                          bool
	err                                                               error
}

func (p *modernHDOrdinaryProbe) stable() [7]int {
	phase, frame, black := p.job.position()
	flag := 0
	if black {
		flag = 1
	}
	return [7]int{phase, frame, flag, int(p.g.nativeRNGState), p.actor.HP, p.actor.MP, p.target.HP}
}
func (p *modernHDOrdinaryProbe) Update() error {
	p.updates++
	if p.err != nil || p.finished {
		return ebiten.Termination
	}
	if p.updates > 1800 {
		p.err = fmt.Errorf("ordinary lifecycle exceeded bound")
		return ebiten.Termination
	}
	f := p.job.modern()
	if f == nil {
		p.err = fmt.Errorf("ordinary sources unavailable")
		return ebiten.Termination
	}
	phase := f.hdPhase
	if p.pending != "" {
		if p.faithful == 0 {
			return nil
		}
		before := p.stable()
		p.g.cycleTheme()
		if p.g.modernStoryPortraits == nil || before != p.stable() {
			p.err = fmt.Errorf("F2 restore changed state")
			return ebiten.Termination
		}
		p.switches[p.pending]++
		p.pending, p.faithful = "", 0
		return nil
	}
	if p.phases[phase] > 0 && p.switches[phase] == 0 {
		before := p.stable()
		p.g.cycleTheme()
		if p.g.modernStoryPortraits != nil || before != p.stable() {
			p.err = fmt.Errorf("F2 faithful changed state")
			return ebiten.Termination
		}
		p.pending = phase
		return nil
	}
	if p.actor.MP != p.lastMP {
		p.mpChanges++
		p.lastMP = p.actor.MP
	}
	if p.target.HP != p.lastHP {
		p.hpChanges++
		p.lastHP = p.target.HP
	}
	p.job.step()
	return nil
}
func (p *modernHDOrdinaryProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *modernHDOrdinaryProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	p.presentation.DrawFinalScreen(screen, offscreen, geometry)
	p.finals++
}
func (p *modernHDOrdinaryProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.finished {
		return
	}
	f := p.job.modern()
	if f == nil {
		p.err = fmt.Errorf("ordinary present unavailable")
		return
	}
	p.g.Draw(screen)
	if p.g.loadErr != "" || !p.job.acknowledged() {
		p.err = fmt.Errorf("Game.Draw failed: %s", p.g.loadErr)
		return
	}
	if p.g.modernStoryPortraits == nil {
		if p.g.modernHDBattleVisible {
			p.err = fmt.Errorf("faithful mode leaked HD")
		}
		p.faithful++
		return
	}
	if !p.g.modernHDBattleVisible {
		p.err = fmt.Errorf("ordinary HD missing in %s", f.hdPhase)
		return
	}
	p.draws++
	p.phases[f.hdPhase]++
	_, frame, _ := p.job.position()
	label := fmt.Sprintf("%s-%03d", f.hdPhase, frame)
	p.seen[label] = true
	if p.command == 6 && f.hdSources != nil {
		for _, layer := range f.hdSources.Layers {
			if layer.PackedShader != 0 {
				p.packedShaders[layer.PackedShader] = true
			}
		}
	}
	if p.out == "" || p.sourceShots[label+".png"] != nil {
		return
	}
	final := ebiten.NewImage(1920, 1200)
	defer final.Dispose()
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	p.presentation.DrawFinalScreen(final, screen, geometry)
	pixels := make([]byte, 1920*1200*4)
	final.ReadPixels(pixels)
	file, err := os.Create(filepath.Join(p.out, label+".png"))
	if err != nil {
		p.err = err
		return
	}
	err = png.Encode(file, &image.RGBA{Pix: pixels, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		p.err = err
		return
	}
	shot := map[string]any{"phase": f.hdPhase, "subtract": f.hdSubtract, "mix": f.hdMix, "layers": modernBattleAuditLayers(f.hdSources)}
	if p.command == 6 {
		palette, err := modernHDBattlePalette(f.hdPalette)
		if err != nil {
			p.err = err
			return
		}
		values := make([]int, len(palette))
		for index, value := range palette {
			values[index] = int(value)
		}
		shot["palette_rgba"] = values
	}
	p.sourceShots[label+".png"] = shot
}

func TestModernHDOrdinaryRemainingRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_ORDINARY_GPU") != "1" {
		t.Skip("dedicated ordinary GPU process required")
	}
	command, err := strconv.Atoi(os.Getenv("FD2_TEST_ORDINARY_COMMAND"))
	if err != nil || (command != 2 && command != 3 && command != 6 && command != 7 && command != 8) {
		t.Fatal("explicit ordinary command required")
	}
	t.Setenv("FD2_MUTE", "1")
	g, actor, target := ordinaryRemainingHDTestGame(t, command)
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	g.modernStoryPortraits = set
	p := &modernHDOrdinaryProbe{g: g, actor: actor, target: target, command: command, out: os.Getenv("FD2_TEST_MODERN_EFFECT_OUT"),
		seen: make(map[string]bool), phases: make(map[string]int), switches: make(map[string]int), sourceShots: make(map[string]any), packedShaders: make(map[int]bool), lastMP: actor.MP, lastHP: target.HP}
	if p.out != "" {
		if err := os.Mkdir(p.out, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := startOrdinaryRemainingHD(g, command, actor, target, func([]battle.NativeCommandDamageResult) { p.callbacks++; p.finished = true }); err != nil {
		t.Fatal(err)
	}
	p.job, p.presentation = ordinaryRemainingOwner(g, command), &modernHDPresentation{Game: g}
	for i := 0; i < 3; i++ {
		p.job.step()
	}
	_, frame, _ := p.job.position()
	if frame != 0 || actor.MP != p.lastMP || target.HP != p.lastHP {
		t.Fatal("owner advanced without Draw")
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if g.modernHDBattleGPU != nil {
		defer g.modernHDBattleGPU.close()
	}
	if p.err != nil || !p.finished || p.callbacks != 1 || p.job.active() {
		t.Fatalf("lifecycle: %v callbacks%d", p.err, p.callbacks)
	}
	phases := []string{"prelude", "actor", "actor-pulse", "handler-front", "target", "handler-tail"}
	if command != 6 {
		phases = append(phases, "tail")
	}
	for _, phase := range phases {
		if p.phases[phase] == 0 || p.switches[phase] != 1 {
			t.Fatalf("missing HD/F2 in %s", phase)
		}
	}
	if len(p.job.plan.Results) > 1 && (p.phases["transition"] == 0 || p.switches["transition"] != 1) {
		t.Fatal("missing multi-target HD/F2 transition")
	}
	stages := map[int]int{2: 6, 3: 13, 6: 5, 7: 5, 8: 16}[command]
	for _, result := range p.job.plan.Results {
		if result.Target.HP != result.HPAfter {
			t.Fatal("target settlement differs")
		}
	}
	if actor.MP != p.job.plan.MPAfter || !actor.Acted || g.nativeRNGState != p.job.plan.RNGAfter ||
		g.rng.Int63() != rand.New(rand.NewSource(32)).Int63() || p.mpChanges != 1 || p.hpChanges != stages {
		t.Fatal("numeric publication, RNG or marker count differs")
	}
	for phase, frames := range map[string][]nativeCompoundPresentedFrame{"prelude": p.job.hd.opening.prelude, "actor": p.job.hd.opening.actor, "handler": p.job.hd.handler, "tail": p.job.hd.tail} {
		for i, f := range frames {
			label := fmt.Sprintf("%s-%03d", phase, i)
			if phase == "handler" {
				label = fmt.Sprintf("%s-%03d", f.hdPhase, i)
			}
			if !p.seen[label] {
				t.Fatalf("unseen %s frame%d", phase, i)
			}
		}
	}
	g.syncModernHDBattleOwner()
	if g.modernHDBattleGPU.owner != nil || len(g.modernHDBattleGPU.sources) != 0 {
		t.Fatal("completed ordinary retained textures")
	}
	report := map[string]any{"status": fmt.Sprintf("PASS-ORDINARY-COMMAND%d-FIXTURE-RUNTIME-E1", command), "command_id": command, "continuations": p.callbacks,
		"go_rng_unchanged": true, "actor_figure": actor.BattleFig, "target_figure": target.BattleFig, "hd_draws": p.draws, "final_draws": p.finals,
		"hd_phase_draws": p.phases, "phase_f2_switches": p.switches, "source_shots": p.sourceShots, "mp_publications": p.mpChanges, "hp_publications": p.hpChanges, "rng": g.nativeRNGState,
		"scope": "同次正式owner的完整Game.Draw fixture，明示FIGANI及命令資料；非正常玩家、原版oracle或PLAYER-E2。"}
	if command == 6 {
		if len(p.packedShaders) != 7 {
			t.Fatal("seven packed shader states were not presented")
		}
		values := make([]int, 0, 7)
		for shade := 2; shade <= 8; shade++ {
			shader := shade<<8 | 0xb0
			if !p.packedShaders[shader] {
				t.Fatal("missing packed shade")
			}
			values = append(values, shader)
		}
		report["packed_shaders"] = values
		report["packed_colour_contract"] = "native guide + four-nearest RGB colour delta; alpha preserved; modern-art approximation"
	}
	if p.out != "" {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(filepath.Join(p.out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("command%d: %d HD draws, all phases/F2, MP1/HP%d and one continuation", command, p.draws, stages)
}
