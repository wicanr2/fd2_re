package ending

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestIndexedCompositorCopiesBlitsAndClampsPalette(t *testing.T) {
	c := NewIndexedCompositor()
	if err := c.Blit(fdother.Frame{X: 1, Y: 1, Width: 2, Height: 1, Pixels: []byte{2, 0, 1, 0, 1, 9}}, c.Offscreen, Width); err != nil {
		t.Fatal(err)
	}
	if err := c.CopyToVGA(c.Offscreen); err != nil {
		t.Fatal(err)
	}
	if c.VGA[Width+1] != 9 || c.VGA[Width+2] != 9 {
		t.Fatalf("blit/copy=%v", c.VGA[Width:Width+4])
	}
	c.Palette[3], c.Palette[4], c.Palette[5] = 2, 62, 63
	if err := c.PaletteDelta(1, 1, 4); err != nil {
		t.Fatal(err)
	}
	if got := c.Palette[3:6]; got[0] != 6 || got[1] != 63 || got[2] != 63 {
		t.Fatalf("palette=%v", got)
	}
}
