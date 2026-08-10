package battle

import (
	"encoding/binary"
	"fmt"
)

// NativeAIIdleRecoveryDecision preserves the raw decision boundary of
// 0x13fd4.  The fields are offsets and values, not normalized status names.
// Accepted is false for an equal-HP record or either non-zero raw gate.
type NativeAIIdleRecoveryDecision struct {
	Unit      int
	Accepted  bool
	CurrentHP uint16
	MaximumHP uint16
	NextHP    uint16
	RawGate25 byte
	RawGate26 byte
}

// PlanNativeAIIdleRecovery is the side-effect-free E0 adapter for 0x13fd4.
// It reads only the raw record fields used by the native function and never
// writes the detached snapshot.  Presentation/wait calls remain outside this
// contract.
func PlanNativeAIIdleRecovery(records []byte, count, unit int) (NativeAIIdleRecoveryDecision, error) {
	decision := NativeAIIdleRecoveryDecision{Unit: unit}
	if count < 0 || count > len(records)/nativeRecordSize || unit < 0 || unit >= count {
		return decision, fmt.Errorf("native AI idle recovery unit is out of bounds")
	}
	record := records[unit*nativeRecordSize : (unit+1)*nativeRecordSize]
	current := binary.LittleEndian.Uint16(record[0x40:0x42])
	maximum := binary.LittleEndian.Uint16(record[0x42:0x44])
	decision.CurrentHP = current
	decision.MaximumHP = maximum
	decision.RawGate25 = record[0x25]
	decision.RawGate26 = record[0x26]
	if current == maximum || record[0x25] != 0 || record[0x26] != 0 {
		return decision, nil
	}
	next := uint32(current) + uint32(maximum)/5
	if next > uint32(maximum) {
		next = uint32(maximum)
	}
	decision.Accepted = true
	decision.NextHP = uint16(next)
	return decision, nil
}

// ApplyNativeAIIdleRecovery commits a previously verified 0x13fd4 decision.
// The decision is recomputed from the same raw snapshot immediately before
// writing, so a caller cannot accidentally apply a stale normalized HP value.
func ApplyNativeAIIdleRecovery(records []byte, count, unit int) (bool, error) {
	decision, err := PlanNativeAIIdleRecovery(records, count, unit)
	if err != nil || !decision.Accepted {
		return false, err
	}
	record := records[unit*nativeRecordSize : (unit+1)*nativeRecordSize]
	binary.LittleEndian.PutUint16(record[0x40:0x42], decision.NextHP)
	return true, nil
}
