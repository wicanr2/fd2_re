package campaign

import (
	"encoding/binary"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func completeNativeCurrentSaveUnit() *battle.Unit {
	return &battle.Unit{
		X: 7, Y: 8, OnField: true, Acted: true,
		NativeMapPresentation:    battle.NativeMapPresentationState{X: 7, Y: 8, Pose: 2, Motion: 3},
		HasNativeMapPresentation: true,
		BattleFig:                9, HasBattleFig: true,
		MapSelectorKey: 9, HasMapSelectorKey: true,
		MapSelectorSlot: 0, HasMapSelectorSlot: true,
		NativeRecordByte5: 0x80, HasNativeRecordByte5: true,
		NativeRecordByte6: 2, HasNativeRecordByte6: true,
		NativeRecordByte8: 4, HasNativeRecordByte8: true,
		NativeRecordRace: 1, HasNativeRecordRace: true,
		NativeRecordClass: 2, HasNativeRecordClass: true,
		NativeRecordByte34: 0x81, HasNativeRecordByte34: true,
		NativeRecordByte35: 5, HasNativeRecordByte35: true,
		NativeRecordByte36: 6, HasNativeRecordByte36: true,
		NativeRecordByte3D: 7, HasNativeRecordByte3D: true,
		NativeRecordDeathEffect: [3]byte{8, 9, 10}, HasNativeRecordDeathEffect: true,
		InventorySlots:       []int{1, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags: []int{0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
		NativeCommandMask:    [5]byte{1, 2, 3, 4, 5}, NativeTransient: [6]byte{6, 7, 8, 9, 10, 11},
		Lv: 12, MV: 5, Exp: 13, DX: 14,
		HP: 15, MaxHP: 16, MP: 17, MaxMP: 18, AP: 19, DP: 20, HIT: 21, EV: 22,
	}
}

func TestBuildNativeCurrentRuntimeRecordsOverlaysLiveFieldsAndPreservesOpaque(t *testing.T) {
	unit := completeNativeCurrentSaveUnit()
	state := &battle.State{
		Units: []*battle.Unit{unit}, HasNativeRuntimeUnitProjection: true,
		NativeRuntimeRecords: []battle.NativeRuntimeRecordState{{SelectorKey: 9, SelectorSlot: 0}},
	}
	for index := range state.NativeRuntimeRecords[0].Raw {
		state.NativeRuntimeRecords[0].Raw[index] = 0xcc
	}
	state.NativeRuntimeRecords[0].Raw[7] = 9
	records, err := BuildNativeCurrentRuntimeRecords(state)
	if err != nil {
		t.Fatal(err)
	}
	raw := records[0].Raw
	if raw[0] != 7 || raw[1] != 8 || raw[3] != 2 || raw[4] != 3 || raw[5] != 0x80 ||
		raw[6] != 2 || raw[7] != 9 || raw[8] != 4 || raw[0x31] != 8 || raw[0x36] != 6 || raw[0x3d] != 7 {
		t.Fatalf("live raw fields=%x", raw[:0x3e])
	}
	if raw[2] != 0xcc || raw[9] != 0xcc || raw[0x28] != 0xcc || raw[0x37] != 0xcc {
		t.Fatal("opaque baseline bytes were not preserved")
	}
	if got := binary.LittleEndian.Uint16(raw[0x40:]); got != 15 {
		t.Fatalf("HP=%d want 15", got)
	}
	unit.HP = 99
	if binary.LittleEndian.Uint16(raw[0x40:]) != 15 {
		t.Fatal("returned record aliases Unit")
	}
}

func TestBuildNativeCurrentRuntimeRecordsFailsClosedOnTopologyOrLiveMismatch(t *testing.T) {
	unit := completeNativeCurrentSaveUnit()
	base := battle.NativeRuntimeRecordState{SelectorKey: 9, SelectorSlot: 0}
	base.Raw[7] = 9
	state := &battle.State{Units: []*battle.Unit{unit}, NativeRuntimeRecords: []battle.NativeRuntimeRecordState{base}}
	if _, err := BuildNativeCurrentRuntimeRecords(state); err == nil {
		t.Fatal("non-CONTINUE roster unexpectedly accepted")
	}
	state.HasNativeRuntimeUnitProjection = true
	unit.X++
	if _, err := BuildNativeCurrentRuntimeRecords(state); err == nil {
		t.Fatal("stale map presentation unexpectedly accepted")
	}
	unit.X--
	unit.Acted = false
	if _, err := BuildNativeCurrentRuntimeRecords(state); err == nil {
		t.Fatal("raw acted mismatch unexpectedly accepted")
	}
}
