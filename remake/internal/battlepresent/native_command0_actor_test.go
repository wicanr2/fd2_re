package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func command0ActorTestInput(side byte) NativeCommand0ActorInput {
	baseBefore := make([]byte, 320*200)
	baseAfter := make([]byte, 320*200)
	baseAfter[1] = 6
	effect := &figani.Animation{Frames: []figani.Frame{
		{X: 10, Y: 10, Width: 1, Height: 1, Pixels: []byte{2}, Mask: []byte{1}, Delay: 1},
		{X: 10, Y: 10, Width: 1, Height: 1, Pixels: []byte{3}, Mask: []byte{1}, Delay: 6, RawByte4: 1},
	}}
	target := &figani.Animation{Frames: []figani.Frame{
		{X: 10, Y: 10, Width: 1, Height: 1, Pixels: []byte{8}, Mask: []byte{1}, Delay: 2},
		{X: 10, Y: 10, Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}, Delay: 2},
	}}
	lut := make([]byte, 256)
	for i := range lut {
		lut[i] = byte(i)
	}
	lut[4], lut[5] = 14, 15
	return NativeCommand0ActorInput{
		BaseBefore: baseBefore, BaseAfter: baseAfter, ActorEffect: effect,
		FirstTargetIdle: target, RawSide: side, LUT: lut,
		Background: fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 4}},
		Platform:   fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 5}},
	}
}

func TestBuildNativeCommand0ActorFramesPublishesOnceAndPulsesSix(t *testing.T) {
	frames, err := BuildNativeCommand0ActorFrames(command0ActorTestInput(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 7 || frames[0].PublishMP || !frames[1].PublishMP {
		t.Fatalf("actor frames/marker=%d %#v %#v", len(frames), frames[0], frames[1])
	}
	for i := range frames {
		wantPulse := i >= 1
		if frames[i].Pulse != wantPulse {
			t.Fatalf("frame %d pulse=%v want=%v", i, frames[i].Pulse, wantPulse)
		}
	}
	if frames[0].Pixels[1] != 0 || frames[1].Pixels[1] != 6 ||
		frames[1].Pixels[50*320] != 14 || frames[1].Pixels[157*320+164] != 15 {
		t.Fatalf("before/after LUT boundary not preserved")
	}
}

func TestBuildNativeCommand0ActorFramesPreservesRawSideLayerOrder(t *testing.T) {
	left, err := BuildNativeCommand0ActorFrames(command0ActorTestInput(1))
	if err != nil {
		t.Fatal(err)
	}
	right, err := BuildNativeCommand0ActorFrames(command0ActorTestInput(0))
	if err != nil {
		t.Fatal(err)
	}
	if left[0].Pixels[10*320+10] != 8 || right[0].Pixels[10*320+10] != 2 {
		t.Fatalf("raw-side order left=%d right=%d", left[0].Pixels[10*320+10], right[0].Pixels[10*320+10])
	}
}

func TestBuildNativeCommand0ActorFramesFailsWithoutUniqueVisibleMPMarker(t *testing.T) {
	in := command0ActorTestInput(1)
	in.ActorEffect.Frames[1].RawByte4 = 0
	if frames, err := BuildNativeCommand0ActorFrames(in); err == nil || frames != nil {
		t.Fatalf("missing marker accepted frames=%v err=%v", frames, err)
	}
	in = command0ActorTestInput(1)
	in.ActorEffect.Frames[1].Delay = 0
	if frames, err := BuildNativeCommand0ActorFrames(in); err == nil || frames != nil {
		t.Fatalf("invisible marker accepted frames=%v err=%v", frames, err)
	}
}
