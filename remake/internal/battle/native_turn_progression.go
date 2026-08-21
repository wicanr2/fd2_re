package battle

import (
	"fmt"
	"strings"
)

type NativeTurnEvent76Plan struct {
	Final bool

	event NativeTurnEvent
}

func validateNativeTurnEvent76(event NativeTurnEvent) error {
	p := event.Progression
	if event.EventID != 76 || event.RawCamp != 2 || !strings.EqualFold(event.Handler, "0x35d60") ||
		p == nil || p.StateIndex != 17 || p.RepeatUntil != 4 || p.MarkUnitIndex != 1 ||
		!strings.EqualFold(p.MarkHelper, "0x13512") || p.ControlSlot != 1 || p.RescheduleDelta != 1 ||
		p.FinalTextIndex != 2 || p.SpawnGroup != 1 || !strings.EqualFold(p.SpawnHelper, "0x10b4e") ||
		p.RawPlacementGate != 0 || p.SpawnCount != 3 || p.BaseStateIndex != 21 ||
		p.FinalActivation != (NativeTurnActivation{Slot: 2, EventID: 79, RawCamp: 0, TurnDelta: 0}) ||
		!strings.EqualFold(p.PulseHandler, "0x35e5a") || p.PulseCount != 6 || p.ExtraDelayMS != 400 ||
		len(p.TailTextIndices) != 4 || p.TailTextIndices[0] != 3 || p.TailTextIndices[1] != 4 ||
		p.TailTextIndices[2] != 5 || p.TailTextIndices[3] != 6 {
		return fmt.Errorf("event76: editable progression differs from recovered handler")
	}
	return nil
}

// PlanNativeTurnEvent76 驗證 raw-camp2 的 live row 與 state byte，不修改單位、
// 排程或 roster。state17 小於4走 repeat branch；等於4才進 final branch。
func PlanNativeTurnEvent76(st *State, event NativeTurnEvent) (NativeTurnEvent76Plan, error) {
	if st == nil || !st.HasNativeTurnEventControlState || st.NativeRoundCounter <= 0 ||
		st.NativeRoundCounter > 0xfe {
		return NativeTurnEvent76Plan{}, fmt.Errorf("event76: native round or live controls unavailable")
	}
	if err := validateNativeTurnEvent76(event); err != nil {
		return NativeTurnEvent76Plan{}, err
	}
	p := event.Progression
	row := st.NativeTurnEventControls[p.ControlSlot]
	if row != (NativeTurnEventControl{Turn: byte(st.NativeRoundCounter), EventID: 76, RawCamp: 2}) {
		return NativeTurnEvent76Plan{}, fmt.Errorf("event76: live row identity mismatch")
	}
	progress := int(st.NativeEventState[p.StateIndex])
	if progress < 0 || progress > p.RepeatUntil {
		return NativeTurnEvent76Plan{}, fmt.Errorf("event76: state progress %d outside recovered range", progress)
	}
	if progress < p.RepeatUntil {
		if p.MarkUnitIndex >= len(st.Units) || st.Units[p.MarkUnitIndex] == nil ||
			!st.Units[p.MarkUnitIndex].HasNativeRecordByte5 {
			return NativeTurnEvent76Plan{}, fmt.Errorf("event76: slot1 raw +5 provenance unavailable")
		}
		if st.HasNativeRuntimeUnitProjection {
			if len(st.NativeRuntimeRecords) != len(st.Units) ||
				st.NativeRuntimeRecords[p.MarkUnitIndex].Raw[5] != st.Units[p.MarkUnitIndex].NativeRecordByte5 {
				return NativeTurnEvent76Plan{}, fmt.Errorf("event76: saved raw slot1 disagrees with typed unit")
			}
		}
	}
	return NativeTurnEvent76Plan{Final: progress == p.RepeatUntil, event: event}, nil
}

// CommitNativeTurnEvent76Repeat 重驗 repeat plan，並原子提交 slot1 raw +5 bit7、
// state17 increment及row1下一回合；若存在CONTINUE raw views則同步雙寫。
func CommitNativeTurnEvent76Repeat(st *State, plan NativeTurnEvent76Plan) error {
	if st == nil || plan.Final || plan.event.Progression == nil {
		return fmt.Errorf("event76: repeat plan unavailable")
	}
	recheck, err := PlanNativeTurnEvent76(st, plan.event)
	if err != nil || recheck.Final {
		return fmt.Errorf("event76: repeat plan changed before commit: %v", err)
	}
	p := plan.event.Progression
	next := st.NativeRoundCounter + p.RescheduleDelta
	if next <= 0 || next > 0xfe {
		return fmt.Errorf("event76: rescheduled turn outside byte range")
	}
	unit := st.Units[p.MarkUnitIndex]
	candidateByte5 := unit.NativeRecordByte5 | 0x80
	candidateState := st.NativeEventState
	candidateRows := st.NativeTurnEventControls
	candidateRawField := append([]byte(nil), st.NativeFieldControlRaw...)
	candidateRuntime := append([]NativeRuntimeRecordState(nil), st.NativeRuntimeRecords...)
	candidateState[p.StateIndex]++
	candidateRows[p.ControlSlot].Turn = byte(next)
	if st.HasNativeFieldControlState {
		offset := 3 + p.ControlSlot*3
		row := st.NativeTurnEventControls[p.ControlSlot]
		if len(candidateRawField) <= offset+2 || candidateRawField[offset] != row.Turn ||
			candidateRawField[offset+1] != row.EventID || candidateRawField[offset+2] != row.RawCamp {
			return fmt.Errorf("event76: raw field row disagrees with typed controls")
		}
		candidateRawField[offset] = byte(next)
	}
	if st.HasNativeRuntimeUnitProjection {
		candidateRuntime[p.MarkUnitIndex].Raw[5] = candidateByte5
	}
	unit.NativeRecordByte5 = candidateByte5
	st.NativeEventState = candidateState
	st.NativeTurnEventControls = candidateRows
	if st.HasNativeFieldControlState {
		st.NativeFieldControlRaw = candidateRawField
	}
	if st.HasNativeRuntimeUnitProjection {
		st.NativeRuntimeRecords = candidateRuntime
	}
	return nil
}
