package figani

import (
	"os"
	"slices"
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
	if a.HeaderByte1 != 0 || a.HeaderByte2 != 0 || a.HeaderByte4 != 0x7e {
		t.Fatalf("header byte1/byte2/byte4=%#x/%#x/%#x, want 0/0/0x7e", a.HeaderByte1, a.HeaderByte2, a.HeaderByte4)
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

func TestParseTreatsPreludeFlagAsByteNotFrameCountHighByte(t *testing.T) {
	// Same valid one-frame resource, with the independently consumed native
	// byte1 prelude flag and byte2 prelude count both set to one.
	raw := []byte{1, 1, 1, 0, 0x7e, 0, 12, 0, 12, 0, 0, 0, 2, 0, 3, 0, 4, 5, 2, 6, 0, 4, 0, 1, 0, 0x00, 7, 0x40, 9, 0xc0}
	a, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Frames) != 1 || a.HeaderByte1 != 1 || a.HeaderByte2 != 1 {
		t.Fatalf("animation=%#v, want one frame and header bytes 1/1", a)
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

func TestDecodeOriginalFIGANIPreludeResource(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FIGANI.DAT"
	a, err := DecodeResource(path, 25)
	if os.IsNotExist(err) {
		t.Skip("player-provided FIGANI.DAT is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Frames) != 19 || a.HeaderByte1 != 1 || a.HeaderByte2 != 14 {
		t.Fatalf("prelude resource frames=%d header=%d/%d, want 19 and 1/14", len(a.Frames), a.HeaderByte1, a.HeaderByte2)
	}
}

func TestNativeCommand24ScheduleMatchesSelector32Resource98(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FIGANI.DAT"
	animation, err := DecodeResource(path, 98)
	if os.IsNotExist(err) {
		t.Skip("player-provided FIGANI.DAT is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := BuildNativeCommand24Schedule(animation)
	if err != nil {
		t.Fatal(err)
	}
	if schedule.TargetStart != 9 || schedule.ActorImpactFrame != 4 ||
		schedule.TargetImpactFrame != 10 || schedule.AudioResource != 53 ||
		schedule.ActorSample != 3 || schedule.TargetSample != 2 ||
		schedule.ShakeOffsets != [6]int{0, 4, 9, 14, 18, 14} {
		t.Fatalf("command24 schedule=%#v", schedule)
	}
	bad := *animation
	bad.Frames = append([]Frame(nil), animation.Frames...)
	bad.Frames[10].RawByte4 = 0
	if _, err := BuildNativeCommand24Schedule(&bad); err == nil {
		t.Fatal("command24 accepted missing target impact marker")
	}
}

func TestNativeCommand29ScheduleMatchesGrowthSelector34Resource104(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FIGANI.DAT"
	animation, err := DecodeResource(path, 104)
	if os.IsNotExist(err) {
		t.Skip("player-provided FIGANI.DAT is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := BuildNativeCommandDerivedStrikeSchedule(animation, 29)
	if err != nil {
		t.Fatal(err)
	}
	if schedule.CommandID != 29 || schedule.TargetStart != 11 ||
		schedule.ActorImpactFrame != 6 || schedule.TargetImpactFrame != 14 ||
		schedule.DamageDenominator != 1 || schedule.AudioResource != 50 ||
		schedule.ActorSample != 1 || schedule.TargetSample != 4 ||
		schedule.UsesTargetBase || schedule.PreludeMode != 1 || !schedule.UsesBGTransition ||
		len(schedule.FrameDelays) != 18 {
		t.Fatalf("command29 schedule=%#v", schedule)
	}
}

func TestNativeCommand28ScheduleKeepsCallerSpecificPresentationBranches(t *testing.T) {
	animation := &Animation{HeaderByte1: 1, HeaderByte2: 2, HeaderByte4: 2, Frames: []Frame{
		{Delay: 1}, {Delay: 2, RawByte4: 1, RawByte5: 3},
		{Delay: 4}, {Delay: 2, RawByte4: 1, RawByte5: 4},
	}}
	schedule, err := BuildNativeCommandDerivedStrikeSchedule(animation, 28)
	if err != nil {
		t.Fatal(err)
	}
	if schedule.DamageDenominator != 8 || !schedule.UsesTargetBase ||
		schedule.PreludeMode != 0 || schedule.UsesBGTransition ||
		schedule.AudioResource != 49 || schedule.ActorSample != 3 || schedule.TargetSample != 4 {
		t.Fatalf("command28 schedule=%#v", schedule)
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

func TestDisplaySchedulerPresentsZeroDelayFrameOnce(t *testing.T) {
	s, err := NewDisplayScheduler([]int{1, 0, 2}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if s.BodyTicks() != 4 {
		t.Fatalf("body ticks=%d, want 4", s.BodyTicks())
	}
	for tick, want := range []int{0, 1, 2, 2} {
		got, presented, _, stepErr := s.Step()
		if stepErr != nil || !presented || got != want {
			t.Fatalf("tick %d frame=%d presented=%v err=%v, want %d", tick, got, presented, stepErr, want)
		}
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

func TestFrameBlitTranslatedClipsAndCanOverrideOpaquePixels(t *testing.T) {
	f := Frame{
		X: 0, Y: 0, Width: 3, Height: 1,
		Pixels: []byte{7, 8, 9}, Mask: []byte{1, 0, 1},
	}
	dst := []byte{1, 1, 1, 1}
	fill := byte(33)
	if err := f.BlitTranslated(dst, 4, -1, 0, &fill); err != nil {
		t.Fatal(err)
	}
	// Source x0 is clipped; transparent source x1 lands at x0 and preserves
	// its destination; opaque source x2 lands at x1 and receives the override.
	if got, want := dst, []byte{1, 33, 1, 1}; !slices.Equal(got, want) {
		t.Fatalf("translated blit=%v want=%v", got, want)
	}
}
