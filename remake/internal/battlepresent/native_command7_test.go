package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func command7CompositorEffect() *figani.Animation {
	frames := make([]figani.Frame, figani.NativeCommand7EffectFrameCount)
	for index := range frames {
		frames[index] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(index + 1)}, Mask: []byte{1}}
	}
	return &figani.Animation{Frames: frames, HeaderByte2: figani.NativeCommand7EffectFrameCount}
}

func TestComposeNativeCommand7TargetDrawsEffectAfterTarget(t *testing.T) {
	target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}, X: 30}
	frame := figani.NativeCommand7Frame{Layers: []figani.NativeCommand7Layer{{Channel: 0, Frame: 4, X: 30}}}
	got, err := ComposeNativeCommand7TargetFrame(make([]byte, 320*200), target, command7CompositorEffect(), frame)
	if err != nil {
		t.Fatal(err)
	}
	if got[30] != 5 {
		t.Fatalf("pixel=%d want effect frame4 after target", got[30])
	}
}

func TestComposeNativeCommand7OrbitPreservesLayerOrder(t *testing.T) {
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}, X: 1}}}
	target := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}, X: 2}}}
	frame := figani.NativeCommand7Frame{Layers: []figani.NativeCommand7Layer{{Frame: 0, X: 2}}}
	got, err := ComposeNativeCommand7OrbitFrame(make([]byte, 320*200), actor, target, command7CompositorEffect(), frame)
	if err != nil {
		t.Fatal(err)
	}
	if got[1] != 7 || got[2] != 1 {
		t.Fatalf("actor/effect pixels=%d,%d", got[1], got[2])
	}
}

func TestComposeNativeCommand7TransitionUsesSelectedOffset(t *testing.T) {
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}}}
	target := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}}}
	got, err := ComposeNativeCommand7TransitionFrame(make([]byte, 320*200), actor, target, command7CompositorEffect(), figani.NativeCommand7TransitionFrame{TargetOffsetX: 4})
	if err != nil || got[4] != 9 {
		t.Fatalf("target pixel=%d err=%v", got[4], err)
	}
}

func TestComposeNativeCommand7RejectsInvalidLayer(t *testing.T) {
	frame := figani.NativeCommand7Frame{Layers: []figani.NativeCommand7Layer{{Frame: figani.NativeCommand7EffectFrameCount}}}
	if _, err := ComposeNativeCommand7TargetFrame(make([]byte, 320*200), figani.Frame{}, command7CompositorEffect(), frame); err == nil {
		t.Fatal("out-of-range command7 layer accepted")
	}
}
