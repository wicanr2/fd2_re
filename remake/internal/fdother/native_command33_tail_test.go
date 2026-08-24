package fdother

import "testing"

func TestBuildNativeCommand33TailScheduleUsesHardCodedCommand13Row(t *testing.T) {
	s, err := BuildNativeCommand33TailSchedule()
	if err != nil {
		t.Fatal(err)
	}
	if s.EffectStart != 0x39 || s.EffectFrames != 7 || s.EffectSample != 12 ||
		s.MaskSample != 1 || s.MaskIndex != 0xc0 || s.MaskPairs != 5 ||
		s.DigitBias != 0x69 || s.DigitFrames != 22 || s.DigitHoldMs != 500 {
		t.Fatalf("command33 tail=%#v", s)
	}
}
