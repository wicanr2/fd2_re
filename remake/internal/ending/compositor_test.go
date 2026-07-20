package ending

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/afm"
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

func TestRecoveredPrefixStopsAtNativeOnlyGate(t *testing.T) {
	frame, transparent := 0, -1
	c := NewIndexedCompositor()
	timeline := Timeline{Segments: []Segment{
		{Op: "blit_frame", Source: "test", Frame: &frame, Target: "offscreen", Stride: Width, Transparent: &transparent},
		{Op: "copy_buffer", Source: "test", Bytes: Bytes, From: "offscreen", To: "vga"},
		{Op: "native_call_opaque", Source: "gate"},
	}}
	frames := []fdother.Frame{{X: 0, Y: 0, Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 7}}}
	stopped, err := c.RunRecoveredPrefix(timeline, frames)
	if err == nil || stopped != 2 || c.VGA[0] != 7 {
		t.Fatalf("stopped=%d err=%v pixel=%d", stopped, err, c.VGA[0])
	}
}

func TestPresentANIReplacesIndexedVGAAndPalette(t *testing.T) {
	c := NewIndexedCompositor()
	frame, palette := make([]byte, Bytes), make([]byte, 768)
	frame[7], palette[9] = 42, 63
	if err := c.PresentANI(frame, palette); err != nil {
		t.Fatal(err)
	}
	if c.VGA[7] != 42 || c.Palette[9] != 63 {
		t.Fatalf("ANI state not presented")
	}
}

func TestPresentANIFrameKeepsFramePalettePair(t *testing.T) {
	c := NewIndexedCompositor()
	clip := &afm.Clip{IndexedFrames: [][]byte{make([]byte, Bytes)}, Palettes: [][]byte{make([]byte, 768)}}
	clip.IndexedFrames[0][3], clip.Palettes[0][4] = 11, 12
	if err := c.PresentANIFrame(clip, 0); err != nil || c.VGA[3] != 11 || c.Palette[4] != 12 {
		t.Fatalf("err=%v", err)
	}
}
