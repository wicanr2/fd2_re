package battle

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func nativeTransientPhaseTestState(units ...*Unit) *State {
	records := make([]NativeRuntimeRecordState, len(units))
	for slot, unit := range units {
		records[slot].Raw[5] = unit.NativeRecordByte5
		records[slot].Raw[6] = unit.NativeRecordByte6
		copy(records[slot].Raw[NativeTransientOffset:NativeTransientOffset+NativeTransientCount], unit.NativeTransient[:])
		binary.LittleEndian.PutUint16(records[slot].Raw[0x40:0x42], uint16(unit.HP))
		binary.LittleEndian.PutUint16(records[slot].Raw[0x42:0x44], uint16(unit.MaxHP))
	}
	return &State{Units: units, NativeRuntimeRecords: records, HasNativeRuntimeUnitProjection: true}
}

func nativeTransientPhaseTestUnit(selector byte, hp, maxHP int, duration byte) *Unit {
	return &Unit{
		HP: hp, MaxHP: maxHP,
		NativeRecordByte5: 0, HasNativeRecordByte5: true,
		NativeRecordByte6: selector, HasNativeRecordByte6: true,
		NativeTransient: [6]byte{0, 0, 0, duration},
	}
}

func TestNativeTransientStorageIsBoundedByOriginalOffsets(t *testing.T) {
	u := &Unit{}
	if u.SetNativeTransientDuration(0x21, 3) || u.SetNativeTransientDuration(0x28, 3) {
		t.Fatal("out-of-range raw offsets must be rejected")
	}
	if !u.SetNativeTransientDuration(0x22, 3) || !u.SetNativeTransientDuration(0x27, 1) {
		t.Fatal("recovered transient range must be writable")
	}
	if got, ok := u.NativeTransientDuration(0x22); !ok || got != 3 {
		t.Fatalf("+0x22 = (%d,%v), want (3,true)", got, ok)
	}
	if _, ok := u.NativeTransientDuration(0x28); ok {
		t.Fatal("out-of-range read must fail closed")
	}
}

func TestTickNativeTransientsUsesRawGates(t *testing.T) {
	active := &Unit{Camp: Enemy, OnField: false, HP: 0, NativeRecordByte5: 0, HasNativeRecordByte5: true, NativeRecordByte6: 7, HasNativeRecordByte6: true, NativeTransient: [6]byte{1, 2, 0, 1, 0, 3}}
	otherSelector := &Unit{NativeRecordByte5: 0, HasNativeRecordByte5: true, NativeRecordByte6: 8, HasNativeRecordByte6: true, NativeTransient: [6]byte{1, 1}}
	blocked := &Unit{NativeRecordByte5: 1, HasNativeRecordByte5: true, NativeRecordByte6: 7, HasNativeRecordByte6: true, NativeTransient: [6]byte{1, 1}}
	missingRaw := &Unit{Camp: Own, OnField: true, HP: 1, NativeTransient: [6]byte{1, 1}}
	st := &State{Units: []*Unit{active, otherSelector, blocked, missingRaw}}

	expired := st.TickNativeTransientsRaw(7)
	if got, want := active.NativeTransient, [6]byte{0, 1, 0, 0, 0, 2}; got != want {
		t.Fatalf("active sweep = %#v, want %#v", got, want)
	}
	if len(expired) != 2 || expired[0].Unit != active || expired[0].Offset != 0x22 || expired[1].Offset != 0x25 {
		t.Fatalf("expiry = %#v, want +0x22/+0x25 for active unit", expired)
	}
	if otherSelector.NativeTransient != [6]byte{1, 1} || blocked.NativeTransient != [6]byte{1, 1} || missingRaw.NativeTransient != [6]byte{1, 1} {
		t.Fatal("units failing the native raw gate must not be decremented")
	}
}

func TestTickNativeTransientsCampWrapperFailsClosed(t *testing.T) {
	u := &Unit{NativeRecordByte6: 1, HasNativeRecordByte6: true, NativeRecordByte5: 0, HasNativeRecordByte5: true, NativeTransient: [6]byte{1}}
	if got := (&State{Units: []*Unit{u}}).TickNativeTransients(Own); got != nil || u.NativeTransient[0] != 1 {
		t.Fatal("normalized Camp must not be guessed as the native selector")
	}
}

func TestAdvanceNativeTransientPhaseAppliesDamageBeforeCountdown(t *testing.T) {
	unit := nativeTransientPhaseTestUnit(2, 100, 100, 2)
	st := nativeTransientPhaseTestState(unit)
	result, err := st.AdvanceNativeTransientPhaseRaw(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Damage) != 1 || result.Damage[0].Slot != 0 ||
		result.Damage[0].Amount != 10 || result.Damage[0].Before != 100 ||
		result.Damage[0].After != 90 || unit.HP != 90 ||
		binary.LittleEndian.Uint16(st.NativeRuntimeRecords[0].Raw[0x40:0x42]) != 90 ||
		unit.NativeTransient[3] != 1 || st.NativeRuntimeRecords[0].Raw[0x25] != 1 ||
		len(result.InactiveSlots) != 0 || len(result.Expired) != 0 {
		t.Fatalf("result=%+v HP=%d duration=%d rawHP=%d rawDuration=%d", result, unit.HP,
			unit.NativeTransient[3], binary.LittleEndian.Uint16(st.NativeRuntimeRecords[0].Raw[0x40:0x42]),
			st.NativeRuntimeRecords[0].Raw[0x25])
	}
}

func TestAdvanceNativeTransientPhaseFatalDamageMarksInactiveAndSkipsCountdown(t *testing.T) {
	for _, tc := range []struct {
		name string
		hp   int
	}{
		{name: "exact", hp: 10},
		{name: "clamped", hp: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unit := nativeTransientPhaseTestUnit(2, tc.hp, 100, 2)
			unit.NativeRecordByte5 = 0x80
			st := nativeTransientPhaseTestState(unit)
			result, err := st.AdvanceNativeTransientPhaseRaw(2)
			if err != nil {
				t.Fatal(err)
			}
			if unit.HP != 0 || unit.NativeRecordByte5 != 1 || st.NativeRuntimeRecords[0].Raw[5] != 1 ||
				unit.NativeTransient[3] != 2 || st.NativeRuntimeRecords[0].Raw[0x25] != 2 ||
				!reflect.DeepEqual(result.InactiveSlots, []int{0}) || len(result.Expired) != 0 {
				t.Fatalf("result=%+v HP=%d byte5=%02x duration=%d", result, unit.HP,
					unit.NativeRecordByte5, unit.NativeTransient[3])
			}
		})
	}
}

func TestAdvanceNativeTransientPhaseZeroDamageStillCountsDown(t *testing.T) {
	unit := nativeTransientPhaseTestUnit(2, 5, 9, 1)
	st := nativeTransientPhaseTestState(unit)
	result, err := st.AdvanceNativeTransientPhaseRaw(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Damage) != 1 || result.Damage[0].Amount != 0 || unit.HP != 5 ||
		unit.NativeTransient[3] != 0 || len(result.Expired) != 1 || result.Expired[0].Offset != 0x25 {
		t.Fatalf("result=%+v HP=%d duration=%d", result, unit.HP, unit.NativeTransient[3])
	}
}

func TestAdvanceNativeTransientPhaseUsesSelectorAndInactiveGates(t *testing.T) {
	selected := nativeTransientPhaseTestUnit(2, 50, 100, 2)
	other := nativeTransientPhaseTestUnit(1, 50, 100, 2)
	inactive := nativeTransientPhaseTestUnit(2, 50, 100, 2)
	inactive.NativeRecordByte5 = 1
	st := nativeTransientPhaseTestState(selected, other, inactive)
	if _, err := st.AdvanceNativeTransientPhaseRaw(2); err != nil {
		t.Fatal(err)
	}
	if selected.HP != 40 || selected.NativeTransient[3] != 1 ||
		other.HP != 50 || other.NativeTransient[3] != 2 ||
		inactive.HP != 50 || inactive.NativeTransient[3] != 2 {
		t.Fatalf("selected=%d/%d other=%d/%d inactive=%d/%d", selected.HP,
			selected.NativeTransient[3], other.HP, other.NativeTransient[3], inactive.HP,
			inactive.NativeTransient[3])
	}
}

func TestAdvanceNativeTransientPhaseMarksMultipleDeathsWithoutDispatchState(t *testing.T) {
	first := nativeTransientPhaseTestUnit(2, 10, 100, 2)
	second := nativeTransientPhaseTestUnit(2, 1, 20, 3)
	survivor := nativeTransientPhaseTestUnit(2, 30, 100, 2)
	st := nativeTransientPhaseTestState(first, second, survivor)
	result, err := st.AdvanceNativeTransientPhaseRaw(2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.InactiveSlots, []int{0, 1}) || len(result.Damage) != 3 ||
		first.NativeRecordByte5 != 1 || second.NativeRecordByte5 != 1 || survivor.HP != 20 ||
		first.NativeTransient[3] != 2 || second.NativeTransient[3] != 3 || survivor.NativeTransient[3] != 1 {
		t.Fatalf("result=%+v first=%+v second=%+v survivor=%+v", result, first, second, survivor)
	}
}

func TestAdvanceNativeTransientPhaseFailsClosedOnRawMismatch(t *testing.T) {
	unit := nativeTransientPhaseTestUnit(2, 50, 100, 2)
	st := nativeTransientPhaseTestState(unit)
	st.NativeRuntimeRecords[0].Raw[0x40]++
	beforeUnit := *unit
	beforeRaw := st.NativeRuntimeRecords[0]
	if _, err := st.AdvanceNativeTransientPhaseRaw(2); err == nil {
		t.Fatal("raw HP mismatch was accepted")
	}
	if !reflect.DeepEqual(*unit, beforeUnit) || !reflect.DeepEqual(st.NativeRuntimeRecords[0], beforeRaw) {
		t.Fatal("rejected phase mutated typed or raw state")
	}
}
