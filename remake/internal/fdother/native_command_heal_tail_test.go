package fdother

import "testing"

func TestBuildNativeCommandHealTailSchedulePreservesRawTables(t *testing.T) {
	for id := 13; id <= 16; id++ {
		s, err := BuildNativeCommandHealTailSchedule(id)
		if err != nil {
			t.Fatalf("id %d: %v", id, err)
		}
		if s.EffectResource != 6 || s.EffectStart != 0x39 || s.EffectFrames != 7 ||
			s.EffectSample != 12 || s.EffectFrameDelayTicks != 1 ||
			s.MaskSample != 1 || s.MaskIndex != 0xc0 || s.MaskPairs != 5 ||
			s.DigitResource != 5 || s.DigitBias != 0x69 || s.DigitFrames != 22 ||
			s.DigitFrameDelayMs != 2 || s.DigitHoldMs != 500 {
			t.Fatalf("id %d schedule=%+v", id, s)
		}
		want := [25]int{15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15}
		if s.DigitVertical != want {
			t.Fatalf("id %d vertical=%v", id, s.DigitVertical)
		}
	}
}

func TestBuildNativeCommandHealTailScheduleRejectsOtherCommands(t *testing.T) {
	for _, id := range []int{-1, 12, 17, 40} {
		if _, err := BuildNativeCommandHealTailSchedule(id); err == nil {
			t.Fatalf("id %d unexpectedly accepted", id)
		}
	}
}
