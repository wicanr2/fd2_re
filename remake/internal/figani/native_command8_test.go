package figani

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func command8TestAnimation() *Animation {
	frames := make([]Frame, NativeCommand8EffectFrameCount)
	for i := range frames {
		frames[i] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(i + 1)}, Mask: []byte{1}}
	}
	return &Animation{Frames: frames, HeaderByte2: NativeCommand8EffectFrameCount}
}

func TestNativeCommand8ScheduleAndMode0(t *testing.T) {
	s, err := BuildNativeCommand8PresentationSchedule(1, command8TestAnimation())
	if err != nil || s.EffectResource != 28 || s.SoundResource != 90 || s.FrameBases != nativeCommand8FrameBases {
		t.Fatalf("schedule=%+v err=%v", s, err)
	}
	z, err := BuildNativeCommand8PresentationSchedule(0, command8TestAnimation())
	if err != nil || z.EffectResource != 30 {
		t.Fatalf("zero=%+v err=%v", z, err)
	}
	if got := NewNativeCommand8State().Counters; got != [16]int{0, -2, -4, -6, -8, -10, -12, -14, -16, -18, -20, -22, -24, -26, -28, -30} {
		t.Fatalf("counters=%v", got)
	}
}

func TestNativeCommand8FirstTargetMarkersAndSamples(t *testing.T) {
	s, _ := BuildNativeCommand8PresentationSchedule(1, command8TestAnimation())
	state := NewNativeCommand8State()
	for i := 0; i < NativeCommand8FrontFrames; i++ {
		f, err := PlanNativeCommand8DrawFrame(state, s)
		if err != nil {
			t.Fatal(err)
		}
		state = f.Next
	}
	frames, _, err := BuildNativeCommand8TargetSequence(state, s)
	if err != nil {
		t.Fatal(err)
	}
	markers := map[int]bool{}
	for i := 0; i <= 30; i += 2 {
		markers[i] = true
	}
	sub1 := map[int]bool{}
	for i := 1; i <= 27; i += 2 {
		sub1[i] = true
	}
	sub2 := map[int]bool{}
	for i := 1; i <= 31; i += 2 {
		sub2[i] = true
	}
	stages := 0
	for i, f := range frames {
		if f.NumericMarker != markers[i] || f.PlaySub1 != sub1[i] || f.PlaySub2 != sub2[i] {
			t.Fatalf("frame %d=%+v", i, f)
		}
		if f.HPStage != 0 {
			stages++
			if f.HPStage != stages {
				t.Fatalf("frame %d stage=%d", i, f.HPStage)
			}
		}
	}
	if stages != 16 {
		t.Fatalf("stages=%d", stages)
	}
}

func TestNativeCommand8TransitionAndTail(t *testing.T) {
	s, _ := BuildNativeCommand8PresentationSchedule(1, command8TestAnimation())
	state := NewNativeCommand8State()
	transition, state, err := BuildNativeCommand8TransitionSequence(state, s, 1)
	if err != nil || len(transition) != 9 || transition[0].TargetOffsetX != -35 || transition[8].TargetOffsetX != 0 {
		t.Fatalf("transition=%+v err=%v", transition, err)
	}
	tail, _, err := BuildNativeCommand8TailSequence(state, s)
	if err != nil || len(tail) != 2 {
		t.Fatalf("tail=%d err=%v", len(tail), err)
	}
}

func TestOriginalFDOTHERCommand8ResourcesMatchRecoveredShapes(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	for _, tc := range []struct {
		resource int
		side     byte
	}{{28, 1}, {30, 0}} {
		animation, err := DecodeResource(path, tc.resource)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil {
			t.Fatalf("resource %d: %v", tc.resource, err)
		}
		schedule, err := BuildNativeCommand8PresentationSchedule(tc.side, animation)
		if err != nil || schedule.EffectResource != tc.resource {
			t.Fatalf("resource %d schedule=%+v err=%v", tc.resource, schedule, err)
		}
	}
	for _, sample := range []int{0, 1, 2} {
		raw, err := fdother.ReadNestedResource(path, NativeCommand8SoundResource, sample)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil || len(raw) == 0 {
			t.Fatalf("resource 90 sample %d len=%d err=%v", sample, len(raw), err)
		}
	}
}
