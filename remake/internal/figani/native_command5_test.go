package figani

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func command5TestAnimation() *Animation {
	frames := make([]Frame, NativeCommand5EffectFrameCount)
	for index := range frames {
		frames[index] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(index + 1)}, Mask: []byte{1}}
	}
	return &Animation{Frames: frames, HeaderByte2: NativeCommand5EffectFrameCount}
}

func TestNativeCommand5SchedulePreservesSideResourcesAndOffsets(t *testing.T) {
	nonzero, err := BuildNativeCommand5PresentationSchedule(1, command5TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	if nonzero.EffectResource != 24 || nonzero.SoundResource != 86 || nonzero.SoundIndices != [2]int{0, 1} ||
		nonzero.XOffsets != [10]int{30, 50, 70, 40, 80, 100, 70, 30, 60, 90} {
		t.Fatalf("nonzero schedule=%+v", nonzero)
	}
	zero, err := BuildNativeCommand5PresentationSchedule(0, command5TestAnimation())
	if err != nil || zero.EffectResource != 25 {
		t.Fatalf("zero schedule=%+v err=%v", zero, err)
	}
}

func TestNativeCommand5Mode0ConsumesSixRNGSteps(t *testing.T) {
	state := NewNativeCommand5State(0x1234)
	wantRNG := uint16(0x1234)
	var wantPhases [6]int
	for channel := range wantPhases {
		wantRNG = fdother.NativeRNGStep(wantRNG)
		wantPhases[channel] = int(wantRNG%2) * 6
	}
	if state.RNG != wantRNG || state.Phases != wantPhases || state.Counters != [6]int{0, -2, -4, -6, -8, -10} ||
		state.Positions != [6]int{0, 1, 2, 3, 4, 5} || state.NextPosition != 6 || state.Stop {
		t.Fatalf("state=%+v", state)
	}
}

func TestNativeCommand5FrontAndTargetPreserveMarkersAndSamples(t *testing.T) {
	schedule, _ := BuildNativeCommand5PresentationSchedule(1, command5TestAnimation())
	state := NewNativeCommand5State(0x4321)
	front, err := PlanNativeCommand5DrawFrame(state, schedule, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(front.Layers) != 1 || !front.PlayPrimary || front.PlaySecondary || front.NumericMarker {
		t.Fatalf("front=%+v", front)
	}
	frames, _, err := BuildNativeCommand5TargetSequence(front.Next, schedule, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantMarkers := map[int]bool{0: true, 2: true, 4: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true}
	published := 0
	for index, frame := range frames {
		if frame.NumericMarker != wantMarkers[index] {
			t.Fatalf("frame %d marker=%v", index, frame.NumericMarker)
		}
		if frame.HPStage != 0 {
			published++
			if frame.HPStage != published {
				t.Fatalf("frame %d stage=%d", index, frame.HPStage)
			}
		}
	}
	if published != NativeCommand5DamageStages {
		t.Fatalf("published=%d", published)
	}
}

func TestNativeCommand5TransitionCarriesStateAndTailStopsReseeding(t *testing.T) {
	schedule, _ := BuildNativeCommand5PresentationSchedule(1, command5TestAnimation())
	front, _ := PlanNativeCommand5DrawFrame(NewNativeCommand5State(7), schedule, 1)
	_, state, err := BuildNativeCommand5TargetSequence(front.Next, schedule, 1)
	if err != nil {
		t.Fatal(err)
	}
	transition, afterTransition, err := BuildNativeCommand5TransitionSequence(state, schedule, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantOffsets := []int{-35, -70, -105, -140, -140, -105, -70, -35, 0}
	for index, frame := range transition {
		if frame.TargetOffsetX != wantOffsets[index] || frame.UseNextTarget != (index >= 4) {
			t.Fatalf("transition %d=%+v", index, frame)
		}
	}
	tail, final, err := BuildNativeCommand5TailSequence(afterTransition, schedule, 1)
	if err != nil || len(tail) != NativeCommand5TailFrames || !final.Stop {
		t.Fatalf("tail=%d final=%+v err=%v", len(tail), final, err)
	}
	for channel, counter := range final.Counters {
		if counter > 14 {
			t.Fatalf("channel %d unexpectedly reset after stop: %d", channel, counter)
		}
	}
}

func TestOriginalFDOTHERCommand5ResourcesMatchRecoveredSignatures(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	for _, tc := range []struct {
		resource int
		side     byte
	}{{24, 1}, {25, 0}} {
		animation, err := DecodeResource(path, tc.resource)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil {
			t.Fatalf("resource %d: %v", tc.resource, err)
		}
		schedule, err := BuildNativeCommand5PresentationSchedule(tc.side, animation)
		if err != nil || schedule.EffectResource != tc.resource {
			t.Fatalf("resource %d schedule=%+v err=%v", tc.resource, schedule, err)
		}
	}
	for _, sample := range []int{0, 1} {
		raw, err := fdother.ReadNestedResource(path, NativeCommand5SoundResource, sample)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil || len(raw) == 0 {
			t.Fatalf("resource 86 sample %d len=%d err=%v", sample, len(raw), err)
		}
	}
}
