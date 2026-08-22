package battle

import (
	"fmt"
)

// NativeCommandDamageResult is one final-effect candidate resolved by the
// verified numeric damage path. Animation and post-resolution messaging are
// intentionally outside this state mutation.
type NativeCommandDamageResult struct {
	Target *Unit
	NativeCommandDamage
	HPBefore int
	HPAfter  int
}

// NativeCommandDamagePlan is the mutation-free half of 0x2A6BD's numeric
// transaction.  Presentation owners use it to publish MP before the target
// loop and HP at sub_26152's seven raw impact markers without rerolling RNG.
type NativeCommandDamagePlan struct {
	Actor                  *Unit
	CommandID              int
	MPBefore, MPAfter      int
	ActorActedBefore       bool
	RNGBefore, RNGAfter    uint16
	Results                []NativeCommandDamageResult
	publishedTargetStages  []int
	mpPublished, completed bool
}

// ExecuteBoundNativeCommand0 uses the state-bound verified resistance table.
// A missing table remains fail-closed rather than falling back to the legacy
// normalized magic approximation.
func (s *State) ExecuteBoundNativeCommand0(actor, confirmed *Unit, rngState uint16) ([]NativeCommandDamageResult, uint16, error) {
	if s == nil || len(s.NativeCommandResistances) == 0 {
		return nil, rngState, fmt.Errorf("native command 0 resistances unavailable")
	}
	return s.ExecuteNativeCommandDamage(actor, confirmed, 0, s.NativeCommandResistances, rngState)
}

// PlanBoundNativeCommand0 validates and resolves command0 without mutation.
func (s *State) PlanBoundNativeCommand0(actor, confirmed *Unit, rngState uint16) (*NativeCommandDamagePlan, error) {
	if s == nil || len(s.NativeCommandResistances) == 0 {
		return nil, fmt.Errorf("native command 0 resistances unavailable")
	}
	return s.PlanNativeCommandDamage(actor, confirmed, 0, s.NativeCommandResistances, rngState)
}

// ExecuteNativeCommandDamage covers the byte-for-byte numeric route proven
// for player-dispatched command IDs 0..12. IDs0..8 dispatch directly to
// 0x2A6BD, which runs sub_2B659's MP event and its final-target loop directly
// calls 0x1C75E(targetSlot, commandID). ID9 invokes 0x1CA89 -> 0x1C75E;
// IDs10..12 run their distinct indexed compositor (0x21548) before the same
// state sequence. Other IDs stay fail-closed.
func (s *State) ExecuteNativeCommandDamage(actor, confirmed *Unit, commandID int, resistByClass map[int]int, rngState uint16) ([]NativeCommandDamageResult, uint16, error) {
	plan, err := s.PlanNativeCommandDamage(actor, confirmed, commandID, resistByClass, rngState)
	if err != nil {
		return nil, rngState, err
	}
	if err := ApplyNativeCommandDamageMP(plan); err != nil {
		return nil, rngState, err
	}
	for index := range plan.Results {
		if err := ApplyNativeCommandDamageStage(plan, index, 7); err != nil {
			return nil, rngState, err
		}
	}
	if err := CompleteNativeCommandDamage(plan); err != nil {
		return nil, rngState, err
	}
	return append([]NativeCommandDamageResult(nil), plan.Results...), plan.RNGAfter, nil
}

// PlanNativeCommandDamage consumes the verified RNG sequence but leaves all
// Unit fields unchanged.  The result captures exact pre/post HP for atomic
// presentation admission and seven-stage publication.
func (s *State) PlanNativeCommandDamage(actor, confirmed *Unit, commandID int, resistByClass map[int]int, rngState uint16) (*NativeCommandDamagePlan, error) {
	if s == nil {
		return nil, fmt.Errorf("missing native command state")
	}
	if commandID < 0 || commandID > 12 || len(s.NativeCommandBook) != 36 || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native command damage record unavailable id=%d", commandID)
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
	// The original table is total for valid runtime class IDs.  Validate every
	// target before 0x1CA89-equivalent MP mutation to keep a missing editable
	// table entry fail-closed rather than making a partial command transaction.
	for _, target := range targets {
		if raw, ok := resistByClass[target.ClassID]; !ok || raw < 0 || raw > 10 {
			return nil, fmt.Errorf("native command damage missing resistance class=%d", target.ClassID)
		}
	}
	if actor == nil || actor.Acted || record.MPCost < 0 || record.MPCost > 0xff || actor.MP < record.MPCost {
		return nil, fmt.Errorf("native command damage insufficient MP")
	}
	plan := &NativeCommandDamagePlan{
		Actor: actor, CommandID: commandID, MPBefore: actor.MP, MPAfter: actor.MP - record.MPCost,
		ActorActedBefore: actor.Acted, RNGBefore: rngState,
		Results: make([]NativeCommandDamageResult, 0, len(targets)), publishedTargetStages: make([]int, len(targets)),
	}
	for _, target := range targets {
		resolved, nextRNG, err := ResolveNativeCommandDamage(record.Damage, record.Hit, resistByClass[target.ClassID], rngState)
		if err != nil {
			return nil, err
		}
		rngState = nextRNG
		hpAfter := target.HP
		if resolved.Hit {
			hpAfter -= resolved.Damage
			if hpAfter < 0 {
				hpAfter = 0
			}
		}
		plan.Results = append(plan.Results, NativeCommandDamageResult{
			Target: target, NativeCommandDamage: resolved, HPBefore: target.HP, HPAfter: hpAfter,
		})
	}
	plan.RNGAfter = rngState
	return plan, nil
}

func ApplyNativeCommandDamageMP(plan *NativeCommandDamagePlan) error {
	if plan == nil || plan.Actor == nil || plan.mpPublished || plan.completed ||
		plan.Actor.MP != plan.MPBefore || plan.Actor.Acted != plan.ActorActedBefore {
		return fmt.Errorf("native command damage MP publish state changed")
	}
	plan.Actor.MP = plan.MPAfter
	plan.mpPublished = true
	return nil
}

// ApplyNativeCommandDamageStage publishes the exact intermediate HP formula
// used by 0x2A6BD with command0's denominator seven.  A caller may jump
// directly to stage7 for state-only execution; presentation advances 1..7.
func ApplyNativeCommandDamageStage(plan *NativeCommandDamagePlan, index, stage int) error {
	if plan == nil || !plan.mpPublished || plan.completed || index < 0 || index >= len(plan.Results) || stage < 1 || stage > 7 {
		return fmt.Errorf("native command damage stage unavailable")
	}
	result := plan.Results[index]
	previous := plan.publishedTargetStages[index]
	if stage < previous || (stage != 7 && stage != previous+1) || result.Target == nil {
		return fmt.Errorf("native command damage stage order changed")
	}
	wantCurrent := result.HPBefore - (result.HPBefore-result.HPAfter)*previous/7
	if result.Target.HP != wantCurrent {
		return fmt.Errorf("native command damage target publish state changed")
	}
	result.Target.HP = result.HPBefore - (result.HPBefore-result.HPAfter)*stage/7
	plan.publishedTargetStages[index] = stage
	return nil
}

func CompleteNativeCommandDamage(plan *NativeCommandDamagePlan) error {
	if plan == nil || plan.Actor == nil || !plan.mpPublished || plan.completed ||
		plan.Actor.MP != plan.MPAfter || plan.Actor.Acted != plan.ActorActedBefore {
		return fmt.Errorf("native command damage completion state changed")
	}
	for index, result := range plan.Results {
		if result.Target == nil || plan.publishedTargetStages[index] != 7 || result.Target.HP != result.HPAfter {
			return fmt.Errorf("native command damage target is not fully published")
		}
	}
	plan.Actor.Acted = true
	plan.completed = true
	return nil
}
