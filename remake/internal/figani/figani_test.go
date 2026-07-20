package figani

import (
	"os"
	"testing"
)

func TestParsePreservesTransparentAndDitherPixels(t *testing.T) {
	// One 4x1 frame: run(7), dither(9), then transparent skip.
	raw := []byte{1, 0, 0, 0, 0, 0, 12, 0, 12, 0, 0, 0, 2, 0, 3, 0, 0, 0, 2, 0, 0, 4, 0, 1, 0, 0x00, 7, 0x40, 9, 0xc0}
	a, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	f := a.Frames[0]
	if f.X != 2 || f.Y != 3 || f.Width != 4 || f.Height != 1 || f.Delay != 2 {
		t.Fatalf("frame=%#v", f)
	}
	dst := make([]byte, 50)
	for i := range dst {
		dst[i] = 1
	}
	if err := f.BlitAt(dst, 10); err != nil {
		t.Fatal(err)
	}
	if got := dst[32:36]; got[0] != 7 || got[1] != 1 || got[2] != 9 || got[3] != 1 {
		t.Fatalf("blit=%v", got)
	}
}

func TestDecodeOriginalFIGANIResource(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FIGANI.DAT"
	a, err := DecodeResource(path, 13)
	if os.IsNotExist(err) {
		t.Skip("player-provided FIGANI.DAT is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Frames) == 0 || a.Frames[0].Width <= 0 || a.Frames[0].Height <= 0 || len(a.Frames[0].Pixels) != a.Frames[0].Width*a.Frames[0].Height {
		t.Fatalf("decoded resource 13 = %#v", a)
	}
}
