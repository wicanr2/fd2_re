package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestComposeNativeCommand2TargetFramePreservesLayerOrder(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 18), HeaderByte2: 18}
	for i := range effect.Frames {
		effect.Frames[i] = figani.Frame{X: -1, Y: 1, Width: 1, Height: 1, Pixels: []byte{byte(i + 1)}, Mask: []byte{1}}
	}
	schedule, err := figani.BuildNativeCommand2PresentationSchedule(1, effect)
	if err != nil {
		t.Fatal(err)
	}
	target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}}
	got, err := ComposeNativeCommand2TargetFrame(make([]byte, 320*200), target, effect, schedule, 1, 16)
	if err != nil {
		t.Fatal(err)
	}
	// mode5 state16 is the final draw at the same pixel.
	if got[0] != 17 {
		t.Fatalf("composed pixel=%d want state16 value17", got[0])
	}
}

func TestComposeNativeCommand2TargetFrameRejectsPartialBank(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 18), HeaderByte2: 18}
	for i := range effect.Frames {
		effect.Frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}}
	}
	schedule, _ := figani.BuildNativeCommand2PresentationSchedule(1, effect)
	effect.Frames = effect.Frames[:17]
	if _, err := ComposeNativeCommand2TargetFrame(make([]byte, 320*200), figani.Frame{Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}}, effect, schedule, 1, 16); err == nil {
		t.Fatal("partial command2 bank accepted")
	}
	effect = &figani.Animation{Frames: make([]figani.Frame, 18), HeaderByte2: 18}
	if _, err := ComposeNativeCommand2TargetFrame(make([]byte, 320*200), figani.Frame{}, effect, schedule, 0, 16); err == nil {
		t.Fatal("raw side accepted the opposite resource schedule")
	}
}

func TestComposeNativeCommand2OrbitPreservesSideLayerOrder(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 18), HeaderByte2: 18}
	for i := range effect.Frames {
		effect.Frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(i + 1)}, Mask: []byte{1}}
	}
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{70}, Mask: []byte{1}}}}
	target := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{80}, Mask: []byte{1}}}}
	helper := figani.NativeCommand2HelperFrame{EffectFrame: 7}
	nonzero, err := ComposeNativeCommand2OrbitFrame(make([]byte, 320*200), actor, target, effect, helper, 1)
	if err != nil || nonzero[0] != 80 {
		t.Fatalf("nonzero pixel=%d err=%v", nonzero[0], err)
	}
	zero, err := ComposeNativeCommand2OrbitFrame(make([]byte, 320*200), actor, target, effect, helper, 0)
	if err != nil || zero[0] != 8 {
		t.Fatalf("zero pixel=%d err=%v", zero[0], err)
	}
}
