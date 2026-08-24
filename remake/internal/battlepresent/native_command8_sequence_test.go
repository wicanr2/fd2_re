package battlepresent

import (
	"github.com/wicanr2/fd2_re/remake/internal/figani"
	"testing"
)

func TestBuildNativeCommand8EffectSequencePrebuildsEveryPhase(t *testing.T) {
	effect := command8CompositorEffect()
	schedule, err := figani.BuildNativeCommand8PresentationSchedule(1, effect)
	if err != nil {
		t.Fatal(err)
	}
	idle := func(pixel byte) *figani.Animation {
		return &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{pixel}, Mask: []byte{1}, Delay: 1}}}
	}
	base := make([]byte, 320*200)
	stages := func() [][]byte {
		out := make([][]byte, figani.NativeCommand8DamageStages+1)
		for i := range out {
			out[i] = append([]byte(nil), base...)
		}
		return out
	}
	seq, err := BuildNativeCommand8EffectSequence(NativeCommand8EffectInput{FrontBase: base, TailBase: base, TargetBases: [][][]byte{stages()}, ActorEffect: idle(3), TargetIdle: []*figani.Animation{idle(4)}, Effect: effect, Schedule: schedule, RawSide: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(seq.Front) != 3 || len(seq.Targets) != 1 || len(seq.Targets[0]) != 34 || len(seq.Transitions) != 0 || len(seq.Tail) != 2 {
		t.Fatalf("shape front=%d targets=%d transition=%d tail=%d", len(seq.Front), len(seq.Targets[0]), len(seq.Transitions), len(seq.Tail))
	}
	for targetIndex, target := range seq.Targets {
		stages := 0
		for _, frame := range target {
			if frame.HPStage != 0 {
				stages++
			}
		}
		if stages != 16 {
			t.Fatalf("target %d stages=%d", targetIndex, stages)
		}
	}
}

func TestBuildNativeCommand8EffectSequenceRejectsLateMissingBase(t *testing.T) {
	effect := command8CompositorEffect()
	schedule, _ := figani.BuildNativeCommand8PresentationSchedule(1, effect)
	idle := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}, Delay: 1}}}
	base := make([]byte, 320*200)
	stages := make([][]byte, figani.NativeCommand8DamageStages+1)
	for i := range stages {
		stages[i] = base
	}
	stages[len(stages)-1] = nil
	if got, err := BuildNativeCommand8EffectSequence(NativeCommand8EffectInput{FrontBase: base, TailBase: base, TargetBases: [][][]byte{stages}, ActorEffect: idle, TargetIdle: []*figani.Animation{idle}, Effect: effect, Schedule: schedule, RawSide: 1}); err == nil || len(got.Front) != 0 || len(got.Targets) != 0 {
		t.Fatalf("partial sequence accepted: %+v err=%v", got, err)
	}
}
