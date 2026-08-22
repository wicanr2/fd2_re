package battle

import (
	"encoding/binary"
	"fmt"
	"math"
)

// NativeCommandModifierTargets validates the recovered player selector for
// command IDs 17..19 without mutating state. The player renderer remains
// intentionally disconnected until its indexed presentation is recovered.
func (s *State) NativeCommandModifierTargets(actor, confirmed *Unit, commandID int) ([]*Unit, error) {
	if s == nil || commandID < 17 || commandID > 19 ||
		len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native command modifier record unavailable id=%d", commandID)
	}
	record := s.NativeCommandBook[commandID]
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
	if err := s.preflightNativeCommandModifierActor(actor, commandID); err != nil {
		return nil, err
	}
	return targets, nil
}

// NativeAICommandModifierTargets rebuilds the final target array through the
// recovered 0x15311 raw-selector owner. It does not reuse the player's
// confirmed-cursor predicate.
func (s *State) NativeAICommandModifierTargets(actor *Unit, commandID int) ([]*Unit, error) {
	if s == nil || actor == nil || !actor.HasNativeRecordByte6 {
		return nil, fmt.Errorf("native AI command modifier selector unavailable")
	}
	if commandID < 17 || commandID > 19 || len(s.NativeCommandBook) != NativeCommandRecordCount ||
		s.NativeCommandBook[commandID].ID != commandID {
		return nil, fmt.Errorf("native AI command modifier record unavailable id=%d", commandID)
	}
	selector := int(actor.NativeRecordByte6)
	if selector != 0 && selector != 1 {
		return nil, fmt.Errorf("native AI command modifier selector=%d is outside 0/1", selector)
	}
	records, err := NativeAIScoringRecords(s.Units)
	if err != nil {
		return nil, err
	}
	record := s.NativeCommandBook[commandID]
	targetCode, err := nativeAIScoredCommandTargetCode(record.TargetCode, selector)
	if err != nil {
		return nil, err
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	indices, err := nativeAIScoredCommandTargetIndices(
		s.W, s.H, records, len(s.Units),
		Cell{X: int(actor.NativeMapPresentation.X), Y: int(actor.NativeMapPresentation.Y)},
		record.EffectMode, targetCode, flags,
	)
	if err != nil {
		return nil, err
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("native AI command modifier target array is empty")
	}
	targets := make([]*Unit, 0, len(indices))
	for _, index := range indices {
		targets = append(targets, s.Units[int(index)])
	}
	if err := s.preflightNativeCommandModifierActor(actor, commandID); err != nil {
		return nil, err
	}
	return targets, nil
}

// ExecuteNativeCommandModifier applies the player selector and the exact
// ID17..19 raw transaction. No presentation or status name is inferred.
func (s *State) ExecuteNativeCommandModifier(actor, confirmed *Unit, commandID int, rngState uint16) (NativeCommandModifierResult, error) {
	targets, err := s.NativeCommandModifierTargets(actor, confirmed, commandID)
	if err != nil {
		return NativeCommandModifierResult{}, err
	}
	return s.executeNativeCommandModifierTargets(actor, targets, commandID, rngState)
}

// ExecuteNativeAICommandModifier consumes only the raw target array rebuilt
// for the recovered mode-11 owner.
func (s *State) ExecuteNativeAICommandModifier(actor *Unit, commandID int, rngState uint16) (NativeCommandModifierResult, error) {
	targets, err := s.NativeAICommandModifierTargets(actor, commandID)
	if err != nil {
		return NativeCommandModifierResult{}, err
	}
	return s.executeNativeCommandModifierTargets(actor, targets, commandID, rngState)
}

func (s *State) preflightNativeCommandModifierActor(actor *Unit, commandID int) error {
	if s == nil || actor == nil || commandID < 17 || commandID > 19 ||
		len(s.NativeCommandBook) != NativeCommandRecordCount {
		return fmt.Errorf("native command modifier transaction unavailable id=%d", commandID)
	}
	debitID := commandID
	if commandID == 17 {
		debitID = 18
	}
	if s.NativeCommandBook[debitID].ID != debitID {
		return fmt.Errorf("native command modifier debit record unavailable id=%d", debitID)
	}
	if actor.MP < s.NativeCommandBook[debitID].MPCost {
		return fmt.Errorf("native command modifier insufficient MP")
	}
	return nil
}

func (s *State) executeNativeCommandModifierTargets(actor *Unit, targets []*Unit, commandID int, rngState uint16) (NativeCommandModifierResult, error) {
	if err := s.preflightNativeCommandModifierActor(actor, commandID); err != nil {
		return NativeCommandModifierResult{}, err
	}
	if len(targets) == 0 || len(targets) > 256 {
		return NativeCommandModifierResult{}, fmt.Errorf("native command modifier target count=%d", len(targets))
	}
	records := make([]byte, len(targets)*nativeRecordSize)
	indices := make([]byte, len(targets))
	for i, target := range targets {
		if target == nil {
			return NativeCommandModifierResult{}, fmt.Errorf("native command modifier target %d missing", i)
		}
		if !target.HasNativeRecordClass || target.Lv < 0 || target.Lv > math.MaxUint8 {
			return NativeCommandModifierResult{}, fmt.Errorf("native command modifier target %d lacks raw class/level provenance", i)
		}
		for _, value := range [...]int{target.AP, target.DP, target.HIT, target.EV} {
			if value < math.MinInt16 || value > math.MaxInt16 {
				return NativeCommandModifierResult{}, fmt.Errorf("native command modifier target %d derived word outside range", i)
			}
		}
		base := i * nativeRecordSize
		records[base+0x20], records[base+0x21] = target.NativeRecordClass, byte(target.Lv)
		copy(records[base+0x22:base+0x28], target.NativeTransient[:])
		for j, value := range [...]int{target.AP, target.DP, target.HIT, target.EV} {
			binary.LittleEndian.PutUint16(records[base+0x48+j*2:], uint16(int16(value)))
		}
		indices[i] = byte(i)
	}
	result, err := ApplyNativeCommandModifier(records, indices, commandID, rngState)
	if err != nil {
		return NativeCommandModifierResult{}, err
	}
	debitID := commandID
	if commandID == 17 {
		debitID = 18
	}
	if !SpendNativeCommandMP(actor, s.NativeCommandBook[debitID].MPCost) {
		return NativeCommandModifierResult{}, fmt.Errorf("native command modifier insufficient MP after preflight")
	}
	for i, target := range targets {
		base := i * nativeRecordSize
		copy(target.NativeTransient[:], records[base+0x22:base+0x28])
		target.AP = int(int16(binary.LittleEndian.Uint16(records[base+0x48:])))
		target.DP = int(int16(binary.LittleEndian.Uint16(records[base+0x4a:])))
		target.HIT = int(int16(binary.LittleEndian.Uint16(records[base+0x4c:])))
		target.EV = int(int16(binary.LittleEndian.Uint16(records[base+0x4e:])))
	}
	actor.Acted = true
	return result, nil
}
