package indexedmap

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

func nativeCommandSamplingTestInput() NativeTransitionFrameInput {
	sprites := make([]fdicon.Sprite, 12)
	for i := range sprites {
		pixels := make([]byte, fdicon.NativeSize*fdicon.NativeSize)
		for p := range pixels {
			pixels[p] = byte((p/fdicon.NativeSize)*24 + p%fdicon.NativeSize)
		}
		sprites[i] = fdicon.Sprite{Pixels: pixels, Mask: make([]byte, len(pixels)), RemapMask: make([]byte, len(pixels))}
	}
	return NativeTransitionFrameInput{
		TerrainBank: &fdicon.Bank{Sprites: sprites}, UnitBank: &fdicon.Bank{}, ForegroundBank: &fdicon.Bank{},
		SelectorCache: &fdicon.NativeSelectorCache{}, Cells: make([]fdicon.NativeTerrainCell, 13*8), Controls: make([]byte, 4),
		TerrainLUT: make([]byte, 256), MapWidth: 13,
	}
}

func TestComposeNativeCommand10To12SurfaceIdentitySampling(t *testing.T) {
	in := nativeCommandSamplingTestInput()
	work, vga := make([]byte, NativeUnitPresentWorkSize), make([]byte, NativeMapVGASize)
	if err := ComposeNativeCommand10To12Surface(work, vga, in, 0, 0, 128); err != nil {
		t.Fatal(err)
	}
	if got := vga[0]; got != 0 {
		t.Fatalf("first pixel=%d", got)
	}
	if got := vga[23]; got != 23 {
		t.Fatalf("tile edge=%d", got)
	}
	if got := vga[viewWidth+24]; got != 24 {
		t.Fatalf("next tile row=%d", got)
	}
}

func TestComposeNativeCommand10To12SurfaceRejectsWithoutMutation(t *testing.T) {
	work, vga := make([]byte, NativeUnitPresentWorkSize), make([]byte, NativeMapVGASize)
	work[0], vga[0] = 7, 9
	if err := ComposeNativeCommand10To12Surface(work, vga, NativeTransitionFrameInput{}, 0, 0, 128); err == nil || work[0] != 7 || vga[0] != 9 {
		t.Fatal("malformed sampling input mutated buffers")
	}
}
