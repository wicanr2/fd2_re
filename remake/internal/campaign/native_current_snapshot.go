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

// BuildNativeCurrentPersistentRecords extends the exact persistent baseline
// only for newly appended own-camp JOIN records. Enemy/ally reinforcements are
// battle-local and do not consume persistent slots. Every new player record is
// rebuilt by the proven JOIN constructor; all pre-existing and unused bytes
// remain byte-identical to the loaded snapshot.
func BuildNativeCurrentPersistentRecords(
	state *battle.State,
	baseline fdsave.CurrentSnapshot,
	table NativeJoinConstructorTable,
	itemRows []byte,
) ([fdsave.RosterUnits]fdsave.PersistentRecord, byte, error) {
	out := baseline.PersistentRecords
	count := int(baseline.Header.PersistentCount)
	oldRuntimeCount := int(baseline.Header.RuntimeCount)
	if state == nil || count < 0 || count > len(out) || oldRuntimeCount < 0 ||
		oldRuntimeCount > len(state.Units) || len(state.Units) != len(state.NativeRuntimeRecords) {
		return out, 0, errors.New("native current save: persistent/runtime baseline is inconsistent")
	}
	identities := make(map[byte]struct{}, count)
	for index := 0; index < count; index++ {
		id := out[index].Raw[8]
		if _, exists := identities[id]; exists {
			return out, 0, fmt.Errorf("native current save: duplicate persistent identity %d", id)
		}
		identities[id] = struct{}{}
	}
	for index := oldRuntimeCount; index < len(state.Units); index++ {
		unit := state.Units[index]
		if unit == nil {
			return out, 0, fmt.Errorf("native current save: appended runtime unit %d is nil", index)
		}
		if unit.Camp != battle.Own {
			continue
		}
		if !unit.HasNativeIdentity || unit.NativeIdentity < 0 || unit.NativeIdentity > 0xff ||
			!unit.HasNativeRecordByte8 || int(unit.NativeRecordByte8) != unit.NativeIdentity {
			return out, 0, fmt.Errorf("native current save: appended own unit %d lacks JOIN identity", index)
		}
		id := byte(unit.NativeIdentity)
		if _, exists := identities[id]; exists {
			return out, 0, fmt.Errorf("native current save: appended own identity %d is not a new JOIN", id)
		}
		if count >= len(out) {
			return out, 0, errors.New("native current save: persistent roster exceeds 32 records")
		}
		record, err := table.MaterializePersistentRecord(int(id), itemRows)
		if err != nil {
			return out, 0, err
		}
		view := record.View()
		if !unit.HasNativeRecordRace || view.Race != unit.NativeRecordRace ||
			!unit.HasNativeRecordClass || view.Class != unit.NativeRecordClass ||
			!unit.HasMapSelectorKey || unit.MapSelectorKey != int(view.RawPresentationKey) {
			return out, 0, fmt.Errorf("native current save: JOIN identity %d constructor/runtime mismatch", id)
		}
		out[count] = record
		count++
		identities[id] = struct{}{}
	}
	return out, byte(count), nil
}
