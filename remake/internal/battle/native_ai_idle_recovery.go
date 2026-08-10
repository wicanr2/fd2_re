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

// NativeAIIdleRecoveryCopy preserves the raw 0x11EB0 copy arguments used by
// the recovery presentation. The names are ABI fields, not renderer or
// gameplay meanings.
type NativeAIIdleRecoveryCopy struct {
	SourceOffset      uint32
	DestinationOffset uint32
	SourceStride      uint32
	DestinationStride uint32
	Rows              uint32
	Length            uint32
}

// NativeAIIdleRecoveryDecode preserves the two 0x1DA16 calls surrounding the
// indexed-buffer copy. ResourceBase remains an expression because IDA proves
// the caller-owned pointer expression, not a high-level asset name.
type NativeAIIdleRecoveryDecode struct {
	ResourceBase string
	Mode         uint32
	Tail         uint32
}

// NativeAIIdleRecoveryCall preserves the raw unit argument of 0x12D7B.
// Address is an original executable address, not a gameplay action name.
type NativeAIIdleRecoveryCall struct {
	Address uint32
	Unit    int
}

// NativeAIIdleRecoveryGlobalWrite preserves the two raw writes to 0x51A83.
// The address/value pair is intentionally not renamed as a status flag.
type NativeAIIdleRecoveryGlobalWrite struct {
	Address uint32
	Value   uint32
}

// NativeAIIdleRecoveryPresentation is the caller-owned presentation contract
// for 0x13FD4. It contains only fixed raw arguments proved by IDA; a renderer
// or audio owner must explicitly consume it before HP is committed.
type NativeAIIdleRecoveryPresentation struct {
	Unit              int
	BeforeHP          uint16
	AfterHP           uint16
	BeforeGlobalWrite NativeAIIdleRecoveryGlobalWrite
	CoordinateCall    NativeAIIdleRecoveryCall
	SampleHandleExpr  string
	SampleIndex       uint32
	SampleLoopCount   uint32
	FirstDecode       NativeAIIdleRecoveryDecode
	SecondDecode      NativeAIIdleRecoveryDecode
	FirstCopy         NativeAIIdleRecoveryCopy
	SecondCopy        NativeAIIdleRecoveryCopy
	WaitTicks         [3]uint32
	AfterGlobalWrite  NativeAIIdleRecoveryGlobalWrite
}

const (
	nativeAIIdleRecoverySampleHandle = "[0x53EEC]"
	nativeAIIdleRecoveryResourceBase = "[0x53A49]+0x8088"
)

// BuildNativeAIIdleRecoveryPresentation materializes the exact raw sequence
// after the HP/gate preflight. The returned value is detached from records so
// the callback cannot mutate battle state through a borrowed slice.
func BuildNativeAIIdleRecoveryPresentation(decision NativeAIIdleRecoveryDecision) (NativeAIIdleRecoveryPresentation, error) {
	if !decision.Accepted || decision.CurrentHP == decision.MaximumHP ||
		decision.RawGate25 != 0 || decision.RawGate26 != 0 {
		return NativeAIIdleRecoveryPresentation{}, fmt.Errorf("native AI idle recovery presentation requires an accepted raw decision")
	}
	copyArgs := NativeAIIdleRecoveryCopy{
		SourceOffset: 0xA0504, DestinationOffset: 0x140,
		SourceStride: 0x1C8, DestinationStride: 0x138,
		Rows: 0xC0, Length: 0x1C8,
	}
	return NativeAIIdleRecoveryPresentation{
		Unit: decision.Unit, BeforeHP: decision.CurrentHP, AfterHP: decision.NextHP,
		BeforeGlobalWrite: NativeAIIdleRecoveryGlobalWrite{Address: 0x51A83, Value: 0},
		CoordinateCall:    NativeAIIdleRecoveryCall{Address: 0x12D7B, Unit: decision.Unit},
		SampleHandleExpr:  nativeAIIdleRecoverySampleHandle, SampleIndex: 4, SampleLoopCount: 1,
		FirstDecode:  NativeAIIdleRecoveryDecode{ResourceBase: nativeAIIdleRecoveryResourceBase, Mode: 2, Tail: 0xFD},
		SecondDecode: NativeAIIdleRecoveryDecode{ResourceBase: nativeAIIdleRecoveryResourceBase, Mode: 0, Tail: 0},
		FirstCopy:    copyArgs, SecondCopy: copyArgs,
		WaitTicks:        [3]uint32{1, 1, 1},
		AfterGlobalWrite: NativeAIIdleRecoveryGlobalWrite{Address: 0x51A83, Value: 1},
	}, nil
}

// ApplyNativeAIIdleRecoveryWithPresentation performs the 0x13FD4 transaction
// only after the caller has acknowledged the complete raw presentation. A
// changed record or a callback error leaves the HP bytes untouched.
func ApplyNativeAIIdleRecoveryWithPresentation(
	records []byte, count, unit int,
	present func(NativeAIIdleRecoveryPresentation) error,
) (bool, error) {
	decision, err := PlanNativeAIIdleRecovery(records, count, unit)
	if err != nil || !decision.Accepted {
		return false, err
	}
	before := append([]byte(nil), records...)
	if present == nil {
		return false, fmt.Errorf("native AI idle recovery presentation owner is unavailable")
	}
	presentation, err := BuildNativeAIIdleRecoveryPresentation(decision)
	if err != nil {
		return false, err
	}
	if err := present(presentation); err != nil {
		copy(records, before)
		return false, err
	}
	current, err := PlanNativeAIIdleRecovery(records, count, unit)
	if err != nil {
		copy(records, before)
		return false, err
	}
	if current != decision {
		copy(records, before)
		return false, fmt.Errorf("native AI idle recovery raw record changed before commit")
	}
	record := records[unit*nativeRecordSize : (unit+1)*nativeRecordSize]
	binary.LittleEndian.PutUint16(record[0x40:0x42], decision.NextHP)
	return true, nil
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
