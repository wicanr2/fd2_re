package dato

import "testing"

func TestConfirmationMouthPostDecrementAndCallerSpecificRestart(t *testing.T) {
	for sample := 0; sample < 30; sample++ {
		s, err := NewConfirmationMouthState(sample)
		if err != nil || s.Open || s.Countdown != sample+2 || s.FrameIndex() != 0 {
			t.Fatal("initial", sample, s, err)
		}
		for old := sample + 2; old >= 0; old-- {
			s, err = s.Tick(0)
			if err != nil || s.Countdown != old-1 || s.Open != (old == 0) {
				t.Fatal("post-decrement", sample, old, s, err)
			}
		}
		if s.FrameIndex() != 3 {
			t.Fatal("open frame", s)
		}
		s, err = s.Tick(sample)
		if err != nil || s.Open || s.Countdown != sample+10 || s.FrameIndex() != 0 {
			t.Fatal("confirmation restart must be 10..39", sample, s, err)
		}
	}
}

func TestConfirmationMouthRejectsInvalidSamplesAndStates(t *testing.T) {
	for _, sample := range []int{-1, 30, 1000} {
		if _, err := NewConfirmationMouthState(sample); err == nil {
			t.Fatal("invalid initial sample accepted", sample)
		}
		if _, err := (ConfirmationMouthState{}).Tick(sample); err == nil {
			t.Fatal("invalid tick sample accepted", sample)
		}
	}
	for _, s := range []ConfirmationMouthState{
		{Countdown: -1}, {Countdown: 40}, {Open: true}, {Open: true, Countdown: -2},
	} {
		if _, err := s.Tick(0); err == nil {
			t.Fatal("invalid state accepted", s)
		}
	}
}
