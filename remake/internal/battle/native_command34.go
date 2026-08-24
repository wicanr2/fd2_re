package battle

import "fmt"

type NativeCommand34TargetState struct {
	Target          *Unit
	NativeTransient [6]byte
	AP, DP, HIT, EV int
}

type NativeCompoundCommand34Result struct {
	Stages      []NativeCommandModifierResult
	StageStates [][]NativeCommand34TargetState
	RNGState    uint16
}

type NativeCompoundCommand34Plan struct {
	Actor     *Unit
	Result    NativeCompoundCommand34Result
	before    []NativeCommand34TargetState
	published int
	completed bool
}

func nativeCommand34States(targets []*Unit) []NativeCommand34TargetState {
	out := make([]NativeCommand34TargetState, len(targets))
	for i, target := range targets {
		out[i] = NativeCommand34TargetState{Target: target, NativeTransient: target.NativeTransient, AP: target.AP, DP: target.DP, HIT: target.HIT, EV: target.EV}
	}
	return out
}

func nativeCommand34Publish(states []NativeCommand34TargetState) {
	for _, state := range states {
		state.Target.NativeTransient = state.NativeTransient
		state.Target.AP, state.Target.DP, state.Target.HIT, state.Target.EV = state.AP, state.DP, state.HIT, state.EV
	}
}

func nativeCommand34EqualCurrent(states []NativeCommand34TargetState) bool {
	for _, state := range states {
		if state.Target == nil || state.Target.NativeTransient != state.NativeTransient || state.Target.AP != state.AP || state.Target.DP != state.DP || state.Target.HIT != state.HIT || state.Target.EV != state.EV {
			return false
		}
	}
	return true
}

// PlanNativeCompoundCommand34 computes the three raw stages without mutation.
// The indexed owner publishes each stage only after its mask frames are drawn.
func (s *State) PlanNativeCompoundCommand34(actor, confirmed *Unit, rngState uint16) (*NativeCompoundCommand34Plan, error) {
	const commandID = 34
	if s == nil || actor == nil || actor.Acted || !actor.HasNativeRecordClass || actor.NativeRecordClass != 19 || !actor.HasBattleFig || !nativeCompoundPlayerSelector(actor.BattleFig) || len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native compound command 34 player provenance unavailable")
	}
	record := s.NativeCommandBook[commandID]
	if record.MPCost < 0 || actor.MP < record.MPCost {
		return nil, fmt.Errorf("native compound command 34 MP gate failed")
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	targets, err := NativeCommandEffectTargets(s.W, s.H, actor, confirmed, record.SelectionMode, record.EffectMode, record.TargetCode, flags, s.Units)
	if err != nil {
		return nil, err
	}
	records, indices, err := nativeCommandModifierRecords(targets)
	if err != nil {
		return nil, err
	}
	plan := &NativeCompoundCommand34Plan{Actor: actor, before: nativeCommand34States(targets)}
	state := rngState
	for _, stageID := range [...]int{17, 18, 19} {
		stage, stageErr := ApplyNativeCommandModifier(records, indices, stageID, state)
		if stageErr != nil {
			return nil, stageErr
		}
		shadow := make([]*Unit, len(targets))
		for i, target := range targets {
			copied := *target
			shadow[i] = &copied
		}
		publishNativeCommandModifierRecords(shadow, records)
		stageStates := nativeCommand34States(shadow)
		for i := range stageStates {
			stageStates[i].Target = targets[i]
		}
		plan.Result.Stages = append(plan.Result.Stages, stage)
		plan.Result.StageStates = append(plan.Result.StageStates, stageStates)
		state = stage.RNGState
	}
	plan.Result.RNGState = state
	return plan, nil
}

func PublishNativeCompoundCommand34Stage(plan *NativeCompoundCommand34Plan, stage int) error {
	if plan == nil || plan.completed || stage != plan.published || stage < 0 || stage >= len(plan.Result.StageStates) {
		return fmt.Errorf("native command34 invalid publish stage %d", stage)
	}
	expected := plan.before
	if stage > 0 {
		expected = plan.Result.StageStates[stage-1]
	}
	if !nativeCommand34EqualCurrent(expected) {
		return fmt.Errorf("native command34 target changed before stage %d", stage)
	}
	nativeCommand34Publish(plan.Result.StageStates[stage])
	plan.published++
	return nil
}

func CompleteNativeCompoundCommand34(plan *NativeCompoundCommand34Plan) error {
	if plan == nil || plan.completed || plan.published != 3 || plan.Actor == nil || plan.Actor.Acted {
		return fmt.Errorf("native command34 completion boundary unavailable")
	}
	plan.Actor.Acted, plan.completed = true, true
	return nil
}

func AbortNativeCompoundCommand34(plan *NativeCompoundCommand34Plan) error {
	if plan == nil {
		return nil
	}
	if plan.completed {
		return fmt.Errorf("native command34 completed plan cannot abort")
	}
	nativeCommand34Publish(plan.before)
	if plan.Actor != nil {
		plan.Actor.Acted = false
	}
	plan.published = 0
	return nil
}

func (s *State) ExecuteNativeCompoundCommand34(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand34Result, error) {
	plan, err := s.PlanNativeCompoundCommand34(actor, confirmed, rngState)
	if err != nil {
		return NativeCompoundCommand34Result{}, err
	}
	for stage := 0; stage < 3; stage++ {
		if err = PublishNativeCompoundCommand34Stage(plan, stage); err != nil {
			_ = AbortNativeCompoundCommand34(plan)
			return NativeCompoundCommand34Result{}, err
		}
	}
	if err = CompleteNativeCompoundCommand34(plan); err != nil {
		_ = AbortNativeCompoundCommand34(plan)
		return NativeCompoundCommand34Result{}, err
	}
	return plan.Result, nil
}

func nativeCompoundPlayerSelector(selector int) bool {
	switch selector {
	case 4, 5, 6, 7, 20:
		return true
	default:
		return false
	}
}
