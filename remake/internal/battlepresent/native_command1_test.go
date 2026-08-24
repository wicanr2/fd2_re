package battlepresent

import (
	"fmt"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestComposeNativeCommand1TargetFramePreservesModeOrder(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 30), HeaderByte2: 30}
	for i := range effect.Frames {
		effect.Frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(i + 1)}, Mask: []byte{1}}
	}
	schedule, err := figani.BuildNativeCommand1PresentationSchedule(1, effect)
	if err != nil {
		t.Fatal(err)
	}
	schedule.AnchorX = 0
	schedule.XOffsets = [8]int{}
	schedule.YOffsets = [8]int{}
	target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}}
	got, err := ComposeNativeCommand1TargetFrame(make([]byte, 320*200), target, effect, schedule, 8)
	if err != nil {
		t.Fatal(err)
	}
	// mode5 是最後一層；step8時 slot4 counter0，最後畫 frame0。
	if got[0] != 1 {
		t.Fatalf("composed pixel=%d want final mode5 frame0", got[0])
	}
}

func TestBuildNativeCommand1TargetSequencePreservesThirtyOneFramesAndEightMarkers(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 30), HeaderByte2: 30}
	for i := range effect.Frames {
		effect.Frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(i + 1)}, Mask: []byte{1}}
	}
	schedule, err := figani.BuildNativeCommand1PresentationSchedule(1, effect)
	if err != nil {
		t.Fatal(err)
	}
	target := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}, Delay: 2}}}
	bases := make([][]byte, 9)
	for index := range bases {
		bases[index] = make([]byte, 320*200)
	}
	sequence, err := BuildNativeCommand1TargetSequence(bases, target, effect, schedule)
	if err != nil {
		t.Fatal(err)
	}
	var hpSteps, sampleSteps []int
	for step := range sequence.Frames {
		if sequence.HPStages[step] != 0 {
			hpSteps = append(hpSteps, step)
		}
		if sequence.SampleMarkers[step] {
			sampleSteps = append(sampleSteps, step)
		}
	}
	if len(sequence.Frames) != 31 || fmt.Sprint(hpSteps) != "[8 10 12 14 16 18 20 22]" ||
		fmt.Sprint(sampleSteps) != "[4 6 8 10 12 14 16 18]" {
		t.Fatalf("frames=%d hp=%v sample=%v", len(sequence.Frames), hpSteps, sampleSteps)
	}
	for stage, step := range hpSteps {
		if sequence.HPStages[step] != stage+1 {
			t.Fatalf("step%d HP stage=%d want=%d", step, sequence.HPStages[step], stage+1)
		}
	}
	if sequence.NextIdleFrame != 0 || sequence.NextIdleRepeat != 1 {
		t.Fatalf("next idle=(%d,%d) want=(0,1)", sequence.NextIdleFrame, sequence.NextIdleRepeat)
	}
}

func TestBuildNativeCommand1TransitionFramesUsesNineRecoveredOffsets(t *testing.T) {
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}}}
	current := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{8}, Mask: []byte{1}}}}
	next := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}}}
	frames, err := BuildNativeCommand1TransitionFrames(make([]byte, 320*200), actor, current, next, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 9 || frames[0][35] != 8 || frames[3][140] != 8 || frames[4][140] != 9 || frames[8][0] != 9 {
		t.Fatalf("command1 transition shape/offsets changed")
	}
}

func TestComposeNativeCommand1TargetFrameKeepsThirtyOneFrameTail(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 30), HeaderByte2: 30}
	for i := range effect.Frames {
		effect.Frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(i + 1)}, Mask: []byte{1}}
	}
	schedule, err := figani.BuildNativeCommand1PresentationSchedule(1, effect)
	if err != nil {
		t.Fatal(err)
	}
	schedule.AnchorX = 0
	schedule.XOffsets = [8]int{}
	schedule.YOffsets = [8]int{}
	target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}}
	got, err := ComposeNativeCommand1TargetFrame(make([]byte, 320*200), target, effect, schedule, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 99 {
		t.Fatalf("frame30 pixel=%d want target idle only", got[0])
	}
}

func TestComposeNativeCommand1TargetFrameRejectsPartialBank(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 30), HeaderByte2: 30}
	for i := range effect.Frames {
		effect.Frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}}
	}
	schedule, _ := figani.BuildNativeCommand1PresentationSchedule(1, effect)
	effect.Frames = effect.Frames[:29]
	if _, err := ComposeNativeCommand1TargetFrame(make([]byte, 320*200), figani.Frame{Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}}, effect, schedule, 0); err == nil {
		t.Fatal("partial command1 bank accepted")
	}
}
