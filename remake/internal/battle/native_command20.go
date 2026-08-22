package battle

import (
	"fmt"
	"math/rand"
)

// NativeCommandRestore is the recovered 0x1C916 HP mutation. Display code in
// the original receives the uncapped rolled amount; this data result instead
// exposes the actual HP delta so an engine caller cannot mistake an over-cap
// display number for a state mutation.
type NativeCommandRestore struct {
	Rolled int
	Actual int
}

// ApplyNativeCommandRestore mirrors 0x1C916 for a supplied raw amount:
// amount*9/10 + (rand()%100)*amount/1000, add current HP and clamp max HP.
func ApplyNativeCommandRestore(target *Unit, amount int, rng *rand.Rand) (NativeCommandRestore, error) {
	if target == nil || rng == nil || amount < 0 || target.MaxHP < 0 || target.HP < 0 || target.HP > target.MaxHP {
		return NativeCommandRestore{}, fmt.Errorf("invalid native command restore state")
	}
	rolled := amount*9/10 + rng.Intn(100)*amount/1000
	before := target.HP
	target.HP += rolled
	if target.HP > target.MaxHP {
		target.HP = target.MaxHP
	}
	return NativeCommandRestore{Rolled: rolled, Actual: target.HP - before}, nil
}

// NativeCommandClearRestoreResult records one final target in the common
// ID20/21 route. Cleared identifies the raw +0x25/+0x26 flag; restore is
// present only if that flag was nonzero.
type NativeCommandClearRestoreResult struct {
	Target  *Unit
	Offset  int
	Cleared bool
	Restore NativeCommandRestore
}

// ExecuteNativeCommandClearRestore mirrors 0x22A85/0x22BC6 -> 0x22AA8 ->
// 0x22AF6 for IDs20/21. The command's own record supplies target/MP data;
// native 0x1C916 is deliberately invoked with record10's damage field, then
// the nonzero target raw interval is cleared. No named status is inferred.
func (s *State) ExecuteNativeCommandClearRestore(actor, confirmed *Unit, commandID int, rng *rand.Rand) ([]NativeCommandClearRestoreResult, error) {
	if rng == nil {
		return nil, fmt.Errorf("missing native command clear/restore state/rng")
	}
	targets, offset, err := s.NativeCommandClearRestoreTargets(actor, confirmed, commandID)
	if err != nil {
		return nil, err
	}
	record := s.NativeCommandBook[commandID]
	if !SpendNativeCommandMP(actor, record.MPCost) {
		return nil, fmt.Errorf("native command clear/restore insufficient MP")
	}
	results := make([]NativeCommandClearRestoreResult, 0, len(targets))
	for _, target := range targets {
		result := NativeCommandClearRestoreResult{Target: target, Offset: offset}
		duration, _ := target.NativeTransientDuration(offset)
		if duration != 0 {
			restore, _ := ApplyNativeCommandRestore(target, s.NativeCommandBook[10].Damage, rng)
			target.SetNativeTransientDuration(offset, 0)
			result.Cleared, result.Restore = true, restore
		}
		results = append(results, result)
	}
	actor.Acted = true
	return results, nil
}

// NativeCommandClearRestoreTargets performs the complete non-mutating
// transaction preflight needed before the player-only 0x1D6C8 presentation.
func (s *State) NativeCommandClearRestoreTargets(actor, confirmed *Unit, commandID int) ([]*Unit, int, error) {
	if s == nil || actor == nil {
		return nil, 0, fmt.Errorf("missing native command clear/restore state/actor")
	}
	offset := 0
	switch commandID {
	case 20:
		offset = 0x25
	case 21:
		offset = 0x26
	default:
		return nil, 0, fmt.Errorf("native command clear/restore unavailable id=%d", commandID)
	}
	if len(s.NativeCommandBook) != 36 || s.NativeCommandBook[commandID].ID != commandID || s.NativeCommandBook[10].ID != 10 {
		return nil, 0, fmt.Errorf("native command clear/restore records unavailable id=%d", commandID)
	}
	record := s.NativeCommandBook[commandID]
	if record.MPCost < 0 || actor.MP < record.MPCost || s.NativeCommandBook[10].Damage < 0 {
		return nil, 0, fmt.Errorf("native command clear/restore cost/amount unavailable")
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, 0, err
	}
	targets, err := NativeCommandEffectTargets(s.W, s.H, actor, confirmed, record.SelectionMode, record.EffectMode, record.TargetCode, flags, s.Units)
	if err != nil {
		return nil, 0, err
	}
	for _, target := range targets {
		if target == nil || target.HP < 0 || target.MaxHP < 0 || target.HP > target.MaxHP {
			return nil, 0, fmt.Errorf("invalid native command clear/restore target state")
		}
	}
	return targets, offset, nil
}
