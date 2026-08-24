package figani

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func command3TestAnimation() *Animation {
	frames := make([]Frame, NativeCommand3EffectFrameCount)
	for index := range frames {
		frames[index] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(index + 1)}, Mask: []byte{1}}
	}
	return &Animation{Frames: frames, HeaderByte2: NativeCommand3EffectFrameCount}
}

func TestNativeCommand3SchedulePreservesResourcesAndTables(t *testing.T) {
	nonzero, err := BuildNativeCommand3PresentationSchedule(1, command3TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	if nonzero.EffectResource != 39 || nonzero.SoundResource != 84 ||
		nonzero.XOffsets != [12]int{30, 0, 70, 40, 130, 70, -30, 30, 110, 80, -10, 30} ||
		nonzero.YOffsets != [12]int{0, 10, 0, 20, 5, 20, 0, 10, 0, 18, 0, 10} ||
		nonzero.FrameBases != [12]int{22, 0, 0, 0, 0, 0, 11, 0, 22, 0, 0, 0} {
		t.Fatalf("nonzero schedule=%+v", nonzero)
	}
	zero, err := BuildNativeCommand3PresentationSchedule(0, command3TestAnimation())
	if err != nil || zero.EffectResource != 43 {
		t.Fatalf("zero schedule=%+v err=%v", zero, err)
	}
}

func TestNativeCommand3Mode0IsDeterministic(t *testing.T) {
	state := NewNativeCommand3State()
	if state.Counters != [12]int{0, -2, -4, -6, -8, -10, -12, -14, -16, -18, -20, -22} ||
		state.Positions != [12]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} ||
		state.NextPosition != 12 || state.Stop || state.Toggle != 0 {
		t.Fatalf("state=%+v", state)
	}
}

func TestNativeCommand3ToggleRepeatsBeforeAdvance(t *testing.T) {
	schedule, _ := BuildNativeCommand3PresentationSchedule(1, command3TestAnimation())
	first, err := PlanNativeCommand3DrawFrame(NewNativeCommand3State(), schedule, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Next.Counters[0] != 0 || first.Next.Toggle != 1 || len(first.Layers) != 1 {
		t.Fatalf("first=%+v", first)
	}
	second, err := PlanNativeCommand3DrawFrame(first.Next, schedule, 1)
	if err != nil {
		t.Fatal(err)
	}
	if second.Next.Counters[0] != 1 || second.Next.Counters[1] != -1 || second.Next.Toggle != 0 || !second.PlaySub2 {
		t.Fatalf("second=%+v", second)
	}
}

func TestNativeCommand3FirstTargetMarkersAndSamples(t *testing.T) {
	schedule, _ := BuildNativeCommand3PresentationSchedule(1, command3TestAnimation())
	state := NewNativeCommand3State()
	for index := 0; index < NativeCommand3FrontFrames; index++ {
		front, err := PlanNativeCommand3DrawFrame(state, schedule, 1)
		if err != nil {
			t.Fatal(err)
		}
		state = front.Next
	}
	frames, _, err := BuildNativeCommand3TargetSequence(state, schedule, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantMarkers := map[int]bool{3: true, 7: true, 11: true, 15: true, 19: true, 23: true, 25: true, 27: true, 29: true, 31: true, 33: true, 35: true, 37: true, 39: true}
	wantSub2 := map[int]bool{23: true, 31: true}
	wantSub1 := map[int]bool{7: true, 11: true, 15: true, 19: true, 23: true, 25: true, 29: true, 31: true, 33: true, 37: true, 39: true}
	published := 0
	for index, frame := range frames {
		if frame.NumericMarker != wantMarkers[index] || frame.PlaySub1 != wantSub1[index] || frame.PlaySub2 != wantSub2[index] {
			t.Fatalf("frame %d=%+v", index, frame)
		}
		if frame.HPStage != 0 {
			published++
			if frame.HPStage != published {
				t.Fatalf("frame %d stage=%d", index, frame.HPStage)
			}
		}
	}
	if published != NativeCommand3DamageStages || frames[39].HPStage != 0 {
		t.Fatalf("published=%d last=%+v", published, frames[39])
	}
}

func TestNativeCommand3TransitionAndTailPreserveState(t *testing.T) {
	schedule, _ := BuildNativeCommand3PresentationSchedule(1, command3TestAnimation())
	state := NewNativeCommand3State()
	for index := 0; index < NativeCommand3FrontFrames; index++ {
		front, _ := PlanNativeCommand3DrawFrame(state, schedule, 1)
		state = front.Next
	}
	_, state, _ = BuildNativeCommand3TargetSequence(state, schedule, 1)
	transition, state, err := BuildNativeCommand3TransitionSequence(state, schedule, 1)
	if err != nil || len(transition) != NativeCommand3TransitionFrames {
		t.Fatalf("transition=%d err=%v", len(transition), err)
	}
	tail, final, err := BuildNativeCommand3TailSequence(state, schedule, 1)
	if err != nil || len(tail) != NativeCommand3TailFrames || !final.Stop {
		t.Fatalf("tail=%d final=%+v err=%v", len(tail), final, err)
	}
}

func TestOriginalFDOTHERCommand3ResourcesMatchRecoveredSignatures(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	for _, tc := range []struct {
		resource int
		side     byte
	}{{39, 1}, {43, 0}} {
		animation, err := DecodeResource(path, tc.resource)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil {
			t.Fatalf("resource %d: %v", tc.resource, err)
		}
		schedule, err := BuildNativeCommand3PresentationSchedule(tc.side, animation)
		if err != nil || schedule.EffectResource != tc.resource {
			t.Fatalf("resource %d schedule=%+v err=%v", tc.resource, schedule, err)
		}
	}
	for _, sample := range []int{0, 1, 2} {
		raw, err := fdother.ReadNestedResource(path, NativeCommand3SoundResource, sample)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil || len(raw) == 0 {
			t.Fatalf("resource 84 sample %d len=%d err=%v", sample, len(raw), err)
		}
	}
}
