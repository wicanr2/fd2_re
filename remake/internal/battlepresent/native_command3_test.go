package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func command3CompositorEffect() *figani.Animation {
	frames := make([]figani.Frame, figani.NativeCommand3EffectFrameCount)
	for index := range frames {
		frames[index] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(index + 1)}, Mask: []byte{1}}
	}
	return &figani.Animation{Frames: frames, HeaderByte2: figani.NativeCommand3EffectFrameCount}
}

func TestComposeNativeCommand3TargetDrawsEffectAfterTarget(t *testing.T) {
	target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}, X: 30, Y: 10}
	frame := figani.NativeCommand3Frame{Layers: []figani.NativeCommand3Layer{{Channel: 0, Frame: 22, X: 30, Y: 10}}}
	got, err := ComposeNativeCommand3TargetFrame(make([]byte, 320*200), target, command3CompositorEffect(), frame)
	if err != nil {
		t.Fatal(err)
	}
	if got[10*320+30] != 23 {
		t.Fatalf("pixel=%d want effect frame22 after target", got[10*320+30])
	}
}

func TestComposeNativeCommand3TransitionUsesSelectedOffset(t *testing.T) {
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}}}
	target := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}}}
	got, err := ComposeNativeCommand3TransitionFrame(make([]byte, 320*200), actor, target, command3CompositorEffect(), figani.NativeCommand3TransitionFrame{TargetOffsetX: 4})
	if err != nil || got[4] != 9 {
		t.Fatalf("target pixel=%d err=%v", got[4], err)
	}
}

func TestComposeNativeCommand3RejectsInvalidLayer(t *testing.T) {
	frame := figani.NativeCommand3Frame{Layers: []figani.NativeCommand3Layer{{Frame: figani.NativeCommand3EffectFrameCount}}}
	if _, err := ComposeNativeCommand3TargetFrame(make([]byte, 320*200), figani.Frame{}, command3CompositorEffect(), frame); err == nil {
		t.Fatal("out-of-range command3 layer accepted")
	}
}
