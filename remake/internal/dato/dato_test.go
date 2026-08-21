package dato

import (
	"encoding/binary"
	"os"
	"testing"
)

func TestParseDATOHighRunAndOpaqueZero(t *testing.T) {
	data := make([]byte, 16+4*(4+4))
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint32(data[i*4:], uint32(16+i*8))
		binary.LittleEndian.PutUint16(data[16+i*8:], 2)
		binary.LittleEndian.PutUint16(data[18+i*8:], 2)
		copy(data[20+i*8:], []byte{0xc2, byte(i + 1), 0})
	}
	frames, err := ParseResource(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 4 || frames[0].Pixels[0] != 1 || frames[0].Pixels[1] != 1 || frames[0].Pixels[2] != 0 || frames[3].Pixels[0] != 4 {
		t.Fatalf("frames=%#v", frames)
	}
	dst := []byte{9, 9, 9, 9}
	if err := frames[0].BlitAt(dst, 2, 0, 0); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 1 || dst[1] != 1 || dst[2] != 0 || dst[3] != 0 {
		t.Fatalf("opaque blit=%v", dst)
	}
}

func TestDATOPlayerResourceIfPresent(t *testing.T) {
	path := "../../../org_game/炎龍騎士團/FLAME2/DATO.DAT"
	if _, err := os.Stat(path); err != nil {
		t.Skip("player-provided DATO.DAT is absent")
	}
	frames, err := DecodeResource(path, 37)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 4 || frames[0].Width != 80 || frames[0].Height != 80 || len(frames[0].Pixels) != 6400 {
		t.Fatalf("DATO#37=%#v", frames)
	}
}

func TestDATOFrameBlitAtOffsetIsOpaque(t *testing.T) {
	dst := make([]byte, 320*20)
	frame := Frame{Width: 2, Height: 2, Pixels: []byte{0, 7, 8, 0}}
	if err := frame.BlitAtOffset(dst, 320, 3*320+4); err != nil {
		t.Fatal(err)
	}
	if dst[3*320+4] != 0 || dst[3*320+5] != 7 || dst[4*320+4] != 8 || dst[4*320+5] != 0 {
		t.Fatalf("DATO offset blit=%v", dst[3*320+4:3*320+6])
	}
}

func TestDATOFrameBlitRightToLeftUsesRightEdgeAnchor(t *testing.T) {
	dst := make([]byte, 8*4)
	frame := Frame{Width: 3, Height: 2, Pixels: []byte{1, 2, 3, 4, 5, 6}}
	if err := frame.BlitRightToLeftAtOffset(dst, 8, 1*8+5); err != nil {
		t.Fatal(err)
	}
	if got := dst[1*8+3 : 1*8+6]; got[0] != 3 || got[1] != 2 || got[2] != 1 {
		t.Fatalf("第一列由右往左寫入=%v", got)
	}
	if got := dst[2*8+3 : 2*8+6]; got[0] != 6 || got[1] != 5 || got[2] != 4 {
		t.Fatalf("第二列stride=%v", got)
	}
}
