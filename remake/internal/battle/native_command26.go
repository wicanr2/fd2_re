package battle

import (
	"fmt"
	"math/rand"
)

// NativeCommandApplicationResult records one final target in the recovered
// command-26/27 application family. Applied means every native gate passed;
// the raw offset is reported instead of assigning a status name.
type NativeCommandApplicationResult struct {
	Target   *Unit
	Offset   int
	Applied  bool
	Damage   int
	Duration byte
}

// ExecuteNativeCommandApplication executes only the byte-identical common
// route used by IDs 22, 26 and 27: 0x22BE1/0x22CBF/0x22E41 -> 0x22CDA ->
// 0x22D1B.
// It first resolves the generic target contract and spends that command's MP.
// Each final target then needs an empty raw interval, class ID other than 25
// or 26, and rand()%100 < 50. Success calls the native damage formula with
// base amount 10, consuming a second RNG draw; integer arithmetic makes the
// rolled damage 9. A third draw writes rand()%4+2 to +0x27 (ID22), +0x25
// (ID26) or +0x26 (ID27). It intentionally does not map any offset onto the
// legacy named status approximation.
func (s *State) ExecuteNativeCommandApplication(actor, confirmed *Unit, commandID int, rng *rand.Rand) ([]NativeCommandApplicationResult, error) {
	if rng == nil {
		return nil, fmt.Errorf("missing native command application state/rng")
	}
	targets, offset, err := s.NativeCommandApplicationTargets(actor, confirmed, commandID)
	if err != nil {
		return nil, err
	}
	record := s.NativeCommandBook[commandID]
	if !SpendNativeCommandMP(actor, record.MPCost) {
		return nil, fmt.Errorf("native command application insufficient MP")
	}
	results := make([]NativeCommandApplicationResult, 0, len(targets))
	for _, target := range targets {
		result := NativeCommandApplicationResult{Target: target, Offset: offset}
		duration, _ := target.NativeTransientDuration(offset)
		if duration == 0 && target.ClassID != 0x19 && target.ClassID != 0x1A && rng.Intn(100) < 50 {
			damage := 10*9/10 + rng.Intn(100)*10/1000
			target.ApplyHPDamage(damage)
			result.Applied = true
			result.Damage = damage
			result.Duration = byte(rng.Intn(4) + 2)
			target.SetNativeTransientDuration(offset, result.Duration)
		}
		results = append(results, result)
	}
	actor.Acted = true
	return results, nil
}

// NativeCommandApplicationTargets performs the complete non-mutating target
// and MP preflight used before the player-only 0x1D6C8 presentation.
func (s *State) NativeCommandApplicationTargets(actor, confirmed *Unit, commandID int) ([]*Unit, int, error) {
	if s == nil || actor == nil {
		return nil, 0, fmt.Errorf("missing native command application state/actor")
	}
	offset := 0
	switch commandID {
	case 22:
		offset = 0x27
	case 26:
		offset = 0x25
	case 27:
		offset = 0x26
	default:
		return nil, 0, fmt.Errorf("native command application unavailable id=%d", commandID)
	}
	if len(s.NativeCommandBook) != 36 || s.NativeCommandBook[commandID].ID != commandID {
		return nil, 0, fmt.Errorf("native command application record unavailable id=%d", commandID)
	}
	record := s.NativeCommandBook[commandID]
	if record.MPCost < 0 || actor.MP < record.MPCost {
		return nil, 0, fmt.Errorf("native command application insufficient MP")
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
		if target == nil || target.HP < 0 {
			return nil, 0, fmt.Errorf("invalid native command application target state")
		}
	}
	return targets, offset, nil
}
