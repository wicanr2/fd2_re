package battle

import "fmt"

// NativeCommand25Result is one final-effect target in the recovered ID25
// route. Cleared reports the direct unit+5 bit0x80 mutation performed by
// 0x22C04; it does not assign a gameplay name to the command.
type NativeCommand25Result struct {
	Target  *Unit
	Cleared bool
}

// ExecuteNativeCommand25 mirrors the verified state portion of
// 0x22C04: generic 0x1CFF0 two-stage targets, record-25 MP debit, then clear
// unit+5 bit0x80 only when it was already set. The wrapper's successful
// command completion still consumes the actor's action after the effect.
// Renderer, message feedback and UI remain outside this fail-closed slice.
func (s *State) ExecuteNativeCommand25(actor, confirmed *Unit) ([]NativeCommand25Result, error) {
	const commandID = 25
	targets, err := s.NativeCommand25Targets(actor, confirmed)
	if err != nil {
		return nil, err
	}
	record := s.NativeCommandBook[commandID]
	if !SpendNativeCommandMP(actor, record.MPCost) {
		return nil, fmt.Errorf("native command 25 insufficient MP")
	}
	results := make([]NativeCommand25Result, 0, len(targets))
	for _, target := range targets {
		cleared := target.NativeRecordByte5&0x80 != 0
		if cleared {
			target.NativeRecordByte5 &^= 0x80
		}
		results = append(results, NativeCommand25Result{Target: target, Cleared: cleared})
	}
	// 0x18D8C sets the invoking actor's completion bit only after the handler
	// returns success. It is independent from the target bit that ID25 clears.
	actor.Acted = true
	return results, nil
}

// NativeCommand25Targets performs the complete non-mutating preflight before
// the player-only 0x1D6C8 presentation. Target raw +5 provenance is mandatory:
// its bit7 must never be projected from or written through Unit.Acted.
func (s *State) NativeCommand25Targets(actor, confirmed *Unit) ([]*Unit, error) {
	const commandID = 25
	if s == nil || actor == nil || len(s.NativeCommandBook) != 36 ||
		s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native command 25 record unavailable")
	}
	record := s.NativeCommandBook[commandID]
	if record.MPCost < 0 || actor.MP < record.MPCost {
		return nil, fmt.Errorf("native command 25 insufficient MP")
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	targets, err := NativeCommandEffectTargets(
		s.W, s.H, actor, confirmed, record.SelectionMode, record.EffectMode,
		record.TargetCode, flags, s.Units,
	)
	if err != nil {
		return nil, err
	}
	for _, target := range targets {
		if target == nil || !target.HasNativeRecordByte5 {
			return nil, fmt.Errorf("native command 25 target lacks raw +5 provenance")
		}
	}
	return targets, nil
}
