package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type modernHDCommand0Probe struct {
	g                                                                 *Game
	actor, target                                                     *battle.Unit
	owner                                                             *nativeCommand0PresentationJob
	presentation                                                      *modernHDPresentation
	out                                                               string
	seen                                                              map[string]bool
	phases, switches                                                  map[string]int
	sourceShots                                                       map[string]any
	pending                                                           string
	draws, updates, faithful, finals, callbacks, mpChanges, hpChanges int
	lastMP, lastHP                                                    int
	finished                                                          bool
	err                                                               error
}

func (p *modernHDCommand0Probe) stable() [7]int {
	j := p.owner
	black := 0
	if j.pulseBlack {
		black = 1
	}
	return [7]int{int(j.phase), j.frame, black, int(p.g.nativeRNGState), p.actor.HP, p.actor.MP, p.target.HP}
}

func (p *modernHDCommand0Probe) Update() error {
	p.updates++
	if p.err != nil || p.finished {
		return ebiten.Termination
	}
	if p.updates > 1600 {
		p.err = fmt.Errorf("ordinary command0 lifecycle exceeded bound")
		return ebiten.Termination
	}
	f := p.owner.modernFrame()
	if f == nil {
		p.err = fmt.Errorf("ordinary command0 sources unavailable")
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
	p.g.stepNativeCommand0Presentation()
	return nil
}
func (p *modernHDCommand0Probe) Layout(int, int) (int, int) { return 640, 400 }
func (p *modernHDCommand0Probe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	p.presentation.DrawFinalScreen(screen, offscreen, geometry)
	p.finals++
}
func (p *modernHDCommand0Probe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.finished {
		return
	}
	f := p.owner.modernFrame()
	if f == nil {
		p.err = fmt.Errorf("ordinary command0 present unavailable")
		return
	}
	p.g.Draw(screen)
	if err := auditCommandPanelGPU(p.g, f); err != nil {
		p.err = err
		return
	}
	if p.g.loadErr != "" || !p.owner.drawn {
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
		p.err = fmt.Errorf("registered command0 HD layer missing in %s", f.hdPhase)
		return
	}
	p.draws++
	p.phases[f.hdPhase]++
	label := fmt.Sprintf("%s-%03d", f.hdPhase, p.owner.frame)
	p.seen[label] = true
	if p.out == "" || p.sourceShots[label+".png"] != nil || !wantCommandPanelCapture(f) {
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
	p.sourceShots[label+".png"] = map[string]any{"phase": f.hdPhase, "subtract": f.hdSubtract, "mix": f.hdMix, "layers": modernBattleAuditLayers(f.hdSources)}
}

func TestModernHDCommand0RuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_COMMAND0_GPU") != "1" {
		t.Skip("dedicated ordinary command0 GPU process required")
	}
	t.Setenv("FD2_MUTE", "1")
	g, actor, target := ordinaryCommand0HDTestGame(t)
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	g.modernStoryPortraits = set
	probe := &modernHDCommand0Probe{g: g, actor: actor, target: target, out: os.Getenv("FD2_TEST_MODERN_EFFECT_OUT"),
		seen: make(map[string]bool), phases: make(map[string]int), switches: make(map[string]int), sourceShots: make(map[string]any), lastMP: actor.MP, lastHP: target.HP}
	if probe.out != "" {
		if err := os.Mkdir(probe.out, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.startNativeCommand0Presentation(actor, target, func([]battle.NativeCommandDamageResult) { probe.callbacks++; probe.finished = true }); err != nil {
		t.Fatal(err)
	}
	probe.owner, probe.presentation = g.nativeCmd0Presentation, &modernHDPresentation{Game: g}
	for i := 0; i < 3; i++ {
		g.stepNativeCommand0Presentation()
	}
	if probe.owner.frame != 0 || probe.owner.mpPublished {
		t.Fatal("owner advanced without Draw")
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if g.modernHDBattleGPU != nil {
		defer g.modernHDBattleGPU.close()
	}
	if probe.err != nil || !probe.finished || probe.callbacks != 1 || g.nativeCmd0Presentation != nil {
		t.Fatalf("lifecycle: %v callbacks%d", probe.err, probe.callbacks)
	}
	for _, phase := range []string{"prelude", "actor", "actor-pulse", "target", "tail"} {
		if probe.phases[phase] == 0 || probe.switches[phase] != 1 {
			t.Fatalf("missing HD/F2 in %s", phase)
		}
	}
	if actor.MP != probe.owner.plan.MPAfter || target.HP != probe.owner.plan.Results[0].HPAfter || !actor.Acted || g.nativeRNGState != probe.owner.plan.RNGAfter ||
		g.rng.Int63() != rand.New(rand.NewSource(32)).Int63() || probe.mpChanges != 1 || probe.hpChanges != 7 {
		t.Fatal("numeric publication, RNG or marker count differs")
	}
	for phase, count := range map[string]int{"prelude": len(probe.owner.prelude), "actor": len(probe.owner.actorSpecs), "target": len(probe.owner.targetFrames), "tail": len(probe.owner.tail)} {
		for i := 0; i < count; i++ {
			if !probe.seen[fmt.Sprintf("%s-%03d", phase, i)] {
				t.Fatalf("unseen %s frame%d", phase, i)
			}
		}
	}
	g.syncModernHDBattleOwner()
	if g.modernHDBattleGPU.owner != nil || len(g.modernHDBattleGPU.sources) != 0 {
		t.Fatal("completed command0 retained textures")
	}
	report := map[string]any{"status": "PASS-ORDINARY-COMMAND0-FIXTURE-RUNTIME-E1", "command_id": 0, "continuations": probe.callbacks, "go_rng_unchanged": true,
		"actor_figure": actor.BattleFig, "target_figure": target.BattleFig,
		"hd_draws": probe.draws, "final_draws": probe.finals, "hd_phase_draws": probe.phases, "phase_f2_switches": probe.switches, "source_shots": probe.sourceShots,
		"mp_publications": probe.mpChanges, "hp_publications": probe.hpChanges, "rng": g.nativeRNGState,
		"scope": "沿既有原始素材及明示BattleFig建構raw placement與command0資料的完整Game.Draw fixture；非正常玩家、原版oracle或PLAYER-E2。"}
	if probe.out != "" {
		appendCommandPanelGPUReport(t, report)
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(filepath.Join(probe.out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d HD draws; full command0 phases/F2, MP1/HP7 and one continuation", probe.draws)
}
