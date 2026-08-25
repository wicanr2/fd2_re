package dato

import "testing"

func TestMouthStateOpenClosesWithNativeRange(t *testing.T) {
	for r := 0; r < 30; r++ {
		next, err := (MouthState{Open: true}).Tick(r)
		if err != nil || next.Open || next.Countdown != r+2 {
			t.Fatalf("r=%d: got %#v err=%v", r, next, err)
		}
	}
}

func TestMouthStateClosedOpensAtZero(t *testing.T) {
	next, err := (MouthState{Countdown: 1}).Tick(0)
	if err != nil || next.Open || next.Countdown != 0 || next.FrameIndex() != 0 {
		t.Fatalf("got %#v err=%v", next, err)
	}
	next, err = (MouthState{}).Tick(0)
	if err != nil || !next.Open {
		t.Fatalf("zero countdown should open: %#v err=%v", next, err)
	}
}

func TestMouthStateClosedCountsDown(t *testing.T) {
	next, err := (MouthState{Countdown: 4}).Tick(29)
	if err != nil || next.Open || next.Countdown != 3 || next.FrameIndex() != 0 {
		t.Fatalf("got %#v err=%v", next, err)
	}
}

func TestMouthStateRejectsInvalidRandom(t *testing.T) {
	for _, r := range []int{-1, 30} {
		if _, err := (MouthState{Open: true}).Tick(r); err == nil {
			t.Fatalf("random=%d accepted", r)
		}
	}
	if _, err := (MouthState{Countdown: -1}).Tick(0); err == nil {
		t.Fatal("negative countdown accepted")
	}
}
