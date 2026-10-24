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

type modernHDCommand5Probe struct {
	g                                                                 *Game
	actor, target                                                     *battle.Unit
	owner                                                             *nativeCommand5PresentationJob
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

func (p *modernHDCommand5Probe) stable() [7]int {
	j := p.owner
	black := 0
	if j.pulseBlack {
		black = 1
	}
	return [7]int{int(j.phase), j.frame, black, int(p.g.nativeRNGState), p.actor.HP, p.actor.MP, p.target.HP}
}

func (p *modernHDCommand5Probe) Update() error {
	p.updates++
	if p.err != nil || p.finished {
		return ebiten.Termination
	}
	if p.updates > 1600 {
		p.err = fmt.Errorf("ordinary command5 lifecycle exceeded bound")
		return ebiten.Termination
	}
	f := p.owner.modernFrame()
	if f == nil {
		p.err = fmt.Errorf("ordinary command5 sources unavailable")
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
	p.g.stepNativeCommand5Presentation()
	return nil
}
func (p *modernHDCommand5Probe) Layout(int, int) (int, int) { return 640, 400 }
func (p *modernHDCommand5Probe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	p.presentation.DrawFinalScreen(screen, offscreen, geometry)
	p.finals++
}
func (p *modernHDCommand5Probe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.finished {
		return
	}
	f := p.owner.modernFrame()
	if f == nil {
		p.err = fmt.Errorf("ordinary command5 present unavailable")
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

func TestModernHDCommand5RuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_COMMAND5_GPU") != "1" {
		t.Skip("dedicated ordinary command5 GPU process required")
	}
	t.Setenv("FD2_MUTE", "1")
	g, actor, target := ordinaryCommand5HDTestGame(t)
	initOrdinarySceneryFixture(t, g, actor, target)
	command := 5
	if os.Getenv("FD2_HD_COMMAND_PANEL_TEXT") == "1" && os.Getenv("FD2_HD_COMMAND_PANEL_OWNER") == "4" {
		command = 4
		record := g.st.NativeCommandBook[5]
		record.ID = 4
		g.st.NativeCommandBook[4] = record
	}
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	g.modernStoryPortraits = set
	probe := &modernHDCommand5Probe{g: g, actor: actor, target: target, out: os.Getenv("FD2_TEST_MODERN_EFFECT_OUT"),
		seen: make(map[string]bool), phases: make(map[string]int), switches: make(map[string]int), sourceShots: make(map[string]any), lastMP: actor.MP, lastHP: target.HP}
	if probe.out != "" {
		if err := os.Mkdir(probe.out, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.startNativeCommand45Presentation(actor, target, command, func([]battle.NativeCommandDamageResult) { probe.callbacks++; probe.finished = true }); err != nil {
		t.Fatal(err)
	}
	probe.owner, probe.presentation = g.nativeCmd5Presentation, &modernHDPresentation{Game: g}
	for i := 0; i < 3; i++ {
		g.stepNativeCommand5Presentation()
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
	if probe.err != nil || !probe.finished || probe.callbacks != 1 || g.nativeCmd5Presentation != nil {
		t.Fatalf("lifecycle: %v callbacks%d", probe.err, probe.callbacks)
	}
	for _, phase := range []string{"prelude", "actor", "actor-pulse", "handler-front", "target", "handler-tail", "tail"} {
		if probe.phases[phase] == 0 || probe.switches[phase] != 1 {
			t.Fatalf("missing HD/F2 in %s", phase)
		}
	}
	if actor.MP != probe.owner.plan.MPAfter || target.HP != probe.owner.plan.Results[0].HPAfter || !actor.Acted || g.nativeRNGState != probe.owner.plan.RNGAfter ||
		g.rng.Int63() != rand.New(rand.NewSource(32)).Int63() || probe.mpChanges != 1 || probe.hpChanges != 6 {
		t.Fatal("numeric publication, RNG or marker count differs")
	}
	for phase, count := range map[string]int{"prelude": len(probe.owner.prelude), "actor": len(probe.owner.actorSpecs), "handler": len(probe.owner.handler), "tail": len(probe.owner.tail)} {
		for i := 0; i < count; i++ {
			label := fmt.Sprintf("%s-%03d", phase, i)
			if phase == "handler" {
				label = fmt.Sprintf("%s-%03d", probe.owner.hd.handler[i].hdPhase, i)
			}
			if !probe.seen[label] {
				t.Fatalf("unseen %s frame%d", phase, i)
			}
		}
	}
	g.syncModernHDBattleOwner()
	if g.modernHDBattleGPU.owner != nil || len(g.modernHDBattleGPU.sources) != 0 {
		t.Fatal("completed command0 retained textures")
	}
	report := map[string]any{"status": fmt.Sprintf("PASS-ORDINARY-COMMAND%d-FIXTURE-RUNTIME-E1", command), "command_id": command, "continuations": probe.callbacks, "go_rng_unchanged": true,
		"actor_figure": actor.BattleFig, "target_figure": target.BattleFig,
		"hd_draws": probe.draws, "final_draws": probe.finals, "hd_phase_draws": probe.phases, "phase_f2_switches": probe.switches, "source_shots": probe.sourceShots,
		"mp_publications": probe.mpChanges, "hp_publications": probe.hpChanges, "rng": g.nativeRNGState,
		"scope": "沿既有原始素材及明示BattleFig建構raw placement與command5資料的完整Game.Draw fixture；非正常玩家、原版oracle或PLAYER-E2。"}
	if probe.out != "" {
		appendCommandPanelGPUReport(t, report)
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(filepath.Join(probe.out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d HD draws; full command5 phases/F2, MP1/HP6 and one continuation", probe.draws)
}

func ordinaryCommand5HDTestGame(t *testing.T) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	g, actor, target := ordinaryCommand0HDTestGame(t)
	g.st.NativeCommandBook[5] = battle.NativeCommandRecord{ID: 5, Damage: 50, Hit: 100, SelectionMode: 1, EffectMode: 1, MPCost: 2, TargetCode: 0}
	return g, actor, target
}
