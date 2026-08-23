package battle

import "fmt"

// NativeCompoundCommand34Result preserves the three directly observed raw
// modifier stages in 0x27FC9 order. It deliberately assigns no gameplay name
// to the affected intervals or derived words.
type NativeCompoundCommand34Result struct {
	Stages   []NativeCommandModifierResult
	RNGState uint16
}

// ExecuteNativeCompoundCommand34 implements only the proven player class-19
// state transaction. The reachable FIGANI resources bypass the sole known MP
// debit sink, so record34 remains an availability gate without mutation.
// Presentation and non-player visual groups remain closed.
func (s *State) ExecuteNativeCompoundCommand34(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand34Result, error) {
	const commandID = 34
	if s == nil || actor == nil || actor.Acted || !actor.HasNativeRecordClass ||
		actor.NativeRecordClass != 19 || !actor.HasBattleFig || !nativeCompoundCommand34PlayerSelector(actor.BattleFig) ||
		len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return NativeCompoundCommand34Result{}, fmt.Errorf("native compound command 34 player provenance unavailable")
	}
	record := s.NativeCommandBook[commandID]
	if record.MPCost < 0 || actor.MP < record.MPCost {
		return NativeCompoundCommand34Result{}, fmt.Errorf("native compound command 34 MP gate failed")
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return NativeCompoundCommand34Result{}, err
	}
	targets, err := NativeCommandEffectTargets(
		s.W, s.H, actor, confirmed, record.SelectionMode, record.EffectMode,
		record.TargetCode, flags, s.Units,
	)
	if err != nil {
		return NativeCompoundCommand34Result{}, err
	}
	records, indices, err := nativeCommandModifierRecords(targets)
	if err != nil {
		return NativeCompoundCommand34Result{}, err
	}
	result := NativeCompoundCommand34Result{Stages: make([]NativeCommandModifierResult, 0, 3)}
	state := rngState
	for _, stageID := range [...]int{17, 18, 19} {
		stage, stageErr := ApplyNativeCommandModifier(records, indices, stageID, state)
		if stageErr != nil {
			return NativeCompoundCommand34Result{}, stageErr
		}
		result.Stages = append(result.Stages, stage)
		state = stage.RNGState
	}
	publishNativeCommandModifierRecords(targets, records)
	actor.Acted = true
	result.RNGState = state
	return result, nil
}

func nativeCompoundCommand34PlayerSelector(selector int) bool {
	switch selector {
	case 4, 5, 6, 7, 20:
		return true
	default:
		return false
	}
}
