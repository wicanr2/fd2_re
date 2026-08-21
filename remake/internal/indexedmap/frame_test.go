package indexedmap

import (
	"bytes"
	"errors"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestSeedNativeCh23StagingUsesZeroOffsetAnd456Stride(t *testing.T) {
	work := make([]byte, NativeUnitPresentWorkSize)
	staging := make([]byte, 312*192)
	for row := 0; row < 192; row++ {
		for col := 0; col < 312; col++ {
			staging[row*312+col] = byte(row + col)
		}
	}
	if err := SeedNativeCh23Staging(work, staging); err != nil {
		t.Fatal(err)
	}
	for _, row := range []int{0, 1, 191} {
		for _, col := range []int{0, 17, 311} {
			if got, want := work[workBase+row*workStride+col], staging[row*312+col]; got != want {
				t.Fatalf("row=%d col=%d got=%d want=%d", row, col, got, want)
			}
		}
	}
	if work[workBase+312] != 0 {
		t.Fatal("ch23 staging overwrote the 456-stride row padding")
	}
	short := make([]byte, NativeUnitPresentWorkSize-1)
	before := append([]byte(nil), short...)
	if err := SeedNativeCh23Staging(short, staging); err == nil || !bytes.Equal(short, before) {
		t.Fatal("invalid ch23 work buffer was accepted or partially mutated")
	}
}

func solid(v byte) fdicon.Sprite {
	pixels, mask := make([]byte, 24*24), make([]byte, 24*24)
	for i := range pixels {
		pixels[i], mask[i] = v, 1
	}
	return fdicon.Sprite{Pixels: pixels, Mask: mask, RemapMask: make([]byte, 24*24)}
}

func bank(n int, v byte) *fdicon.Bank {
	b := &fdicon.Bank{Sprites: make([]fdicon.Sprite, n)}
	for i := range b.Sprites {
		b.Sprites[i] = solid(v)
	}
	return b
}

func TestComposeFramePreservesNativeLayerOrder(t *testing.T) {
	cache := &fdicon.NativeSelectorCache{}
	if _, err := cache.SlotFor(0); err != nil {
		t.Fatal(err)
	}
	foreground := bank(12, 0)
	foreground.Sprites[1] = solid(4) // tile 0 + foreground index-one rule
	work, vga := make([]byte, 456*300), make([]byte, NativeMapVGASize)
	cells := make([]fdicon.NativeTerrainCell, 13*8)
	for i := range cells {
		cells[i].BlitMode = 0xff
	}
	in := FrameInput{
		TerrainBank: bank(12, 1), RangeBank: bank(20, 2), UnitBank: bank(12, 3), ForegroundBank: foreground,
		SelectorCache: cache, Cells: cells, Controls: []byte{0x80, 0, 0, 0}, LUT: make([]byte, 256), MapWidth: 13,
		RangeMode: 1, Units: []fdicon.NativeUnitLayerEntry{{X: 0, Y: 0, Slot: 0}}, ForegroundUnits: []fdicon.NativeForegroundLayerEntry{{X: 0, Y: 0}},
	}
	if err := ComposeFrame(work, vga, in, func(frame []byte) error {
		off := workBase
		if frame[off] != 4 { // terrain -> range -> unit -> foreground
			t.Fatalf("pre-HUD pixel=%d, want foreground 4", frame[off])
		}
		frame[off] = 5
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if vga[steadyViewportOffset] != 5 {
		t.Fatalf("viewport pixel=%d, want HUD 5", vga[steadyViewportOffset])
	}
	if vga[steadyViewportOffset-1] != 0 || vga[steadyViewportOffset+steadyViewportWidth] != 0 {
		t.Fatal("steady viewport overwrote the native four-pixel border")
	}
}

func TestComposeFrameRejectsMissingHUDBeforeMutation(t *testing.T) {
	work, vga := make([]byte, 456*300), make([]byte, NativeMapVGASize)
	beforeWork, beforeVGA := append([]byte(nil), work...), append([]byte(nil), vga...)
	if err := ComposeFrame(work, vga, FrameInput{}, nil); err == nil {
		t.Fatal("incomplete input accepted")
	}
	if string(work) != string(beforeWork) {
		t.Fatal("rejected input mutated work buffer")
	}
	if string(vga) != string(beforeVGA) {
		t.Fatal("rejected input mutated VGA buffer")
	}
}

func TestComposeFrameMode6MutatesBetweenTerrainAndForegroundAtomically(t *testing.T) {
	cache := &fdicon.NativeSelectorCache{}
	if _, err := cache.SlotFor(0); err != nil {
		t.Fatal(err)
	}
	makeInput := func(cells []fdicon.NativeTerrainCell) FrameInput {
		return FrameInput{
			TerrainBank: bank(12, 1), RangeBank: bank(20, 2),
			UnitBank: bank(12, 3), ForegroundBank: bank(12, 4),
			SelectorCache: cache, Cells: cells,
			Controls: []byte{0, 0, 0, 0}, LUT: make([]byte, 256),
			MapWidth: 13, RangeMode: 6, CursorX: 1, CursorY: 0,
		}
	}
	cells := make([]fdicon.NativeTerrainCell, 13*8)
	for i := range cells {
		cells[i].BlitMode = 0xff
	}
	work, vga := make([]byte, workStride*320), make([]byte, NativeMapVGASize)
	if err := ComposeFrame(work, vga, makeInput(cells), func([]byte) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if cells[1].BlitMode != 0 || cells[0].BlitMode != 0xff {
		t.Fatalf("mode6 cells=%#v/%#v", cells[0], cells[1])
	}

	failedCells := make([]fdicon.NativeTerrainCell, 13*8)
	for i := range failedCells {
		failedCells[i].BlitMode = 0xff
	}
	beforeWork, beforeVGA := append([]byte(nil), work...), append([]byte(nil), vga...)
	if err := ComposeFrame(work, vga, makeInput(failedCells), func([]byte) error {
		return errors.New("HUD fail")
	}); err == nil {
		t.Fatal("mode6 frame accepted failed HUD")
	}
	if failedCells[1].BlitMode != 0xff ||
		string(work) != string(beforeWork) || string(vga) != string(beforeVGA) {
		t.Fatal("failed mode6 frame leaked field or framebuffer mutation")
	}
}

func TestComposeNativeFrameBindsRecoveredHUDInsteadOfCallback(t *testing.T) {
	// 0x11cfa passes work+0x8088 to 0x1acf3. The HUD offsets are relative to
	// that pointer and the following viewport copy uses the same base.
	work := make([]byte, workStride*320)
	vga := make([]byte, NativeMapVGASize)
	terrain := bank(2, 0)
	terrain.Sprites[0] = solid(1)
	terrain.Sprites[1] = solid(0x66)
	rangeBank := bank(20, 0)
	units := bank(12, 0)
	units.Sprites[0] = solid(2)
	units.Sprites[1] = solid(0x77)
	cache := &fdicon.NativeSelectorCache{}
	if _, err := cache.SlotFor(0); err != nil {
		t.Fatal(err)
	}
	cells := make([]fdicon.NativeTerrainCell, 13*8)
	for i := range cells {
		cells[i].BlitMode = 0xff
	}
	in := NativeFrameInput{
		Frame: FrameInput{
			TerrainBank: terrain, RangeBank: rangeBank, UnitBank: units, ForegroundBank: terrain, SelectorCache: cache,
			Cells: cells, Controls: []byte{0, 2, 0, 0, 0, 2, 0, 0}, MapWidth: 13,
			RangeMode: 1, Units: []fdicon.NativeUnitLayerEntry{{X: 0, Y: 0, Slot: 0}},
		},
		HUD: NativeMapHUDInput{DisplayGateA: true, DisplayGateB: true, AnchorX: 136, TerrainDescriptor: 1, TerrainControl: 2,
			OptionalUnit: &NativeMapHUDOptionalUnit{SelectorSlot: 0, RawState: 3, Current: 7, Maximum: 8}},
		Frames: hudFrames(), HUDTerrain: terrain, HUDUnits: units, HUDCache: cache,
	}
	if err := ComposeNativeFrame(work, vga, in); err != nil {
		t.Fatal(err)
	}
	layout, _ := fdicon.NativeMapHUDLayoutFor(136, workStride)
	if layout.Terrain != layout.Unit || work[workBase+layout.Frame] != 0x5a ||
		work[workBase+layout.Terrain] != 0x77 ||
		work[workBase+layout.Unit] != 0x77 ||
		work[workBase+layout.HP] != 0x70 {
		t.Fatalf("native HUD missing from viewport-relative work: %#x/%#x/%#x/%#x",
			work[workBase+layout.Frame], work[workBase+layout.Terrain],
			work[workBase+layout.Unit], work[workBase+layout.HP])
	}
	// 0x11eb0 places the same viewport-relative HUD position at VGA (4,4).
	if got := vga[(157+4)*viewWidth+136+4]; got != 0x5a {
		t.Fatalf("native HUD did not reach viewport: %#x", got)
	}
}

func TestComposeNativeTransitionFramePreservesNativeLayerOrder(t *testing.T) {
	work := make([]byte, workStride*320)
	vga := make([]byte, viewWidth*viewHeight)
	cells := make([]fdicon.NativeTerrainCell, 13*8)
	for i := range cells {
		cells[i].BlitMode = 0xff
	}
	cache := &fdicon.NativeSelectorCache{}
	if _, err := cache.SlotFor(0); err != nil {
		t.Fatal(err)
	}
	identity := make([]byte, 256)
	for i := range identity {
		identity[i] = byte(i)
	}
	pass, err := fdother.BuildNativeIndexedTransitionPass(6, 6, 10, 0, 192)
	if err != nil {
		t.Fatal(err)
	}
	in := NativeTransitionFrameInput{
		TerrainBank: bank(12, 7), UnitBank: bank(12, 0), ForegroundBank: bank(12, 0), SelectorCache: cache,
		Cells: cells, Controls: []byte{0, 0, 0, 0}, TerrainLUT: identity, MapWidth: 13,
	}
	if err := ComposeNativeTransitionFrame(work, vga, in, pass, identity); err != nil {
		t.Fatal(err)
	}
	if got := vga[0]; got != 7 {
		t.Fatalf("transition viewport first pixel=%d want terrain 7", got)
	}
}

func TestComposeNativeTransitionFrameRejectsMissingRawInputAtomically(t *testing.T) {
	work := make([]byte, workStride*320)
	vga := make([]byte, viewWidth*viewHeight)
	work[0], vga[0] = 9, 8
	pass, err := fdother.BuildNativeIndexedTransitionPass(6, 6, 10, 0, 192)
	if err != nil {
		t.Fatal(err)
	}
	if err := ComposeNativeTransitionFrame(work, vga, NativeTransitionFrameInput{}, pass, nil); err == nil {
		t.Fatal("missing transition input accepted")
	}
	if work[0] != 9 || vga[0] != 8 {
		t.Fatal("rejected transition mutated caller buffers")
	}
}

func TestNativeUnitPresentSnapshotRedrawAndViewportStaySeparated(t *testing.T) {
	work := make([]byte, NativeUnitPresentWorkSize)
	vga := make([]byte, viewWidth*viewHeight)
	cells := make([]fdicon.NativeTerrainCell, 13*8)
	for i := range cells {
		cells[i].BlitMode = 0xff
	}
	cache := &fdicon.NativeSelectorCache{}
	if _, err := cache.SlotFor(0); err != nil {
		t.Fatal(err)
	}
	identity := make([]byte, 256)
	for i := range identity {
		identity[i] = byte(i)
	}
	foreground := bank(12, 0)
	foreground.Sprites[1] = solid(4)
	in := NativeTransitionFrameInput{
		TerrainBank: bank(12, 1), UnitBank: bank(12, 3), ForegroundBank: foreground,
		SelectorCache: cache, Cells: cells, Controls: []byte{0x80, 0, 0, 0},
		TerrainLUT: identity, MapWidth: 13,
		Units:           []fdicon.NativeUnitLayerEntry{{X: 0, Y: 0, Slot: 0}},
		ForegroundUnits: []fdicon.NativeForegroundLayerEntry{{X: 0, Y: 0}},
	}
	if err := ComposeNativeUnitPresentTerrainSnapshot(work, in); err != nil {
		t.Fatal(err)
	}
	if got := work[workBase]; got != 1 {
		t.Fatalf("terrain snapshot pixel=%d", got)
	}
	if err := RedrawNativeUnitPresentObjects(work, in); err != nil {
		t.Fatal(err)
	}
	if got := work[workBase]; got != 4 {
		t.Fatalf("object redraw pixel=%d", got)
	}
	for y := 0; y < viewHeight; y++ {
		for x := 312; x < viewWidth; x++ {
			vga[y*viewWidth+x] = 0xee
		}
	}
	if err := CopyNativeUnitPresentViewport(vga, work); err != nil {
		t.Fatal(err)
	}
	if vga[0] != 4 || vga[312] != 0xee {
		t.Fatalf("unit-present viewport=%d tail=%#x", vga[0], vga[312])
	}
}

func TestNativeUnitPresentSnapshotRejectsWrongAllocationAtomically(t *testing.T) {
	work := make([]byte, NativeUnitPresentWorkSize-1)
	work[0] = 9
	if err := ComposeNativeUnitPresentTerrainSnapshot(work, NativeTransitionFrameInput{}); err == nil || work[0] != 9 {
		t.Fatalf("short snapshot mutation/error=%v first=%d", err, work[0])
	}
	if err := RedrawNativeUnitPresentObjects(work, NativeTransitionFrameInput{}); err == nil || work[0] != 9 {
		t.Fatalf("short redraw mutation/error=%v first=%d", err, work[0])
	}
}

func TestComposeNativeUnitPresentIntroAndLUTFramesProduceViewport(t *testing.T) {
	work := make([]byte, NativeUnitPresentWorkSize)
	snapshot := make([]byte, NativeUnitPresentWorkSize)
	vga := make([]byte, viewWidth*viewHeight)
	cells := make([]fdicon.NativeTerrainCell, 13*8)
	for i := range cells {
		cells[i].BlitMode = 0xff
	}
	cache := &fdicon.NativeSelectorCache{}
	if _, err := cache.SlotFor(0); err != nil {
		t.Fatal(err)
	}
	identity := make([]byte, 256)
	for i := range identity {
		identity[i] = byte(i)
	}
	in := NativeTransitionFrameInput{
		TerrainBank: bank(12, 1), UnitBank: bank(12, 3), ForegroundBank: bank(12, 0),
		SelectorCache: cache, Cells: cells, Controls: []byte{0, 0, 0, 0},
		TerrainLUT: identity, MapWidth: 13,
	}
	if err := ComposeNativeUnitPresentTerrainSnapshot(snapshot, in); err != nil {
		t.Fatal(err)
	}
	entry := fdother.LMI1Entry{Width: 1, Height: 1, Pixels: []byte{7}}
	// Native origin for map(0,0), camera(0,0) is viewport x=0,y=1.
	if err := ComposeNativeUnitPresentIntroFrame(work, vga, snapshot, in, entry, 0, 0); err != nil {
		t.Fatal(err)
	}
	if got := vga[viewWidth]; got != 7 {
		t.Fatalf("intro viewport LMI pixel=%d", got)
	}
	lutSnapshot := make([]byte, NativeUnitPresentWorkSize)
	if err := ComposeNativeUnitPresentLUTSnapshot(lutSnapshot, snapshot, in, entry, 0, 0); err != nil {
		t.Fatal(err)
	}
	lmiOrigin := fdother.NativeUnitPresentByteOrigin(0, 0, 0, 0)
	if lutSnapshot[lmiOrigin] != 7 || snapshot[lmiOrigin] == 7 {
		t.Fatalf("LUT snapshot did not preserve separate terrain+final-LMI phase: lut=%d terrain=%d",
			lutSnapshot[lmiOrigin], snapshot[lmiOrigin])
	}
	frames, err := fdother.NativeUnitPresentLUTFrames(3, 2)
	if err != nil {
		t.Fatal(err)
	}
	lut := make([]byte, 256)
	for i := range lut {
		lut[i] = byte(i)
	}
	if err := ComposeNativeUnitPresentLUTFrame(work, vga, lutSnapshot, lut, in, frames[5]); err != nil {
		t.Fatal(err)
	}
	if vga[0] != 1 {
		t.Fatalf("LUT viewport terrain pixel=%d", vga[0])
	}
}

func TestComposeNativeUnitPresentLUTSnapshotRejectsInvalidInputAtomically(t *testing.T) {
	dst := make([]byte, NativeUnitPresentWorkSize)
	dst[0] = 9
	entry := fdother.LMI1Entry{Width: 1, Height: 1, Pixels: []byte{7}}
	if err := ComposeNativeUnitPresentLUTSnapshot(
		dst, make([]byte, NativeUnitPresentWorkSize-1),
		NativeTransitionFrameInput{}, entry, 0, 0,
	); err == nil || dst[0] != 9 {
		t.Fatalf("invalid LUT snapshot input mutated destination: err=%v first=%d", err, dst[0])
	}
}

func TestNativeUnitPresentStripLayoutMatches22390(t *testing.T) {
	top := NativeUnitPresentStripLayoutFor(7, 3, 5, 3)
	if top != (NativeUnitPresentStripLayout{
		WorkOffset: workBase + 2*24,
		VGAOffset:  2 * 24,
		Rows:       18,
	}) {
		t.Fatalf("top-row layout=%+v", top)
	}
	lower := NativeUnitPresentStripLayoutFor(7, 6, 5, 3)
	if lower != (NativeUnitPresentStripLayout{
		WorkOffset: workBase + 2*24 + (3*24-6)*workStride,
		VGAOffset:  2*24 + (3*24-6)*viewWidth,
		Rows:       24,
	}) {
		t.Fatalf("lower layout=%+v", lower)
	}
}

func TestRunNativeUnitPresentStripBridgeCopiesRowsProgressively(t *testing.T) {
	work := make([]byte, NativeUnitPresentWorkSize)
	vga := make([]byte, viewWidth*viewHeight)
	layout := NativeUnitPresentStripLayoutFor(2, 2, 0, 0)
	for row := 0; row < layout.Rows; row++ {
		for x := 0; x < 24; x++ {
			work[layout.WorkOffset+row*workStride+x] = byte(row + 1)
		}
	}
	var delayed []int
	err := RunNativeUnitPresentStripBridge(work, vga, 2, 2, 0, 0, func(row int) error {
		delayed = append(delayed, row)
		if got := vga[layout.VGAOffset+row*viewWidth]; got != byte(row+1) {
			t.Fatalf("row %d was not visible before delay: %d", row, got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(delayed) != 24 {
		t.Fatalf("delay calls=%d", len(delayed))
	}
	if vga[layout.VGAOffset+23*viewWidth+23] != 24 {
		t.Fatal("last 24-byte row was not copied")
	}
}

func TestRunNativeUnitPresentStripBridgePreflightsBounds(t *testing.T) {
	work := make([]byte, NativeUnitPresentWorkSize)
	vga := make([]byte, viewWidth*viewHeight)
	vga[0] = 9
	calls := 0
	err := RunNativeUnitPresentStripBridge(work, vga, 0, 0, 5, 5, func(int) error {
		calls++
		return nil
	})
	if err == nil || calls != 0 || vga[0] != 9 {
		t.Fatalf("out-of-range bridge was not atomic: err=%v calls=%d first=%d", err, calls, vga[0])
	}
}

func TestComposeNativeUnitPresentStripBridgeUsesNoFullViewportCopy(t *testing.T) {
	work := make([]byte, NativeUnitPresentWorkSize)
	snapshot := make([]byte, NativeUnitPresentWorkSize)
	vga := make([]byte, viewWidth*viewHeight)
	cells := make([]fdicon.NativeTerrainCell, 13*8)
	for i := range cells {
		cells[i].BlitMode = 0xff
	}
	cache := &fdicon.NativeSelectorCache{}
	if _, err := cache.SlotFor(0); err != nil {
		t.Fatal(err)
	}
	identity := make([]byte, 256)
	for i := range identity {
		identity[i] = byte(i)
	}
	in := NativeTransitionFrameInput{
		UnitBank: bank(12, 3), ForegroundBank: bank(12, 0),
		SelectorCache: cache, Cells: cells, Controls: []byte{0, 0, 0, 0},
		TerrainLUT: identity, MapWidth: 13,
	}
	layout := NativeUnitPresentStripLayoutFor(2, 2, 0, 0)
	for row := 0; row < layout.Rows; row++ {
		for x := 0; x < 24; x++ {
			snapshot[layout.WorkOffset+row*workStride+x] = byte(row + 1)
		}
	}
	vga[0] = 0xa5 // outside the revealed strip; a full viewport copy would overwrite it.
	delays := 0
	err := ComposeNativeUnitPresentStripBridge(
		work, vga, snapshot, identity, in,
		2, 2, 2, 2,
		func(int) error { delays++; return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if delays != 24 || vga[0] != 0xa5 {
		t.Fatalf("bridge delays=%d untouched VGA=%#x", delays, vga[0])
	}
	if got := vga[layout.VGAOffset]; got != 1 {
		t.Fatalf("first revealed row=%d", got)
	}
}

func TestBuildNativeTerrainCellsRequiresExporterArrays(t *testing.T) {
	cells, err := BuildNativeTerrainCells([]int{1, 2}, []byte{0xff, 0x00})
	if err != nil || len(cells) != 2 || cells[0].Tile != 1 || cells[1].BlitMode != 0 {
		t.Fatalf("cells=%#v err=%v", cells, err)
	}
	for _, tc := range []struct {
		tiles []int
		modes []byte
	}{
		{[]int{1}, nil},
		{[]int{0x400}, []byte{0}},
	} {
		if _, err := BuildNativeTerrainCells(tc.tiles, tc.modes); err == nil {
			t.Fatalf("accepted incomplete/invalid arrays: %#v", tc)
		}
	}
}
