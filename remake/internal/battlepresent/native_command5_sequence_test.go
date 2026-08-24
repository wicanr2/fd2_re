package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestBuildNativeCommand5EffectSequencePrebuildsAndCarriesState(t *testing.T) {
	effect := command5CompositorEffect()
	schedule, err := figani.BuildNativeCommand5PresentationSchedule(1, effect)
	if err != nil {
		t.Fatal(err)
	}
	idle := func(pixel byte) *figani.Animation {
		return &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{pixel}, Mask: []byte{1}, Delay: 1}}}
	}
	base := make([]byte, 320*200)
	stages := func() [][]byte {
		out := make([][]byte, figani.NativeCommand5DamageStages+1)
		for index := range out {
			out[index] = append([]byte(nil), base...)
		}
		return out
	}
	sequence, err := BuildNativeCommand5EffectSequence(NativeCommand5EffectInput{
		FrontBase: base, TailBase: base, TargetBases: [][][]byte{stages(), stages()}, TransitionBases: [][]byte{base},
		ActorEffect: idle(3), TargetIdle: []*figani.Animation{idle(4), idle(5)}, Effect: effect,
		Schedule: schedule, RawSide: 1, VisualRNG: 17,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sequence.Front) != 1 || len(sequence.Targets) != 2 || len(sequence.Targets[0]) != 12 ||
		len(sequence.Targets[1]) != 12 || len(sequence.Transitions) != 1 || len(sequence.Transitions[0]) != 9 || len(sequence.Tail) != 8 {
		t.Fatalf("sequence shape front=%d targets=%v transitions=%d/%d tail=%d", len(sequence.Front), []int{len(sequence.Targets[0]), len(sequence.Targets[1])}, len(sequence.Transitions), len(sequence.Transitions[0]), len(sequence.Tail))
	}
	for targetIndex, target := range sequence.Targets {
		stages := 0
		for _, frame := range target {
			if frame.HPStage != 0 {
				stages++
			}
		}
		if stages != figani.NativeCommand5DamageStages {
			t.Fatalf("target %d stages=%d", targetIndex, stages)
		}
	}
}

func TestBuildNativeCommand5EffectSequenceRejectsLateMissingBase(t *testing.T) {
	effect := command5CompositorEffect()
	schedule, _ := figani.BuildNativeCommand5PresentationSchedule(1, effect)
	idle := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}, Delay: 1}}}
	base := make([]byte, 320*200)
	stages := make([][]byte, figani.NativeCommand5DamageStages+1)
	for index := range stages {
		stages[index] = base
	}
	stages[len(stages)-1] = nil
	if got, err := BuildNativeCommand5EffectSequence(NativeCommand5EffectInput{
		FrontBase: base, TailBase: base, TargetBases: [][][]byte{stages}, ActorEffect: idle,
		TargetIdle: []*figani.Animation{idle}, Effect: effect, Schedule: schedule, RawSide: 1,
	}); err == nil || len(got.Front) != 0 || len(got.Targets) != 0 {
		t.Fatalf("partial sequence accepted: %+v err=%v", got, err)
	}
}
