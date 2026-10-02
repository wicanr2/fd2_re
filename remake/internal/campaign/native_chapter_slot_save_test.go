package campaign

import (
	"encoding/binary"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// #87：未出戰record16的五個呈現欄位須留在原始持續槽，
// 同一角色仍可在戰後商店更改物品與能力值。
func TestEncodeNativePersistentRecordPreservesUnmaterializedPresentation(t *testing.T) {
	baseline := fdsave.PersistentRecord{}
	baseline.Raw[0], baseline.Raw[1] = 133, 216
	baseline.Raw[2], baseline.Raw[3], baseline.Raw[4] = 204, 189, 189
	baseline.Raw[7], baseline.Raw[8] = 15, 4
	unit := completeNativeCurrentSaveUnit()
	unit.X, unit.Y, unit.BattleFig = 0, 0, 0
	unit.HasNativeMapPresentation, unit.HasMapSelectorSlot, unit.HasBattleFig = false, false, false
	unit.NativeMapPresentation = battle.NativeMapPresentationState{}
	unit.InventorySlots[0] = 24
	unit.HP = 27
	got, err := EncodeNativePersistentRecord(baseline, unit)
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range []int{0, 1, 2, 3, 4, 7} {
		if got.Raw[offset] != baseline.Raw[offset] {
			t.Fatalf("未物化+%x=%d want原始%d", offset, got.Raw[offset], baseline.Raw[offset])
		}
	}
	if got.Raw[0x0b] != 24 || binary.LittleEndian.Uint16(got.Raw[0x40:]) != 27 {
		t.Fatal("未出戰者的物品或能力值沒有寫回")
	}
	// OnField不代表有raw呈現來源，不能用它覆寫保留欄位。
	unit.OnField = false
	again, err := EncodeNativePersistentRecord(baseline, unit)
	if err != nil || again != got {
		t.Fatalf("OnField改變存檔出處政策：err=%v", err)
	}
}

// TestBuildNativeChapterSlotOverlaysRosterOnBaseline 釘住酒店存檔寫回：已證實的欄位
// 由隊伍覆寫，其餘 byte 照讀檔基底，metadata 只改章節、筆數與金幣。第四章 r9 收據
// （state-sample9/FD2.SAV）與重製端 r38 寫出的整檔 sha256 相同就是這條規則的來源。
func TestBuildNativeChapterSlotOverlaysRosterOnBaseline(t *testing.T) {
	table, err := LoadNativeJoinConstructorTable(filepath.Join("..", "..", "assets", "data", "native_join_constructor.json"))
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(filepath.Join("..", "..", "assets", "data", "native_item_effect_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	baseline := fdsave.ChapterSlotSnapshot{Slot: 0}
	baseline.Verified.RosterCount = 1
	for i := range baseline.Records[0].Raw {
		baseline.Records[0].Raw[i] = 0xa0 + byte(i) // 不透明 byte 都要留下來
	}
	baseline.Records[0].Raw[8] = 4
	for i := range baseline.Metadata {
		baseline.Metadata[i] = 0xb0 + byte(i)
	}
	unit := completeNativeCurrentSaveUnit()
	unit.NativeIdentity, unit.HasNativeIdentity = 4, true
	unit.MapSelectorSlot = 3
	unit.NativeMapPresentation.Pose, unit.NativeMapPresentation.Motion = 0, 0
	unit.BaseAP, unit.BaseDP, unit.EquipmentBaseSet = 78, 48, true
	roster := map[int]battle.Unit{4: *unit}
	slot, err := BuildNativeChapterSlot(baseline, roster, []int{4}, 4, 2037, table, itemRows)
	if err != nil {
		t.Fatal(err)
	}
	rec := slot.Roster[:fdsave.UnitSize]
	if rec[0] != 7 || rec[1] != 8 || rec[2] != 3 || rec[3] != 0 || rec[4] != 0 || rec[5] != 0x80 ||
		rec[6] != 2 || rec[7] != 9 || rec[8] != 4 {
		t.Fatalf("header=% x", rec[:9])
	}
	if rec[0x0a] != 0x40 || rec[0x0b] != 1 || rec[0x0c] != 0x80 || rec[0x0d] != 0xff || rec[0x21] != 12 {
		t.Fatalf("inventory/level=% x %d", rec[0x0a:0x0e], rec[0x21])
	}
	if rec[0x28] != 0xa0+0x28 || rec[0x30] != 0xa0+0x30 || rec[0x3d] != 7 {
		t.Fatalf("opaque bytes changed: +28=%#x +30=%#x +3d=%#x", rec[0x28], rec[0x30], rec[0x3d])
	}
	if binary.LittleEndian.Uint16(rec[0x37:]) != 78 || binary.LittleEndian.Uint16(rec[0x39:]) != 48 ||
		rec[0x3c] != 13 || binary.LittleEndian.Uint16(rec[0x40:]) != 15 || binary.LittleEndian.Uint16(rec[0x48:]) != 19 {
		t.Fatalf("stats=% x", rec[0x37:0x50])
	}
	for i := fdsave.UnitSize; i < fdsave.RosterSize; i++ {
		if slot.Roster[i] != 0 {
			t.Fatalf("unused record byte %#x changed", i)
		}
	}
	if slot.Metadata[0] != 4 || slot.Metadata[1] != 1 || binary.LittleEndian.Uint32(slot.Metadata[2:]) != 2037 ||
		slot.Metadata[6] != 0xb6 || slot.Metadata[0x27] != 0xb0+0x27 {
		t.Fatalf("metadata=% x", slot.Metadata[:8])
	}

	// 隊伍裡找不到基底身分、或多出來的成員不在加入順序裡，都整筆拒絕。
	if _, err := BuildNativeChapterSlot(baseline, map[int]battle.Unit{}, nil, 4, 0, table, itemRows); err == nil {
		t.Fatal("缺少基底身分應拒絕")
	}
	extra := *unit
	extra.NativeIdentity, extra.NativeRecordByte8 = 12, 12
	if _, err := BuildNativeChapterSlot(baseline, map[int]battle.Unit{4: *unit, 12: extra}, []int{4}, 4, 0, table, itemRows); err == nil {
		t.Fatal("不在加入順序的成員應拒絕")
	}
	joined, err := BuildNativeChapterSlot(baseline, map[int]battle.Unit{4: *unit, 12: extra}, []int{4, 12}, 4, 0, table, itemRows)
	if err != nil {
		t.Fatal(err)
	}
	if joined.Metadata[1] != 2 || joined.Roster[fdsave.UnitSize+8] != 12 {
		t.Fatalf("JOIN 成員未補在第 2 筆：count=%d +8=%d", joined.Metadata[1], joined.Roster[fdsave.UnitSize+8])
	}
}
