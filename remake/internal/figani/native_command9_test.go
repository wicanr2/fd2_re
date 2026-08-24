package figani

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func command9Animation(count int) *Animation {
	return &Animation{Frames: make([]Frame, count), HeaderByte2: byte(count)}
}

func TestNativeCommand9AIScheduleFailsClosedByRawSideAndShape(t *testing.T) {
	if _, err := BuildNativeCommand9AISchedule(1, command9Animation(31)); err == nil {
		t.Fatal("unproven nonzero raw side was accepted")
	}
	if _, err := BuildNativeCommand9AISchedule(0, command9Animation(30)); err == nil {
		t.Fatal("short effect was accepted")
	}
}

func TestNativeCommand9AITargetMarkersAndSamples(t *testing.T) {
	s, err := BuildNativeCommand9AISchedule(0, command9Animation(31))
	if err != nil {
		t.Fatal(err)
	}
	if s.SlideInHoldMS != 500 {
		t.Fatalf("slide-in hold=%d", s.SlideInHoldMS)
	}
	frames, next, err := BuildNativeCommand9AITargetSequence(NewNativeCommand9AIState(), s)
	if err != nil {
		t.Fatal(err)
	}
	markers, stages := 0, 0
	for index, frame := range frames {
		if frame.NumericMarker {
			markers++
		}
		if frame.HPStage != 0 {
			stages++
		}
		if frame.PlaySub1 != (index == 5) || frame.PlaySub2 != (index == 35) {
			t.Fatalf("frame %d samples=%t/%t", index, frame.PlaySub1, frame.PlaySub2)
		}
	}
	if len(frames) != 60 || markers != 27 || stages != 20 || next.Counter != 61 {
		t.Fatalf("frames=%d markers=%d stages=%d next=%#v", len(frames), markers, stages, next)
	}
}

func TestNativeCommand9AIOrbitAlternatesFrameZero(t *testing.T) {
	s, _ := BuildNativeCommand9AISchedule(0, command9Animation(31))
	state := NewNativeCommand9AIState()
	for index := 0; index < 4; index++ {
		frame, err := PlanNativeCommand9AIOrbitFrame(state, s)
		if err != nil {
			t.Fatal(err)
		}
		if (len(frame.EffectFrames) == 1) != (index%2 == 0) {
			t.Fatalf("orbit frame %d=%#v", index, frame)
		}
		state = frame.Next
	}
}

func TestNativeCommand9AIOriginalAssets(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(path); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	effect, err := DecodeResource(path, NativeCommand9AIEffectResource)
	if err != nil {
		t.Fatal(err)
	}
	if len(effect.Frames) != 31 {
		t.Fatalf("FDOTHER #44 frames=%d, want 31", len(effect.Frames))
	}
	if _, err := BuildNativeCommand9AISchedule(0, effect); err != nil {
		t.Fatal(err)
	}
	for sample := 0; sample <= 2; sample++ {
		raw, err := fdother.ReadNestedResource(path, NativeCommand9AISoundResource, sample)
		if err != nil || len(raw) == 0 {
			t.Fatalf("FDOTHER #90 sub%d len=%d err=%v", sample, len(raw), err)
		}
	}
}
