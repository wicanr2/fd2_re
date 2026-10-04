package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func command6SequenceAnimation(frames int) *figani.Animation {
	out := &figani.Animation{Frames: make([]figani.Frame, frames), HeaderByte2: byte(frames)}
	for index := range out.Frames {
		out.Frames[index] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(index + 1)}, Mask: []byte{1}, Delay: 1}
	}
	return out
}

func TestBuildNativeCommand6EffectSequencePrebuildsEveryTarget(t *testing.T) {
	effect := command6SequenceAnimation(figani.NativeCommand6EffectFrameCount)
	for index := range effect.Frames {
		effect.Frames[index].Y = 20
	}
	schedule, err := figani.BuildNativeCommand6PresentationSchedule(1, effect)
	if err != nil {
		t.Fatal(err)
	}
	base := make([]byte, 320*200)
	targetBases := make([][][]byte, 2)
	for target := range targetBases {
		targetBases[target] = make([][]byte, figani.NativeCommand6DamageStages+1)
		for stage := range targetBases[target] {
			targetBases[target][stage] = append([]byte(nil), base...)
		}
	}
	sequence, err := BuildNativeCommand6EffectSequence(NativeCommand6EffectInput{
		FrontBase: base, TailBase: base, TargetBases: targetBases, TransitionBases: [][]byte{base},
		ActorEffect: command6SequenceAnimation(2), TargetIdle: []*figani.Animation{command6SequenceAnimation(2), command6SequenceAnimation(2)},
		Effect: effect, Schedule: schedule, RawSide: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sequence.Front) != 7 || len(sequence.Targets) != 2 || len(sequence.Targets[0].Frames) != 12 ||
		len(sequence.Transitions) != 1 || len(sequence.Transitions[0]) != 9 || len(sequence.Tail) != 7 {
		t.Fatalf("sequence shape front=%d targets=%d/%d transitions=%d/%d tail=%d", len(sequence.Front), len(sequence.Targets), len(sequence.Targets[0].Frames), len(sequence.Transitions), len(sequence.Transitions[0]), len(sequence.Tail))
	}
	// #156：依原始 mode3/mode5 及 12+9 次 counter 推進手算。
	// #162：__CHP向零截斷，首目標channel4=(0,42)；後續frame2加(3,3)。
	for _, sample := range []struct {
		target, x, y int
		pixel        byte
	}{
		{0, 0, 62, 5},
		{1, 3, 65, 3},
		{1, 74, 58, 2},
	} {
		if got := sequence.Targets[sample.target].Frames[0][sample.y*320+sample.x]; got != sample.pixel {
			t.Fatalf("target%d pixel(%d,%d)=%d want%d", sample.target, sample.x, sample.y, got, sample.pixel)
		}
	}
	for target, frames := range sequence.Targets {
		published := 0
		for _, stage := range frames.HPStages {
			if stage != 0 {
				published++
			}
		}
		if published != figani.NativeCommand6DamageStages {
			t.Fatalf("target %d published stages=%d", target, published)
		}
	}
}

func TestBuildNativeCommand6EffectSequenceFailsBeforePartialOutput(t *testing.T) {
	effect := command6SequenceAnimation(figani.NativeCommand6EffectFrameCount)
	schedule, _ := figani.BuildNativeCommand6PresentationSchedule(1, effect)
	base := make([]byte, 320*200)
	if got, err := BuildNativeCommand6EffectSequence(NativeCommand6EffectInput{
		FrontBase: base, TailBase: base, TargetBases: [][][]byte{{base}},
		ActorEffect: command6SequenceAnimation(1), TargetIdle: []*figani.Animation{command6SequenceAnimation(1)}, Effect: effect, Schedule: schedule, RawSide: 1,
	}); err == nil || len(got.Front) != 0 {
		t.Fatalf("malformed stage bases accepted: %+v err=%v", got, err)
	}
}
