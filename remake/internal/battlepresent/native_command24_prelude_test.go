package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func command24PreludeDAC(value byte) []byte {
	dac := make([]byte, 256*3)
	for i := range dac {
		dac[i] = value
	}
	return dac
}

func TestBuildNativeCommand24PreludeFramesMatchesBoth29164Branches(t *testing.T) {
	base := make([]byte, NativeCommand24BackgroundWidth*NativeCommand24BackgroundHeight)
	for i := range base {
		base[i] = 1
	}
	idle := figani.Frame{X: 100, Y: 10, Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}
	platform := fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 7}}

	right, err := BuildNativeCommand24PreludeFrames(NativeCommand24PreludeInput{
		Base: base, ActorIdle: idle, RawSide: 0, BaselineDAC: command24PreludeDAC(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(right) != 9 || right[0].Stage != 8 || right[8].Stage != 0 ||
		right[0].Pixels[10*320+20] != 9 || right[8].Pixels[10*320+100] != 9 ||
		right[0].DAC[0] != 12 || right[8].DAC[0] != 60 {
		t.Fatalf("right branch does not match 0x29164: first=%#v last=%#v", right[0], right[8])
	}

	left, err := BuildNativeCommand24PreludeFrames(NativeCommand24PreludeInput{
		Base: base, ActorIdle: idle, Platform: &platform, RawSide: 2, BaselineDAC: command24PreludeDAC(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 9 || left[0].Pixels[10*320+180] != 9 || left[8].Pixels[10*320+100] != 9 ||
		left[0].Pixels[157*320+244] != 7 || left[8].Pixels[157*320+164] != 7 ||
		left[0].DAC[0] != 12 || left[8].DAC[0] != 60 {
		t.Fatalf("left branch does not match 0x29164: first=%#v last=%#v", left[0], left[8])
	}
}

func TestBuildNativeCommand24PreludeFramesFailsClosed(t *testing.T) {
	base := make([]byte, NativeCommand24BackgroundWidth*NativeCommand24BackgroundHeight)
	idle := figani.Frame{X: 100, Y: 10, Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}
	if frames, err := BuildNativeCommand24PreludeFrames(NativeCommand24PreludeInput{
		Base: base, ActorIdle: idle, RawSide: 1, BaselineDAC: command24PreludeDAC(60),
	}); err == nil || frames != nil {
		t.Fatalf("missing nonzero-side TAI accepted: frames=%v err=%v", frames, err)
	}
	badDAC := command24PreludeDAC(60)
	badDAC[0] = 64
	if frames, err := BuildNativeCommand24PreludeFrames(NativeCommand24PreludeInput{
		Base: base, ActorIdle: idle, RawSide: 0, BaselineDAC: badDAC,
	}); err == nil || frames != nil {
		t.Fatalf("invalid native DAC accepted: frames=%v err=%v", frames, err)
	}
}

func TestBuildNativeCommandPreludeFramesMatches29164ModeZeroBothSides(t *testing.T) {
	base := make([]byte, NativeCommand24BackgroundWidth*NativeCommand24BackgroundHeight)
	actor := figani.Frame{X: 100, Y: 10, Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}
	target := figani.Frame{X: 40, Y: 20, Width: 1, Height: 1, Pixels: []byte{8}, Mask: []byte{1}}
	platform := fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 7}}

	right, err := BuildNativeCommandPreludeFrames(NativeCommandPreludeInput{
		Base: base, ActorIdle: actor, FirstTargetIdle: target, Platform: &platform,
		RawSide: 0, Mode: 0, BaselineDAC: command24PreludeDAC(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(right) != 9 || right[0].Pixels[10*320+20] != 9 || right[8].Pixels[10*320+100] != 9 ||
		right[0].Pixels[20*320+40] != 8 || right[8].Pixels[157*320+164] != 7 {
		t.Fatalf("mode0 right branch mismatch first=%#v last=%#v", right[0], right[8])
	}

	left, err := BuildNativeCommandPreludeFrames(NativeCommandPreludeInput{
		Base: base, ActorIdle: actor, FirstTargetIdle: target, Platform: &platform,
		RawSide: 2, Mode: 0, BaselineDAC: command24PreludeDAC(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 9 || left[0].Pixels[10*320+180] != 9 || left[8].Pixels[10*320+100] != 9 ||
		left[0].Pixels[20*320+40] != 8 || left[0].Pixels[157*320+244] != 7 {
		t.Fatalf("mode0 left branch mismatch first=%#v last=%#v", left[0], left[8])
	}
}

func TestBuildNativeCommandPreludeFramesModeZeroFailsWithoutTargetOrTAI(t *testing.T) {
	base := make([]byte, NativeCommand24BackgroundWidth*NativeCommand24BackgroundHeight)
	actor := figani.Frame{X: 1, Y: 1, Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}
	if frames, err := BuildNativeCommandPreludeFrames(NativeCommandPreludeInput{
		Base: base, ActorIdle: actor, RawSide: 0, Mode: 0, BaselineDAC: command24PreludeDAC(60),
	}); err == nil || frames != nil {
		t.Fatalf("mode0 accepted missing TAI/target: frames=%v err=%v", frames, err)
	}
	platform := fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 7}}
	if frames, err := BuildNativeCommandPreludeFrames(NativeCommandPreludeInput{
		Base: base, ActorIdle: actor, Platform: &platform, RawSide: 0, Mode: 0,
		BaselineDAC: command24PreludeDAC(60),
	}); err == nil || frames != nil {
		t.Fatalf("mode0 accepted missing target: frames=%v err=%v", frames, err)
	}
}
