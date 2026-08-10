package battle

import "fmt"

// nextNativeAIMode11Plan lowers the direct 0x13A9F mode-11 branch without
// routing it through 0x14EF0.  The two producers are evaluated before any
// action is committed, then their raw gate order is preserved in
// NativeMode11Stages.  A missing producer/table/record is an error rather
// than permission to use the normalized planner.
func (s *State) nextNativeAIMode11Plan(u *Unit) (*AIPlan, bool, error) {
	if s == nil || u == nil || !u.HasNativeRecordByte34 || u.NativeRecordByte34&0x0f != 11 {
		return nil, false, nil
	}
	if len(s.NativeCommandBook) != NativeCommandRecordCount || len(s.nativeFutureItemRows) == 0 {
		return nil, true, fmt.Errorf("native AI mode 11 command/item producer tables are unavailable")
	}
	actor, selector, records, actorRecord, baseFlags, costRow, err := s.nativeAIModeRuntimeContext(u)
	if err != nil {
		return nil, true, err
	}
	physical, physicalOK, err := s.nativeAI14EF0PhysicalSelection(
		actor, selector, records, actorRecord, baseFlags, costRow,
	)
	if err != nil {
		return nil, true, fmt.Errorf("native AI mode 11 0x14237: %w", err)
	}
	command, err := ScoreNativeAI1598A(
		s.W, s.H, records, len(s.Units), actor, selector, u,
		s.NativeCommandBook, baseFlags, s.NativeTerrainMoveCodes,
		costRow, nil,
	)
	if err != nil {
		return nil, true, fmt.Errorf("native AI mode 11 0x1598A: %w", err)
	}
	tx, err := SelectNativeAIMode11Transaction(
		int32(command.MaxScore), int32(nativeAIPhysicalGate(physical, physicalOK)),
		true, true,
	)
	if err != nil {
		return nil, true, err
	}
	stages, err := tx.Stages()
	if err != nil {
		return nil, true, err
	}
	plan := &AIPlan{U: u, SpellID: -1, NativeMode11Stages: make([]NativeAIMode11StagePlan, 0, len(stages))}
	for _, stage := range stages {
		stagePlan := NativeAIMode11StagePlan{Stage: stage}
		switch stage.Route {
		case NativeAIMode11Call15311:
			if !command.HasPositiveWinner || command.MaxScore < 6 || len(command.PositiveWinner.TargetIndices) == 0 {
				return nil, true, fmt.Errorf("native AI mode 11 0x15311 winner provenance is unavailable")
			}
			winner := command.PositiveWinner
			if winner.CommandID < 0 || winner.CommandID >= len(s.NativeCommandBook) {
				return nil, true, fmt.Errorf("native AI mode 11 command id=%d is unavailable", winner.CommandID)
			}
			targetIndex := int(winner.TargetIndices[0])
			if targetIndex < 0 || targetIndex >= len(s.Units) || s.Units[targetIndex] == nil {
				return nil, true, fmt.Errorf("native AI mode 11 command target provenance is unavailable")
			}
			action, err := s.nativeAIPlanForDestination(
				u, actorRecord, selector,
				Cell{X: winner.X, Y: winner.Y}, targetIndex, costRow,
			)
			if err != nil {
				return nil, true, fmt.Errorf("native AI mode 11 0x15311 path: %w", err)
			}
			action.NativeActionKind = NativeAIActionCommand
			action.NativeCommandID = winner.CommandID
			action.NativeActionScore = command.MaxScore
			action.NativeScoredCommands = s.nativeAIPlanScoredCommands(u)
			stagePlan.Action = action

		case NativeAIMode11Call1548E:
			if !physicalOK || physical.Ranking.Priority < 6 {
				return nil, true, fmt.Errorf("native AI mode 11 0x1548e physical winner provenance is unavailable")
			}
			action, err := s.nativeAIPlanForDestination(
				u, actorRecord, selector,
				Cell{X: physical.Candidate.DestinationX, Y: physical.Candidate.DestinationY},
				physical.Candidate.TargetIndex, costRow,
			)
			if err != nil {
				return nil, true, fmt.Errorf("native AI mode 11 0x1548e path: %w", err)
			}
			action.NativeActionKind = NativeAIActionPhysical
			action.NativeActionScore = physical.Ranking.Priority
			action.NativeScoredCommands = s.nativeAIPlanScoredCommands(u)
			stagePlan.Action = action

		case NativeAIMode11Call14121:
			action, recovery, err := s.nativeAIMode11BlockedOrRecovery(
				u, actor, selector, records, actorRecord, baseFlags, costRow,
			)
			if err != nil {
				return nil, true, err
			}
			stagePlan.Action, stagePlan.Recovery = action, recovery
		default:
			return nil, true, fmt.Errorf("native AI mode 11 route %#x is not executable", stage.Route)
		}
		plan.NativeMode11Stages = append(plan.NativeMode11Stages, stagePlan)
	}
	if len(plan.NativeMode11Stages) == 0 {
		return nil, true, fmt.Errorf("native AI mode 11 has no stages")
	}
	return plan, true, nil
}

// nativeAIPhysicalGate is the raw [0x53C4F] threshold producer.  The helper
// keeps the no-candidate value below the signed >=6 gate; it never invents a
// target when 0x14237 did not return one.
func nativeAIPhysicalGate(selection NativePhysicalAttackSelection, ok bool) int {
	if !ok {
		return 0
	}
	return selection.Ranking.Priority
}

// nativeAIMode11BlockedOrRecovery is the direct 0x14121 -> 0x13FD4 edge.  A
// found raw blocked cell becomes movement-only.  If no cell is found, the
// raw HP/gate decision is retained; an unaccepted decision is a legitimate
// common-tail no-op, not a normalized wait or attack.
func (s *State) nativeAIMode11BlockedOrRecovery(
	u *Unit, actor, selector int, records, actorRecord, baseFlags, costRow []byte,
) (*AIPlan, *NativeAIIdleRecoveryDecision, error) {
	flags, err := nativeAIModeBlockedSearchFlags(s.W, s.H, records, len(s.Units), selector, baseFlags)
	if err != nil {
		return nil, nil, err
	}
	blocked, found, err := NativePathBlockedCoordinate(
		s.W, s.H, Cell{X: int(actorRecord[0]), Y: int(actorRecord[1])}, 28,
		flags, s.NativeTerrainMoveCodes, costRow,
	)
	if err != nil {
		return nil, nil, err
	}
	if found {
		action, err := s.nativeAIPlanTowardRawDestination(
			u, actor, selector, records, actorRecord, baseFlags, costRow, blocked,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("native AI mode 11 0x14121 path: %w", err)
		}
		action.NativeModeFallbackActive = true
		action.NativeModeFallback = 11
		action.NativeModeWriteRangeZero = true
		action.Target = nil
		return action, nil, nil
	}
	decision, err := PlanNativeAIIdleRecovery(records, len(s.Units), actor)
	if err != nil {
		return nil, nil, fmt.Errorf("native AI mode 11 0x13fd4: %w", err)
	}
	if !decision.Accepted {
		return nil, nil, nil
	}
	return nil, &decision, nil
}
