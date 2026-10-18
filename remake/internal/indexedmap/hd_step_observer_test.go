package indexedmap

import (
	"bytes"
	"errors"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

func TestNativeStepTerrainObserverPreservesBothCallersAndAtomicity(t *testing.T) {
	in := FrameInput{TerrainBank: bank(12, 7), UnitBank: bank(12, 8), ForegroundBank: bank(12, 9),
		SelectorCache: &fdicon.NativeSelectorCache{}, MapWidth: 13, CameraY: 1,
		Cells: make([]fdicon.NativeTerrainCell, 13*10), Controls: []byte{0, 0, 0, 0}, LUT: make([]byte, 256)}
	for i := range in.Cells {
		in.Cells[i].BlitMode = 0xff
	}
	for _, up := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "up-shift"}[up], func(t *testing.T) {
			work := bytes.Repeat([]byte{99}, NativeUnitPresentWorkSize)
			vga := bytes.Repeat([]byte{77}, NativeMapVGASize)
			observedWork, observedVGA := append([]byte(nil), work...), append([]byte(nil), vga...)
			calls := 0
			observer := func(stage FrameStage, w, v []byte) error {
				if stage != FrameStageTerrain {
					t.Fatal("unexpected diagnostic stage")
				}
				calls++
				w[0] = 201
				v[0] = 202
				return nil
			}
			var first, second error
			if up {
				first = ComposeNativeUpStepFrame(work, vga, in, 2, 3)
				second = ComposeNativeUpStepFrameObserved(observedWork, observedVGA, in, 2, 3, observer)
				_, _, rows, base, err := NativeUpStepLayout(in, 2, 3)
				if err != nil || rows != 9 || base != workBase-3*fdicon.NativeMotionPixels*workStride {
					t.Fatal("up-copy geometry changed")
				}
			} else {
				first = ComposeNativeStepFrame(work, vga, in)
				second = ComposeNativeStepFrameObserved(observedWork, observedVGA, in, observer)
			}
			if first != nil || second != nil || calls != 1 || !bytes.Equal(work, observedWork) || !bytes.Equal(vga, observedVGA) {
				t.Fatalf("observer changed original output: %v %v", first, second)
			}
			beforeWork, beforeVGA := append([]byte(nil), observedWork...), append([]byte(nil), observedVGA...)
			refuse := func(FrameStage, []byte, []byte) error { return errors.New("reject diagnostic stage") }
			if up {
				second = ComposeNativeUpStepFrameObserved(observedWork, observedVGA, in, 2, 3, refuse)
			} else {
				second = ComposeNativeStepFrameObserved(observedWork, observedVGA, in, refuse)
			}
			if second == nil || !bytes.Equal(beforeWork, observedWork) || !bytes.Equal(beforeVGA, observedVGA) {
				t.Fatal("rejected stage published partial pixels")
			}
		})
	}
}
