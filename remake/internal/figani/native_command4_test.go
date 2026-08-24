package figani

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func command4TestAnimation() *Animation {
	frames := make([]Frame, 14)
	for i := range frames {
		frames[i] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(i)}, Mask: []byte{1}}
	}
	return &Animation{HeaderByte2: 14, Frames: frames}
}

func TestNativeCommand4ScheduleAndMarkers(t *testing.T) {
	s, err := BuildNativeCommand4PresentationSchedule(1, command4TestAnimation())
	if err != nil || s.CommandID != 4 || s.EffectResource != 22 || s.SoundResource != 85 ||
		s.FrontFrames != 2 || s.TargetFrames != 12 || s.TailFrames != 8 || s.MarkerCounter != 3 {
		t.Fatalf("command4 schedule=%+v err=%v", s, err)
	}
	state := NewNativeCommand5StateForSchedule(0x1234, s)
	for i := 0; i < s.FrontFrames; i++ {
		f, e := PlanNativeCommand5DrawFrame(state, s, 1)
		if e != nil {
			t.Fatal(e)
		}
		state = f.Next
	}
	frames, _, err := BuildNativeCommand5TargetSequence(state, s, 1)
	if err != nil {
		t.Fatal(err)
	}
	stages := 0
	for _, frame := range frames {
		if frame.HPStage != 0 {
			stages++
		}
	}
	if len(frames) != 12 || stages != 6 {
		t.Fatalf("frames=%d stages=%d", len(frames), stages)
	}
}

func TestOriginalFDOTHERCommand4ResourcesMatchRecoveredSignatures(t *testing.T) {
	path := filepath.Join("..", "..", "..", "org_game", "炎龍騎士團", "FLAME2", "FDOTHER.DAT")
	for _, tc := range []struct {
		side     byte
		resource int
	}{{1, 22}, {0, 23}} {
		effect, err := DecodeResource(path, tc.resource)
		if err != nil {
			t.Fatal(err)
		}
		s, err := BuildNativeCommand4PresentationSchedule(tc.side, effect)
		if err != nil || s.EffectResource != tc.resource {
			t.Fatalf("resource %d schedule=%+v err=%v", tc.resource, s, err)
		}
	}
	for sample := 0; sample <= 1; sample++ {
		raw, err := fdother.ReadNestedResource(path, 85, sample)
		if err != nil || len(raw) == 0 {
			t.Fatalf("#85 sub%d len=%d err=%v", sample, len(raw), err)
		}
	}
}
