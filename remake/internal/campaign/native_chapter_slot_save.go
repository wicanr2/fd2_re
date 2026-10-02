package campaign

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// native_chapter_slot_save.go — 酒店存檔（0x30012）寫回原版四槽的 roster 與 metadata。
//
// 原版把記憶體裡的 32 筆持續紀錄整段抄進 `0x312B + slot×0xA28`，再寫 metadata
// `+0` 章節、`+1` 筆數、`+2..+5` 金幣（`+6..+9` 由系統設定寫，這裡沿用 baseline）。
// 重製端的持續隊伍是 battle.Unit 投影，沒有保留每一個 byte，所以每筆紀錄以
// 讀檔時的 raw baseline 為底，只覆寫已證實的欄位；沒有 raw 來源的 byte 一律不動。

// EncodeNativePersistentRecord 把一個持續隊伍單位覆寫到它讀檔時的 raw 紀錄上。
// 覆寫的欄位與 BuildNativeCurrentRuntimeRecords 同一組（座標、+5、+6..+8、物品欄、
// 指令位元、種族／職業／等級、暫時狀態、+0x34..+0x36、+0x3B..+0x4F），再加上升級
// 會動的 +0x37／+0x39 基礎攻防（`0x1E529` 寫的兩個 word）。
func EncodeNativePersistentRecord(baseline fdsave.PersistentRecord, unit *battle.Unit) (fdsave.PersistentRecord, error) {
	if unit == nil {
		return baseline, errors.New("native chapter slot save: unit is nil")
	}
	if unit.X < 0 || unit.X > 0xff || unit.Y < 0 || unit.Y > 0xff || !unit.HasNativeRecordByte5 ||
		unit.BaseAP < -0x8000 || unit.BaseAP > 0x7fff || unit.BaseDP < -0x8000 || unit.BaseDP > 0x7fff {
		return baseline, errors.New("native chapter slot save: unit lacks raw provenance for the persistent record")
	}
	panel, err := battle.NativeItemPanelRecordForUnit(unit)
	if err != nil {
		return baseline, fmt.Errorf("native chapter slot save: %w", err)
	}
	if panel[8] != baseline.Raw[8] {
		return baseline, fmt.Errorf("native chapter slot save: identity %d does not match baseline %d", panel[8], baseline.Raw[8])
	}
	record := baseline
	// JOIN 之後尚未被 0x11506 同步的記錄：+0..+4、+0x34..+0x36、+0x3d 是建構器沒碰的殘值，
	// 要等 0x11506 整筆抄回才會有場上的值（第七章 r6 凱麗）。
	pending := unit.NativeJoinPersistentPending
	if !pending {
		// #87：11506只抄回raw身分匹配的runtime紀錄。未出戰者的
		// LOAD投影沒有呈現來源，保留持續槽的原始欄位。
		if unit.HasNativeMapPresentation {
			record.Raw[0] = byte(unit.X)
			record.Raw[1] = byte(unit.Y)
			record.Raw[3] = unit.NativeMapPresentation.Pose
			record.Raw[4] = unit.NativeMapPresentation.Motion
		}
		if unit.HasMapSelectorSlot && unit.MapSelectorSlot >= 0 && unit.MapSelectorSlot <= 0xff {
			record.Raw[2] = byte(unit.MapSelectorSlot)
		}
	}
	record.Raw[5] = unit.NativeRecordByte5
	record.Raw[6], record.Raw[8] = panel[6], panel[8]
	if unit.HasBattleFig {
		record.Raw[7] = panel[7]
	}
	copy(record.Raw[0x0a:0x28], panel[0x0a:0x28])
	if unit.HasNativeRecordDeathEffect && !pending {
		// 建構器只寫 +0x31=0xff，+0x32／+0x33 留殘值。
		copy(record.Raw[0x31:0x34], unit.NativeRecordDeathEffect[:])
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
		if present.has && !pending {
			record.Raw[offset] = present.value
		}
	}
	if unit.EquipmentBaseSet {
		binary.LittleEndian.PutUint16(record.Raw[0x37:], uint16(int16(unit.BaseAP)))
		binary.LittleEndian.PutUint16(record.Raw[0x39:], uint16(int16(unit.BaseDP)))
	}
	copy(record.Raw[0x3b:0x3d], panel[0x3b:0x3d])
	copy(record.Raw[0x3e:0x50], panel[0x3e:0x50])
	return record, nil
}

// BuildNativeChapterSlot 由讀檔時的槽快照、目前的持續隊伍與金幣組出要寫回的槽。
// baseline 裡的每一筆有效紀錄都要在隊伍裡找得到同一個 +8 身分；隊伍多出來的身分
// 依加入順序以 JOIN 建構列補在後面。不在場的 byte 全部照 baseline。
func BuildNativeChapterSlot(
	baseline fdsave.ChapterSlotSnapshot,
	roster map[int]battle.Unit,
	joinOrder []int,
	chapter byte,
	gold uint32,
	table NativeJoinConstructorTable,
	itemRows []byte,
) (fdsave.Slot, error) {
	count := int(baseline.Verified.RosterCount)
	if count < 0 || count > fdsave.RosterUnits {
		return fdsave.Slot{}, fmt.Errorf("native chapter slot save: baseline count %d outside 0..%d", count, fdsave.RosterUnits)
	}
	byIdentity := make(map[int]*battle.Unit, len(roster))
	for id := range roster {
		unit := roster[id]
		if !unit.HasNativeIdentity {
			return fdsave.Slot{}, fmt.Errorf("native chapter slot save: roster member %d lacks native identity", id)
		}
		if _, dup := byIdentity[unit.NativeIdentity]; dup {
			return fdsave.Slot{}, fmt.Errorf("native chapter slot save: duplicate native identity %d", unit.NativeIdentity)
		}
		byIdentity[unit.NativeIdentity] = &unit
	}
	records := baseline.Records
	seen := make(map[int]bool, count)
	for index := 0; index < count; index++ {
		identity := int(records[index].Raw[8])
		unit, ok := byIdentity[identity]
		if !ok {
			return fdsave.Slot{}, fmt.Errorf("native chapter slot save: baseline record %d identity %d is not in the roster", index, identity)
		}
		encoded, err := EncodeNativePersistentRecord(records[index], unit)
		if err != nil {
			return fdsave.Slot{}, fmt.Errorf("record %d: %w", index, err)
		}
		records[index] = encoded
		seen[identity] = true
	}
	for _, id := range joinOrder {
		unit, ok := roster[id]
		if !ok || seen[unit.NativeIdentity] {
			continue
		}
		if count >= fdsave.RosterUnits {
			return fdsave.Slot{}, errors.New("native chapter slot save: persistent roster exceeds 32 records")
		}
		// sub_112A5 寫進 count 那一格，沒寫到的 byte 是那一格 LOAD 時的殘值（第七章
		// 凱麗：JOIN12 的身分與場上 raw +8=140 不符，0x11506 不抄）。0x11506 按 raw +8
		// 匹配，不要求參戰或特定 camp；第二章新登場的 JOIN8 也會立即被同步。同步後
		// 的場上記錄覆蓋建構殘值；這份具型別投影以零基底補建，已知欄位由 unit 寫回。
		residual := fdsave.PersistentRecord{}
		if unit.NativeJoinPersistentPending {
			residual = records[count]
		}
		constructed, err := table.MaterializePersistentRecordOn(residual, unit.NativeIdentity, itemRows)
		if err != nil {
			return fdsave.Slot{}, err
		}
		encoded, err := EncodeNativePersistentRecord(constructed, &unit)
		if err != nil {
			return fdsave.Slot{}, fmt.Errorf("joined record %d: %w", count, err)
		}
		records[count] = encoded
		seen[unit.NativeIdentity] = true
		count++
	}
	if len(seen) != len(byIdentity) {
		return fdsave.Slot{}, fmt.Errorf("native chapter slot save: %d roster members are not in the join order", len(byIdentity)-len(seen))
	}
	slot := fdsave.Slot{
		Roster:   make([]byte, 0, fdsave.RosterSize),
		Metadata: append([]byte(nil), baseline.Metadata[:]...),
	}
	for index := range records {
		slot.Roster = append(slot.Roster, records[index].Raw[:]...)
	}
	slot.Metadata[0] = chapter
	slot.Metadata[1] = byte(count)
	binary.LittleEndian.PutUint32(slot.Metadata[2:], gold)
	return slot, nil
}
