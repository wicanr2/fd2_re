package battle

import (
	"fmt"
	"math/rand"
)

// NativeCommandHealResult is one final target in the shared recovered
// ID13..16 HP-restore route. The command family has separate indexed
// presentation wrappers, which are intentionally not represented here.
type NativeCommandHealResult struct {
	Target  *Unit
	Restore NativeCommandRestore
}

// NativeCommandHealTargets validates the recovered command-13..16 selector
// and MP boundary without mutating combat state. The presentation owner uses
// this preflight so a missing indexed adapter cannot debit MP or change HP.
func (s *State) NativeCommandHealTargets(actor, confirmed *Unit, commandID int) ([]*Unit, error) {
	if s == nil {
		return nil, fmt.Errorf("missing native command heal state")
	}
	if commandID < 13 || commandID > 16 || len(s.NativeCommandBook) != 36 || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native command heal record unavailable id=%d", commandID)
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
	if actor == nil || actor.MP < record.MPCost {
		return nil, fmt.Errorf("native command heal insufficient MP")
	}
	return targets, nil
}

// NativeAICommandHealTargets reproduces the 0x15311 consumer after movement:
// it rebuilds the 0x14818 target array at the actor's selected destination,
// using the raw +6 selector transform recovered for the 0x1598A/0x15311 path.
// It does not reuse the player's confirmed-cursor predicate.
func (s *State) NativeAICommandHealTargets(actor *Unit, commandID int) ([]*Unit, error) {
	if s == nil || actor == nil || !actor.HasNativeRecordByte6 {
		return nil, fmt.Errorf("native AI command heal selector unavailable")
	}
	if commandID < 13 || commandID > 16 || len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native AI command heal record unavailable id=%d", commandID)
	}
	selector := int(actor.NativeRecordByte6)
	if selector != 0 && selector != 1 {
		return nil, fmt.Errorf("native AI command heal selector=%d is outside 0/1", selector)
	}
	records, err := NativeAIScoringRecords(s.Units)
	if err != nil {
		return nil, err
	}
	targetCode, err := nativeAIScoredCommandTargetCode(s.NativeCommandBook[commandID].TargetCode, selector)
	if err != nil {
		return nil, err
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	indices, err := nativeAIScoredCommandTargetIndices(
		s.W, s.H, records, len(s.Units), Cell{
			X: int(actor.NativeMapPresentation.X), Y: int(actor.NativeMapPresentation.Y),
		},
		s.NativeCommandBook[commandID].EffectMode, targetCode, flags,
	)
	if err != nil {
		return nil, err
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("native AI command heal target array is empty")
	}
	targets := make([]*Unit, 0, len(indices))
	for _, index := range indices {
		targets = append(targets, s.Units[int(index)])
	}
	if actor.MP < s.NativeCommandBook[commandID].MPCost {
		return nil, fmt.Errorf("native AI command heal insufficient MP")
	}
	return targets, nil
}

// ExecuteNativeCommandHeal executes the state portion shared by native IDs
// 13..16 (0x21AD9/0x21B99/0x2211C/0x22153 -> 0x21B18): generic two-stage
// targets, that record's MP debit, and per-target 0x1C916 restore using the
// same record's u16 damage field. It remains fail-closed outside this exact
// bounded family and does not attach its renderer/SFX/UI.
func (s *State) ExecuteNativeCommandHeal(actor, confirmed *Unit, commandID int, rng *rand.Rand) ([]NativeCommandHealResult, error) {
	if s == nil || rng == nil {
		return nil, fmt.Errorf("missing native command heal state/rng")
	}
	if commandID < 13 || commandID > 16 || len(s.NativeCommandBook) != 36 || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native command heal record unavailable id=%d", commandID)
	}
	targets, err := s.NativeCommandHealTargets(actor, confirmed, commandID)
	if err != nil {
		return nil, err
	}
	return s.executeNativeCommandHealTargets(actor, targets, commandID, rng)
}

// ExecuteNativeAICommandHeal consumes only the target array rebuilt by the
// recovered 0x15311 AI owner. Presentation remains owned by the caller.
func (s *State) ExecuteNativeAICommandHeal(actor *Unit, commandID int, rng *rand.Rand) ([]NativeCommandHealResult, error) {
	if s == nil || rng == nil {
		return nil, fmt.Errorf("missing native AI command heal state/rng")
	}
	targets, err := s.NativeAICommandHealTargets(actor, commandID)
	if err != nil {
		return nil, err
	}
	return s.executeNativeCommandHealTargets(actor, targets, commandID, rng)
}

func (s *State) executeNativeCommandHealTargets(actor *Unit, targets []*Unit, commandID int, rng *rand.Rand) ([]NativeCommandHealResult, error) {
	if s == nil || rng == nil || commandID < 13 || commandID > 16 ||
		len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native command heal target transaction unavailable id=%d", commandID)
	}
	record := s.NativeCommandBook[commandID]
	if !SpendNativeCommandMP(actor, record.MPCost) {
		return nil, fmt.Errorf("native command heal insufficient MP")
	}
	results := make([]NativeCommandHealResult, 0, len(targets))
	for _, target := range targets {
		restore, err := ApplyNativeCommandRestore(target, record.Damage, rng)
		if err != nil {
			return nil, err
		}
		results = append(results, NativeCommandHealResult{Target: target, Restore: restore})
	}
	actor.Acted = true
	return results, nil
}
