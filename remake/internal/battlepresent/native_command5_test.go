package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func command5CompositorEffect() *figani.Animation {
	frames := make([]figani.Frame, figani.NativeCommand5EffectFrameCount)
	for index := range frames {
		frames[index] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(index + 1)}, Mask: []byte{1}}
	}
	return &figani.Animation{Frames: frames, HeaderByte2: figani.NativeCommand5EffectFrameCount}
}

func TestComposeNativeCommand5TargetDrawsEffectAfterTarget(t *testing.T) {
	target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}, X: 30}
	frame := figani.NativeCommand5Frame{Layers: []figani.NativeCommand5Layer{{Channel: 0, Frame: 5, X: 30}}}
	got, err := ComposeNativeCommand5TargetFrame(make([]byte, 320*200), target, command5CompositorEffect(), frame)
	if err != nil {
		t.Fatal(err)
	}
	if got[30] != 6 {
		t.Fatalf("pixel=%d want effect frame5 after target", got[30])
	}
}

func TestComposeNativeCommand5OrbitPreservesActorTargetEffectOrder(t *testing.T) {
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}, X: 1}}}
	target := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}, X: 2}}}
	frame := figani.NativeCommand5Frame{Layers: []figani.NativeCommand5Layer{{Frame: 0, X: 2}}}
	got, err := ComposeNativeCommand5OrbitFrame(make([]byte, 320*200), actor, target, command5CompositorEffect(), frame)
	if err != nil {
		t.Fatal(err)
	}
	if got[1] != 7 || got[2] != 1 {
		t.Fatalf("actor/effect pixels=%d,%d", got[1], got[2])
	}
}

func TestComposeNativeCommand5TransitionUsesSelectedOffset(t *testing.T) {
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}}}
	target := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}}}
	got, err := ComposeNativeCommand5TransitionFrame(make([]byte, 320*200), actor, target, command5CompositorEffect(), figani.NativeCommand5TransitionFrame{TargetOffsetX: 4})
	if err != nil || got[4] != 9 {
		t.Fatalf("target pixel=%d err=%v", got[4], err)
	}
}

func TestComposeNativeCommand5RejectsInvalidLayer(t *testing.T) {
	frame := figani.NativeCommand5Frame{Layers: []figani.NativeCommand5Layer{{Frame: figani.NativeCommand5EffectFrameCount}}}
	if _, err := ComposeNativeCommand5TargetFrame(make([]byte, 320*200), figani.Frame{}, command5CompositorEffect(), frame); err == nil {
		t.Fatal("out-of-range command5 layer accepted")
	}
}
