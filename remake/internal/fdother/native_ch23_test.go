package fdother

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestDecodeAndBlitNativeCh23StageFromPlayerArchive(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(datPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	frame, err := DecodeNativeCh23Stage(datPath)
	if err != nil {
		t.Fatal(err)
	}
	staging := make([]byte, NativeCh23StageStride*NativeCh23StageHeight)
	if err := BlitNativeCh23Stage(frame, staging); err != nil {
		t.Fatal(err)
	}
	separated, err := LoadSeparatedNativeCh23Stage(
		"../../generated-assets/fd2-original-b97caf22/surfaces",
	)
	if err != nil {
		t.Skipf("separated ch23 stage is absent: %v", err)
	}
	wantIndexed, wantMask, err := frame.IndexedLayers()
	if err != nil {
		t.Fatal(err)
	}
	if separated.X != frame.X || separated.Y != frame.Y ||
		separated.Width != frame.Width || separated.Height != frame.Height ||
		!bytes.Equal(separated.Indexed, wantIndexed) ||
		!bytes.Equal(separated.Mask, wantMask) {
		t.Fatal("separated ch23 stage differs from the original archive")
	}
}

func TestSeparatedNativeCh23StageFailsClosedWithoutPack(t *testing.T) {
	if _, err := LoadSeparatedNativeCh23Stage(t.TempDir()); err == nil {
		t.Fatal("missing separated ch23 stage was accepted")
	}
}

func TestBlitNativeCh23StageRejectsWrongSurfaceWithoutMutation(t *testing.T) {
	frame := Frame{Width: NativeCh23StageWidth, Height: NativeCh23StageHeight, Pixels: []byte{0, 0, 0, 0}}
	staging := make([]byte, NativeCh23StageStride*NativeCh23StageHeight-1)
	staging[0] = 7
	if err := BlitNativeCh23Stage(frame, staging); err == nil {
		t.Fatal("wrong staging surface accepted")
	}
	if staging[0] != 7 {
		t.Fatalf("rejected staging surface mutated: %d", staging[0])
	}
}

func TestBlitNativeCh23StageRejectsMalformedRLEWithoutMutation(t *testing.T) {
	frame := Frame{Width: NativeCh23StageWidth, Height: NativeCh23StageHeight, Pixels: []byte{0, 0, 0, 0}}
	staging := make([]byte, NativeCh23StageStride*NativeCh23StageHeight)
	staging[0], staging[len(staging)-1] = 13, 17
	if err := BlitNativeCh23Stage(frame, staging); err == nil {
		t.Fatal("malformed ch23 RLE accepted")
	}
	if staging[0] != 13 || staging[len(staging)-1] != 17 {
		t.Fatalf("malformed RLE partially mutated staging: %d/%d", staging[0], staging[len(staging)-1])
	}
}

func TestRotateNativeCh23RowsWrapsBottomRowsToTop(t *testing.T) {
	buf := make([]byte, NativeCh23StageStride*NativeCh23StageHeight)
	for row := 0; row < NativeCh23StageHeight; row++ {
		buf[row*NativeCh23StageStride] = byte(row)
	}
	if err := RotateNativeCh23Rows(buf, 2); err != nil {
		t.Fatal(err)
	}
	if buf[0] != 190 || buf[NativeCh23StageStride] != 191 || buf[2*NativeCh23StageStride] != 0 || buf[3*NativeCh23StageStride] != 1 {
		t.Fatalf("rotated rows=%d,%d,%d,%d", buf[0], buf[NativeCh23StageStride], buf[2*NativeCh23StageStride], buf[3*NativeCh23StageStride])
	}
}

func TestRotateNativeCh23RowsRejectsInvalidWithoutMutation(t *testing.T) {
	buf := make([]byte, NativeCh23StageStride*NativeCh23StageHeight)
	buf[0], buf[len(buf)-1] = 7, 9
	want := append([]byte(nil), buf...)
	if err := RotateNativeCh23Rows(buf, NativeCh23StageHeight+1); err == nil {
		t.Fatal("invalid latch accepted")
	}
	for i := range buf {
		if buf[i] != want[i] {
			t.Fatalf("buffer mutated at %d", i)
		}
	}
}

func TestApplyNativeDACPaletteCycleE0EFWritesOnlyNativeRange(t *testing.T) {
	dac := make([]byte, 256*3)
	for i := range dac {
		dac[i] = 63
	}
	if err := ApplyNativeDACPaletteCycleE0EF(dac, 0); err != nil {
		t.Fatal(err)
	}
	if dac[0] != 63 || dac[0xdf*3] != 63 || dac[0xf0*3] != 63 {
		t.Fatal("palette cycle modified an entry outside 0xe0..0xef")
	}
	if got := dac[0xe0*3 : 0xe0*3+3]; got[0] != 0x0e || got[1] != 0x15 || got[2] != 0x26 {
		t.Fatalf("palette phase0 entry=%#v", got)
	}
	if err := ApplyNativeDACPaletteCycleE0EF(dac, 1); err != nil {
		t.Fatal(err)
	}
	if got := dac[0xe0*3 : 0xe0*3+3]; got[0] != 0x0d || got[1] != 0x14 || got[2] != 0x25 {
		t.Fatalf("palette phase1 entry=%#v", got)
	}
}

func TestApplyNativeDACPaletteCycleE0EFRejectsInvalidAtomically(t *testing.T) {
	dac := make([]byte, 256*3)
	dac[0xe0*3] = 11
	want := append([]byte(nil), dac...)
	if err := ApplyNativeDACPaletteCycleE0EF(dac, 16); err == nil {
		t.Fatal("invalid palette phase accepted")
	}
	for i := range dac {
		if dac[i] != want[i] {
			t.Fatalf("palette mutated at %d", i)
		}
	}
}

func TestAdvanceNativeDACPaletteCycleE0EFMatchesUnsignedTickGate(t *testing.T) {
	dac := make([]byte, 256*3)
	phase, tick, advanced, err := AdvanceNativeDACPaletteCycleE0EF(dac, 0, 0, 1)
	if err != nil || advanced || phase != 0 || tick != 0 {
		t.Fatalf("sub-threshold state=(%d,%d,%v) err=%v", phase, tick, advanced, err)
	}
	phase, tick, advanced, err = AdvanceNativeDACPaletteCycleE0EF(dac, phase, tick, 2)
	if err != nil || !advanced || phase != 1 || tick != 2 {
		t.Fatalf("accepted state=(%d,%d,%v) err=%v", phase, tick, advanced, err)
	}
	if got := dac[0xe0*3 : 0xe0*3+3]; got[0] != 0x0d || got[1] != 0x14 || got[2] != 0x25 {
		t.Fatalf("phase1 entry=%#v", got)
	}
	phase, tick, advanced, err = AdvanceNativeDACPaletteCycleE0EF(dac, 15, 0x7fff, -0x7fff)
	if err != nil || !advanced || phase != 0 || tick != -0x7fff {
		t.Fatalf("wrapped state=(%d,%d,%v) err=%v", phase, tick, advanced, err)
	}
}

func TestAdvanceNativeDACPaletteCycleE0EFRejectsAtomically(t *testing.T) {
	dac := make([]byte, 256*3)
	dac[0xe0*3] = 0x3f
	want := append([]byte(nil), dac...)
	if _, _, _, err := AdvanceNativeDACPaletteCycleE0EF(dac, 16, 0, 2); err == nil {
		t.Fatal("invalid phase accepted")
	}
	if !bytes.Equal(dac, want) {
		t.Fatal("rejected cycle mutated DAC")
	}
}

func TestRunNativeCh23InitialLoopPreservesRawSchedule(t *testing.T) {
	staging := make([]byte, NativeCh23StageStride*NativeCh23StageHeight)
	for row := 0; row < NativeCh23StageHeight; row++ {
		staging[row*NativeCh23StageStride] = byte(row)
	}
	draws, ticks, palettes := 0, 0, 0
	firstDrawRow := -1
	var latches []int
	err := RunNativeCh23Loop(NativeCh23LoopSpec{
		Phase:       "initial",
		Repeat:      30,
		StageValues: []int{2, 3, 4, 5, 6, 7, 8, 9},
	}, staging, nil, NativeCh23LoopHooks{
		Latch: func(stage int) error {
			latches = append(latches, stage)
			return nil
		},
		Draw: func() error {
			if draws == 0 {
				firstDrawRow = int(staging[0])
			}
			draws++
			return nil
		},
		Tick:    func() error { ticks++; return nil },
		Palette: func(int) error { palettes++; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if draws != 8*30 || ticks != draws || palettes != 0 || firstDrawRow != 0 {
		t.Fatalf("initial schedule draws=%d ticks=%d palettes=%d firstRow=%d", draws, ticks, palettes, firstDrawRow)
	}
	if !reflect.DeepEqual(latches, []int{2, 3, 4, 5, 6, 7, 8, 9}) {
		t.Fatalf("raw latch stages=%v", latches)
	}
}

func TestRunNativeCh23RequiresRawLatchCallback(t *testing.T) {
	staging := make([]byte, NativeCh23StageStride*NativeCh23StageHeight)
	err := RunNativeCh23Loop(NativeCh23LoopSpec{
		Phase:       "initial",
		Repeat:      30,
		StageValues: []int{2, 3, 4, 5, 6, 7, 8, 9},
	}, staging, nil, NativeCh23LoopHooks{
		Draw: func() error { return nil },
		Tick: func() error { return nil },
	})
	if err == nil {
		t.Fatal("missing raw latch callback accepted")
	}
}

func TestRunNativeCh23PaletteLoopKeepsRawESIOrder(t *testing.T) {
	staging := make([]byte, NativeCh23StageStride*NativeCh23StageHeight)
	dac := bytes.Repeat([]byte{0x5a}, 256*3)
	var rawESI []int
	draws, ticks := 0, 0
	err := RunNativeCh23Loop(NativeCh23LoopSpec{
		Phase:       "palette",
		Repeat:      12,
		StageValues: []int{10, 11, 12, 13, 14},
		Palette:     true,
	}, staging, dac, NativeCh23LoopHooks{
		Latch:   func(int) error { return nil },
		Palette: func(value int) error { rawESI = append(rawESI, value); return nil },
		Draw:    func() error { draws++; return nil },
		Tick:    func() error { ticks++; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rawESI) != 60 || draws != 60 || ticks != 60 {
		t.Fatalf("palette schedule rawESI=%d draws=%d ticks=%d", len(rawESI), draws, ticks)
	}
	for i, value := range rawESI {
		if value != i {
			t.Fatalf("raw ESI[%d]=%d", i, value)
		}
	}
}

func TestRunNativeCh23LoopRollsBackOnCallbackFailure(t *testing.T) {
	staging := bytes.Repeat([]byte{0x31}, NativeCh23StageStride*NativeCh23StageHeight)
	dac := bytes.Repeat([]byte{0x42}, 256*3)
	beforeStage, beforeDAC := append([]byte(nil), staging...), append([]byte(nil), dac...)
	err := RunNativeCh23Loop(NativeCh23LoopSpec{
		Phase:       "palette",
		Repeat:      12,
		StageValues: []int{10, 11, 12, 13, 14},
		Palette:     true,
	}, staging, dac, NativeCh23LoopHooks{
		Latch: func(stage int) error {
			staging[0] = byte(stage)
			return nil
		},
		Palette: func(value int) error {
			if value == 7 {
				return errors.New("synthetic raw palette failure")
			}
			return nil
		},
		Draw: func() error { return nil },
		Tick: func() error { return nil },
	})
	if err == nil {
		t.Fatal("callback failure accepted")
	}
	if !bytes.Equal(staging, beforeStage) || !bytes.Equal(dac, beforeDAC) {
		t.Fatal("callback failure mutated raw buffers")
	}
}
