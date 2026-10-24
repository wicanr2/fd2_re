package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestNativeCompoundStageHoldRequiresNewDraw(t *testing.T) {
	frames := []nativeCompoundPresentedFrame{{delay: 1}, {delay: 1}, {delay: 1}}
	t.Run("command34", func(t *testing.T) {
		j := &nativeCommand34PresentationJob{frames: frames, holding: true, hold: 0, holdStage: 0, stageEnd: [3]int{1, 9, 10}, drawn: true}
		g := &Game{nativeCmd34Presentation: j}
		g.stepNativeCommand34Presentation()
		if j.frame != 1 || j.drawn {
			t.Fatal("hold reused previous frame Draw acknowledgement")
		}
		g.stepNativeCommand34Presentation()
		if j.frame != 1 || j.repeat != 0 || j.published != 0 {
			t.Fatal("next stage advanced without its own Draw")
		}
	})
	t.Run("command35", func(t *testing.T) {
		j := &nativeCommand35PresentationJob{frames: frames, holding: true, hold: 0, holdStage: 0, stageEnd: [3]int{1, 9, 10}, drawn: true}
		g := &Game{nativeCmd35Presentation: j}
		g.stepNativeCommand35Presentation()
		if j.frame != 1 || j.drawn {
			t.Fatal("hold reused previous frame Draw acknowledgement")
		}
		g.stepNativeCommand35Presentation()
		if j.frame != 1 || j.repeat != 0 || j.published != 0 {
			t.Fatal("next stage advanced without its own Draw")
		}
	})
}

// 每個command獨立Go程序，沿既有原版素材fixture，保留所有delay及三段hold。
func TestModernHDStagedSummonRuntimeGPU(t *testing.T) {
	command, _ := strconv.Atoi(os.Getenv("FD2_HD_STAGED_COMMAND"))
	if command != 34 && command != 35 {
		t.Skip("dedicated command34 or command35 GPU lifecycle required")
	}
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Fatal("explicit staged summon catalog and pack required")
	}
	t.Setenv("FD2_MUTE", "1")
	var g *Game
	var actor, target *battle.Unit
	if command == 34 {
		g, actor, target = nativeCommand34PresentationFixture(t)
	} else {
		g, actor, target = nativeCommand35PresentationFixture(t)
	}
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	entry := set.battleEffects[command+33]
	if entry == nil {
		t.Fatal("missing complete staged summon")
	}
	g.modernStoryPortraits = set
	beforeActor, beforeTarget := *actor, *target
	work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	seed := g.nativeRNGState
	probe := &modernHDCompoundProbe{g: g, actor: actor, target: target, command: command,
		before: [4]int{int(seed), actor.HP, actor.MP, target.HP}, out: os.Getenv("FD2_TEST_MODERN_EFFECT_OUT"), shots: make(map[string]bool)}
	start := func(done func()) error {
		if command == 34 {
			return g.startNativeCommand34Presentation(actor, target, func(result battle.NativeCompoundCommand34Result) {
				if len(result.StageStates) != 3 {
					t.Errorf("command34 stage count=%d", len(result.StageStates))
				}
				done()
			})
		}
		return g.startNativeCommand35Presentation(actor, target, func(result battle.NativeCompoundCommand35Result) {
			if len(result.StageStates) != 3 {
				t.Errorf("command35 stage count=%d", len(result.StageStates))
			}
			done()
		})
	}
	last := len(entry.paths) - 1
	saved := entry.paths[last]
	entry.paths[last] = filepath.Join(t.TempDir(), "missing.png")
	if err := start(func() {}); err == nil || g.nativeCmd34Presentation != nil || g.nativeCmd35Presentation != nil ||
		!reflect.DeepEqual(beforeActor, *actor) || !reflect.DeepEqual(beforeTarget, *target) || g.nativeRNGState != seed ||
		!bytes.Equal(work, g.nativeMapWork) || !bytes.Equal(vga, g.nativeMapVGA) || entry.Frames[0] != nil {
		t.Fatal("missing last staged summon frame partially published transaction or art")
	}
	entry.paths[last] = saved
	if probe.out != "" {
		if err := os.Mkdir(probe.out, 0755); err != nil {
			t.Fatal(err)
		}
	}
	callbacks := 0
	if err := start(func() { callbacks++; probe.finished = true }); err != nil {
		t.Fatal(err)
	}
	owner34, owner35 := g.nativeCmd34Presentation, g.nativeCmd35Presentation
	seenStages := [3]bool{}
	probe.check = func() error {
		stage, frame, boundaries := 0, 0, [3]int{}
		if command == 34 {
			stage, frame, boundaries = owner34.published, owner34.frame, owner34.publishAt
		} else {
			stage, frame, boundaries = owner35.published, owner35.frame, owner35.publishAt
		}
		for index, boundary := range boundaries {
			if (index < stage) != (frame >= boundary) {
				return fmt.Errorf("command%d stage%d published away from original Draw boundary", command, index)
			}
		}
		if stage == 0 {
			if !reflect.DeepEqual(beforeActor, *actor) || !reflect.DeepEqual(beforeTarget, *target) {
				return fmt.Errorf("staged summon changed units before first publication")
			}
			return nil
		}
		seenStages[stage-1] = true
		if actor.Acted || actor.MP != beforeActor.MP {
			return fmt.Errorf("staged summon acted or changed class19 MP too early")
		}
		if command == 34 {
			if g.nativeRNGState != owner34.plan.Result.Stages[stage-1].RNGState {
				return fmt.Errorf("command34 RNG publication differs")
			}
			for _, state := range owner34.plan.Result.StageStates[stage-1] {
				u := state.Target
				if u.NativeTransient != state.NativeTransient || u.AP != state.AP || u.DP != state.DP || u.HIT != state.HIT || u.EV != state.EV {
					return fmt.Errorf("command34 stage%d stats differ", stage)
				}
			}
		} else {
			if g.nativeRNGState != owner35.plan.Result.Stages[stage-1].RNGState {
				return fmt.Errorf("command35 RNG publication differs")
			}
			for _, state := range owner35.plan.Result.StageStates[stage-1] {
				if state.Target.HP != state.HP || state.Target.NativeTransient != state.NativeTransient {
					return fmt.Errorf("command35 stage%d HP/transient differ", stage)
				}
			}
		}
		return nil
	}
	count := 0
	if command == 34 {
		count = len(owner34.frames)
	} else {
		count = len(owner35.frames)
	}
	probe.seen = make([]bool, count)
	probe.presentation = &modernHDPresentation{Game: g}
	for i := 0; i < 3; i++ {
		if command == 34 {
			g.stepNativeCommand34Presentation()
		} else {
			g.stepNativeCommand35Presentation()
		}
	}
	index, repeat, published, _, _ := probe.current()
	if index != 0 || repeat != 0 || published {
		t.Fatal("staged summon advanced without Draw")
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
	if probe.err != nil || !probe.finished || callbacks != 1 || g.nativeCmd34Presentation != nil || g.nativeCmd35Presentation != nil {
		t.Fatalf("command%d lifecycle: %v callbacks%d", command, probe.err, callbacks)
	}
	for index, seen := range probe.seen {
		if !seen {
			t.Fatalf("staged summon present%d skipped", index)
		}
	}
	if seenStages != [3]bool{true, true, true} || !actor.Acted || actor.MP != beforeActor.MP ||
		g.rng.Int63() != rand.New(rand.NewSource(int64(command))).Int63() {
		t.Fatal("missing stage or final action/MP/Go RNG differs")
	}
	if err := probe.requireCasterPhases(); err != nil {
		t.Fatal(err)
	}
	if probe.hd == 0 || probe.faithful == 0 || !probe.on || probe.finals == 0 {
		t.Fatal("missing staged summon HD/F2/final presentation")
	}
	g.syncModernHDBattleOwner()
	if g.modernHDBattleGPU.owner != nil || len(g.modernHDBattleGPU.sources) != 0 {
		t.Fatal("completed staged summon retained GPU textures")
	}
	report := map[string]any{"status": "PASS-SCOPED-RUNTIME-E1", "command": command, "scope": "既有class19／BattleFig4修改fixture，保留原始delay及三段hold，非普通玩家或原版oracle。",
		"present_count": count, "hd_draws": probe.hd, "faithful_draws": probe.faithful, "final_draws": probe.finals, "continuations": callbacks,
		"stages": seenStages, "screenshots": probe.shots, "hd_phase_draws": probe.phaseDraws, "phase_f2_switches": probe.phaseSwitches, "source_shots": probe.sourceShots, "actor_hp": actor.HP, "target_hp": target.HP, "mp": actor.MP, "native_rng": g.nativeRNGState, "go_rng_unchanged": true}
	if probe.out != "" {
		appendCommandPanelGPUReport(t, report)
		appendCompoundMapGPUReport(t, report)
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(filepath.Join(probe.out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("command%d %d original presents; %d HD / %d faithful draws; three publications and holds, F2, missing-last-frame atomicity and one continuation; fixture only", command, count, probe.hd, probe.faithful)
}
