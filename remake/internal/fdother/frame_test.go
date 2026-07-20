package fdother

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func testFrame(x, y, w, h int, rle []byte) []byte {
	d := make([]byte, 13)
	binary.LittleEndian.PutUint16(d, uint16(x))
	binary.LittleEndian.PutUint16(d[2:], uint16(y))
	binary.LittleEndian.PutUint16(d[9:], uint16(w))
	binary.LittleEndian.PutUint16(d[11:], uint16(h))
	return append(d, rle...)
}

func testContainer(frames ...[]byte) []byte {
	d := make([]byte, 8+4*len(frames))
	binary.LittleEndian.PutUint16(d, uint16(len(frames)))
	off := len(d)
	for i, f := range frames {
		binary.LittleEndian.PutUint32(d[8+4*i:], uint32(off))
		d = append(d, f...)
		off += len(f)
	}
	return d
}

func TestFrameBlitPreservesTransparentDestination(t *testing.T) {
	// run(2,5), dither(1,8), literal(1,9), skip(1) -> 5,5,old,8,9,old.
	data := testContainer(testFrame(2, 1, 6, 1, []byte{1, 5, 0x40, 8, 0x80, 9, 0xc0}))
	frames, err := ParseFrames(data)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]byte, 30)
	for i := range dst {
		dst[i] = 7
	}
	if err := frames[0].Blit(dst, 10, -1); err != nil {
		t.Fatal(err)
	}
	want := []byte{5, 5, 7, 8, 9, 7}
	for i, v := range want {
		if got := dst[12+i]; got != v {
			t.Fatalf("pixel %d = %d, want %d", i, got, v)
		}
	}
}

func TestFrameRejectsMalformedRLE(t *testing.T) {
	frames, err := ParseFrames(testContainer(testFrame(0, 0, 2, 1, []byte{1})))
	if err != nil {
		t.Fatal(err)
	}
	if err := frames[0].Blit(make([]byte, 2), 2, -1); err == nil {
		t.Fatal("truncated RLE was accepted")
	}
}

func TestFDOTHER054FrameTable(t *testing.T) {
	const path = "../../../extracted/raw/FDOTHER/FDOTHER_054.bin"
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("player-provided FDOTHER_054 asset is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	frames, err := ParseFrames(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 111 {
		t.Fatalf("frame count = %d, want 111", len(frames))
	}
	for index, want := range map[int][4]int{0: {0, 23, 320, 132}, 9: {116, 39, 86, 81}, 108: {116, 39, 86, 81}, 110: {0, 0, 320, 200}} {
		got := frames[index]
		if [4]int{got.X, got.Y, got.Width, got.Height} != want {
			t.Fatalf("frame %d = %#v, want %v", index, got, want)
		}
	}
	// Each ending caller uses the verified transparent branch against a
	// 320-wide framebuffer. Decoding every native frame here catches a wrong
	// RLE grammar or frame-table boundary without committing original art.
	for i, frame := range frames {
		if err := frame.Blit(make([]byte, 320*200), 320, -1); err != nil {
			t.Fatalf("frame %d does not decode into 320x200: %v", i, err)
		}
	}
}

func TestFDOTHER054ArchiveLoader(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	const rawPath = "../../../extracted/raw/FDOTHER/FDOTHER_054.bin"
	raw, err := os.ReadFile(rawPath)
	if os.IsNotExist(err) {
		t.Skip("player-provided FDOTHER_054 extraction is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	archive, err := os.ReadFile(datPath)
	if os.IsNotExist(err) {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	entry, err := archiveEntry(archive, 0x36)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(entry, raw) {
		t.Fatal("FDOTHER archive entry 0x36 does not equal raw FDOTHER_054")
	}
	frames, err := DecodeResource(datPath, 0x36)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 111 {
		t.Fatalf("loader frame count = %d, want 111", len(frames))
	}
}

func TestFDOTHER056SingleFramePayload(t *testing.T) {
	const path = "../../../extracted/raw/FDOTHER/FDOTHER_056.bin"
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("player-provided FDOTHER_056 asset is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 4 || binary.LittleEndian.Uint16(data) != 320 || binary.LittleEndian.Uint16(data[2:]) != 200 {
		t.Fatalf("#56 header=% x", data[:min(4, len(data))])
	}
	frame, err := ParseSingleFrame(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := frame.Blit(make([]byte, 320*200), 320, -1); err != nil {
		t.Fatalf("#56 single-frame RLE decode: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
