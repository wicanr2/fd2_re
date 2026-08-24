package battlepresent

import (
	"github.com/wicanr2/fd2_re/remake/internal/figani"
	"testing"
)

func command8CompositorEffect() *figani.Animation {
	frames := make([]figani.Frame, figani.NativeCommand8EffectFrameCount)
	for i := range frames {
		frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(i + 1)}, Mask: []byte{1}}
	}
	return &figani.Animation{Frames: frames, HeaderByte2: figani.NativeCommand8EffectFrameCount}
}

func TestComposeNativeCommand8TargetDrawsEffectAfterTarget(t *testing.T) {
	target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}, X: 30, Y: 10}
	frame := figani.NativeCommand8Frame{Layers: []figani.NativeCommand8Layer{{Channel: 0, Frame: 8}}}
	got, err := ComposeNativeCommand8TargetFrame(make([]byte, 320*200), target, command8CompositorEffect(), frame)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 9 {
		t.Fatalf("pixel=%d want effect frame8 after target", got[0])
	}
}

func TestComposeNativeCommand8TransitionUsesSelectedOffset(t *testing.T) {
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}}}
	target := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}}}
	got, err := ComposeNativeCommand8TransitionFrame(make([]byte, 320*200), actor, target, command8CompositorEffect(), figani.NativeCommand8TransitionFrame{TargetOffsetX: 4})
	if err != nil || got[4] != 9 {
		t.Fatalf("target pixel=%d err=%v", got[4], err)
	}
}

func TestComposeNativeCommand8RejectsInvalidLayer(t *testing.T) {
	frame := figani.NativeCommand8Frame{Layers: []figani.NativeCommand8Layer{{Frame: figani.NativeCommand8EffectFrameCount}}}
	if _, err := ComposeNativeCommand8TargetFrame(make([]byte, 320*200), figani.Frame{}, command8CompositorEffect(), frame); err == nil {
		t.Fatal("out-of-range command8 layer accepted")
	}
}
