package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func command9PresentAnimation(count int, delay int) *figani.Animation {
	frames := make([]figani.Frame, count)
	for index := range frames {
		frames[index] = figani.Frame{Width: 1, Height: 1, X: 1, Y: 1, Delay: delay, Pixels: []byte{byte(index + 1)}, Mask: []byte{1}}
	}
	return &figani.Animation{Frames: frames, HeaderByte2: byte(count)}
}

func TestBuildNativeCommand9AISequencePreservesAllPhases(t *testing.T) {
	effect := command9PresentAnimation(31, 1)
	schedule, err := figani.BuildNativeCommand9AISchedule(0, effect)
	if err != nil {
		t.Fatal(err)
	}
	bases := make([][]byte, figani.NativeCommand9AIDamageStages+1)
	for index := range bases {
		bases[index] = make([]byte, 320*200)
	}
	seq, err := BuildNativeCommand9AISequence(NativeCommand9AIInput{BeforeBase: bases[0], AfterBase: bases[len(bases)-1], TargetBases: bases, Actor: command9PresentAnimation(1, 1), Target: command9PresentAnimation(2, 1), Effect: effect, Schedule: schedule})
	if err != nil {
		t.Fatal(err)
	}
	if len(seq.SlideIn) != 11 || len(seq.Front) != 20 || len(seq.Target) != 60 || len(seq.Tail) != 20 || len(seq.SlideOut) != 8 {
		t.Fatalf("sequence lengths=%d/%d/%d/%d/%d", len(seq.SlideIn), len(seq.Front), len(seq.Target), len(seq.Tail), len(seq.SlideOut))
	}
	stages := 0
	for _, frame := range seq.Target {
		if frame.HPStage != 0 {
			stages++
		}
	}
	if stages != 20 {
		t.Fatalf("HP stages=%d", stages)
	}
}

func TestBuildNativeCommand9AISequenceRejectsMissingStage(t *testing.T) {
	effect := command9PresentAnimation(31, 1)
	schedule, _ := figani.BuildNativeCommand9AISchedule(0, effect)
	if _, err := BuildNativeCommand9AISequence(NativeCommand9AIInput{BeforeBase: make([]byte, 320*200), AfterBase: make([]byte, 320*200), TargetBases: make([][]byte, 20), Actor: command9PresentAnimation(1, 1), Target: command9PresentAnimation(1, 1), Effect: effect, Schedule: schedule}); err == nil {
		t.Fatal("missing HP stage was accepted")
	}
}
