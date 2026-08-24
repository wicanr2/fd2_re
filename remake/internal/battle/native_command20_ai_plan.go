package battle

import "fmt"

// NativeAICommand2022Plan is the state-only transaction for the enemy routes
// 0x22A85, 0x22BC6 and 0x22BE1.  It deliberately uses the raw selector owner
// and the native 16-bit RNG adapters; presentation remains a caller concern.
type NativeAICommand2022Plan struct {
	Actor     *Unit
	CommandID int
	Targets   []*Unit
	Results   []NativeAICommand2022Result
	RNGState  uint16
	mpBefore  int
	mpAfter   int
	Before    []NativeAICommand2022TargetState
	After     []NativeAICommand2022TargetState
	published bool
	completed bool
}

type NativeAICommand2022Result struct {
	Target  *Unit
	Offset  int
	Restore *NativeRawFlagRestoreResult
	Apply   *NativeRawApplicationResult
}

type NativeAICommand2022TargetState struct {
	Target          *Unit
	HP              int
	NativeTransient [6]byte
}

func (s *State) NativeAICommand2022Targets(actor *Unit, commandID int) ([]*Unit, error) {
	if s == nil || actor == nil || !actor.HasNativeRecordByte6 ||
		(commandID < 20 || commandID > 22) || len(s.NativeCommandBook) != NativeCommandRecordCount ||
		s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native AI command 20-22 record unavailable id=%d", commandID)
	}
	selector := int(actor.NativeRecordByte6)
	if selector != 0 && selector != 1 {
		return nil, fmt.Errorf("native AI command 20-22 selector=%d is outside 0/1", selector)
	}
	records, err := NativeAIScoringRecords(s.Units)
	if err != nil {
		return nil, err
	}
	if actor.Acted || s.NativeCommandBook[commandID].MPCost < 0 {
		return nil, fmt.Errorf("native AI command 20-22 actor/cost gate failed")
	}
	targetCode, err := nativeAIScoredCommandTargetCode(s.NativeCommandBook[commandID].TargetCode, selector)
	if err != nil {
		return nil, err
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	indices, err := nativeAIScoredCommandTargetIndices(s.W, s.H, records, len(s.Units),
		Cell{X: int(actor.NativeMapPresentation.X), Y: int(actor.NativeMapPresentation.Y)},
		s.NativeCommandBook[commandID].EffectMode, targetCode, flags)
	if err != nil || len(indices) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("native AI command 20-22 target array is empty")
	}
	targets := make([]*Unit, 0, len(indices))
	for _, index := range indices {
		targets = append(targets, s.Units[int(index)])
	}
	if actor.MP < s.NativeCommandBook[commandID].MPCost {
		return nil, fmt.Errorf("native AI command 20-22 insufficient MP")
	}
	return targets, nil
}

func nativeAICommand2022States(targets []*Unit) []NativeAICommand2022TargetState {
	out := make([]NativeAICommand2022TargetState, len(targets))
	for i, target := range targets {
		out[i] = NativeAICommand2022TargetState{Target: target, HP: target.HP, NativeTransient: target.NativeTransient}
	}
	return out
}

func nativeAICommand2022Equal(states []NativeAICommand2022TargetState) bool {
	for _, state := range states {
		if state.Target == nil || state.Target.HP != state.HP || state.Target.NativeTransient != state.NativeTransient {
			return false
		}
	}
	return true
}

func nativeAICommand2022Publish(states []NativeAICommand2022TargetState) {
	for _, state := range states {
		state.Target.HP, state.Target.NativeTransient = state.HP, state.NativeTransient
	}
}

func (s *State) PlanNativeAICommand2022(actor *Unit, commandID int, rngState uint16) (*NativeAICommand2022Plan, error) {
	targets, err := s.NativeAICommand2022Targets(actor, commandID)
	if err != nil {
		return nil, err
	}
	records, indices, err := nativeCompoundCommandTargetRecords(targets)
	if err != nil {
		return nil, err
	}
	marker := 0
	switch commandID {
	case 20:
		marker = 0x25
	case 21:
		marker = 0x26
	case 22:
		marker = 0x27
	default:
		return nil, fmt.Errorf("native AI command 20-22 unsupported id=%d", commandID)
	}
	plan := &NativeAICommand2022Plan{Actor: actor, CommandID: commandID, Targets: append([]*Unit(nil), targets...), Before: nativeAICommand2022States(targets), mpBefore: actor.MP, mpAfter: actor.MP - s.NativeCommandBook[commandID].MPCost}
	shadow := append([]byte(nil), records...)
	if commandID == 20 || commandID == 21 {
		if s.NativeCommandBook[10].ID != 10 || s.NativeCommandBook[10].Damage < 0 {
			return nil, fmt.Errorf("native AI command restore amount unavailable")
		}
		results, next, _, err := ApplyNativeRawFlagRestore(shadow, indices, marker, rngState)
		if err != nil {
			return nil, err
		}
		plan.RNGState = next
		for index := range results {
			result := &results[index]
			entry := NativeAICommand2022Result{Target: targets[result.TargetIndex], Offset: marker, Restore: result}
			plan.Results = append(plan.Results, entry)
		}
	} else {
		results, next, _, err := ApplyNativeRawApplication(shadow, indices, marker, rngState)
		if err != nil {
			return nil, err
		}
		plan.RNGState = next
		for index := range results {
			result := &results[index]
			entry := NativeAICommand2022Result{Target: targets[result.TargetIndex], Offset: marker, Apply: result}
			plan.Results = append(plan.Results, entry)
		}
	}
	plan.After = nativeAICommand2022StatesFromRecords(targets, shadow)
	return plan, nil
}

func nativeAICommand2022StatesFromRecords(targets []*Unit, records []byte) []NativeAICommand2022TargetState {
	out := make([]NativeAICommand2022TargetState, len(targets))
	for i, target := range targets {
		base := i * nativeRecordSize
		out[i] = NativeAICommand2022TargetState{Target: target, HP: int(records[base+0x40]) | int(records[base+0x41])<<8}
		copy(out[i].NativeTransient[:], records[base+0x22:base+0x28])
	}
	return out
}

func PublishNativeAICommand2022(plan *NativeAICommand2022Plan) error {
	if plan == nil || plan.Actor == nil || plan.published || plan.completed || plan.Actor.Acted || plan.Actor.MP != plan.mpBefore || !nativeAICommand2022Equal(plan.Before) {
		return fmt.Errorf("native AI command 20-22 publish boundary unavailable")
	}
	nativeAICommand2022Publish(plan.After)
	plan.Actor.MP = plan.mpAfter
	plan.published = true
	return nil
}

func CompleteNativeAICommand2022(plan *NativeAICommand2022Plan) error {
	if plan == nil || !plan.published || plan.completed || plan.Actor == nil || plan.Actor.Acted || plan.Actor.MP != plan.mpAfter || !nativeAICommand2022Equal(plan.After) {
		return fmt.Errorf("native AI command 20-22 completion boundary unavailable")
	}
	plan.Actor.Acted = true
	plan.completed = true
	return nil
}

func AbortNativeAICommand2022(plan *NativeAICommand2022Plan) error {
	if plan == nil {
		return nil
	}
	if plan.completed {
		return fmt.Errorf("native AI command 20-22 completed plan cannot abort")
	}
	if plan.published {
		nativeAICommand2022Publish(plan.Before)
		plan.Actor.MP = plan.mpBefore
	}
	if plan.Actor != nil {
		plan.Actor.Acted = false
	}
	plan.published = false
	return nil
}
