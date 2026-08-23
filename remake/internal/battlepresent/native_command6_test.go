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
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}}}
	frame := figani.NativeCommand6OrbitFrame{
		First:  []figani.NativeCommand6Layer{{Mode: 1, Frame: 4, X: 1, Y: 1}},
		Second: []figani.NativeCommand6Layer{{Mode: 2, Frame: 4, X: 2, Y: 1}},
	}
	got, err := ComposeNativeCommand6OrbitFrame(make([]byte, 320*200), actor, nil, effect, frame)
	if err != nil {
		t.Fatal(err)
	}
	if got[1*320+1] != 7 || got[1*320+2] != 7 {
		t.Fatalf("orbit pixels=%d,%d", got[1*320+1], got[1*320+2])
	}
	if got[0] != 9 {
		t.Fatalf("orbit actor pixel=%d", got[0])
	}
}

func TestComposeNativeCommand6TailOrbitRequiresAndDrawsTarget(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, figani.NativeCommand6EffectFrameCount)}
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}}}
	target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}, X: 2}
	frame := figani.NativeCommand6OrbitFrame{DrawTarget: true}
	if _, err := ComposeNativeCommand6OrbitFrame(make([]byte, 320*200), actor, nil, effect, frame); err == nil {
		t.Fatal("tail without target accepted")
	}
	got, err := ComposeNativeCommand6OrbitFrame(make([]byte, 320*200), actor, &target, effect, frame)
	if err != nil || got[2] != 9 {
		t.Fatalf("tail target pixel=%d err=%v", got[2], err)
	}
}

func TestComposeNativeCommand6TransitionFrameUsesLastActorAndSelectedTarget(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, figani.NativeCommand6EffectFrameCount)}
	actor := &figani.Animation{Frames: []figani.Frame{
		{Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}},
		{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}},
	}}
	target := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}}}
	got, err := ComposeNativeCommand6TransitionFrame(make([]byte, 320*200), actor, target, effect, figani.NativeCommand6TransitionFrame{TargetOffsetX: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 7 || got[2] != 9 {
		t.Fatalf("transition actor/target pixels=%d,%d", got[0], got[2])
	}
}
