package campaign

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// BuildNativeCurrentRuntimeRecords overlays the proven live Unit fields on an
// exact current-runtime raw baseline. Unknown bytes remain byte-identical.
// This is intentionally narrower than a general Unit serializer: it accepts
// only a stable CONTINUE-projected roster in original record order.
func BuildNativeCurrentRuntimeRecords(
	state *battle.State,
) ([]fdsave.PersistentRecord, error) {
	if state == nil || !state.HasNativeRuntimeUnitProjection ||
		len(state.Units) == 0 || len(state.NativeRuntimeRecords) != len(state.Units) {
		return nil, errors.New("native current save: runtime roster lacks exact CONTINUE provenance")
	}
	out := make([]fdsave.PersistentRecord, len(state.Units))
	for index, unit := range state.Units {
		if unit == nil || !unit.HasNativeMapPresentation ||
			unit.X < 0 || unit.X > 0xff || unit.Y < 0 || unit.Y > 0xff ||
			int(unit.NativeMapPresentation.X) != unit.X ||
			int(unit.NativeMapPresentation.Y) != unit.Y ||
			!unit.HasNativeRecordByte5 ||
			(unit.NativeRecordByte5&1 == 0) != unit.OnField ||
			(unit.NativeRecordByte5&0x80 != 0) != unit.Acted {
			return nil, fmt.Errorf("native current save: runtime unit %d has inconsistent live provenance", index)
		}
		panel, err := battle.NativeItemPanelRecordForUnit(unit)
		if err != nil {
			return nil, fmt.Errorf("native current save: runtime unit %d: %w", index, err)
		}
		baseline := state.NativeRuntimeRecords[index]
		if baseline.SelectorKey != byte(unit.BattleFig) ||
			!unit.HasMapSelectorKey || unit.MapSelectorKey != int(baseline.SelectorKey) ||
			!unit.HasMapSelectorSlot || unit.MapSelectorSlot != int(baseline.SelectorSlot) {
			return nil, fmt.Errorf("native current save: runtime unit %d selector identity changed", index)
		}

		record := baseline.Raw
		record[0] = byte(unit.X)
		record[1] = byte(unit.Y)
		record[3] = unit.NativeMapPresentation.Pose
		record[4] = unit.NativeMapPresentation.Motion
		record[5] = unit.NativeRecordByte5
		copy(record[6:9], panel[6:9])
		copy(record[0x0a:0x28], panel[0x0a:0x28])
		if unit.HasNativeRecordDeathEffect {
			copy(record[0x31:0x34], unit.NativeRecordDeathEffect[:])
		}
		for offset, present := range map[int]struct {
			value byte
			has   bool
		}{
			0x34: {unit.NativeRecordByte34, unit.HasNativeRecordByte34},
			0x35: {unit.NativeRecordByte35, unit.HasNativeRecordByte35},
			0x36: {unit.NativeRecordByte36, unit.HasNativeRecordByte36},
			0x3d: {unit.NativeRecordByte3D, unit.HasNativeRecordByte3D},
		} {
			if present.has {
				record[offset] = present.value
			}
		}
		copy(record[0x3b:0x3d], panel[0x3b:0x3d])
		copy(record[0x3e:0x50], panel[0x3e:0x50])
		out[index].Raw = record
	}
	return out, nil
}
