package fdother

import "testing"

func TestBuildNativeAIItemDamageTailSchedulePreservesRawContract(t *testing.T) {
	for _, commandID := range []int{0, 2, 3} {
		s, err := BuildNativeAIItemDamageTailSchedule(commandID)
		if err != nil {
			t.Fatalf("command %d: %v", commandID, err)
		}
		if s.EffectResource != 6 || s.EffectStart != 0x31 || s.EffectFrames != 8 || s.EffectSample != 6 ||
			s.BlendFrames != 10 || s.RawBase != 0x20 || s.Blend != ([10]int{7, 6, 5, 4, 3, 2, 1, 0, 7, 6}) ||
			s.DigitResource != 5 || s.DigitBias != 0x5e || s.DigitFrames != 22 || s.DigitHoldMs != 500 {
			t.Fatalf("command %d schedule=%+v", commandID, s)
		}
	}
}

func TestBuildNativeAIItemDamageTailScheduleRejectsUnprovenCommands(t *testing.T) {
	for _, commandID := range []int{-1, 1, 4, 32} {
		if _, err := BuildNativeAIItemDamageTailSchedule(commandID); err == nil {
			t.Fatalf("command %d accepted", commandID)
		}
	}
}
