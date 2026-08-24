package figani

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestBuildNativeCompoundPresentationSchedulePreservesRawTables(t *testing.T) {
	effect := &Animation{Frames: make([]Frame, 8)}
	wantColors := [4][3]byte{{63, 63, 63}, {51, 57, 63}, {53, 0, 0}, {53, 58, 9}}
	for id := 32; id <= 35; id++ {
		s, err := BuildNativeCompoundPresentationSchedule(id, effect)
		if err != nil {
			t.Fatal(err)
		}
		if s.EffectResource != id+33 || s.SoundResource != id+59 || s.ActorSlideFrames != 8 ||
			s.ActorSlideStepX != 20 || s.EffectSlideFrames != 9 || s.EffectSlideStepX != 30 ||
			s.PreludeHoldTicks != 6 || s.MainDelayTicks != 2 || s.MainFrameCount != 7 ||
			s.PaletteRGB != wantColors[id-32] {
			t.Fatalf("command%d schedule=%#v", id, s)
		}
	}
	s32, _ := BuildNativeCompoundPresentationSchedule(32, effect)
	s33, _ := BuildNativeCompoundPresentationSchedule(33, effect)
	s34, _ := BuildNativeCompoundPresentationSchedule(34, effect)
	s35, _ := BuildNativeCompoundPresentationSchedule(35, effect)
	if !s32.TailEnabled || s32.MixerSample2Frame != 1 || !s33.OverlayFirstFrame || s33.Sample1Frame != 6 ||
		!s34.OverlayFirstFrame || s34.Sample1Frame != 2 || !s35.TailEnabled || s35.MixerSample2Frame != 1 {
		t.Fatalf("command-specific markers: 32=%#v 33=%#v 34=%#v 35=%#v", s32, s33, s34, s35)
	}
}

func TestNativeCompoundPresentationOriginalResources(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(path); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	for id := 32; id <= 35; id++ {
		effect, err := DecodeResource(path, id+33)
		if err != nil {
			t.Fatalf("command%d effect: %v", id, err)
		}
		if _, err := BuildNativeCompoundPresentationSchedule(id, effect); err != nil {
			t.Fatalf("command%d schedule: %v", id, err)
		}
		samples := []int{1}
		if id == 32 || id == 35 {
			samples = append(samples, 2)
		}
		for _, sample := range samples {
			raw, err := fdother.ReadNestedResource(path, id+59, sample)
			if err != nil || len(raw) == 0 {
				t.Fatalf("command%d sound%d len=%d err=%v", id, sample, len(raw), err)
			}
		}
	}
}
