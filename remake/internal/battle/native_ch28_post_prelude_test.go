package battle

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func nativeCh28PostPreludeFixture(frontier int) *State {
	units := make([]*Unit, frontier)
	for index := range units {
		group := 0
		if index >= 20 && index < 76 {
			group = 8
		} else if index >= 76 {
			group = []int{4, 4, 5, 5, 6, 6, 7, 7, 1, 1, 1}[index-76]
		}
		units[index] = &Unit{
			Group: group, HP: 0x1234,
			NativeMapPresentation:    NativeMapPresentationState{X: byte(index), Y: 2, Pose: 3},
			HasNativeMapPresentation: true,
			NativeRecordByte5:        0, HasNativeRecordByte5: true,
		}
	}
	records := make([]NativeRuntimeRecordState, frontier)
	for index := range records {
		records[index].Raw[0] = units[index].NativeMapPresentation.X
		records[index].Raw[1] = units[index].NativeMapPresentation.Y
		records[index].Raw[3] = units[index].NativeMapPresentation.Pose
		records[index].Raw[5] = units[index].NativeRecordByte5
		binary.LittleEndian.PutUint16(records[index].Raw[0x40:0x42], uint16(units[index].HP))
	}
	return &State{
		Units:                          units,
		Roster:                         []*Unit{{Group: 2}, {Group: 3}, {Group: 9}},
		NativeRuntimeRecords:           records,
		HasNativeRuntimeUnitProjection: true,
	}
}

func TestNativeCh28PostFrontiersFollowRecoveredAppendPrefixes(t *testing.T) {
	for _, frontier := range []int{76, 78, 80, 82, 84, 87} {
		if err := ValidateNativeCh28PostFrontier(nativeCh28PostPreludeFixture(frontier)); err != nil {
			t.Errorf("frontier %d: %v", frontier, err)
		}
	}
	invalid := nativeCh28PostPreludeFixture(82)
	invalid.Units[80].Group = 7
	if err := ValidateNativeCh28PostFrontier(invalid); err == nil {
		t.Fatal("out-of-order dynamic group was accepted")
	}
}

func TestNativeCh28PostRawPreludeClearsTailAndPatchesSlot20(t *testing.T) {
	st := nativeCh28PostPreludeFixture(87)
	st.NativeRuntimeRecords[19].Raw[0x40] = 0xaa
	st.Units[19].HP = 0x12aa
	if err := ApplyNativeCh28PostRawPrelude(st); err != nil {
		t.Fatal(err)
	}
	if st.NativeRuntimeRecords[19].Raw[0x40] != 0xaa {
		t.Fatal("raw prelude cleared a record before slot20")
	}
	for index := 20; index < len(st.NativeRuntimeRecords); index++ {
		if st.NativeRuntimeRecords[index].Raw[0x40] != 0 || st.NativeRuntimeRecords[index].Raw[0x41] != 0 {
			t.Fatalf("slot%d raw word40 was not cleared", index)
		}
	}
	if st.NativeRuntimeRecords[20].Raw[7] != 0x7e || st.NativeRuntimeRecords[20].Raw[8] != 0x7e ||
		st.Units[20].HP != 0 || st.Units[19].HP != 0x12aa ||
		!st.Units[20].HasBattleFig || st.Units[20].BattleFig != 0x7e ||
		!st.Units[20].HasNativeRecordByte8 || st.Units[20].NativeRecordByte8 != 0x7e {
		t.Fatalf("slot20 raw patch record=%#v unit=%#v", st.NativeRuntimeRecords[20].Raw[7:9], st.Units[20])
	}
}

func TestNativeCh28PostRawPreludeSupportsTypedNormalBattleState(t *testing.T) {
	st := nativeCh28PostPreludeFixture(76)
	st.NativeRuntimeRecords = nil
	st.HasNativeRuntimeUnitProjection = false
	if err := ApplyNativeCh28PostRawPrelude(st); err != nil {
		t.Fatal(err)
	}
	if st.Units[19].HP != 0x1234 || st.Units[20].HP != 0 || st.Units[75].HP != 0 ||
		st.Units[20].BattleFig != 0x7e || st.Units[20].NativeRecordByte8 != 0x7e {
		t.Fatalf("typed normal-battle transaction mismatch: slot19=%+v slot20=%+v slot75=%+v", st.Units[19], st.Units[20], st.Units[75])
	}
}

func TestNativeCh28PostRawPreludeRejectsIncompleteProjectionAtomically(t *testing.T) {
	st := nativeCh28PostPreludeFixture(84)
	st.NativeRuntimeRecords = st.NativeRuntimeRecords[:83]
	beforeUnits := append([]*Unit(nil), st.Units...)
	beforeRecords := append([]NativeRuntimeRecordState(nil), st.NativeRuntimeRecords...)
	if err := ApplyNativeCh28PostRawPrelude(st); err == nil {
		t.Fatal("incomplete raw projection was accepted")
	}
	if !reflect.DeepEqual(st.Units, beforeUnits) || !reflect.DeepEqual(st.NativeRuntimeRecords, beforeRecords) {
		t.Fatal("failed raw prelude mutated battle state")
	}
}

func TestNativeCh28PostRawPreludeRejectsProjectionDisagreementAtomically(t *testing.T) {
	st := nativeCh28PostPreludeFixture(76)
	st.NativeRuntimeRecords[8].Raw[0x40]++
	beforeUnits := append([]*Unit(nil), st.Units...)
	beforeRecords := append([]NativeRuntimeRecordState(nil), st.NativeRuntimeRecords...)
	if err := ApplyNativeCh28PostRawPrelude(st); err == nil {
		t.Fatal("raw/typed disagreement was accepted")
	}
	if !reflect.DeepEqual(st.Units, beforeUnits) || !reflect.DeepEqual(st.NativeRuntimeRecords, beforeRecords) {
		t.Fatal("projection disagreement mutated battle state")
	}
}
