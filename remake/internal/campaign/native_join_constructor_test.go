package campaign

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func TestNativeJoinConstructorMaterializesKeliFromRawTables(t *testing.T) {
	table, err := LoadNativeJoinConstructorTable(filepath.Join("..", "..", "assets", "data", "native_join_constructor.json"))
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(filepath.Join("..", "..", "assets", "data", "native_item_effect_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	unit, err := table.MaterializePersistentUnit(12, battle.Unit{
		Name: "凱麗", HP: 90, MaxHP: 90, MP: 7, MaxMP: 7, AP: 44, DP: 33,
		Exp: 42, NativeCommandMask: [5]byte{9, 9, 9, 9, 9},
		NativeTransient: [6]byte{1, 2, 3, 4, 5, 6},
	}, itemRows)
	if err != nil {
		t.Fatal(err)
	}
	if unit.Name != "凱麗" || unit.Lv != 10 || unit.ClassID != 8 || unit.MV != 5 ||
		unit.HP != 151 || unit.MaxHP != 151 || unit.MP != 0 || unit.MaxMP != 0 ||
		unit.NativeRecordWord42 != 151 || !unit.HasNativeRecordWord42 ||
		unit.NativeRecordWord46 != 0 || !unit.HasNativeRecordWord46 ||
		unit.NativeIdentity != 12 || !unit.HasNativeIdentity ||
		unit.NativeRecordByte5 != 0 || !unit.HasNativeRecordByte5 ||
		unit.NativeRecordByte6 != 2 || !unit.HasNativeRecordByte6 ||
		unit.MapSelectorKey != 12 || !unit.HasMapSelectorKey ||
		unit.BattleFig != 12 || !unit.HasBattleFig || unit.Exp != 0 ||
		unit.NativeCommandMask != [5]byte{} || unit.NativeTransient != [6]byte{} {
		t.Fatalf("unexpected Keli JOIN projection: %+v", unit)
	}
	if unit.AP != 100 || unit.DP != 79 || unit.HIT != 110 || unit.EV != 15 ||
		unit.DX != 10 || unit.BaseAP != 80 || unit.BaseDP != 69 ||
		unit.BaseHIT != 10 || unit.BaseEV != 10 || !unit.EquipmentBaseSet {
		t.Fatalf("unexpected Keli 0x1145A projection: %+v", unit)
	}
	if !reflect.DeepEqual(unit.NativeInventoryFlags, []int{0x40, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}) ||
		!reflect.DeepEqual(unit.InventorySlots, []int{0x3e, 0xac, 0xff, 0xff, 0xff, 0xff, 0, 0}) ||
		!reflect.DeepEqual(unit.Inventory, []int{0x3e, 0xac}) ||
		!reflect.DeepEqual(unit.Equipped, []bool{true, true}) {
		t.Fatalf("unexpected Keli JOIN inventory: slots=%#v flags=%#v inventory=%#v equipped=%#v",
			unit.InventorySlots, unit.NativeInventoryFlags, unit.Inventory, unit.Equipped)
	}
}

func TestNativeJoinConstructorRejectsWrongExecutableIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "join.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":1,"source":{"exe_size":1},"evidence_level":"已證實","rows":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNativeJoinConstructorTable(path); err == nil {
		t.Fatal("wrong executable identity was accepted")
	}
}

func TestNativeJoinConstructorRejectsUnknownIdentity(t *testing.T) {
	table, err := LoadNativeJoinConstructorTable(filepath.Join("..", "..", "assets", "data", "native_join_constructor.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := table.MaterializePersistentUnit(32, battle.Unit{}, nil); err == nil {
		t.Fatal("unknown JOIN identity was accepted")
	}
}

func TestNativeJoinConstructorRejectsMissingEquippedItemRowAtomically(t *testing.T) {
	table, err := LoadNativeJoinConstructorTable(filepath.Join("..", "..", "assets", "data", "native_join_constructor.json"))
	if err != nil {
		t.Fatal(err)
	}
	base := battle.Unit{Name: "不可變", AP: 777, Inventory: []int{9}}
	if unit, err := table.MaterializePersistentUnit(12, base, make([]byte, battle.NativeItemEffectRowSize)); err == nil {
		t.Fatalf("short item table was accepted: %+v", unit)
	}
	if base.AP != 777 || !bytes.Equal([]byte{byte(base.Inventory[0])}, []byte{9}) {
		t.Fatalf("failed transaction mutated caller base: %+v", base)
	}
}

func TestNativeJoinConstructorMaterializesAllKnownRows(t *testing.T) {
	table, err := LoadNativeJoinConstructorTable(filepath.Join("..", "..", "assets", "data", "native_join_constructor.json"))
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(filepath.Join("..", "..", "assets", "data", "native_item_effect_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	for id := 0; id < 32; id++ {
		unit, err := table.MaterializePersistentUnit(id, battle.Unit{}, itemRows)
		if err != nil {
			t.Fatalf("row %d: %v", id, err)
		}
		if unit.NativeIdentity != id || len(unit.InventorySlots) != 8 ||
			len(unit.NativeInventoryFlags) != 8 || !unit.EquipmentBaseSet {
			t.Fatalf("row %d incomplete projection: %+v", id, unit)
		}
	}
}

func TestNativeJoinConstructorRejectsOffsetDrift(t *testing.T) {
	source := filepath.Join("..", "..", "assets", "data", "native_join_constructor.json")
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(raw, []byte(`"default_file_offset": "0x55ba1"`), []byte(`"default_file_offset": "0x55ba2"`), 1)
	if bytes.Equal(raw, tampered) {
		t.Fatal("fixture did not contain row-zero source offset")
	}
	path := filepath.Join(t.TempDir(), "join.json")
	if err := os.WriteFile(path, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNativeJoinConstructorTable(path); err == nil {
		t.Fatal("shifted source offset was accepted")
	}
}

func TestNativeJoinConstructorClearsItemFlagsAndTransientOverResidual(t *testing.T) {
	// 0x1138B：有物品的格旗標寫 0；0x113C9：+0x22..+0x27 清零。第十章 r4 酒店存檔 id 11 的
	// +0x0E 殘值 0xF8 被寫成 0。
	table, err := LoadNativeJoinConstructorTable(filepath.Join("..", "..", "assets", "data", "native_join_constructor.json"))
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(filepath.Join("..", "..", "assets", "data", "native_item_effect_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	var residual fdsave.PersistentRecord
	for i := range residual.Raw {
		residual.Raw[i] = 0xf8
	}
	record, err := table.MaterializePersistentRecordOn(residual, 11, itemRows)
	if err != nil {
		t.Fatal(err)
	}
	raw := record.Raw
	for slot := 0; slot < 4; slot++ {
		cell := 0x0e + slot*2
		want := byte(0)
		if raw[cell+1] == 0xff {
			want = 0x80
		}
		if raw[cell] != want {
			t.Fatalf("slot %d flag=%#x item=%#x want %#x", slot, raw[cell], raw[cell+1], want)
		}
	}
	if !bytes.Equal(raw[0x22:0x28], make([]byte, 6)) {
		t.Fatalf("transient bytes kept residual: % x", raw[0x22:0x28])
	}
}
