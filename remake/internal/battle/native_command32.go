package battle

import "fmt"

// NativeCompoundCommand32Target preserves the per-target 0x1C75E result
// without claiming the still-unrestored indexed success/failure presentation.
type NativeCompoundCommand32Target struct {
	Target   *Unit
	Damage   NativeCommandDamage
	HPBefore int
	HPAfter  int
}

// NativeCompoundCommand32Result is the bounded class-19 player state result.
type NativeCompoundCommand32Result struct {
	Targets  []NativeCompoundCommand32Target
	RNGState uint16
}

// ExecuteNativeCompoundCommand32 stages the proven 0x2111A -> 0x1C75E target
// loop privately. The five reachable FIGANI resources bypass the known MP
// debit sink, so record32 is an availability gate only on this player path.
func (s *State) ExecuteNativeCompoundCommand32(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand32Result, error) {
	const commandID = 32
	if s == nil || actor == nil || actor.Acted || !actor.HasNativeRecordClass ||
		actor.NativeRecordClass != 19 || !actor.HasBattleFig || !nativeCompoundPlayerSelector(actor.BattleFig) ||
		len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return NativeCompoundCommand32Result{}, fmt.Errorf("native compound command 32 player provenance unavailable")
	}
	record := s.NativeCommandBook[commandID]
	if record.MPCost < 0 || actor.MP < record.MPCost || record.Damage < 0 || record.Hit < 0 || record.Hit > 100 {
		return NativeCompoundCommand32Result{}, fmt.Errorf("native compound command 32 record gate failed")
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return NativeCompoundCommand32Result{}, err
	}
	targets, err := NativeCommandEffectTargets(
		s.W, s.H, actor, confirmed, record.SelectionMode, record.EffectMode,
		record.TargetCode, flags, s.Units,
	)
	if err != nil {
		return NativeCompoundCommand32Result{}, err
	}
	if _, _, err := nativeCompoundCommandTargetRecords(targets); err != nil {
		return NativeCompoundCommand32Result{}, err
	}
	for index, target := range targets {
		resistance, ok := s.NativeCommandResistances[target.ClassID]
		if !ok || resistance < 0 || resistance > 10 {
			return NativeCompoundCommand32Result{}, fmt.Errorf("native compound command 32 target %d resistance unavailable", index)
		}
	}

	result := NativeCompoundCommand32Result{Targets: make([]NativeCompoundCommand32Target, 0, len(targets))}
	state := rngState
	for _, target := range targets {
		damage, nextState, resolveErr := ResolveNativeCommandDamage(
			record.Damage, record.Hit, s.NativeCommandResistances[target.ClassID], state,
		)
		if resolveErr != nil {
			return NativeCompoundCommand32Result{}, resolveErr
		}
		hpAfter := target.HP
		if damage.Hit {
			hpAfter -= damage.Damage
			if hpAfter < 0 {
				hpAfter = 0
			}
		}
		result.Targets = append(result.Targets, NativeCompoundCommand32Target{
			Target: target, Damage: damage, HPBefore: target.HP, HPAfter: hpAfter,
		})
		state = nextState
	}
	for _, targetResult := range result.Targets {
		targetResult.Target.HP = targetResult.HPAfter
	}
	actor.Acted = true
	result.RNGState = state
	return result, nil
}
