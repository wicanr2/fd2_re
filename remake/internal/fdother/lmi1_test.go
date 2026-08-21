package fdother

import (
	"encoding/binary"
	"os"
	"testing"
)

func TestParseLMI1NativeCodec(t *testing.T) {
	// Two entries: literal 0, literal 0xc0, then a 0xc3 repeat of 7.
	data := make([]byte, 6+2*4)
	copy(data, "LMI1")
	binary.LittleEndian.PutUint16(data[4:], 2)
	first, second := 14, 22
	binary.LittleEndian.PutUint32(data[6:], uint32(first))
	binary.LittleEndian.PutUint32(data[10:], uint32(second))
	data = append(data, 3, 0, 1, 0, 0, 0xc0, 1, 2)
	data = append(data, 2, 0, 1, 0, 0xc2, 7)
	entries, err := ParseLMI1(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Width != 3 || entries[0].Height != 1 {
		t.Fatalf("entries=%#v", entries)
	}
	want0 := []byte{0, 0xc0, 1}
	for i, v := range want0 {
		if entries[0].Pixels[i] != v {
			t.Fatalf("entry0 pixel %d=%#x, want %#x", i, entries[0].Pixels[i], v)
		}
	}
	if entries[1].Width != 2 || entries[1].Height != 1 || len(entries[1].Pixels) != 2 || entries[1].Pixels[0] != 7 || entries[1].Pixels[1] != 7 {
		t.Fatalf("entry1=%#v", entries[1])
	}
}

func TestParseLMI1RejectsMalformedCodec(t *testing.T) {
	data := make([]byte, 14)
	copy(data, "LMI1")
	binary.LittleEndian.PutUint16(data[4:], 1)
	binary.LittleEndian.PutUint32(data[6:], 10)
	binary.LittleEndian.PutUint16(data[10:], 2)
	binary.LittleEndian.PutUint16(data[12:], 1)
	if _, err := ParseLMI1(data); err == nil {
		t.Fatal("truncated LMI1 stream must fail closed")
	}
}

func TestParseOpaqueRunCellNativeCodec(t *testing.T) {
	// The opaque 0x4e8af path has no LMI1 container: its entry begins
	// directly with u16 width/height followed by the same high-run stream.
	data := []byte{3, 0, 1, 0, 0, 0xc0, 0xc2, 7}
	entry, err := ParseOpaqueRunCell(data)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Width != 3 || entry.Height != 1 {
		t.Fatalf("geometry=%dx%d, want 3x1", entry.Width, entry.Height)
	}
	if got, want := entry.Pixels, []byte{0, 0xc0, 7}; string(got) != string(want) {
		t.Fatalf("pixels=%#v, want %#v", got, want)
	}
}

func TestParseOpaqueRunCellRejectsMalformedInput(t *testing.T) {
	for _, data := range [][]byte{
		nil,
		{1, 0, 0, 0},
		{1, 0, 1, 0, 0xc2}, // repeat lacks its palette-index byte
	} {
		if _, err := ParseOpaqueRunCell(data); err == nil {
			t.Fatalf("malformed opaque-run cell %#v was accepted", data)
		}
	}
}

func TestLMI1BlitPreservesTransparentAndMirrors(t *testing.T) {
	e := LMI1Entry{Width: 3, Height: 1, Pixels: []byte{1, 0, 2}}
	dst := make([]byte, 16)
	for i := range dst {
		dst[i] = 9
	}
	if err := e.BlitAt(dst, 8, 1, 0, false); err != nil {
		t.Fatal(err)
	}
	if got, want := dst[:5], []byte{9, 1, 9, 2, 9}; string(got) != string(want) {
		t.Fatalf("forward blit=%v, want %v", got, want)
	}
	if err := e.BlitAt(dst, 8, 1, 1, true); err != nil {
		t.Fatal(err)
	}
	if got, want := dst[8:13], []byte{9, 2, 9, 1, 9}; string(got) != string(want) {
		t.Fatalf("mirrored blit=%v, want %v", got, want)
	}
}

func TestLMI1OpaqueBlitMatches4E8AFZeroWrites(t *testing.T) {
	e := LMI1Entry{Width: 3, Height: 1, Pixels: []byte{1, 0, 2}}
	dst := []byte{9, 9, 9, 9, 9, 9, 9, 9}
	if err := e.BlitOpaqueAt(dst, 8, 1, 0, false); err != nil {
		t.Fatal(err)
	}
	if got, want := dst[:5], []byte{9, 1, 0, 2, 9}; string(got) != string(want) {
		t.Fatalf("opaque blit=%v, want %v", got, want)
	}
	if err := e.BlitOpaqueAt(dst, 8, 4, 0, true); err != nil {
		t.Fatal(err)
	}
	if got, want := dst[4:7], []byte{2, 0, 1}; string(got) != string(want) {
		t.Fatalf("opaque mirror=%v, want %v", got, want)
	}
}

func TestFDOTHER005LMI1UIContainer(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(datPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	entries, err := DecodeLMI1Resource(datPath, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 138 {
		t.Fatalf("FDOTHER#5 LMI1 entry count=%d, want 138", len(entries))
	}
	// Native 0x1f42d selects LMI1 entry #0x52 for its battle-entry split slide.
	if e := entries[0x52]; e.Width != 72 || e.Height != 14 || len(e.Pixels) != 72*14 {
		t.Fatalf("FDOTHER#5 LMI1 entry#0x52=%dx%d pixels=%d, want 72x14", e.Width, e.Height, len(e.Pixels))
	}
}

func TestFDOTHER005MapHUDUses4ModeFrameEntries(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(datPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	// 0x1aeb1 has literal native immediates 0x83/0x84, so these are
	// hexadecimal directory indices 131/132, not decimal 83/84.
	for index, want := range map[int][2]int{0x83: {6, 7}, 0x84: {6, 5}, 130: {69, 34}} {
		frame, err := DecodeLMI1FrameResource(datPath, 5, index)
		if err != nil {
			t.Fatalf("entry %#x: %v", index, err)
		}
		if [2]int{frame.Width, frame.Height} != want {
			t.Fatalf("entry %#x geometry=%dx%d, want %dx%d", index, frame.Width, frame.Height, want[0], want[1])
		}
		if err := frame.BlitAt(make([]byte, 320*200), 320, 320*157+1, -1); err != nil {
			t.Fatalf("entry %#x 0x4e63d decode: %v", index, err)
		}
	}
}

func TestFDOTHER005DialogueFrameUsesRawLMI1EntryPath(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(datPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	cell, err := DecodeLMI1RawEntry(datPath, 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if cell.Width != 3 || cell.Height != 3 || len(cell.Pixels) != 9 || cell.Pixels[0] != 0x60 || cell.Pixels[1] != 0xbe || cell.Pixels[2] != 0xbd {
		t.Fatalf("raw dialogue cell=%#v", cell)
	}
	// 0xbe/bd are literal indexed pixels on this native path, not repeat codes.
	if err := cell.BlitAt(make([]byte, 16), 4, 0, 0); err != nil {
		t.Fatal(err)
	}
}

func TestParseLMI1FrameEntryRejectsOtherContainerOrIndex(t *testing.T) {
	if _, err := ParseLMI1FrameEntry([]byte("bad"), 0); err == nil {
		t.Fatal("non-LMI1 container was accepted")
	}
	data := make([]byte, 10)
	copy(data, "LMI1")
	binary.LittleEndian.PutUint16(data[4:], 1)
	binary.LittleEndian.PutUint32(data[6:], 10)
	if _, err := ParseLMI1FrameEntry(data, 1); err == nil {
		t.Fatal("out-of-range LMI1 index was accepted")
	}
}

func TestFDOTHER006NativeUnitPresentBank(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(datPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	entries, err := DecodeLMI1Resource(datPath, 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 230 {
		t.Fatalf("FDOTHER#6 LMI1 entry count=%d, want 230", len(entries))
	}
	// 0x22470 reads #6 entries 0x72..0x7c; 0x22253's +0x1f6
	// directory pointer is entry 0x7c. These geometry anchors keep the
	// native unit-present resource separate from #81's unused allocation.
	if e := entries[0x72]; e.Width != 12 || e.Height != 21 {
		t.Fatalf("FDOTHER#6 entry#0x72=%dx%d, want 12x21", e.Width, e.Height)
	}
	for index := 0x73; index <= 0x7b; index++ {
		if e := entries[index]; e.Width != 20 || e.Height != 22 {
			t.Fatalf("FDOTHER#6 entry#%#x=%dx%d, want 20x22", index, e.Width, e.Height)
		}
	}
	if e := entries[0x7c]; e.Width != 24 || e.Height != 23 {
		t.Fatalf("FDOTHER#6 entry#0x7c=%dx%d, want 24x23", e.Width, e.Height)
	}
}

func TestFDOTHER005And006CommandHealTailDescriptors(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(datPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	digits, err := DecodeLMI1Resource(datPath, NativeCommandHealTailDigitResource)
	if err != nil {
		t.Fatal(err)
	}
	effect, err := DecodeLMI1Resource(datPath, NativeCommandHealTailEffectResource)
	if err != nil {
		t.Fatal(err)
	}
	if len(digits) <= NativeCommandHealTailDigitBias+9 {
		t.Fatalf("FDOTHER#5 count=%d lacks descriptors %#x..%#x", len(digits), NativeCommandHealTailDigitBias, NativeCommandHealTailDigitBias+9)
	}
	if len(effect) < NativeCommandHealTailEffectStart+NativeCommandHealTailEffectFrames {
		t.Fatalf("FDOTHER#6 count=%d lacks descriptors %#x..%#x", len(effect), NativeCommandHealTailEffectStart, NativeCommandHealTailEffectStart+NativeCommandHealTailEffectFrames-1)
	}
	for index := NativeCommandHealTailDigitBias; index <= NativeCommandHealTailDigitBias+9; index++ {
		if entry := digits[index]; entry.Width <= 0 || entry.Height <= 0 || len(entry.Pixels) != entry.Width*entry.Height {
			t.Fatalf("FDOTHER#5 descriptor %#x malformed", index)
		}
	}
	for index := NativeCommandHealTailEffectStart; index < NativeCommandHealTailEffectStart+NativeCommandHealTailEffectFrames; index++ {
		if entry := effect[index]; entry.Width <= 0 || entry.Height <= 0 || len(entry.Pixels) != entry.Width*entry.Height {
			t.Fatalf("FDOTHER#6 descriptor %#x malformed", index)
		}
	}
}
