package battle

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestNativeCommand23RelocationUsesSpellGatesAndIsAtomic(t *testing.T) {
	book, err := LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, mp := range []uint16{19, 30} {
		records := make([]byte, 3*nativeRecordSize)
		records[0x08] = 7 // The spell handler does not require item caster24.
		binary.LittleEndian.PutUint16(records[0x44:], mp)
		target := records[nativeRecordSize : 2*nativeRecordSize]
		target[0], target[1], target[0x20], target[0x21] = 1, 0, 9, 7
		before := append([]byte(nil), records...)
		got, err := ApplyNativeCommand23Relocation(records, 0, []byte{1, 2}, 8, 7, book)
		if mp < 20 {
			if err == nil || !bytes.Equal(before, records) {
				t.Fatal("insufficient spell MP changed records")
			}
			continue
		}
		if err != nil || got.AccumulatorDelta != 370 || got.RemainingMP != 10 || target[0] != 8 || target[1] != 7 || !bytes.Equal(records[2*nativeRecordSize:], before[2*nativeRecordSize:]) {
			t.Fatalf("spell relocation: %+v error%v", got, err)
		}
	}
}

func nativeRelocationCommandBook() []NativeCommandRecord {
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[23].MPCost = 20
	return book
}

func TestNativeType23RelocatesFirstTargetAndWrapsMP(t *testing.T) {
	records := make([]byte, 3*nativeRecordSize)
	actor := records[:nativeRecordSize]
	actor[0x08] = 24
	binary.LittleEndian.PutUint16(actor[0x44:], 10)
	binary.LittleEndian.PutUint16(actor[0x46:], 20)
	target := records[nativeRecordSize : 2*nativeRecordSize]
	target[0], target[1], target[0x20], target[0x21] = 4, 5, 9, 7
	other := records[2*nativeRecordSize:]
	other[0], other[1] = 8, 9
	route, ok := NativeItemRelocationRouteForType(23, 1)
	if !ok || route.CommandID != 23 || route.Presentation != 0x22253 || route.ConsumesSource {
		t.Fatalf("route = %#v, %v", route, ok)
	}
	got, err := ApplyNativeItemRelocation(records, 0, []byte{1, 2}, 12, 13, route, nativeRelocationCommandBook())
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetUnit != 1 || got.FromX != 4 || got.FromY != 5 ||
		got.RemainingMP != 0xfff6 || got.AccumulatorDelta != 370 {
		t.Fatalf("result = %#v", got)
	}
	if binary.LittleEndian.Uint16(actor[0x44:]) != 0xfff6 ||
		target[0] != 12 || target[1] != 13 || other[0] != 8 || other[1] != 9 {
		t.Fatalf("actorMP=%#x target=%d,%d other=%d,%d",
			binary.LittleEndian.Uint16(actor[0x44:]), target[0], target[1], other[0], other[1])
	}
}

func TestNativeType23PreflightsIdentityAndMaxMP(t *testing.T) {
	for _, tc := range []struct {
		name       string
		identity   byte
		maxMP      uint16
		wantTarget int
	}{
		{"identity", 23, 20, 1},
		{"max_mp", 24, 19, 1},
		{"target", 24, 20, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records := make([]byte, 2*nativeRecordSize)
			records[0x08] = tc.identity
			binary.LittleEndian.PutUint16(records[0x44:], 30)
			binary.LittleEndian.PutUint16(records[0x46:], tc.maxMP)
			route, _ := NativeItemRelocationRouteForType(23, 1)
			if _, err := ApplyNativeItemRelocation(
				records, 0, []byte{byte(tc.wantTarget)}, 9, 10, route, nativeRelocationCommandBook(),
			); err == nil {
				t.Fatal("invalid relocation unexpectedly accepted")
			}
			if binary.LittleEndian.Uint16(records[0x44:]) != 30 {
				t.Fatal("invalid relocation mutated MP")
			}
		})
	}
}
