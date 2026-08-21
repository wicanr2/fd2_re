package battle

import (
	"encoding/binary"
	"fmt"
)

var nativeCh28PostFrontiers = map[int]struct{}{
	76: {}, 78: {}, 80: {}, 82: {}, 84: {}, 87: {},
}

// ValidateNativeCh28PostFrontier accepts only the materialization prefixes
// produced by the recovered map28 chain: 20 persistent records, group8's 56
// rows, groups4..7 in two-row order, then group1's three rows. Groups2/3 are
// source-only and group9 must still be a single pending constructor row.
func ValidateNativeCh28PostFrontier(st *State) error {
	if st == nil {
		return fmt.Errorf("ch28 post: battle state unavailable")
	}
	if _, ok := nativeCh28PostFrontiers[len(st.Units)]; !ok {
		return fmt.Errorf("ch28 post: runtime frontier %d is not recovered", len(st.Units))
	}
	if len(st.Units) < 76 {
		return fmt.Errorf("ch28 post: base frontier unavailable")
	}
	for index := 20; index < 76; index++ {
		if st.Units[index] == nil || st.Units[index].Group != 8 {
			return fmt.Errorf("ch28 post: slot%d is not the recovered group8 prefix", index)
		}
	}
	wantTail := []int{4, 4, 5, 5, 6, 6, 7, 7, 1, 1, 1}
	for offset, unit := range st.Units[76:] {
		if unit == nil || unit.Group != wantTail[offset] {
			return fmt.Errorf("ch28 post: slot%d breaks the recovered dynamic group order", 76+offset)
		}
	}
	pendingGroup9 := 0
	for _, unit := range st.Roster {
		if unit != nil && unit.Group == 9 {
			pendingGroup9++
		}
	}
	if pendingGroup9 != 1 {
		return fmt.Errorf("ch28 post: pending group9 rows=%d want 1", pendingGroup9)
	}
	return nil
}

// ApplyNativeCh28PostRawPrelude reproduces only the proven raw writers before
// sub_1DB65: clear word +0x40 from slot20 through the current tail, then set
// slot20 raw +7/+8 to 0x7e. The 0x1DB65 indexed presenter is a separate gate;
// callers must not treat this transaction as the complete postbattle beat.
func ApplyNativeCh28PostRawPrelude(st *State) error {
	if err := ValidateNativeCh28PostFrontier(st); err != nil {
		return err
	}
	for index, unit := range st.Units {
		if unit == nil || !unit.HasNativeMapPresentation || !unit.HasNativeRecordByte5 ||
			unit.HP < 0 || unit.HP > 0xffff {
			return fmt.Errorf("ch28 post: slot%d lacks presenter raw-field provenance", index)
		}
	}
	if st.HasNativeRuntimeUnitProjection {
		if len(st.NativeRuntimeRecords) != len(st.Units) {
			return fmt.Errorf("ch28 post: complete 0x50 runtime projection is inconsistent")
		}
		for index, unit := range st.Units {
			record := st.NativeRuntimeRecords[index].Raw
			if record[0] != unit.NativeMapPresentation.X ||
				record[1] != unit.NativeMapPresentation.Y ||
				record[3] != unit.NativeMapPresentation.Pose ||
				record[5] != unit.NativeRecordByte5 ||
				binary.LittleEndian.Uint16(record[0x40:0x42]) != uint16(unit.HP) {
				return fmt.Errorf("ch28 post: slot%d raw projection disagrees with typed presenter fields", index)
			}
		}
	} else if len(st.NativeRuntimeRecords) != 0 {
		return fmt.Errorf("ch28 post: unproven partial 0x50 runtime projection")
	}

	units := append([]*Unit(nil), st.Units...)
	for index, unit := range st.Units {
		patched := *unit
		if index >= 20 {
			patched.HP = 0
		}
		if index == 20 {
			patched.BattleFig = 0x7e
			patched.HasBattleFig = true
			patched.NativeRecordByte8 = 0x7e
			patched.HasNativeRecordByte8 = true
		}
		units[index] = &patched
	}

	if st.HasNativeRuntimeUnitProjection {
		records := append([]NativeRuntimeRecordState(nil), st.NativeRuntimeRecords...)
		for index := 20; index < len(records); index++ {
			binary.LittleEndian.PutUint16(records[index].Raw[0x40:0x42], 0)
		}
		records[20].Raw[7] = 0x7e
		records[20].Raw[8] = 0x7e
		st.NativeRuntimeRecords = records
	}
	st.Units = units
	return nil
}
