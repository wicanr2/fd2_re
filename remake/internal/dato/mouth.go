package dato

import "fmt"

// MouthState is the small state machine used by the native dialogue loop.
// The original presents frame 0 while closed and frame 3 for the brief open
// tick; the other DATO frames are resource variants, not timing states.
type MouthState struct {
	Open      bool
	Countdown int
}

// FrameIndex returns the native DATO frame selected for the current state.
func (s MouthState) FrameIndex() int {
	if s.Open {
		return 3
	}
	return 0
}

// Tick advances one native mouth tick. randomMod30 is the already sampled
// rand()%0x1e value used only when closing an open mouth. Keeping the sample
// outside this adapter makes tests deterministic without claiming parity with
// the executable's global RNG stream.
func (s MouthState) Tick(randomMod30 int) (MouthState, error) {
	if randomMod30 < 0 || randomMod30 >= 30 {
		return MouthState{}, fmt.Errorf("dato: random mouth value %d outside [0,30)", randomMod30)
	}
	if s.Countdown < 0 {
		return MouthState{}, fmt.Errorf("dato: mouth countdown %d is negative", s.Countdown)
	}
	if s.Open {
		return MouthState{Countdown: randomMod30 + 2}, nil
	}
	next := s
	// sub_16C57 採用 post-decrement：只有舊值為零才張嘴；正值則減一後存回。
	if next.Countdown == 0 {
		next.Open = true
		return next, nil
	}
	next.Countdown--
	return next, nil
}
