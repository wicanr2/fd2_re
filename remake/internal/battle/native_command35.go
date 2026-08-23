package battle

import (
	"encoding/binary"
	"fmt"
)

// NativeCompoundCommand35Stage preserves one directly observed 0x22D1B call
// without assigning a gameplay name to its marker byte.
type NativeCompoundCommand35Stage struct {
	CommandID    int
	MarkerOffset int
	Results      []NativeRawApplicationResult
	Accumulator  int
	RNGState     uint16
}

// NativeCompoundCommand35Result exposes the three raw stages in 0x27FC9
// order. Indexed presentation and the score/EXP consumer remain outside this
// bounded player transaction.
type NativeCompoundCommand35Result struct {
	Stages   []NativeCompoundCommand35Stage
	RNGState uint16
}

// ExecuteNativeCompoundCommand35 implements only the proven player class-19
// transaction. The reachable FIGANI resources bypass the sole known MP debit
// sink, so record35 remains an availability gate without mutation.
func (s *State) ExecuteNativeCompoundCommand35(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand35Result, error) {
	const commandID = 35
	if s == nil || actor == nil || actor.Acted || !actor.HasNativeRecordClass ||
		actor.NativeRecordClass != 19 || !actor.HasBattleFig || !nativeCompoundPlayerSelector(actor.BattleFig) ||
		len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return NativeCompoundCommand35Result{}, fmt.Errorf("native compound command 35 player provenance unavailable")
	}
	record := s.NativeCommandBook[commandID]
	if record.MPCost < 0 || actor.MP < record.MPCost {
		return NativeCompoundCommand35Result{}, fmt.Errorf("native compound command 35 MP gate failed")
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return NativeCompoundCommand35Result{}, err
	}
	targets, err := NativeCommandEffectTargets(
		s.W, s.H, actor, confirmed, record.SelectionMode, record.EffectMode,
		record.TargetCode, flags, s.Units,
	)
	if err != nil {
		return NativeCompoundCommand35Result{}, err
	}
	records, indices, err := nativeCompoundCommandTargetRecords(targets)
	if err != nil {
		return NativeCompoundCommand35Result{}, err
	}
	plan, ok := NativeCompoundCommandPlan(commandID)
	if !ok || len(plan) != 3 {
		return NativeCompoundCommand35Result{}, fmt.Errorf("native compound command 35 plan unavailable")
	}
	result := NativeCompoundCommand35Result{Stages: make([]NativeCompoundCommand35Stage, 0, len(plan))}
	state := rngState
	for _, step := range plan {
		if step.Callee != 0x22d1b || step.MarkerOffset < 0x25 || step.MarkerOffset > 0x27 {
			return NativeCompoundCommand35Result{}, fmt.Errorf("native compound command 35 plan diverged")
		}
		applications, nextState, accumulator, stageErr := ApplyNativeRawApplication(
			records, indices, step.MarkerOffset, state,
		)
		if stageErr != nil {
			return NativeCompoundCommand35Result{}, stageErr
		}
		state = nextState
		result.Stages = append(result.Stages, NativeCompoundCommand35Stage{
			CommandID: step.CommandID, MarkerOffset: step.MarkerOffset,
			Results: applications, Accumulator: accumulator, RNGState: state,
		})
	}
	for index, target := range targets {
		base := index * nativeRecordSize
		copy(target.NativeTransient[:], records[base+0x22:base+0x28])
		target.HP = int(binary.LittleEndian.Uint16(records[base+0x40 : base+0x42]))
	}
	actor.Acted = true
	result.RNGState = state
	return result, nil
}
