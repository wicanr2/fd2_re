package battle

import (
	"encoding/binary"
	"fmt"
)

type NativeCompoundCommand35Stage struct {
	CommandID    int
	MarkerOffset int
	Results      []NativeRawApplicationResult
	Accumulator  int
	RNGState     uint16
}

type NativeCommand35TargetState struct {
	Target          *Unit
	HP              int
	NativeTransient [6]byte
}

type NativeCompoundCommand35Result struct {
	Stages      []NativeCompoundCommand35Stage
	StageStates [][]NativeCommand35TargetState
	RNGState    uint16
}

type NativeCompoundCommand35Plan struct {
	Actor     *Unit
	Result    NativeCompoundCommand35Result
	before    []NativeCommand35TargetState
	published int
	completed bool
}

func nativeCommand35States(targets []*Unit) []NativeCommand35TargetState {
	out := make([]NativeCommand35TargetState, len(targets))
	for index, target := range targets {
		out[index] = NativeCommand35TargetState{Target: target, HP: target.HP, NativeTransient: target.NativeTransient}
	}
	return out
}

func nativeCommand35Publish(states []NativeCommand35TargetState) {
	for _, state := range states {
		state.Target.HP, state.Target.NativeTransient = state.HP, state.NativeTransient
	}
}

func nativeCommand35EqualCurrent(states []NativeCommand35TargetState) bool {
	for _, state := range states {
		if state.Target == nil || state.Target.HP != state.HP || state.Target.NativeTransient != state.NativeTransient {
			return false
		}
	}
	return true
}

func (s *State) PlanNativeCompoundCommand35(actor, confirmed *Unit, rngState uint16) (*NativeCompoundCommand35Plan, error) {
	const commandID = 35
	if s == nil || actor == nil || actor.Acted || !actor.HasNativeRecordClass || actor.NativeRecordClass != 19 || !actor.HasBattleFig || !nativeCompoundPlayerSelector(actor.BattleFig) || len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native compound command 35 player provenance unavailable")
	}
	record := s.NativeCommandBook[commandID]
	if record.MPCost < 0 || actor.MP < record.MPCost {
		return nil, fmt.Errorf("native compound command 35 MP gate failed")
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	targets, err := NativeCommandEffectTargets(s.W, s.H, actor, confirmed, record.SelectionMode, record.EffectMode, record.TargetCode, flags, s.Units)
	if err != nil {
		return nil, err
	}
	records, indices, err := nativeCompoundCommandTargetRecords(targets)
	if err != nil {
		return nil, err
	}
	steps, ok := NativeCompoundCommandPlan(commandID)
	if !ok || len(steps) != 3 {
		return nil, fmt.Errorf("native compound command 35 plan unavailable")
	}
	plan := &NativeCompoundCommand35Plan{Actor: actor, before: nativeCommand35States(targets)}
	state := rngState
	for _, step := range steps {
		if step.Callee != 0x22d1b || step.MarkerOffset < 0x25 || step.MarkerOffset > 0x27 {
			return nil, fmt.Errorf("native compound command 35 plan diverged")
		}
		applications, nextState, accumulator, stageErr := ApplyNativeRawApplication(records, indices, step.MarkerOffset, state)
		if stageErr != nil {
			return nil, stageErr
		}
		stage := NativeCompoundCommand35Stage{CommandID: step.CommandID, MarkerOffset: step.MarkerOffset, Results: applications, Accumulator: accumulator, RNGState: nextState}
		stageStates := make([]NativeCommand35TargetState, len(targets))
		for index, target := range targets {
			base := index * nativeRecordSize
			stageStates[index] = NativeCommand35TargetState{Target: target, HP: int(binary.LittleEndian.Uint16(records[base+0x40 : base+0x42]))}
			copy(stageStates[index].NativeTransient[:], records[base+0x22:base+0x28])
		}
		plan.Result.Stages = append(plan.Result.Stages, stage)
		plan.Result.StageStates = append(plan.Result.StageStates, stageStates)
		state = nextState
	}
	plan.Result.RNGState = state
	return plan, nil
}

func PublishNativeCompoundCommand35Stage(plan *NativeCompoundCommand35Plan, stage int) error {
	if plan == nil || plan.completed || stage != plan.published || stage < 0 || stage >= len(plan.Result.StageStates) {
		return fmt.Errorf("native command35 invalid publish stage %d", stage)
	}
	expected := plan.before
	if stage > 0 {
		expected = plan.Result.StageStates[stage-1]
	}
	if !nativeCommand35EqualCurrent(expected) {
		return fmt.Errorf("native command35 target changed before stage %d", stage)
	}
	nativeCommand35Publish(plan.Result.StageStates[stage])
	plan.published++
	return nil
}

func CompleteNativeCompoundCommand35(plan *NativeCompoundCommand35Plan) error {
	if plan == nil || plan.completed || plan.published != 3 || plan.Actor == nil || plan.Actor.Acted {
		return fmt.Errorf("native command35 completion boundary unavailable")
	}
	plan.Actor.Acted, plan.completed = true, true
	return nil
}

func AbortNativeCompoundCommand35(plan *NativeCompoundCommand35Plan) error {
	if plan == nil {
		return nil
	}
	if plan.completed {
		return fmt.Errorf("native command35 completed plan cannot abort")
	}
	nativeCommand35Publish(plan.before)
	if plan.Actor != nil {
		plan.Actor.Acted = false
	}
	plan.published = 0
	return nil
}

func (s *State) ExecuteNativeCompoundCommand35(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand35Result, error) {
	plan, err := s.PlanNativeCompoundCommand35(actor, confirmed, rngState)
	if err != nil {
		return NativeCompoundCommand35Result{}, err
	}
	for stage := 0; stage < 3; stage++ {
		if err = PublishNativeCompoundCommand35Stage(plan, stage); err != nil {
			_ = AbortNativeCompoundCommand35(plan)
			return NativeCompoundCommand35Result{}, err
		}
	}
	if err = CompleteNativeCompoundCommand35(plan); err != nil {
		_ = AbortNativeCompoundCommand35(plan)
		return NativeCompoundCommand35Result{}, err
	}
	return plan.Result, nil
}
