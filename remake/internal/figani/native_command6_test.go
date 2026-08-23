package figani

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func command6TestAnimation() *Animation {
	frames := make([]Frame, NativeCommand6EffectFrameCount)
	for i := range frames {
		frames[i] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(i)}, Mask: []byte{1}}
	}
	return &Animation{Frames: frames, HeaderByte2: NativeCommand6EffectFrameCount}
}

func TestNativeCommand6SchedulePreservesRawSideTables(t *testing.T) {
	nonzero, err := BuildNativeCommand6PresentationSchedule(1, command6TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	if nonzero.EffectResource != 32 || nonzero.SoundResource != 87 || nonzero.SoundIndices != [3]int{1, 2, 3} ||
		nonzero.BaseByte != 30 || nonzero.DwordTable != [5]int{10, 8, 3, 0, 0} || nonzero.ByteTable != [5]byte{10, 8, 3, 0, 0} {
		t.Fatalf("nonzero side schedule=%+v", nonzero)
	}
	zero, err := BuildNativeCommand6PresentationSchedule(0, command6TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	if zero.EffectResource != 33 || zero.BaseByte != 90 || zero.DwordTable != [5]int{-10, -8, -3, 0, 0} ||
		zero.ByteTable != [5]byte{0, 0, 0, 0, 0} {
		t.Fatalf("zero side schedule=%+v", zero)
	}
}

func TestNativeCommand6CoordinatesPreserveFivePointFormula(t *testing.T) {
	want := [5]NativeCommand6Point{{40, 30}, {33, 41}, {22, 37}, {22, 23}, {33, 19}}
	if got := NativeCommand6Coordinates(10, 30); got != want {
		t.Fatalf("radius10 coordinates=%v want=%v", got, want)
	}
	wantZeroSide := [5]NativeCommand6Point{{100, 30}, {93, 41}, {82, 37}, {82, 23}, {93, 19}}
	if got := NativeCommand6Coordinates(10, 90); got != wantZeroSide {
		t.Fatalf("zero-side coordinates=%v want=%v", got, wantZeroSide)
	}
}

func TestNativeCommand6TargetPlanCompletesOnSecondFrame(t *testing.T) {
	schedule, _ := BuildNativeCommand6PresentationSchedule(1, command6TestAnimation())
	points := NativeCommand6Coordinates(10, schedule.BaseByte)
	first, err := PlanNativeCommand6TargetFrame(NewNativeCommand6TargetState(), schedule, points, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Complete || len(first.Mode4) != 2 || len(first.Mode5) != 2 || first.Next.Counters != [5]int{1, 0, -1, -2, -3} {
		t.Fatalf("first target frame=%+v", first)
	}
	second, err := PlanNativeCommand6TargetFrame(first.Next, schedule, points, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Complete || second.Next.Counters != [5]int{2, 1, 0, -1, -2} {
		t.Fatalf("second target frame=%+v", second)
	}
	if len(second.Mode5) != 3 || !second.Mode5[2].Secondary || second.Mode5[2].Frame != 5 || second.Mode5[2].Channel != 0 {
		t.Fatalf("second secondary layers=%+v", second.Mode5)
	}
}

func TestNativeCommand6ZeroSideDrawsOnlyMode5Main(t *testing.T) {
	schedule, _ := BuildNativeCommand6PresentationSchedule(0, command6TestAnimation())
	frame, err := PlanNativeCommand6TargetFrame(NewNativeCommand6TargetState(), schedule, NativeCommand6Coordinates(10, schedule.BaseByte), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame.Mode4) != 0 || len(frame.Mode5) != 5 {
		t.Fatalf("zero-side layers mode4=%d mode5=%d", len(frame.Mode4), len(frame.Mode5))
	}
}

func TestOriginalFDOTHERCommand6ResourcesMatchRecoveredSignatures(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	for _, tc := range []struct {
		resource int
		side     byte
	}{{32, 1}, {33, 0}} {
		animation, err := DecodeResource(path, tc.resource)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil {
			t.Fatalf("resource %d: %v", tc.resource, err)
		}
		schedule, err := BuildNativeCommand6PresentationSchedule(tc.side, animation)
		if err != nil || schedule.EffectResource != tc.resource {
			t.Fatalf("resource %d schedule=%+v err=%v", tc.resource, schedule, err)
		}
	}
	for _, sample := range []int{1, 2, 3} {
		raw, err := fdother.ReadNestedResource(path, NativeCommand6SoundResource, sample)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil || len(raw) == 0 {
			t.Fatalf("resource 87 sample %d len=%d err=%v", sample, len(raw), err)
		}
	}
}
