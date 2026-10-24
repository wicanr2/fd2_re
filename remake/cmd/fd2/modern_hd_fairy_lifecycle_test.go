package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestModernHDFairyRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_FAIRY_GPU") != "1" {
		t.Skip("dedicated fairy GPU lifecycle required")
	}
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Fatal("explicit fairy catalog and pack required")
	}
	t.Setenv("FD2_MUTE", "1")
	g, actor, target := nativeCommand33PresentationFixture(t)
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	entry := set.battleEffects[66]
	if entry == nil {
		t.Fatal("missing complete effect66")
	}
	g.modernStoryPortraits = set
	work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	saved := entry.paths[13]
	entry.paths[13] = filepath.Join(t.TempDir(), "missing.png")
	if err := g.startNativeCommand33Presentation(actor, target, nil); err == nil || g.nativeCmd33Presentation != nil ||
		actor.HP != 40 || target.HP != 10 || actor.MP != 60 || g.nativeRNGState != 7 ||
		!bytes.Equal(work, g.nativeMapWork) || !bytes.Equal(vga, g.nativeMapVGA) || entry.Frames[0] != nil {
		t.Fatal("missing last fairy frame partially published transaction or art")
	}
	entry.paths[13] = saved
	probe := &modernHDCompoundProbe{g: g, actor: actor, target: target, command: 33,
		before: [4]int{int(g.nativeRNGState), actor.HP, actor.MP, target.HP},
		out:    os.Getenv("FD2_TEST_MODERN_EFFECT_OUT"), shots: make(map[string]bool)}
	if probe.out != "" {
		if err := os.Mkdir(probe.out, 0755); err != nil {
			t.Fatal(err)
		}
	}
	callbacks := 0
	if err := g.startNativeCommand33Presentation(actor, target, func(result battle.NativeCompoundCommand33Result) {
		if len(result.Targets) != 2 {
			t.Errorf("fairy callback targets=%d", len(result.Targets))
		}
		callbacks++
		probe.finished = true
	}); err != nil {
		t.Fatal(err)
	}
	owner := g.nativeCmd33Presentation
	probe.seen = make([]bool, len(owner.frames))
	probe.presentation = &modernHDPresentation{Game: g}
	for i := 0; i < 3; i++ {
		g.stepNativeCommand33Presentation()
	}
	if owner.frame != 0 || owner.repeat != 0 || owner.published {
		t.Fatal("fairy advanced without Draw")
	}
	defer func() {
		if probe.readback != nil {
			probe.readback.Dispose()
		}
	}()
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if g.modernHDBattleGPU != nil {
		defer g.modernHDBattleGPU.close()
	}
	if probe.err != nil || !probe.finished || callbacks != 1 || g.nativeCmd33Presentation != nil {
		t.Fatalf("fairy lifecycle: %v callbacks%d", probe.err, callbacks)
	}
	for index, seen := range probe.seen {
		if !seen {
			t.Fatalf("fairy present %d skipped", index)
		}
	}
	if actor.HP != actor.MaxHP || target.HP != target.MaxHP || actor.MP != 60 || !actor.Acted ||
		actor.NativeTransient[3] != 0 || target.NativeTransient[5] != 0 ||
		g.nativeRNGState != owner.plan.Result.Restore.RNGState || g.rng.Int63() != rand.New(rand.NewSource(33)).Int63() {
		t.Fatal("fairy transaction, transient reset or RNG differs")
	}
	if err := probe.requireCasterPhases(); err != nil {
		t.Fatal(err)
	}
	if probe.hd == 0 || probe.faithful == 0 || !probe.on || probe.finals == 0 {
		t.Fatal("missing fairy HD/F2/final presentation")
	}
	g.syncModernHDBattleOwner()
	if g.modernHDBattleGPU.owner != nil || len(g.modernHDBattleGPU.sources) != 0 {
		t.Fatal("completed fairy retained GPU textures")
	}
	report := map[string]any{"status": "PASS-SCOPED-RUNTIME-E1", "scope": "既有command33 class19／BattleFig4修改fixture；原生delay及hold保持，非一般玩家、自然存檔或原版oracle。",
		"present_count": len(probe.seen), "hd_draws": probe.hd, "faithful_draws": probe.faithful, "final_draws": probe.finals,
		"continuations": callbacks, "screenshots": probe.shots, "hd_phase_draws": probe.phaseDraws, "phase_f2_switches": probe.phaseSwitches, "source_shots": probe.sourceShots, "actor_hp": actor.HP, "target_hp": target.HP, "mp": actor.MP,
		"native_rng": g.nativeRNGState, "go_rng_unchanged": true}
	if probe.out != "" {
		appendCommandPanelGPUReport(t, report)
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(filepath.Join(probe.out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d original presents; %d HD / %d faithful draws; all fairy phases, F2, atomic missing-last-frame, publication and one continuation; fixture only", len(probe.seen), probe.hd, probe.faithful)
}
