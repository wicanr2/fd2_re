package main

import (
	"fmt"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func ordinaryRemainingHDTestGame(t *testing.T, command int) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	g, actor, target := ordinaryCommand0HDTestGame(t)
	resistances, err := battle.LoadNativeCommandResistances("../../assets/data/native_command_resistances.json")
	if err != nil {
		t.Fatal(err)
	}
	g.st.NativeCommandResistances = resistances
	target.BattleFig = 102
	mode, cost := 0, 6
	switch command {
	case 2, 3:
		mode = 1
	case 6, 7:
		mode = 2
	case 8:
		actor.BattleFig = 21
	case 9:
		actor.Camp, target.Camp = battle.Enemy, battle.Own
		actor.NativeRecordByte6, target.NativeRecordByte6 = 0, 2
		actor.HasNativeRecordByte6, target.HasNativeRecordByte6 = true, true
		actor.BattleFig, target.BattleFig = 90, 0
		g.nativeRNGState = 3
	default:
		t.Fatal("unsupported ordinary test command")
	}
	g.st.NativeCommandBook[command] = battle.NativeCommandRecord{ID: command, Damage: 50, Hit: 100, SelectionMode: 8, EffectMode: mode, MPCost: cost, TargetCode: 0}
	return g, actor, target
}

func startOrdinaryRemainingHD(g *Game, command int, actor, target *battle.Unit, then func([]battle.NativeCommandDamageResult)) error {
	switch command {
	case 2:
		return g.startNativeCommand2Presentation(actor, target, then)
	case 3:
		return g.startNativeCommand3Presentation(actor, target, then)
	case 6:
		return g.startNativeCommand6Presentation(actor, target, then)
	case 7:
		return g.startNativeCommand7Presentation(actor, target, then)
	case 8:
		return g.startNativeCommand8Presentation(actor, target, then)
	case 9:
		return g.startNativeCommand9AIPresentation(actor, target, then)
	}
	return fmt.Errorf("unsupported ordinary command")
}

type ordinaryRemainingTestOwner struct {
	owner        any
	hd           *modernHDOrdinaryFrames
	plan         *battle.NativeCommandDamagePlan
	step         func()
	active       func() bool
	modern       func() *nativeCompoundPresentedFrame
	position     func() (int, int, bool)
	acknowledged func() bool
}

func ordinaryRemainingOwner(g *Game, command int) ordinaryRemainingTestOwner {
	switch command {
	case 9:
		j := g.nativeCmd9AIPresentation
		if j == nil {
			return ordinaryRemainingTestOwner{}
		}
		return ordinaryRemainingTestOwner{owner: j, hd: j.hd, plan: j.plan, step: g.stepNativeCommand9AIPresentation,
			active: func() bool { return g.nativeCmd9AIPresentation != nil }, modern: j.modernFrame,
			position:     func() (int, int, bool) { return int(j.phase), j.frame, j.pulseBlack },
			acknowledged: func() bool { return j.drawn }}

	case 2:
		j := g.nativeCmd2Presentation
		if j == nil {
			return ordinaryRemainingTestOwner{}
		}
		return ordinaryRemainingTestOwner{owner: j, hd: j.hd, plan: j.plan, step: g.stepNativeCommand2Presentation,
			active: func() bool { return g.nativeCmd2Presentation != nil }, modern: j.modernFrame,
			position:     func() (int, int, bool) { return int(j.phase), j.frame, j.pulseBlack },
			acknowledged: func() bool { return j.drawn }}

	case 3:
		j := g.nativeCmd3Presentation
		if j == nil {
			return ordinaryRemainingTestOwner{}
		}
		return ordinaryRemainingTestOwner{owner: j, hd: j.hd, plan: j.plan, step: g.stepNativeCommand3Presentation,
			active: func() bool { return g.nativeCmd3Presentation != nil }, modern: j.modernFrame,
			position:     func() (int, int, bool) { return int(j.phase), j.frame, j.pulseBlack },
			acknowledged: func() bool { return j.drawn }}
	case 6:
		j := g.nativeCmd6Presentation
		if j == nil {
			return ordinaryRemainingTestOwner{}
		}
		return ordinaryRemainingTestOwner{owner: j, hd: j.hd, plan: j.plan, step: g.stepNativeCommand6Presentation,
			active: func() bool { return g.nativeCmd6Presentation != nil }, modern: j.modernFrame,
			position:     func() (int, int, bool) { return int(j.phase), j.frame, j.pulseBlack },
			acknowledged: func() bool { return j.drawn }}

	case 7:
		j := g.nativeCmd7Presentation
		if j == nil {
			return ordinaryRemainingTestOwner{}
		}
		return ordinaryRemainingTestOwner{owner: j, hd: j.hd, plan: j.plan, step: g.stepNativeCommand7Presentation,
			active: func() bool { return g.nativeCmd7Presentation != nil }, modern: j.modernFrame,
			position:     func() (int, int, bool) { return int(j.phase), j.frame, j.pulseBlack },
			acknowledged: func() bool { return j.drawn }}

	case 8:
		j := g.nativeCmd8Presentation
		if j == nil {
			return ordinaryRemainingTestOwner{}
		}
		return ordinaryRemainingTestOwner{owner: j, hd: j.hd, plan: j.plan, step: g.stepNativeCommand8Presentation,
			active: func() bool { return g.nativeCmd8Presentation != nil }, modern: j.modernFrame,
			position:     func() (int, int, bool) { return int(j.phase), j.frame, j.pulseBlack },
			acknowledged: func() bool { return j.drawn }}

	}
	return ordinaryRemainingTestOwner{}
}

func TestModernHDOrdinaryRemainingPreflightAndF2Atomic(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit full theme required")
	}
	t.Setenv("FD2_MUTE", "1")
	template, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []int{2, 3, 6, 7, 8, 9} {
		t.Run(fmt.Sprint(command), func(t *testing.T) {
			g, actor, target := ordinaryRemainingHDTestGame(t, command)
			set := cloneModernHDBanksForTest(template)
			g.modernStoryPortraits = set
			a, b, seed := *actor, *target, g.nativeRNGState
			_, actual, err := figani.LoadSeparatedResourceWithZeroHeaderFallbackSource(separatedAssetPath("animations"), actor.BattleFig*3+2)
			if err != nil {
				t.Fatal(err)
			}
			entry := set.battleAnimations[actual]
			if entry == nil {
				t.Fatal("registered actual cast source required")
			}
			last := len(entry.paths) - 1
			saved := entry.paths[last]
			entry.paths[last] = filepath.Join(t.TempDir(), "missing.png")
			if err := startOrdinaryRemainingHD(g, command, actor, target, nil); err == nil || !strings.Contains(err.Error(), "missing.png") ||
				ordinaryRemainingOwner(g, command).owner != nil || entry.Frames[0] != nil ||
				!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
				t.Fatalf("preflight err=%v owner=%v bankLoaded=%v actorEqual=%v targetEqual=%v rng=%d want=%d", err,
					ordinaryRemainingOwner(g, command).owner, entry.Frames[0] != nil, reflect.DeepEqual(a, *actor), reflect.DeepEqual(b, *target), g.nativeRNGState, seed)
			}
			entry.paths[last] = saved
			if err := startOrdinaryRemainingHD(g, command, actor, target, nil); err != nil {
				t.Fatal(err)
			}
			j := ordinaryRemainingOwner(g, command)
			unique := map[int]bool{actor.BattleFig * 3: true, actual: true}
			for _, result := range j.plan.Results {
				unique[result.Target.BattleFig*3] = true
			}
			resources := make([]int, 0, len(unique))
			for resource := range unique {
				resources = append(resources, resource)
			}
			sort.Ints(resources)
			if !reflect.DeepEqual(j.hd.resources(), resources) {
				t.Fatalf("wrong actual sources: %v", j.hd.resources())
			}
			if err := g.applyTheme(nil); err != nil {
				t.Fatal(err)
			}
			reload := cloneModernHDBanksForTest(template)
			reload.battleAnimations[actual].paths[last] = filepath.Join(t.TempDir(), "missing-f2.png")
			if err := g.applyTheme(reload); err == nil || g.modernStoryPortraits != nil || ordinaryRemainingOwner(g, command).owner != j.owner ||
				reload.battleAnimations[actor.BattleFig*3].Frames[0] != nil || reload.battleAnimations[target.BattleFig*3].Frames[0] != nil ||
				!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
				t.Fatal("failed F2 changed owner, state or partial bank")
			}
			if err := g.applyTheme(set); err != nil {
				t.Fatal(err)
			}
			j.step()
			_, frame, _ := j.position()
			if frame != 0 || actor.MP != a.MP || target.HP != b.HP {
				t.Fatal("unacknowledged frame changed state")
			}
		})
	}
}

func appendModernOrdinaryRemainingGPUCases(t *testing.T, cases []modernHDBattleGPUCase) []modernHDBattleGPUCase {
	t.Helper()
	_, palette, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	actor, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), 272)
	if err != nil {
		t.Fatal(err)
	}
	var targets []*figani.Animation
	for _, id := range []int{0, 24} {
		idle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), id)
		if err != nil {
			t.Fatal(err)
		}
		targets = append(targets, idle)
	}
	for _, command := range []int{2, 3, 6, 7, 8} {
		stages := map[int]int{2: 6, 3: 13, 6: 5, 7: 5, 8: 16}[command]
		count := 2
		if command == 8 {
			count = 1
		}
		bases := make([][][]byte, count)
		for target := range bases {
			for stage := 0; stage <= stages; stage++ {
				base := make([]byte, 64000)
				for i := range base {
					base[i] = byte((i + stage*3 + target*13) % 251)
				}
				bases[target] = append(bases[target], base)
			}
		}
		for _, side := range []byte{0, 2} {
			effectID := map[int]int{2: 26, 3: 39, 6: 32, 7: 37, 8: 28}[command]
			if side == 0 {
				effectID = map[int]int{2: 27, 3: 43, 6: 33, 7: 38, 8: 30}[command]
			}
			effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", effectID)
			if err != nil {
				t.Fatal(err)
			}
			var sources []battlepresent.NativeCommandSources
			var pixels [][]byte
			collect := func(source battlepresent.NativeCommandSources, p []byte) {
				sources = append(sources, source)
				pixels = append(pixels, p)
			}
			transitionBases := [][]byte{bases[0][stages]}
			if count == 1 {
				transitionBases = nil
			}
			switch command {
			case 2:
				schedule, err := figani.BuildNativeCommand2PresentationSchedule(side, effect)
				if err != nil {
					t.Fatal(err)
				}
				front, end, err := figani.BuildNativeCommand2FrontSequence(side, effect)
				if err != nil {
					t.Fatal(err)
				}
				for _, helper := range front {
					p, s, err := battlepresent.ComposeNativeCommand2OrbitFrameWithSources(bases[0][0], actor, targets[0], effect, helper, side)
					if err != nil {
						t.Fatal(err)
					}
					s.Phase = "handler-front"
					collect(s, p)
				}
				for target := 0; target < count; target++ {
					frames, last, err := figani.BuildNativeCommand2TargetSequence()
					if err != nil {
						t.Fatal(err)
					}
					for _, helper := range frames {
						p, s, err := battlepresent.ComposeNativeCommand2TargetFrameWithSources(bases[target][helper.HPStage], targets[target].Frames[0], effect, schedule, side, helper.EffectFrame)
						if err != nil {
							t.Fatal(err)
						}
						s.TargetIndex = target
						collect(s, p)
					}
					if target+1 < count {
						direction := 1
						if side == 0 {
							direction = -1
						}
						for step, helper := range figani.BuildNativeCommand2TransitionSequence(last) {
							which, offset := target, direction*35*(step+1)
							if step >= 4 {
								which++
								offset = direction * 35 * (8 - step)
							}
							p, s, err := battlepresent.ComposeNativeCommand2TransitionFrameWithSources(bases[target][stages], actor, targets[which], effect, helper, side, offset)
							if err != nil {
								t.Fatal(err)
							}
							s.TargetIndex = which
							collect(s, p)
						}
					}
				}
				tail, err := figani.BuildNativeCommand2TailSequence(side, end.Repeat, effect)
				if err != nil {
					t.Fatal(err)
				}
				for _, helper := range tail {
					p, s, err := battlepresent.ComposeNativeCommand2OrbitFrameWithSources(bases[count-1][stages], actor, targets[count-1], effect, helper, side)
					if err != nil {
						t.Fatal(err)
					}
					s.TargetIndex = count - 1
					s.Phase = "handler-tail"
					collect(s, p)
				}

			case 3:
				schedule, err := figani.BuildNativeCommand3PresentationSchedule(side, effect)
				if err != nil {
					t.Fatal(err)
				}
				sequence, err := battlepresent.BuildNativeCommand3EffectSequence(battlepresent.NativeCommand3EffectInput{
					FrontBase: bases[0][0], TailBase: bases[count-1][stages], TargetBases: bases, TransitionBases: transitionBases,
					ActorEffect: actor, TargetIdle: targets[:count], Effect: effect, Schedule: schedule, RawSide: side})
				if err != nil {
					t.Fatal(err)
				}
				for _, frame := range sequence.Front {
					collect(frame.Sources, frame.Pixels)
				}
				for i, frames := range sequence.Targets {
					for _, frame := range frames {
						collect(frame.Sources, frame.Pixels)
					}
					if i < len(sequence.Transitions) {
						for _, frame := range sequence.Transitions[i] {
							collect(frame.Sources, frame.Pixels)
						}
					}
				}
				for _, frame := range sequence.Tail {
					collect(frame.Sources, frame.Pixels)
				}

			case 6:
				schedule, err := figani.BuildNativeCommand6PresentationSchedule(side, effect)
				if err != nil {
					t.Fatal(err)
				}
				for _, hits := range [][]bool{{true, false}, {false, true}} {
					sequence, err := battlepresent.BuildNativeCommand6EffectSequence(battlepresent.NativeCommand6EffectInput{
						FrontBase: bases[0][0], TailBase: bases[count-1][stages], TargetBases: bases, TransitionBases: transitionBases,
						ActorEffect: actor, TargetIdle: targets[:count], Effect: effect, Schedule: schedule, RawSide: side,
						TargetHits: hits, TargetNumericRNG: []uint16{7, 9}})
					if err != nil {
						t.Fatal(err)
					}
					for i, frame := range sequence.Front {
						collect(sequence.FrontSources[i], frame)
					}
					for target, frames := range sequence.Targets {
						for i, frame := range frames.Frames {
							collect(frames.Sources[i], frame)
						}
						if target < len(sequence.Transitions) {
							for i, frame := range sequence.Transitions[target] {
								collect(sequence.TransitionSources[target][i], frame)
							}
						}
					}
					for i, frame := range sequence.Tail {
						collect(sequence.TailSources[i], frame)
					}
				}
			case 7:
				schedule, err := figani.BuildNativeCommand7PresentationSchedule(side, effect)
				if err != nil {
					t.Fatal(err)
				}
				sequence, err := battlepresent.BuildNativeCommand7EffectSequence(battlepresent.NativeCommand7EffectInput{
					FrontBase: bases[0][0], TailBase: bases[count-1][stages], TargetBases: bases, TransitionBases: transitionBases,
					ActorEffect: actor, TargetIdle: targets[:count], Effect: effect, Schedule: schedule, RawSide: side})
				if err != nil {
					t.Fatal(err)
				}
				for _, frame := range sequence.Front {
					collect(frame.Sources, frame.Pixels)
				}
				for i, frames := range sequence.Targets {
					for _, frame := range frames {
						collect(frame.Sources, frame.Pixels)
					}
					if i < len(sequence.Transitions) {
						for _, frame := range sequence.Transitions[i] {
							collect(frame.Sources, frame.Pixels)
						}
					}
				}
				for _, frame := range sequence.Tail {
					collect(frame.Sources, frame.Pixels)
				}

			case 8:
				schedule, err := figani.BuildNativeCommand8PresentationSchedule(side, effect)
				if err != nil {
					t.Fatal(err)
				}
				sequence, err := battlepresent.BuildNativeCommand8EffectSequence(battlepresent.NativeCommand8EffectInput{
					FrontBase: bases[0][0], TailBase: bases[count-1][stages], TargetBases: bases, TransitionBases: transitionBases,
					ActorEffect: actor, TargetIdle: targets[:count], Effect: effect, Schedule: schedule, RawSide: side})
				if err != nil {
					t.Fatal(err)
				}
				for _, frame := range sequence.Front {
					collect(frame.Sources, frame.Pixels)
				}
				for i, frames := range sequence.Targets {
					for _, frame := range frames {
						collect(frame.Sources, frame.Pixels)
					}
					if i < len(sequence.Transitions) {
						for _, frame := range sequence.Transitions[i] {
							collect(frame.Sources, frame.Pixels)
						}
					}
				}
				for _, frame := range sequence.Tail {
					collect(frame.Sources, frame.Pixels)
				}

			}
			bound, err := bindModernHDOrdinarySources(sources, actor, 272, targets[:count], []int{0, 24}[:count], effect, effectID, palette)
			if err != nil || len(bound) != len(pixels) {
				t.Fatalf("ordinary%d binding: %v", command, err)
			}
			for i, f := range bound {
				plan, err := (&Game{}).modernHDBattlePlan(f.hdSources, f.hdPalette, 0)
				if err != nil {
					t.Fatal(err)
				}
				cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("ordinary%d/%d/%s/%d", command, side, f.hdPhase, i), plan: plan, indexed: pixels[i]})
			}
			bad := append([]battlepresent.NativeCommandSources(nil), sources...)
			bad[len(bad)-1].TargetIndex = count
			if result, err := bindModernHDOrdinarySources(bad, actor, 272, targets[:count], []int{0, 24}[:count], effect, effectID, palette); err == nil || result != nil {
				t.Fatal("late source failure returned partial batch")
			}
			bad = append([]battlepresent.NativeCommandSources(nil), sources...)
			bad[len(bad)-1].After = []battlepresent.NativeCommandEffectSource{{Frame: 999}}
			if result, err := bindModernHDOrdinarySources(bad, actor, 272, targets[:count], []int{0, 24}[:count], effect, effectID, palette); err == nil || result != nil {
				t.Fatal("late effect failure returned partial batch")
			}
		}
	}
	return cases
}
