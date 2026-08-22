package main

import (
	"encoding/binary"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// applyNativeTransientPhase owns one proven sub_1A866(selector) phase sweep.
// It decrements raw +0x22..+0x27 and, when any byte expires, runs the proven
// 0x1B750 equipment/derived-stat recalculation before publishing the state.
func (g *Game) applyNativeTransientPhase(selector byte) ([]battle.NativeTransientExpiry, error) {
	if g == nil || g.st == nil || !g.st.HasNativeRuntimeUnitProjection ||
		len(g.st.Units) != len(g.st.NativeRuntimeRecords) {
		return nil, fmt.Errorf("native transient phase: runtime raw projection is incomplete")
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	)
	if err != nil {
		return nil, fmt.Errorf("native transient phase: item rows: %w", err)
	}
	candidate := *g.st
	candidate.Units = make([]*battle.Unit, len(g.st.Units))
	for index, source := range g.st.Units {
		if source == nil {
			return nil, fmt.Errorf("native transient phase: unit %d is nil", index)
		}
		clone := *source
		candidate.Units[index] = &clone
	}
	candidate.NativeRuntimeRecords = append(
		[]battle.NativeRuntimeRecordState(nil), g.st.NativeRuntimeRecords...,
	)
	expired := candidate.TickNativeTransientsRaw(selector)
	expiredUnits := make(map[*battle.Unit]struct{}, len(expired))
	for _, event := range expired {
		expiredUnits[event.Unit] = struct{}{}
	}
	for index, unit := range candidate.Units {
		if !unit.HasNativeRecordByte6 || unit.NativeRecordByte6 != selector ||
			!unit.HasNativeRecordByte5 || unit.NativeRecordByte5&1 != 0 {
			continue
		}
		panel, err := battle.NativeItemPanelRecordForUnit(unit)
		if err != nil {
			return nil, fmt.Errorf("native transient phase: unit %d record: %w", index, err)
		}
		record := append([]byte(nil), candidate.NativeRuntimeRecords[index].Raw[:]...)
		copy(record[6:0x28], panel[6:0x28])
		copy(record[0x3b:0x48], panel[0x3b:0x48])
		if _, didExpire := expiredUnits[unit]; didExpire {
			if err := battle.ApplyNativeRuntimeEquipmentRecalc(record, itemRows); err != nil {
				return nil, fmt.Errorf("native transient phase: unit %d recompute: %w", index, err)
			}
			unit.AP = int(int16(binary.LittleEndian.Uint16(record[0x48:])))
			unit.DP = int(int16(binary.LittleEndian.Uint16(record[0x4a:])))
			unit.HIT = int(int16(binary.LittleEndian.Uint16(record[0x4c:])))
			unit.EV = int(int16(binary.LittleEndian.Uint16(record[0x4e:])))
		}
		copy(candidate.NativeRuntimeRecords[index].Raw[:], record)
	}
	*g.st = candidate
	return expired, nil
}
