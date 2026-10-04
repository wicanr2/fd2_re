package battlepresent

import (
	"encoding/binary"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func solidNativeCommand24Layer(value byte) fdother.Frame {
	pixels := make([]byte, 4, 4+100*10)
	binary.LittleEndian.PutUint16(pixels, 320)
	binary.LittleEndian.PutUint16(pixels[2:], 100)
	for y := 0; y < 100; y++ {
		for run := 0; run < 5; run++ {
			pixels = append(pixels, 63, value)
		}
	}
	return fdother.Frame{Width: 320, Height: 100, Pixels: pixels}
}

func TestBuildNativeCommand24BackgroundFramesMatches29C90Viewports(t *testing.T) {
	source := make([]byte, 320*200)
	target := make([]byte, 320*200)
	for i := range source {
		source[i], target[i] = 40, 50
	}
	idle := figani.Frame{X: 10, Y: 10, Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}}
	frames, err := BuildNativeCommand24BackgroundFrames(NativeCommand24BackgroundInputs{
		Layers: [3]fdother.Frame{solidNativeCommand24Layer(10), solidNativeCommand24Layer(20), solidNativeCommand24Layer(30)},
		Source: source, Target: target, TargetIdle: idle,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 20 {
		t.Fatalf("frame count=%d want 20", len(frames))
	}
	row := 50 * 320
	if frames[0][row] != 10 || frames[0][row+31] != 10 || frames[0][row+32] != 40 {
		t.Fatalf("first slide boundary=%v", frames[0][row:row+34])
	}
	if frames[9][row] != 10 || frames[9][row+319] != 10 {
		t.Fatal("first slide did not end on BG resource0")
	}
	if frames[10][row] != 50 || frames[10][row+31] != 50 || frames[10][row+32] != 30 {
		t.Fatalf("second slide boundary=%v", frames[10][row:row+34])
	}
	if frames[19][10+10*320] != 99 || frames[19][319] != 50 {
		t.Fatal("second slide did not end on target base plus idle frame")
	}
}

func TestBuildNativeCommand24BackgroundFramesRejectsBeforePartialOutput(t *testing.T) {
	_, err := BuildNativeCommand24BackgroundFrames(NativeCommand24BackgroundInputs{
		Layers: [3]fdother.Frame{solidNativeCommand24Layer(1), solidNativeCommand24Layer(2), {}},
		Source: make([]byte, 320*200), Target: make([]byte, 320*200),
		TargetIdle: figani.Frame{X: 0, Y: 0, Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}},
	})
	if err == nil {
		t.Fatal("malformed BG layer was accepted")
	}
}

func TestNativePhysicalRightBackgroundKeeps29DEDAsymmetry(t *testing.T) {
	source, target := make([]byte, 320*200), make([]byte, 320*200)
	for i := range source {
		source[i], target[i] = 40, 50
	}
	in := NativeCommand24BackgroundInputs{
		Layers: [3]fdother.Frame{solidNativeCommand24Layer(10), solidNativeCommand24Layer(20), solidNativeCommand24Layer(30)},
		Source: source, Target: target,
		TargetIdle: figani.Frame{X: 10, Y: 10, Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}},
	}
	frames, err := BuildNativePhysicalRightBackgroundFrames(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 19 {
		t.Fatalf("29DED presents=%d; original is 9+10", len(frames))
	}
	row := 50 * 320
	// 第一段 i=1：offset32，因此前288 pixels仍是source，末32為BG1。
	if frames[0][row+287] != 40 || frames[0][row+288] != 20 || frames[0][row+319] != 20 {
		t.Fatal("29DED first source boundary or BG1 changed")
	}
	// 第一段沒有offset320；最後仍有32 pixels的source。
	if frames[8][row+31] != 40 || frames[8][row+32] != 10 {
		t.Fatal("29DED unexpectedly gained a tenth first slide")
	}
	if frames[9][row+287] != 30 || frames[9][row+288] != 50 ||
		frames[18][10+10*320] != 99 || frames[18][319] != 50 {
		t.Fatal("29DED target viewport or idle origin changed")
	}
	in.Layers[2] = fdother.Frame{}
	if partial, err := BuildNativePhysicalRightBackgroundFrames(in); err == nil || partial != nil {
		t.Fatal("malformed physical scroll returned a partial schedule")
	}
}
