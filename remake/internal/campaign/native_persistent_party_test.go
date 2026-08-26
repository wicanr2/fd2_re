package campaign

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func TestMaterializeNativePersistentPartyOriginalSnapshot(t *testing.T) {
	path := os.Getenv("FD2_NATIVE_SAVE_FIXTURE")
	if path == "" {
		t.Skip("set FD2_NATIVE_SAVE_FIXTURE to a user-provided original FD2.SAV")
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadNativeCharacterCatalog(
		"../../assets/data/native_character_catalog.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	records := snapshot.ActivePersistentRecords()
	got := make([]string, len(records))
	for index, record := range records {
		unit, err := MaterializeNativePersistentPartyRecord(record, catalog)
		if err != nil {
			t.Fatalf("record %d: %v", index, err)
		}
		got[index] = unit.Name
	}
	want := []string{"索爾", "悠妮", "亞雷斯", "蓋亞"}
	if len(got) != len(want) {
		t.Fatalf("party=%v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("party=%v, want %v", got, want)
		}
	}
}

func TestNativeCharacterCatalogAssetIsComplete(t *testing.T) {
	catalog, err := LoadNativeCharacterCatalog(
		"../../assets/data/native_character_catalog.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	for identity, want := range map[int]string{
		0: "索爾", 4: "亞雷斯", 9: "悠妮", 30: "蓋亞",
	} {
		if got := catalog.identityNames[identity]; got != want {
			t.Fatalf("identity %d=%q, want %q", identity, got, want)
		}
	}
	if got := catalog.classNames[27]; got != "　　" {
		t.Fatalf("class 27=%q, want two full-width spaces", got)
	}
	if got := catalog.classNames[28]; got != "？？？" {
		t.Fatalf("class 28=%q, want %q", got, "？？？")
	}
}

func TestMaterializeNativePersistentPartyRecordPreservesProvenFields(t *testing.T) {
	catalog, err := LoadNativeCharacterCatalog(
		"../../assets/data/native_character_catalog.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	var record fdsave.PersistentRecord
	record.Raw[5], record.Raw[6], record.Raw[7], record.Raw[8] = 0x81, 2, 0x44, 9
	for slot := 0; slot < 8; slot++ {
		record.Raw[0x0a+slot*2] = 0x80
		record.Raw[0x0b+slot*2] = 0xff
	}
	record.Raw[0x0a], record.Raw[0x0b] = 0x40, 0x12
	copy(record.Raw[0x1a:0x1f], []byte{1, 2, 3, 4, 5})
	record.Raw[0x1f], record.Raw[0x20], record.Raw[0x21] = 1, 5, 7
	copy(record.Raw[0x22:0x28], []byte{6, 7, 8, 9, 10, 11})
	record.Raw[0x34], record.Raw[0x35], record.Raw[0x36] = 0x81, 0x22, 0x33
	record.Raw[0x3b], record.Raw[0x3c] = 6, 42
	for offset, value := range map[int]int16{
		0x37: 10, 0x39: 11, 0x3e: 12,
		0x40: 13, 0x42: 14, 0x44: 15, 0x46: 16,
		0x48: 17, 0x4a: 18, 0x4c: 19, 0x4e: 20,
	} {
		binary.LittleEndian.PutUint16(record.Raw[offset:], uint16(value))
	}

	unit, err := MaterializeNativePersistentPartyRecord(record, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if unit.Name != "悠妮" || unit.ClsName != "法師" ||
		unit.NativeIdentity != 9 || unit.ClassID != 5 ||
		!unit.HasMapSelectorKey || unit.MapSelectorKey != 0x44 ||
		unit.HP != 13 || unit.MaxHP != 14 || unit.AP != 17 ||
		unit.BaseAP != 10 || unit.BaseHIT != 12 ||
		!unit.HasNativeRecordWord42 || unit.NativeRecordWord42 != 14 ||
		!unit.HasNativeRecordWord46 || unit.NativeRecordWord46 != 16 ||
		!unit.HasNativeRecordByte34 || unit.NativeRecordByte34 != 0x81 ||
		!unit.HasNativeRecordByte35 || unit.NativeRecordByte35 != 0x22 ||
		!unit.HasNativeRecordByte36 || unit.NativeRecordByte36 != 0x33 ||
		unit.DX != 12 || unit.Exp != 42 ||
		len(unit.Inventory) != 1 || unit.Inventory[0] != 0x12 ||
		!unit.Equipped[0] {
		t.Fatalf("materialized unit=%#v", unit)
	}
	if unit.Portrait != 0 || unit.Fig != 0 || unit.OnField ||
		unit.HasBattleFig {
		t.Fatalf("unproven presentation fields were materialized: %#v", unit)
	}
}

func TestMaterializeNativePersistentPartyRecordRejectsUnknownClass(t *testing.T) {
	catalog, err := LoadNativeCharacterCatalog(
		"../../assets/data/native_character_catalog.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	var record fdsave.PersistentRecord
	record.Raw[8] = 0
	record.Raw[0x20] = 0xff
	if _, err := MaterializeNativePersistentPartyRecord(record, catalog); err == nil {
		t.Fatal("unknown class unexpectedly accepted")
	}
}

func TestNativeCharacterCatalogRejectsReorderedIdentity(t *testing.T) {
	catalog, err := LoadNativeCharacterCatalog(
		"../../assets/data/native_character_catalog.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	catalog.Identities[9].ID = 10
	if err := catalog.validate(); err == nil {
		t.Fatal("reordered identity unexpectedly accepted")
	}
}

func TestNativeCharacterCatalogRejectsWrongEXEVersion(t *testing.T) {
	catalog, err := LoadNativeCharacterCatalog(
		"../../assets/data/native_character_catalog.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	catalog.Source.EXESHA256 = "wrong"
	if err := catalog.validate(); err == nil {
		t.Fatal("wrong EXE version unexpectedly accepted")
	}
}
