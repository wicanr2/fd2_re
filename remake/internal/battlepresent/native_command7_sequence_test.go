package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestBuildNativeCommand7EffectSequencePrebuildsEveryPhase(t *testing.T) {
	effect := command7CompositorEffect()
	for index := range effect.Frames {
		effect.Frames[index].Delay = []int{2, 2, 2, 2, 4}[index]
	}
	schedule, err := figani.BuildNativeCommand7PresentationSchedule(1, effect)
	if err != nil {
		t.Fatal(err)
	}
	idle := func(pixel byte) *figani.Animation {
		return &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{pixel}, Mask: []byte{1}, Delay: 1}}}
	}
	base := make([]byte, 320*200)
	stages := func() [][]byte {
		out := make([][]byte, figani.NativeCommand7DamageStages+1)
		for index := range out {
			out[index] = append([]byte(nil), base...)
		}
		return out
	}
	sequence, err := BuildNativeCommand7EffectSequence(NativeCommand7EffectInput{
		FrontBase: base, TailBase: base, TargetBases: [][][]byte{stages(), stages()}, TransitionBases: [][]byte{base},
		ActorEffect: idle(3), TargetIdle: []*figani.Animation{idle(4), idle(5)}, Effect: effect, Schedule: schedule, RawSide: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sequence.Front) != 2 || len(sequence.Targets) != 2 || len(sequence.Targets[0]) != 32 ||
		len(sequence.Targets[1]) != 32 || len(sequence.Transitions) != 1 || len(sequence.Transitions[0]) != 9 || len(sequence.Tail) != 16 {
		t.Fatalf("shape front=%d targets=%d/%d transition=%d tail=%d", len(sequence.Front), len(sequence.Targets[0]), len(sequence.Targets[1]), len(sequence.Transitions[0]), len(sequence.Tail))
	}
	for targetIndex, target := range sequence.Targets {
		stages := 0
		for _, frame := range target {
			if frame.HPStage != 0 {
				stages++
			}
		}
		if stages != figani.NativeCommand7DamageStages {
			t.Fatalf("target %d stages=%d", targetIndex, stages)
		}
	}
}

func TestBuildNativeCommand7EffectSequenceRejectsLateMissingBase(t *testing.T) {
	effect := command7CompositorEffect()
	schedule, _ := figani.BuildNativeCommand7PresentationSchedule(1, effect)
	idle := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}, Delay: 1}}}
	base := make([]byte, 320*200)
	stages := make([][]byte, figani.NativeCommand7DamageStages+1)
	for index := range stages {
		stages[index] = base
	}
	stages[len(stages)-1] = nil
	if got, err := BuildNativeCommand7EffectSequence(NativeCommand7EffectInput{
		FrontBase: base, TailBase: base, TargetBases: [][][]byte{stages}, ActorEffect: idle,
		TargetIdle: []*figani.Animation{idle}, Effect: effect, Schedule: schedule, RawSide: 1,
	}); err == nil || len(got.Front) != 0 || len(got.Targets) != 0 {
		t.Fatalf("partial sequence accepted: %+v err=%v", got, err)
	}
}
