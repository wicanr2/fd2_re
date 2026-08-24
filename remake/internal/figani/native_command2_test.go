package figani

import (
	"fmt"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func command2TestAnimation() *Animation {
	frames := make([]Frame, NativeCommand2EffectFrameCount)
	for i := range frames {
		frames[i] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(i)}, Mask: []byte{1}}
	}
	return &Animation{Frames: frames, HeaderByte2: NativeCommand2EffectFrameCount}
}

func command2SequenceAnimation(delay int) *Animation {
	frames := make([]Frame, NativeCommand2EffectFrameCount)
	for index := range frames {
		frames[index] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(index)}, Mask: []byte{1}, Delay: delay}
	}
	return &Animation{Frames: frames, HeaderByte2: NativeCommand2EffectFrameCount}
}

func TestNativeCommand2SequencesPreservePhaseBudgetsAndMarkers(t *testing.T) {
	front, frontEnd, err := BuildNativeCommand2FrontSequence(0, command2SequenceAnimation(4))
	if err != nil || len(front) != NativeCommand2FrontFrames || frontEnd.Frame != 7 || frontEnd.Repeat != 1 {
		t.Fatalf("front len=%d end=%+v err=%v", len(front), frontEnd, err)
	}
	if !front[28].Sample1 {
		t.Fatal("raw-side-zero front omitted the pre-draw frame7 sample1 marker")
	}
	target, targetEnd, err := BuildNativeCommand2TargetSequence()
	if err != nil || len(target) != NativeCommand2TargetFrames || targetEnd.Frame != 16 {
		t.Fatalf("target len=%d end=%+v err=%v", len(target), targetEnd, err)
	}
	var hpSteps []int
	for step, frame := range target {
		if frame.HPStage != 0 {
			hpSteps = append(hpSteps, step)
			if !frame.Sample2 || frame.HPStage != len(hpSteps) {
				t.Fatalf("target step%d frame=%+v", step, frame)
			}
		}
	}
	if fmt.Sprint(hpSteps) != "[0 2 4 6 8 10]" {
		t.Fatalf("HP steps=%v", hpSteps)
	}
	transition := BuildNativeCommand2TransitionSequence(targetEnd)
	var sampleSteps []int
	for step, frame := range transition {
		if frame.Sample2 {
			sampleSteps = append(sampleSteps, step)
		}
		if frame.NumericMarker || frame.HPStage != 0 {
			t.Fatalf("boundary step%d leaked numeric marker", step)
		}
	}
	if fmt.Sprint(sampleSteps) != "[0 2 4 6 8]" {
		t.Fatalf("boundary sample steps=%v", sampleSteps)
	}
	tail, err := BuildNativeCommand2TailSequence(1, frontEnd.Repeat, command2SequenceAnimation(4))
	if err != nil || len(tail) != NativeCommand2TailFrames || tail[0].EffectFrame != 10 {
		t.Fatalf("tail len=%d first=%+v err=%v", len(tail), tail[0], err)
	}
}

func TestNativeCommand2SchedulePreservesRawResources(t *testing.T) {
	for _, tc := range []struct {
		side     byte
		resource int
	}{{0, 27}, {1, 26}, {2, 26}} {
		schedule, err := BuildNativeCommand2PresentationSchedule(tc.side, command2TestAnimation())
		if err != nil {
			t.Fatal(err)
		}
		if schedule.EffectResource != tc.resource || schedule.SoundResource != 83 || schedule.SoundIndices != [3]int{1, 2, 3} {
			t.Fatalf("side=%d schedule=%+v", tc.side, schedule)
		}
	}
}

func TestOriginalFDOTHERCommand2EffectBanksMatchRecoveredSignatures(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	for _, tc := range []struct {
		resource int
		side     byte
	}{{26, 1}, {27, 0}} {
		animation, err := DecodeResource(path, tc.resource)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil {
			t.Fatalf("resource %d: %v", tc.resource, err)
		}
		schedule, err := BuildNativeCommand2PresentationSchedule(tc.side, animation)
		if err != nil || schedule.EffectResource != tc.resource {
			t.Fatalf("resource %d schedule=%+v err=%v", tc.resource, schedule, err)
		}
		front, end, err := BuildNativeCommand2FrontSequence(tc.side, animation)
		if err != nil || len(front) != NativeCommand2FrontFrames {
			t.Fatalf("resource %d front len=%d end=%+v err=%v", tc.resource, len(front), end, err)
		}
		if tail, err := BuildNativeCommand2TailSequence(tc.side, end.Repeat, animation); err != nil || len(tail) != NativeCommand2TailFrames {
			t.Fatalf("resource %d tail len=%d err=%v", tc.resource, len(tail), err)
		}
	}
	for _, sample := range []int{1, 2, 3} {
		raw, err := fdother.ReadNestedResource(path, NativeCommand2SoundResource, sample)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil || len(raw) == 0 {
			t.Fatalf("resource 83 sample %d len=%d err=%v", sample, len(raw), err)
		}
	}
}
