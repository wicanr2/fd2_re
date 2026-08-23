package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestComposeNativeCommand6TargetFramePreservesModeOrder(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 10), HeaderByte2: 10}
	for index := range effect.Frames {
		effect.Frames[index] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(index + 1)}, Mask: []byte{1}}
	}
	target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}}
	frame := figani.NativeCommand6TargetFrame{
		Mode4: []figani.NativeCommand6Layer{{Mode: 4, Channel: 0, Frame: 4}},
		Mode5: []figani.NativeCommand6Layer{{Mode: 5, Channel: 0, Frame: 5}},
	}
	got, err := ComposeNativeCommand6TargetFrame(make([]byte, 320*200), target, effect, frame)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 6 {
		t.Fatalf("composed pixel=%d want final mode5 frame5", got[0])
	}
}

func TestComposeNativeCommand6TargetFrameRejectsInvalidLayer(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 10), HeaderByte2: 10}
	frame := figani.NativeCommand6TargetFrame{Mode4: []figani.NativeCommand6Layer{{Frame: 10}}}
	if _, err := ComposeNativeCommand6TargetFrame(make([]byte, 320*200), figani.Frame{}, effect, frame); err == nil {
		t.Fatal("out-of-range command6 layer accepted")
	}
}

func TestComposeNativeCommand6OrbitFramePreservesLayerOrder(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, figani.NativeCommand6EffectFrameCount), HeaderByte2: figani.NativeCommand6EffectFrameCount}
	effect.Frames[4] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}
	frame := figani.NativeCommand6OrbitFrame{
		First:  []figani.NativeCommand6Layer{{Mode: 1, Frame: 4, X: 1, Y: 1}},
		Second: []figani.NativeCommand6Layer{{Mode: 2, Frame: 4, X: 2, Y: 1}},
	}
	got, err := ComposeNativeCommand6OrbitFrame(make([]byte, 320*200), effect, frame)
	if err != nil {
		t.Fatal(err)
	}
	if got[1*320+1] != 7 || got[1*320+2] != 7 {
		t.Fatalf("orbit pixels=%d,%d", got[1*320+1], got[1*320+2])
	}
}
