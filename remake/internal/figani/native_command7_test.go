package figani

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func command7TestAnimation() *Animation {
	frames := make([]Frame, NativeCommand7EffectFrameCount)
	for index := range frames {
		frames[index] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(index + 1)}, Mask: []byte{1}, Delay: []int{2, 2, 2, 2, 4}[index]}
	}
	return &Animation{Frames: frames, HeaderByte2: NativeCommand7EffectFrameCount}
}

func TestNativeCommand7SchedulePreservesResourcesAndOffsets(t *testing.T) {
	nonzero, err := BuildNativeCommand7PresentationSchedule(1, command7TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	if nonzero.EffectResource != 37 || nonzero.SoundResource != 88 || nonzero.SoundIndices != [2]int{0, 1} ||
		nonzero.XOffsets != [10]int{30, -10, 70, 20, 100, 130, 40, 80, 110, 60} {
		t.Fatalf("nonzero schedule=%+v", nonzero)
	}
	zero, err := BuildNativeCommand7PresentationSchedule(0, command7TestAnimation())
	if err != nil || zero.EffectResource != 38 {
		t.Fatalf("zero schedule=%+v err=%v", zero, err)
	}
}

func TestNativeCommand7Mode0PreservesUnusedFourthGroup(t *testing.T) {
	state := NewNativeCommand7State()
	if state.Counters != [4]int{0, -3, -6, -9} || state.Positions != [4]int{0, 1, 2, 3} ||
		state.NextPosition != 4 || state.Stop || state.Toggle != 0 {
		t.Fatalf("state=%+v", state)
	}
}

func TestNativeCommand7ToggleRepeatsBeforeAdvance(t *testing.T) {
	schedule, _ := BuildNativeCommand7PresentationSchedule(1, command7TestAnimation())
	first, err := PlanNativeCommand7DrawFrame(NewNativeCommand7State(), schedule, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Next.Counters != [4]int{0, -3, -6, -9} || first.Next.Toggle != 1 || len(first.Layers) != 1 {
		t.Fatalf("first=%+v", first)
	}
	second, err := PlanNativeCommand7DrawFrame(first.Next, schedule, 1)
	if err != nil {
		t.Fatal(err)
	}
	if second.Next.Counters != [4]int{1, -2, -5, -9} || second.Next.Toggle != 0 || len(second.Layers) != 1 {
		t.Fatalf("second=%+v", second)
	}
}

func TestNativeCommand7FirstTargetMarkersAndSamples(t *testing.T) {
	schedule, _ := BuildNativeCommand7PresentationSchedule(1, command7TestAnimation())
	state := NewNativeCommand7State()
	for index := 0; index < NativeCommand7FrontFrames; index++ {
		front, err := PlanNativeCommand7DrawFrame(state, schedule, 1)
		if err != nil {
			t.Fatal(err)
		}
		state = front.Next
	}
	frames, _, err := BuildNativeCommand7TargetSequence(state, schedule, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantMarkers := map[int]bool{1: true, 7: true, 13: true, 15: true, 21: true, 27: true, 29: true}
	wantA := map[int]bool{1: true, 15: true, 29: true}
	wantB := map[int]bool{7: true, 21: true}
	published := 0
	for index, frame := range frames {
		if frame.NumericMarker != wantMarkers[index] || frame.PlayHandleA != wantA[index] || frame.PlayHandleB != wantB[index] {
			t.Fatalf("frame %d=%+v", index, frame)
		}
		if frame.HPStage != 0 {
			published++
			if frame.HPStage != published {
				t.Fatalf("frame %d stage=%d", index, frame.HPStage)
			}
		}
	}
	if published != NativeCommand7DamageStages {
		t.Fatalf("published=%d", published)
	}
}

func TestNativeCommand7TransitionAndTailPreserveState(t *testing.T) {
	schedule, _ := BuildNativeCommand7PresentationSchedule(1, command7TestAnimation())
	state := NewNativeCommand7State()
	for index := 0; index < NativeCommand7FrontFrames; index++ {
		front, _ := PlanNativeCommand7DrawFrame(state, schedule, 1)
		state = front.Next
	}
	_, state, _ = BuildNativeCommand7TargetSequence(state, schedule, 1)
	transition, state, err := BuildNativeCommand7TransitionSequence(state, schedule, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantOffsets := []int{-35, -70, -105, -140, -140, -105, -70, -35, 0}
	for index, frame := range transition {
		if frame.TargetOffsetX != wantOffsets[index] || frame.UseNextTarget != (index >= 4) {
			t.Fatalf("transition %d=%+v", index, frame)
		}
	}
	tail, final, err := BuildNativeCommand7TailSequence(state, schedule, 1)
	if err != nil || len(tail) != NativeCommand7TailFrames || !final.Stop {
		t.Fatalf("tail=%d final=%+v err=%v", len(tail), final, err)
	}
}

func TestOriginalFDOTHERCommand7ResourcesMatchRecoveredSignatures(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	for _, tc := range []struct {
		resource int
		side     byte
	}{{37, 1}, {38, 0}} {
		animation, err := DecodeResource(path, tc.resource)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil {
			t.Fatalf("resource %d: %v", tc.resource, err)
		}
		schedule, err := BuildNativeCommand7PresentationSchedule(tc.side, animation)
		if err != nil || schedule.EffectResource != tc.resource {
			t.Fatalf("resource %d schedule=%+v err=%v", tc.resource, schedule, err)
		}
	}
	for _, sample := range []int{0, 1} {
		raw, err := fdother.ReadNestedResource(path, NativeCommand7SoundResource, sample)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil || len(raw) == 0 {
			t.Fatalf("resource 88 sample %d len=%d err=%v", sample, len(raw), err)
		}
	}
}
