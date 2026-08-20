package fdother

import "testing"

func TestBuildNativeCommandHealPresentationSchedulePreserves21EB1Loops(t *testing.T) {
	tests := []struct {
		id, first, last int
	}{
		{13, 1, 17}, {14, 2, 34}, {15, 8, 40}, {16, 6, 54},
	}
	for _, tt := range tests {
		s, err := BuildNativeCommandHealPresentationSchedule(tt.id)
		if err != nil {
			t.Fatalf("id %d: %v", tt.id, err)
		}
		if len(s.Frames) != 16 || s.MidFrame != 9 || s.MidDelayMs != 200 ||
			s.TailDelayMs != 200 || s.SampleIndex != 11 {
			t.Fatalf("id %d schedule=%#v", tt.id, s)
		}
		if got := s.Frames[0]; got.LUTIndex != 9 || got.Radius != tt.first || got.FrameDelayMs != 5 {
			t.Fatalf("id %d first=%#v", tt.id, got)
		}
		if got := s.Frames[8]; got.LUTIndex != 1 || got.Radius != tt.last {
			t.Fatalf("id %d expansion tail=%#v", tt.id, got)
		}
		if got := s.Frames[9]; got.LUTIndex != 3 || got.Radius != tt.last {
			t.Fatalf("id %d release start=%#v", tt.id, got)
		}
		if got := s.Frames[15]; got.LUTIndex != 9 || got.Radius != tt.last {
			t.Fatalf("id %d release tail=%#v", tt.id, got)
		}
	}
}

func TestBuildNativeCommandHealPresentationScheduleRejectsOtherCommands(t *testing.T) {
	for _, id := range []int{-1, 12, 17, 255} {
		if _, err := BuildNativeCommandHealPresentationSchedule(id); err == nil {
			t.Fatalf("id %d unexpectedly accepted", id)
		}
	}
}
