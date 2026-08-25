package fdother

import "testing"

func validType21Entries(count int) []LMI1Entry {
	out := make([]LMI1Entry, count)
	for index := range out {
		out[index] = LMI1Entry{Width: 1, Height: 1, Pixels: []byte{byte(index)}}
	}
	return out
}

func TestBuildNativeAIItemType21TailSchedulePreservesCallerTables(t *testing.T) {
	effect, digits := validType21Entries(0x4c), validType21Entries(0x68)
	for _, tc := range []struct{ id, start, frames, sample int }{{1, 0x31, 8, 6}, {6, 0x40, 10, 9}, {7, 0x40, 10, 9}} {
		s, err := BuildNativeAIItemType21TailSchedule(tc.id, effect, digits)
		if err != nil {
			t.Fatalf("command %d: %v", tc.id, err)
		}
		if s.EffectStart != tc.start || s.EffectFrames != tc.frames || s.EffectSample != tc.sample ||
			s.ToggleA != 0x4a || s.ToggleB != 0x4b || s.TogglePairs != 4 || s.ToggleDelayMS != 90 ||
			s.DigitBias != 0x5e || s.DigitFrames != 22 || s.DigitHoldMS != 500 {
			t.Fatalf("command %d schedule=%+v", tc.id, s)
		}
	}
}

func TestBuildNativeAIItemType21TailScheduleFailsClosed(t *testing.T) {
	effect, digits := validType21Entries(0x4c), validType21Entries(0x68)
	for _, commandID := range []int{0, 2, 3, 32} {
		if _, err := BuildNativeAIItemType21TailSchedule(commandID, effect, digits); err == nil {
			t.Fatalf("command %d accepted", commandID)
		}
	}
	effect[0x4a] = LMI1Entry{}
	if _, err := BuildNativeAIItemType21TailSchedule(1, effect, digits); err == nil {
		t.Fatal("invalid toggle accepted")
	}
}
