package dato

import "fmt"

// ConfirmationMouthState belongs to sub_19953, independently of sub_16C57.
// Evidence/spec: docs/data/ida/fd2_confirmation_mouth_runtime_20261005.json.
type ConfirmationMouthState struct {
	Open      bool
	Countdown int
}

func NewConfirmationMouthState(randomMod30 int) (ConfirmationMouthState, error) {
	if randomMod30 < 0 || randomMod30 >= 30 {
		return ConfirmationMouthState{}, fmt.Errorf("dato: confirmation random value %d outside [0,30)", randomMod30)
	}
	return ConfirmationMouthState{Countdown: randomMod30 + 2}, nil
}

func (s ConfirmationMouthState) FrameIndex() int {
	if s.Open {
		return 3
	}
	return 0
}

// Tick consumes a fresh randomMod30 only when closing an open mouth. Native
// post-decrement leaves EBP=-1 during the open tick; retain that exact state.
func (s ConfirmationMouthState) Tick(randomMod30 int) (ConfirmationMouthState, error) {
	if randomMod30 < 0 || randomMod30 >= 30 ||
		(s.Open && s.Countdown != -1) || (!s.Open && (s.Countdown < 0 || s.Countdown > 39)) {
		return ConfirmationMouthState{}, fmt.Errorf("dato: invalid confirmation mouth state/sample: %+v, %d", s, randomMod30)
	}
	if s.Open {
		return ConfirmationMouthState{Countdown: randomMod30 + 10}, nil
	}
	s.Countdown--
	if s.Countdown == -1 {
		s.Open = true
	}
	return s, nil
}
