package battle

import (
	"fmt"
	"math/rand"
)

// NativeCommandHealResult is one final target in the shared recovered
// ID13..16 HP-restore route. The command family has separate indexed
// presentation wrappers, which are intentionally not represented here.
type NativeCommandHealResult struct {
	Target  *Unit
	Restore NativeCommandRestore
}

// NativeCommandHealTargets validates the recovered command-13..16 selector
// and MP boundary without mutating combat state. The presentation owner uses
// this preflight so a missing indexed adapter cannot debit MP or change HP.
func (s *State) NativeCommandHealTargets(actor, confirmed *Unit, commandID int) ([]*Unit, error) {
	if s == nil {
		return nil, fmt.Errorf("missing native command heal state")
	}
	if commandID < 13 || commandID > 16 || len(s.NativeCommandBook) != 36 || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native command heal record unavailable id=%d", commandID)
	}
	record := s.NativeCommandBook[commandID]
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	targets, err := NativeCommandEffectTargets(s.W, s.H, actor, confirmed, record.SelectionMode, record.EffectMode, record.TargetCode, flags, s.Units)
	if err != nil {
		return nil, err
	}
	if actor == nil || actor.MP < record.MPCost {
		return nil, fmt.Errorf("native command heal insufficient MP")
	}
	return targets, nil
}

// ExecuteNativeCommandHeal executes the state portion shared by native IDs
// 13..16 (0x21AD9/0x21B99/0x2211C/0x22153 -> 0x21B18): generic two-stage
// targets, that record's MP debit, and per-target 0x1C916 restore using the
// same record's u16 damage field. It remains fail-closed outside this exact
// bounded family and does not attach its renderer/SFX/UI.
func (s *State) ExecuteNativeCommandHeal(actor, confirmed *Unit, commandID int, rng *rand.Rand) ([]NativeCommandHealResult, error) {
	if s == nil || rng == nil {
		return nil, fmt.Errorf("missing native command heal state/rng")
	}
	if commandID < 13 || commandID > 16 || len(s.NativeCommandBook) != 36 || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native command heal record unavailable id=%d", commandID)
	}
	record := s.NativeCommandBook[commandID]
	targets, err := s.NativeCommandHealTargets(actor, confirmed, commandID)
	if err != nil {
		return nil, err
	}
	if !SpendNativeCommandMP(actor, record.MPCost) {
		return nil, fmt.Errorf("native command heal insufficient MP")
	}
	results := make([]NativeCommandHealResult, 0, len(targets))
	for _, target := range targets {
		restore, err := ApplyNativeCommandRestore(target, record.Damage, rng)
		if err != nil {
			return nil, err
		}
		results = append(results, NativeCommandHealResult{Target: target, Restore: restore})
	}
	actor.Acted = true
	return results, nil
}
