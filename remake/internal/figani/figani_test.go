package figani

import (
	"os"
	"testing"
)

func TestParsePreservesTransparentAndDitherPixels(t *testing.T) {
	// One 4x1 frame: run(7), dither(9), then transparent skip.
	raw := []byte{1, 0, 0, 0, 0x7e, 0, 12, 0, 12, 0, 0, 0, 2, 0, 3, 0, 4, 5, 2, 6, 0, 4, 0, 1, 0, 0x00, 7, 0x40, 9, 0xc0}
	a, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	f := a.Frames[0]
	if f.X != 2 || f.Y != 3 || f.Width != 4 || f.Height != 1 || f.Delay != 2 || f.RawByte4 != 4 || f.RawByte5 != 5 || f.RawByte7 != 6 {
		t.Fatalf("frame=%#v", f)
	}
	if a.HeaderByte1 != 0 || a.HeaderByte4 != 0x7e {
		t.Fatalf("header byte1/byte4=%#x/%#x, want 0/0x7e", a.HeaderByte1, a.HeaderByte4)
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

func TestDecodeOriginalPlayerClass19HeaderFlags(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FIGANI.DAT"
	// Player-reachable class-19 sources use visual groups 4..7 (optional
	// class change) or 20 (Sara's initial class). 0x2b659 receives group*3+1.
	for resource, want := range map[int]byte{13: 2, 16: 2, 19: 2, 22: 5, 61: 5} {
		a, err := DecodeResource(path, resource)
		if os.IsNotExist(err) {
			t.Skip("player-provided FIGANI.DAT is absent")
		}
		if err != nil {
			t.Fatalf("resource %d: %v", resource, err)
		}
		if a.HeaderByte4 != want {
			t.Errorf("resource %d HeaderByte4=%d, want %d", resource, a.HeaderByte4, want)
		}
	}
}

func TestNativeSchedulerInitializesWithoutRenderingAndAdvancesAfterRender(t *testing.T) {
	a := &Animation{Frames: []Frame{{Delay: 1}, {Delay: 2}, {Delay: 1}}}
	s := NativeScheduler{}
	if index, rendered, err := s.Step(a, false); err != nil || index != 0 || rendered || s.Subframe != 0 {
		t.Fatalf("init index/render/state=%d/%v/%#v err=%v", index, rendered, s, err)
	}
	if index, rendered, err := s.Step(a, true); err != nil || index != 0 || !rendered || s.FrameIndex != 1 || s.Subframe != 0 {
		t.Fatalf("first index/render/state=%d/%v/%#v err=%v", index, rendered, s, err)
	}
	if index, _, err := s.Step(a, true); err != nil || index != 1 || s.FrameIndex != 1 || s.Subframe != 1 {
		t.Fatalf("second state=%#v index=%d err=%v", s, index, err)
	}
	if index, _, err := s.Step(a, true); err != nil || index != 1 || s.FrameIndex != 2 || s.Subframe != 0 {
		t.Fatalf("third state=%#v index=%d err=%v", s, index, err)
	}
	if index, _, err := s.Step(a, true); err != nil || index != 2 || s.FrameIndex != 0 {
		t.Fatalf("wrap state=%#v index=%d err=%v", s, index, err)
	}
}

func TestDisplaySchedulerHonoursNativeDelaysAndScale(t *testing.T) {
	s, err := NewDisplayScheduler([]int{1, 2, 1}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if s.BodyTicks() != 8 {
		t.Fatalf("body ticks=%d, want 8", s.BodyTicks())
	}
	for frame, want := range []int{0, 0, 1, 1, 1, 1, 2, 2} {
		got, presented, done, stepErr := s.Step()
		if stepErr != nil || !presented || got != want {
			t.Fatalf("tick %d frame/present/done=%d/%v/%v err=%v, want frame %d", frame, got, presented, done, stepErr, want)
		}
		if frame < 7 && done {
			t.Fatalf("tick %d ended the body early", frame)
		}
		if frame == 7 && !done {
			t.Fatal("last body tick did not report done")
		}
	}
	if got, presented, done, err := s.Step(); err != nil || presented || !done || got != 2 {
		t.Fatalf("tail step frame/present/done=%d/%v/%v err=%v", got, presented, done, err)
	}
	for _, want := range []struct {
		frame int
		start int
	}{{0, 0}, {1, 2}, {2, 6}} {
		if got, ok := s.FrameStart(want.frame); !ok || got != want.start {
			t.Fatalf("frame %d start=%d/%v, want %d/true", want.frame, got, ok, want.start)
		}
	}
}

func TestDisplaySchedulerRejectsUnknownDelayState(t *testing.T) {
	for name, delays := range map[string][]int{
		"empty":    nil,
		"zero":     {1, 0},
		"negative": {1, -1},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewDisplayScheduler(delays, 1); err == nil {
				t.Fatal("invalid delay state was accepted")
			}
		})
	}
	if _, err := NewDisplayScheduler([]int{1}, 0); err == nil {
		t.Fatal("zero display scale was accepted")
	}
}

func TestFrameBlitAtBaseShiftsNativeWorkSurface(t *testing.T) {
	f := Frame{X: 2, Y: 3, Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}
	dst := make([]byte, 640*5)
	if err := f.BlitAtBase(dst, 640, 80); err != nil {
		t.Fatal(err)
	}
	if got := dst[80+3*640+2]; got != 9 {
		t.Fatalf("shifted pixel=%d", got)
	}
}
