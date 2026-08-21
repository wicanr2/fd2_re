package battle

import (
	"fmt"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func validateNativeTurnEvent79(event NativeTurnEvent) error {
	p := event.PairMutation
	if event.EventID != 79 || event.RawCamp != 0 || !strings.EqualFold(event.Handler, "0x35ee6") ||
		p == nil || p.BaseStateIndex != 21 || p.Group != 1 || p.Count != 3 || p.Modulo != 3 ||
		p.SecondOffset != 1 || !strings.EqualFold(p.MarkHelper, "0x13512") ||
		p.ControlSlot != 2 || p.RescheduleDelta != 1 {
		return fmt.Errorf("event79: editable pair mutation differs from recovered handler")
	}
	return nil
}

// ApplyNativeTurnEvent79 驗證並一次提交row2、兩個raw +5 bit7與process-wide RNG。
// RNG由caller持有；錯誤時回傳原值，State保持不變。
func ApplyNativeTurnEvent79(st *State, event NativeTurnEvent, rngState uint16) (uint16, [2]int, error) {
	if st == nil || !st.HasNativeTurnEventControlState || st.NativeRoundCounter <= 0 ||
		st.NativeRoundCounter > 0xfe {
		return rngState, [2]int{}, fmt.Errorf("event79: native round or live controls unavailable")
	}
	if err := validateNativeTurnEvent79(event); err != nil {
		return rngState, [2]int{}, err
	}
	p := event.PairMutation
	row := st.NativeTurnEventControls[p.ControlSlot]
	if row != (NativeTurnEventControl{Turn: byte(st.NativeRoundCounter), EventID: 79, RawCamp: 0}) {
		return rngState, [2]int{}, fmt.Errorf("event79: live row identity mismatch")
	}
	base := int(st.NativeEventState[p.BaseStateIndex])
	if base < 0 || base+p.Count > len(st.Units) {
		return rngState, [2]int{}, fmt.Errorf("event79: group1 base %d outside runtime", base)
	}
	for index := base; index < base+p.Count; index++ {
		unit := st.Units[index]
		if unit == nil || unit.Group != p.Group || !unit.HasNativeRecordByte5 {
			return rngState, [2]int{}, fmt.Errorf("event79: slot%d lacks group1 raw +5 provenance", index)
		}
		if st.HasNativeRuntimeUnitProjection &&
			(len(st.NativeRuntimeRecords) != len(st.Units) || st.NativeRuntimeRecords[index].Raw[5] != unit.NativeRecordByte5) {
			return rngState, [2]int{}, fmt.Errorf("event79: saved raw slot%d disagrees with typed unit", index)
		}
	}
	nextRNG := fdother.NativeRNGStep(rngState)
	targets := [2]int{
		base + int(nextRNG%uint16(p.Modulo)),
		base + (int(nextRNG)+p.SecondOffset)%p.Modulo,
	}
	nextRound := st.NativeRoundCounter + p.RescheduleDelta
	if nextRound <= 0 || nextRound > 0xfe || targets[0] == targets[1] {
		return rngState, [2]int{}, fmt.Errorf("event79: next row or pair is invalid")
	}
	candidateRows := st.NativeTurnEventControls
	candidateRows[p.ControlSlot].Turn = byte(nextRound)
	candidateField := append([]byte(nil), st.NativeFieldControlRaw...)
	candidateRuntime := append([]NativeRuntimeRecordState(nil), st.NativeRuntimeRecords...)
	if st.HasNativeFieldControlState {
		offset := 3 + p.ControlSlot*3
		if len(candidateField) <= offset+2 || candidateField[offset] != row.Turn ||
			candidateField[offset+1] != row.EventID || candidateField[offset+2] != row.RawCamp {
			return rngState, [2]int{}, fmt.Errorf("event79: raw field row disagrees with typed controls")
		}
		candidateField[offset] = byte(nextRound)
	}
	for _, target := range targets {
		if st.HasNativeRuntimeUnitProjection {
			candidateRuntime[target].Raw[5] |= 0x80
		}
	}
	for _, target := range targets {
		st.Units[target].NativeRecordByte5 |= 0x80
	}
	st.NativeTurnEventControls = candidateRows
	if st.HasNativeFieldControlState {
		st.NativeFieldControlRaw = candidateField
	}
	if st.HasNativeRuntimeUnitProjection {
		st.NativeRuntimeRecords = candidateRuntime
	}
	return nextRNG, targets, nil
}
