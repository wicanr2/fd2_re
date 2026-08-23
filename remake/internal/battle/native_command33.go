package battle

import (
	"encoding/binary"
	"fmt"
	"math"
)

// NativeCompoundCommand33Result exposes only the proven raw HP batch and RNG
// boundary. The wrapper's indexed presentation and score consumer stay owned
// by their still-unresolved callers.
type NativeCompoundCommand33Result struct {
	Restore NativeRawRestoreBatch
}

// ExecuteNativeCompoundCommand33 implements the bounded player class-19 state
// transaction at 0x285A1..0x285ED. Proven player FIGANI resources bypass the
// known MP debit sink, so record33 remains an availability gate only.
func (s *State) ExecuteNativeCompoundCommand33(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand33Result, error) {
	const commandID = 33
	if s == nil || actor == nil || actor.Acted || !actor.HasNativeRecordClass ||
		actor.NativeRecordClass != 19 || !actor.HasBattleFig || !nativeCompoundPlayerSelector(actor.BattleFig) ||
		len(s.NativeCommandBook) != NativeCommandRecordCount || s.NativeCommandBook[commandID].ID != commandID {
		return NativeCompoundCommand33Result{}, fmt.Errorf("native compound command 33 player provenance unavailable")
	}
	record := s.NativeCommandBook[commandID]
	if record.MPCost < 0 || actor.MP < record.MPCost {
		return NativeCompoundCommand33Result{}, fmt.Errorf("native compound command 33 MP gate failed")
	}
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return NativeCompoundCommand33Result{}, err
	}
	targets, err := NativeCommandEffectTargets(
		s.W, s.H, actor, confirmed, record.SelectionMode, record.EffectMode,
		record.TargetCode, flags, s.Units,
	)
	if err != nil {
		return NativeCompoundCommand33Result{}, err
	}
	records, indices, err := nativeCompoundCommandTargetRecords(targets)
	if err != nil {
		return NativeCompoundCommand33Result{}, err
	}
	for index := range targets {
		base := index * nativeRecordSize
		clear(records[base+0x25 : base+0x28])
	}
	restore, err := ApplyNativeRawHPRestoreList(records, indices, 0x320, rngState)
	if err != nil {
		return NativeCompoundCommand33Result{}, err
	}
	for index, target := range targets {
		base := index * nativeRecordSize
		copy(target.NativeTransient[:], records[base+0x22:base+0x28])
		target.HP = int(binary.LittleEndian.Uint16(records[base+0x40 : base+0x42]))
	}
	actor.Acted = true
	return NativeCompoundCommand33Result{Restore: restore}, nil
}

func nativeCompoundCommandTargetRecords(targets []*Unit) ([]byte, []byte, error) {
	if len(targets) == 0 || len(targets) > 256 {
		return nil, nil, fmt.Errorf("native compound command target count=%d", len(targets))
	}
	records := make([]byte, len(targets)*nativeRecordSize)
	indices := make([]byte, len(targets))
	for index, target := range targets {
		if target == nil || !target.HasNativeRecordClass || !target.HasBattleFig ||
			target.Lv < 0 || target.Lv > math.MaxUint8 || target.BattleFig < 0 || target.BattleFig > math.MaxUint8 ||
			target.HP < 0 || target.HP > math.MaxUint16 || target.MaxHP < 0 || target.MaxHP > math.MaxUint16 {
			return nil, nil, fmt.Errorf("native compound command target %d lacks raw provenance", index)
		}
		base := index * nativeRecordSize
		records[base+7] = byte(target.BattleFig)
		records[base+0x20], records[base+0x21] = target.NativeRecordClass, byte(target.Lv)
		copy(records[base+0x22:base+0x28], target.NativeTransient[:])
		binary.LittleEndian.PutUint16(records[base+0x40:base+0x42], uint16(target.HP))
		binary.LittleEndian.PutUint16(records[base+0x42:base+0x44], uint16(target.MaxHP))
		indices[index] = byte(index)
	}
	return records, indices, nil
}
