package battle

import "fmt"

// NativeCompoundCommand32Target preserves the per-target 0x1C75E transaction
// result consumed by the separate indexed presentation owner.
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

// NativeCompoundCommand32Plan separates the proven per-target 0x1C75E
// publication markers from final actor completion so the indexed owner can
// fail closed without rerolling RNG or partially committing HP.
type NativeCompoundCommand32Plan struct {
	Actor            *Unit
	Result           NativeCompoundCommand32Result
	ActorActedBefore bool
	RNGBefore        uint16
	targets          []NativeCompoundCommand32Target
	published        []bool
	completed        bool
}

// ExecuteNativeCompoundCommand32 stages the proven 0x2111A -> 0x1C75E target
// loop privately. The five reachable FIGANI resources bypass the known MP
// debit sink, so record32 is an availability gate only on this player path.
func (s *State) ExecuteNativeCompoundCommand32(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand32Result, error) {
	plan, err := s.PlanNativeCompoundCommand32(actor, confirmed, rngState)
	if err != nil {
		return NativeCompoundCommand32Result{}, err
	}
	for index := range plan.Result.Targets {
		if err := ApplyNativeCompoundCommand32Target(plan, index); err != nil {
			_ = AbortNativeCompoundCommand32(plan)
			return NativeCompoundCommand32Result{}, err
		}
	}
	if err := CompleteNativeCompoundCommand32(plan); err != nil {
		_ = AbortNativeCompoundCommand32(plan)
		return NativeCompoundCommand32Result{}, err
	}
	return plan.Result, nil
}

// PlanNativeCompoundCommand32 performs the complete target/RNG preflight
// without mutating actor, targets, or the caller-owned RNG state.
func (s *State) PlanNativeCompoundCommand32(actor, confirmed *Unit, rngState uint16) (*NativeCompoundCommand32Plan, error) {
	const commandID = 32
	if s == nil || actor == nil || actor.Acted || !actor.HasNativeRecordClass ||
		actor.NativeRecordClass != 19 || !actor.HasBattleFig || !nativeCompoundPlayerSelector(actor.BattleFig) ||
		len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native compound command 32 player provenance unavailable")
	}
	record := s.NativeCommandBook[commandID]
	if record.MPCost < 0 || actor.MP < record.MPCost || record.Damage < 0 || record.Hit < 0 || record.Hit > 100 {
		return nil, fmt.Errorf("native compound command 32 record gate failed")
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
	if _, _, err := nativeCompoundCommandTargetRecords(targets); err != nil {
		return nil, err
	}
	seenTargets := make(map[*Unit]struct{}, len(targets))
	for index, target := range targets {
		if _, duplicate := seenTargets[target]; duplicate {
			return nil, fmt.Errorf("native compound command 32 target %d duplicated", index)
		}
		seenTargets[target] = struct{}{}
		resistance, ok := s.NativeCommandResistances[target.ClassID]
		if !ok || resistance < 0 || resistance > 10 {
			return nil, fmt.Errorf("native compound command 32 target %d resistance unavailable", index)
		}
	}

	result := NativeCompoundCommand32Result{Targets: make([]NativeCompoundCommand32Target, 0, len(targets))}
	state := rngState
	for _, target := range targets {
		damage, nextState, resolveErr := ResolveNativeCommandDamage(
			record.Damage, record.Hit, s.NativeCommandResistances[target.ClassID], state,
		)
		if resolveErr != nil {
			return nil, resolveErr
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
	result.RNGState = state
	plannedTargets := append([]NativeCompoundCommand32Target(nil), result.Targets...)
	return &NativeCompoundCommand32Plan{
		Actor: actor, Result: result, ActorActedBefore: actor.Acted, RNGBefore: rngState,
		targets:   plannedTargets,
		published: make([]bool, len(result.Targets)),
	}, nil
}

func ApplyNativeCompoundCommand32Target(plan *NativeCompoundCommand32Plan, index int) error {
	if plan == nil || plan.Actor == nil || plan.completed || plan.Actor.Acted != plan.ActorActedBefore ||
		index < 0 || index >= len(plan.targets) || len(plan.published) != len(plan.targets) || plan.published[index] {
		return fmt.Errorf("native compound command 32 target publication unavailable")
	}
	target := plan.targets[index]
	if target.Target == nil || target.Target.HP != target.HPBefore {
		return fmt.Errorf("native compound command 32 target state changed")
	}
	target.Target.HP = target.HPAfter
	plan.published[index] = true
	return nil
}

func CompleteNativeCompoundCommand32(plan *NativeCompoundCommand32Plan) error {
	if plan == nil || plan.Actor == nil || plan.completed || plan.Actor.Acted != plan.ActorActedBefore ||
		len(plan.targets) == 0 || len(plan.published) != len(plan.targets) {
		return fmt.Errorf("native compound command 32 completion unavailable")
	}
	for index, published := range plan.published {
		if !published || plan.targets[index].Target == nil || plan.targets[index].Target.HP != plan.targets[index].HPAfter {
			return fmt.Errorf("native compound command 32 target %d incomplete", index)
		}
	}
	plan.Actor.Acted = true
	plan.completed = true
	return nil
}

// AbortNativeCompoundCommand32 restores every already-published target after
// preflighting the complete rollback. It never overwrites state changed by an
// outside owner and therefore remains fail closed at presentation boundaries.
func AbortNativeCompoundCommand32(plan *NativeCompoundCommand32Plan) error {
	if plan == nil || plan.Actor == nil || plan.completed || plan.Actor.Acted != plan.ActorActedBefore ||
		len(plan.published) != len(plan.targets) {
		return fmt.Errorf("native compound command 32 rollback unavailable")
	}
	for index, published := range plan.published {
		if published && (plan.targets[index].Target == nil || plan.targets[index].Target.HP != plan.targets[index].HPAfter) {
			return fmt.Errorf("native compound command 32 target %d rollback state changed", index)
		}
	}
	for index := len(plan.published) - 1; index >= 0; index-- {
		if plan.published[index] {
			plan.targets[index].Target.HP = plan.targets[index].HPBefore
			plan.published[index] = false
		}
	}
	return nil
}
