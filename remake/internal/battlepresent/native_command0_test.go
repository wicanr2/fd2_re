package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestComposeNativeCommand0TargetFramePreservesBackTargetFrontOrder(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 16), HeaderByte2: 16}
	for i := range effect.Frames {
		effect.Frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(10 + i)}, Mask: []byte{1}}
	}
	effect.Frames[9].RawByte4 = 1
	schedule, err := figani.BuildNativeCommand0PresentationSchedule(1, effect)
	if err != nil {
		t.Fatal(err)
	}
	// Collapse all active layers at step 12 onto one pixel. Back layers 2/4
	// are covered by the target; front layers 0/1/3/5/6 then cover it in
	// native order, so layer6's frame0 is the final value 10.
	schedule.XOffsets = [7]int{}
	schedule.YOffsets = [7]int{}
	target := figani.Frame{X: 0, Y: 0, Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}}
	got, err := ComposeNativeCommand0TargetFrame(make([]byte, 320*200), target, effect, schedule, 12)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 10 {
		t.Fatalf("composed pixel=%d want layer6 frame0 value10", got[0])
	}
}

func TestComposeNativeCommand0TargetFrameUsesSideShiftAndRejectsPartialPublish(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 16), HeaderByte2: 16}
	for i := range effect.Frames {
		effect.Frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}
	}
	effect.Frames[9].RawByte4 = 1
	schedule, err := figani.BuildNativeCommand0PresentationSchedule(0, effect)
	if err != nil {
		t.Fatal(err)
	}
	target := figani.Frame{X: 1, Y: 1, Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}
	base := make([]byte, 320*200)
	got, err := ComposeNativeCommand0TargetFrame(base, target, effect, schedule, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got[40+148] != 7 || got[1+320] != 9 {
		t.Fatalf("side-shift effect=%d target=%d", got[40+148], got[1+320])
	}
	bad := *effect
	bad.Frames = bad.Frames[:15]
	if _, err := ComposeNativeCommand0TargetFrame(base, target, &bad, schedule, 0); err == nil {
		t.Fatal("partial effect bank accepted")
	}
}
