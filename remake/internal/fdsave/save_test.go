package fdsave

import (
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
)

func TestNativeEnvelopeRoundTripAndChecksum(t *testing.T) {
	plain := make([]byte, FileSize)
	for i := range plain {
		plain[i] = byte(i*37 + 11)
	}
	stored, err := Encode(plain)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) == string(plain) {
		t.Fatal("envelope did not transform input")
	}
	got, err := Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]byte(nil), plain...)
	sum, _ := Checksum(want)
	binary.LittleEndian.PutUint32(want[ChecksumOff:], sum)
	if string(got) != string(want) {
		t.Fatal("round-trip plaintext mismatch")
	}
}

func TestDecodeRejectsTamper(t *testing.T) {
	stored, err := Encode(make([]byte, FileSize))
	if err != nil {
		t.Fatal(err)
	}
	stored[0x123] ^= 1
	if _, err := Decode(stored); err == nil {
		t.Fatal("tampered save accepted")
	}
}

func TestSlotBoundsAndVerifiedMetadata(t *testing.T) {
	if start, end, err := SlotBounds(0); err != nil || start != 0x312b || end != 0x3b53 {
		t.Fatalf("slot0 bounds=%#x..%#x err=%v", start, end, err)
	}
	if _, _, err := SlotBounds(4); err == nil {
		t.Fatal("out-of-range slot accepted")
	}
	plain := make([]byte, FileSize)
	start, _, _ := SlotBounds(2)
	plain[start+RosterSize] = 0xff
	plain[start+RosterSize+1] = 7
	binary.LittleEndian.PutUint32(plain[start+RosterSize+2:], 0x12345678)
	copy(plain[start+RosterSize+6:], []byte{0xa1, 0xa2, 0xa3, 0xa4})
	meta, err := ReadVerifiedMetadata(plain, 2)
	if err != nil || meta != (VerifiedMetadata{
		Chapter: 0xff, RosterCount: 7, Currency: 0x12345678,
		HUDGateA: 0xa1, Raw53AF9: 0xa2, Raw51E61: 0xa3, Raw51E62: 0xa4,
	}) {
		t.Fatalf("metadata=%#v err=%v", meta, err)
	}
}

func TestWriteSlotPreservesOtherSlotsAndUsesOpaqueRegions(t *testing.T) {
	plain := make([]byte, FileSize)
	for i := range plain {
		plain[i] = byte(i * 13)
	}
	replacement := Slot{Roster: make([]byte, RosterSize), Metadata: make([]byte, MetadataSize)}
	for i := range replacement.Roster {
		replacement.Roster[i] = 0xa5
	}
	for i := range replacement.Metadata {
		replacement.Metadata[i] = 0x5a
	}
	got, err := WriteSlot(plain, 2, replacement)
	if err != nil {
		t.Fatal(err)
	}
	start, end, _ := SlotBounds(2)
	if got[start] != 0xa5 || got[end-1] != 0x5a {
		t.Fatal("replacement did not reach requested raw slot")
	}
	otherStart, _, _ := SlotBounds(1)
	if got[otherStart] != plain[otherStart] {
		t.Fatal("write changed a different slot")
	}
	if plain[start] == got[start] {
		t.Fatal("write unexpectedly mutated caller image")
	}
	if _, err := WriteSlot(plain, 0, Slot{Roster: []byte{1}, Metadata: make([]byte, MetadataSize)}); err == nil {
		t.Fatal("short roster unexpectedly accepted")
	}
}

func TestInspectChapterSlotPreservesFixedRosterAndOpaqueMetadata(t *testing.T) {
	plain := make([]byte, FileSize)
	start, _, err := SlotBounds(1)
	if err != nil {
		t.Fatal(err)
	}
	for record := 0; record < RosterUnits; record++ {
		for offset := 0; offset < UnitSize; offset++ {
			plain[start+record*UnitSize+offset] = byte(record*17 + offset)
		}
	}
	metadata := start + RosterSize
	plain[metadata] = 3
	plain[metadata+1] = 2
	binary.LittleEndian.PutUint32(plain[metadata+2:], 123456)
	copy(plain[metadata+6:], []byte{0xa1, 0xa2, 0xa3, 0xa4})
	plain[metadata+MetadataSize-1] = 0xef

	got, err := InspectChapterSlot(plain, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Slot != 1 ||
		got.Verified != (VerifiedMetadata{
			Chapter: 3, RosterCount: 2, Currency: 123456,
			HUDGateA: 0xa1, Raw53AF9: 0xa2, Raw51E61: 0xa3, Raw51E62: 0xa4,
		}) {
		t.Fatalf("snapshot header=%#v", got)
	}
	if got.Metadata[6] != 0xa1 || got.Metadata[MetadataSize-1] != 0xef {
		t.Fatal("opaque metadata was not preserved")
	}
	if got.Records[31].Raw[49] != byte((31*17+49)%256) {
		t.Fatal("fixed-capacity roster tail was not preserved")
	}
	active := got.ActiveRecords()
	if len(active) != 2 || active[1].Raw[8] != byte(17+8) {
		t.Fatalf("active records=%#v", active)
	}
	active[0].Raw[0] ^= 0xff
	if active[0].Raw[0] == got.Records[0].Raw[0] {
		t.Fatal("active records unexpectedly alias the snapshot")
	}
}

func TestInspectChapterSlotFailsClosedForEmptyAndOversizedCount(t *testing.T) {
	plain := make([]byte, FileSize)
	start, _, err := SlotBounds(0)
	if err != nil {
		t.Fatal(err)
	}
	metadata := start + RosterSize
	plain[metadata] = 0xff
	if _, err := InspectChapterSlot(plain, 0); !errors.Is(err, ErrEmptyChapterSlot) {
		t.Fatalf("empty slot error=%v", err)
	}
	plain[metadata] = 1
	plain[metadata+1] = RosterUnits + 1
	if _, err := InspectChapterSlot(plain, 0); err == nil {
		t.Fatal("oversized roster count unexpectedly accepted")
	}
}

func TestPersistentRecordViewUsesProvenOffsetsAndSignedWords(t *testing.T) {
	var record PersistentRecord
	record.Raw[5] = 0x81
	record.Raw[6] = 2
	record.Raw[7] = 0x34
	record.Raw[8] = 9
	for slot := 0; slot < 8; slot++ {
		record.Raw[0x0a+slot*2] = byte(0x40 + slot)
		record.Raw[0x0b+slot*2] = byte(0x20 + slot)
	}
	copy(record.Raw[0x1a:0x1f], []byte{1, 2, 3, 4, 5})
	record.Raw[0x1f] = 6
	record.Raw[0x20] = 7
	record.Raw[0x21] = 8
	copy(record.Raw[0x22:0x28], []byte{9, 10, 11, 12, 13, 14})
	record.Raw[0x34], record.Raw[0x35], record.Raw[0x36] = 0x81, 0x22, 0x33
	record.Raw[0x3b] = 15
	record.Raw[0x3c] = 16
	for offset, value := range map[int]int16{
		0x37: -17,
		0x39: 18,
		0x3e: -19,
		0x40: 20,
		0x42: 21,
		0x44: 22,
		0x46: 23,
		0x48: 24,
		0x4a: 25,
		0x4c: -26,
		0x4e: 27,
	} {
		binary.LittleEndian.PutUint16(record.Raw[offset:], uint16(value))
	}

	got := record.View()
	if got.RawByte5 != 0x81 || got.RawCamp != 2 ||
		got.RawPresentationKey != 0x34 || got.RawIdentity != 9 ||
		got.Inventory[7] != (PersistentInventoryCell{Flags: 0x47, ItemID: 0x27}) ||
		got.CommandMask != ([5]byte{1, 2, 3, 4, 5}) ||
		got.Race != 6 || got.Class != 7 || got.Level != 8 ||
		got.Transient != ([6]byte{9, 10, 11, 12, 13, 14}) ||
		got.RawByte34 != 0x81 || got.RawByte35 != 0x22 || got.RawByte36 != 0x33 ||
		got.BaseAP != -17 || got.BaseDP != 18 ||
		got.Movement != 15 || got.Experience != 16 ||
		got.DX != -19 || got.HP != 20 || got.MaxHP != 21 ||
		got.MP != 22 || got.MaxMP != 23 || got.AP != 24 ||
		got.DP != 25 || got.HIT != -26 || got.EV != 27 {
		t.Fatalf("persistent view=%#v", got)
	}
	if record.Raw[0x37] == 0 {
		t.Fatal("view unexpectedly mutated the raw record")
	}
}

func TestInspectCurrentSnapshotUsesIDA10010Offsets(t *testing.T) {
	plain := make([]byte, FileSize)
	plain[CurrentFieldControlOffset] = 0x11
	plain[CurrentFieldControlOffset+CurrentFieldControlSize-1] = 0x22
	plain[CurrentNativeEventStateOffset] = 0x33
	plain[CurrentNativeEventStateOffset+CurrentNativeEventStateSize-1] = 0x44
	header := plain[CurrentRuntimeHeaderOffset : CurrentRuntimeHeaderOffset+CurrentRuntimeHeaderSize]
	copy(header, []byte{
		3, 2, 7,
		1, 13, 8, 17, 7, 4,
		1,
		0x78, 0x56, 0x34, 0x12,
		0xaa, 1, 0xbb, 0xcc,
	})
	plain[CurrentPersistentRosterOffset+8] = 9
	plain[CurrentRuntimeRosterOffset+8] = 4
	plain[CurrentRuntimeRosterOffset+UnitSize+8] = 30

	got, err := InspectCurrentSnapshot(plain)
	if err != nil {
		t.Fatal(err)
	}
	if got.Header.TurnCounter != 3 || got.Header.RuntimeCount != 2 ||
		got.Header.PersistentCount != 1 || got.Header.Chapter != 7 ||
		got.Header.Currency != 0x12345678 || got.Header.HUDGateA != 1 {
		t.Fatalf("current header=%#v", got.Header)
	}
	if active := got.ActivePersistentRecords(); len(active) != 1 ||
		active[0].View().RawIdentity != 9 {
		t.Fatalf("current persistent records=%#v", active)
	}
	if len(got.RuntimeRecords) != 2 ||
		got.RuntimeRecords[0].View().RawIdentity != 4 ||
		got.RuntimeRecords[1].View().RawIdentity != 30 {
		t.Fatalf("current runtime records=%#v", got.RuntimeRecords)
	}
	if got.NativeFieldControl[0] != 0x11 ||
		got.NativeFieldControl[len(got.NativeFieldControl)-1] != 0x22 ||
		got.NativeEventState[0] != 0x33 ||
		got.NativeEventState[len(got.NativeEventState)-1] != 0x44 {
		t.Fatalf(
			"current raw regions battle=%#x/%#x event-state=%#x/%#x",
			got.NativeFieldControl[0],
			got.NativeFieldControl[len(got.NativeFieldControl)-1],
			got.NativeEventState[0],
			got.NativeEventState[len(got.NativeEventState)-1],
		)
	}
	plain[CurrentFieldControlOffset] = 0
	plain[CurrentNativeEventStateOffset] = 0
	if got.NativeFieldControl[0] != 0x11 || got.NativeEventState[0] != 0x33 {
		t.Fatal("current snapshot raw regions alias caller plaintext")
	}
}

func TestInspectCurrentSnapshotRejectsImpossibleCounts(t *testing.T) {
	plain := make([]byte, FileSize)
	plain[CurrentRuntimeHeaderOffset+1] = RosterUnits*3 + 1
	if _, err := InspectCurrentSnapshot(plain); err == nil {
		t.Fatal("oversized runtime count unexpectedly accepted")
	}
	plain[CurrentRuntimeHeaderOffset+1] = 0
	plain[CurrentRuntimeHeaderOffset+9] = RosterUnits + 1
	if _, err := InspectCurrentSnapshot(plain); err == nil {
		t.Fatal("oversized persistent count unexpectedly accepted")
	}
}

func TestWriteCurrentSnapshotRoundTripPreservesOpaqueAndChapterSlots(t *testing.T) {
	plain := make([]byte, FileSize)
	for index := range plain {
		plain[index] = byte(index*29 + 7)
	}
	snapshot := CurrentSnapshot{
		Header: CurrentRuntimeHeader{
			TurnCounter: 17, RuntimeCount: 2, Chapter: 9,
			CameraX: 3, CameraY: 4, CursorX: 10, CursorY: 11,
			VisibleCursorX: 7, VisibleCursorY: 7, PersistentCount: 1,
			Currency: 0x12345678, Raw53AF9: 1, HUDGateA: 0,
			Raw51E61: 1, Raw51E62: 0,
		},
		RuntimeRecords: make([]PersistentRecord, 2),
	}
	snapshot.NativeFieldControl[0] = 0x31
	snapshot.NativeFieldControl[2] = 1
	snapshot.NativeFieldControl[len(snapshot.NativeFieldControl)-1] = 0x32
	snapshot.PersistentRecords[0].Raw[8] = 9
	snapshot.PersistentRecords[len(snapshot.PersistentRecords)-1].Raw[0x4f] = 0xa5
	snapshot.RuntimeRecords[0].Raw[8] = 4
	snapshot.RuntimeRecords[1].Raw[8] = 5
	snapshot.NativeEventState[0] = 0x61
	snapshot.NativeEventState[len(snapshot.NativeEventState)-1] = 0x62

	beforeSlot := append([]byte(nil), plain[SlotOffset:ChecksumOff]...)
	beforeRuntimeTail := append(
		[]byte(nil),
		plain[CurrentRuntimeRosterOffset+2*UnitSize:CurrentNativeEventStateOffset]...,
	)
	gotPlain, err := WriteCurrentSnapshot(plain, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plain[SlotOffset:ChecksumOff], beforeSlot) {
		t.Fatal("writer mutated caller plaintext")
	}
	if !reflect.DeepEqual(gotPlain[SlotOffset:ChecksumOff], beforeSlot) {
		t.Fatal("writer changed native chapter slots or their opaque metadata")
	}
	if !reflect.DeepEqual(
		gotPlain[CurrentRuntimeRosterOffset+2*UnitSize:CurrentNativeEventStateOffset],
		beforeRuntimeTail,
	) {
		t.Fatal("writer changed unused current-runtime capacity")
	}
	stored, err := Encode(gotPlain)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	got, err := InspectCurrentSnapshot(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, snapshot) {
		t.Fatalf("current snapshot round-trip mismatch\n got=%#v\nwant=%#v", got, snapshot)
	}
}

func TestWriteCurrentSnapshotRejectsMalformedInputWithoutAliasing(t *testing.T) {
	plain := make([]byte, FileSize)
	snapshot := CurrentSnapshot{
		Header:         CurrentRuntimeHeader{RuntimeCount: 2},
		RuntimeRecords: make([]PersistentRecord, 1),
	}
	if _, err := WriteCurrentSnapshot(plain, snapshot); err == nil {
		t.Fatal("runtime count mismatch unexpectedly accepted")
	}
	if plain[CurrentRuntimeHeaderOffset] != 0 {
		t.Fatal("failed write mutated caller plaintext")
	}
	snapshot.Header.RuntimeCount = 1
	snapshot.Header.PersistentCount = RosterUnits + 1
	if _, err := WriteCurrentSnapshot(plain, snapshot); err == nil {
		t.Fatal("oversized persistent count unexpectedly accepted")
	}
	snapshot.Header.PersistentCount = 0
	snapshot.NativeFieldControl[2] = CurrentFieldControlUnitCap + 1
	if _, err := WriteCurrentSnapshot(plain, snapshot); err == nil {
		t.Fatal("oversized field unit count unexpectedly accepted")
	}
}

func validContinueSnapshot() CurrentSnapshot {
	snapshot := CurrentSnapshot{
		Header: CurrentRuntimeHeader{
			RuntimeCount: 3, Chapter: 1,
			CameraX: 1, CameraY: 2,
			CursorX: 8, CursorY: 7,
			VisibleCursorX: 7, VisibleCursorY: 5,
			PersistentCount: 2,
		},
		RuntimeRecords: make([]PersistentRecord, 3),
	}
	snapshot.NativeFieldControl[2] = 3
	snapshot.NativeFieldControl[0] = 4
	snapshot.NativeFieldControl[1] = 5
	snapshot.NativeFieldControl[3] = 6
	snapshot.NativeFieldControl[4] = 7
	snapshot.NativeFieldControl[5] = 8
	snapshot.NativeFieldControl[0x33] = 9
	snapshot.NativeFieldControl[0x34] = 10
	snapshot.NativeFieldControl[0x53] = 11
	binary.LittleEndian.PutUint16(snapshot.NativeFieldControl[0x54:0x56], 0x1234)
	snapshot.NativeFieldControl[CurrentFieldControlUnitOffset] = 12
	snapshot.NativeFieldControl[CurrentFieldControlUnitOffset+2*CurrentFieldControlUnitSize+
		CurrentFieldControlUnitSize-1] = 13
	for i := range snapshot.RuntimeRecords {
		snapshot.RuntimeRecords[i].Raw[0] = byte(3 + i)
		snapshot.RuntimeRecords[i].Raw[1] = byte(4 + i)
		snapshot.RuntimeRecords[i].Raw[3] = byte(i)
		snapshot.RuntimeRecords[i].Raw[4] = byte(i + 1)
	}
	snapshot.RuntimeRecords[0].Raw[2] = 99
	snapshot.RuntimeRecords[0].Raw[7] = 5
	snapshot.RuntimeRecords[1].Raw[2] = 98
	snapshot.RuntimeRecords[1].Raw[7] = 7
	snapshot.RuntimeRecords[2].Raw[2] = 97
	snapshot.RuntimeRecords[2].Raw[7] = 5
	snapshot.PersistentRecords[0].Raw[8] = 9
	snapshot.NativeEventState[12] = 1
	return snapshot
}

func validContinueContext() ContinueRuntimeContext {
	return ContinueRuntimeContext{
		Chapter: 1, FieldWidth: 30, FieldHeight: 20,
		SelectorGroupCount: 16, TitleTimerTick: -123, HasTitleTimerTick: true,
	}
}

func TestBuildContinueRuntimeInputRebuildsSelectorSlotsFromRuntimeOrder(t *testing.T) {
	snapshot := validContinueSnapshot()
	input, err := BuildContinueRuntimeInput(snapshot, validContinueContext())
	if err != nil {
		t.Fatal(err)
	}
	if got := []byte{
		input.RuntimeRecords[0].SelectorSlot,
		input.RuntimeRecords[1].SelectorSlot,
		input.RuntimeRecords[2].SelectorSlot,
	}; !reflect.DeepEqual(got, []byte{0, 1, 0}) {
		t.Fatalf("selector slots=%v, want [0 1 0]", got)
	}
	if input.RuntimeRecords[0].SelectorKey != 5 ||
		input.RuntimeRecords[1].SelectorKey != 7 ||
		input.RuntimeRecords[0].Raw.Raw[2] != 99 {
		t.Fatalf("runtime input lost raw provenance: %#v", input.RuntimeRecords)
	}
	if got := input.MapPresentation; got.OpeningRangeMode != 0 ||
		got.InteractiveRangeMode != 1 || got.HUDGateB != 1 ||
		got.HUDAnchorX != 1 {
		t.Fatalf("CONTINUE map presentation=%#v", got)
	}
	if got := input.MapTimingSeed; got != (ContinueMapTimingSeed{
		SpriteLastTimerTick: -123, TerrainPhaseOverride: -1,
	}) {
		t.Fatalf("CONTINUE map timing seed=%#v", got)
	}
	if got := input.FieldControl; got.RawMapSelector != 4 ||
		got.RawOwnDeployCount != 5 || got.RawUnitCount != 3 ||
		got.TurnEvents[0] != (ContinueTurnEventControl{Turn: 6, EventID: 7, RawCamp: 8}) ||
		got.FieldEvents[0] != (ContinueFieldEventControl{EventID: 9, Selector: 10}) ||
		got.Chests[0] != (ContinueChestControl{RawType: 11, Value: 0x1234}) ||
		len(got.Units) != 3 || got.Units[0].Raw[0] != 12 ||
		got.Units[2].Raw[CurrentFieldControlUnitSize-1] != 13 {
		t.Fatalf("CONTINUE field control=%#v", got)
	}
	wantOwners := []ContinueRuntimeOwner{
		ContinueOwnerPendingGroupBinding,
		ContinueOwnerBattleControllerHandoff,
	}
	if input.ReadyForContinue() ||
		!reflect.DeepEqual(input.UnresolvedOwners, wantOwners) {
		t.Fatalf("CONTINUE readiness=%v owners=%v", input.ReadyForContinue(), input.UnresolvedOwners)
	}

	snapshot.RuntimeRecords[0].Raw[7] = 3
	snapshot.NativeFieldControl[0] = 0xff
	snapshot.NativeEventState[12] = 0
	snapshot.PersistentRecords[0].Raw[8] = 0
	if input.RuntimeRecords[0].SelectorKey != 5 ||
		input.NativeFieldControl[0] != 4 ||
		input.FieldControl.RawMapSelector != 4 ||
		input.NativeEventState[12] != 1 ||
		input.PersistentRecords[0].Raw[8] != 9 {
		t.Fatal("CONTINUE runtime input aliases caller snapshot")
	}
}

func TestBuildContinueRuntimeInputAppliesTitleHUDAnchorBranches(t *testing.T) {
	tests := []struct {
		name               string
		visibleX, visibleY byte
		wantAnchor         int
	}{
		{name: "left lower region", visibleX: 2, visibleY: 6, wantAnchor: 0xf2},
		{name: "left boundary retained", visibleX: 3, visibleY: 6, wantAnchor: 1},
		{name: "middle lower region", visibleX: 7, visibleY: 6, wantAnchor: 1},
		{name: "right boundary retained", visibleX: 9, visibleY: 6, wantAnchor: 1},
		{name: "right lower region", visibleX: 10, visibleY: 6, wantAnchor: 1},
		{name: "upper region", visibleX: 2, visibleY: 5, wantAnchor: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := validContinueSnapshot()
			snapshot.Header.VisibleCursorX = tt.visibleX
			snapshot.Header.VisibleCursorY = tt.visibleY
			snapshot.Header.CursorX = snapshot.Header.CameraX + tt.visibleX
			snapshot.Header.CursorY = snapshot.Header.CameraY + tt.visibleY
			input, err := BuildContinueRuntimeInput(snapshot, validContinueContext())
			if err != nil {
				t.Fatal(err)
			}
			if input.MapPresentation.HUDAnchorX != tt.wantAnchor {
				t.Fatalf(
					"HUD anchor=%#x, want %#x",
					input.MapPresentation.HUDAnchorX, tt.wantAnchor,
				)
			}
		})
	}
}

func TestDecodeContinueFieldControlUsesExclusiveNativeCount(t *testing.T) {
	var raw [CurrentFieldControlSize]byte
	raw[2] = 30
	lastLive := CurrentFieldControlUnitOffset + 29*CurrentFieldControlUnitSize
	firstPadding := CurrentFieldControlUnitOffset + 30*CurrentFieldControlUnitSize
	raw[lastLive] = 0x1d
	raw[firstPadding] = 0xee

	got := decodeContinueFieldControl(raw)
	if got.RawUnitCount != 30 || len(got.Units) != 30 {
		t.Fatalf("field control count=%d units=%d", got.RawUnitCount, len(got.Units))
	}
	if got.Units[29].Raw[0] != 0x1d {
		t.Fatal("last live row was not decoded")
	}
	for _, unit := range got.Units {
		if unit.Raw[0] == 0xee {
			t.Fatal("resource padding was exposed as a live unit")
		}
	}
}

func TestBuildContinueRuntimeInputRejectsMalformedPreconditionsAtomically(t *testing.T) {
	tests := map[string]func(*CurrentSnapshot, *ContinueRuntimeContext){
		"chapter mismatch": func(_ *CurrentSnapshot, context *ContinueRuntimeContext) {
			context.Chapter = 2
		},
		"small field": func(_ *CurrentSnapshot, context *ContinueRuntimeContext) {
			context.FieldWidth = 12
		},
		"selector group count": func(_ *CurrentSnapshot, context *ContinueRuntimeContext) {
			context.SelectorGroupCount = 0
		},
		"missing title timer seed": func(_ *CurrentSnapshot, context *ContinueRuntimeContext) {
			context.HasTitleTimerTick = false
		},
		"invalid title timer seed": func(_ *CurrentSnapshot, context *ContinueRuntimeContext) {
			context.TitleTimerTick = 0x8000
		},
		"runtime count": func(snapshot *CurrentSnapshot, _ *ContinueRuntimeContext) {
			snapshot.Header.RuntimeCount++
		},
		"persistent count": func(snapshot *CurrentSnapshot, _ *ContinueRuntimeContext) {
			snapshot.Header.PersistentCount = RosterUnits + 1
		},
		"field control count": func(snapshot *CurrentSnapshot, _ *ContinueRuntimeContext) {
			snapshot.NativeFieldControl[2] = CurrentFieldControlUnitCap + 1
		},
		"camera identity": func(snapshot *CurrentSnapshot, _ *ContinueRuntimeContext) {
			snapshot.Header.VisibleCursorX++
		},
		"FDICON key": func(snapshot *CurrentSnapshot, _ *ContinueRuntimeContext) {
			snapshot.RuntimeRecords[1].Raw[7] = 16
		},
		"active coordinate": func(snapshot *CurrentSnapshot, context *ContinueRuntimeContext) {
			snapshot.RuntimeRecords[1].Raw[0] = byte(context.FieldWidth)
		},
		"active pose": func(snapshot *CurrentSnapshot, _ *ContinueRuntimeContext) {
			snapshot.RuntimeRecords[1].Raw[3] = 4
		},
		"active motion": func(snapshot *CurrentSnapshot, _ *ContinueRuntimeContext) {
			snapshot.RuntimeRecords[1].Raw[4] = 7
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			snapshot, context := validContinueSnapshot(), validContinueContext()
			mutate(&snapshot, &context)
			got, err := BuildContinueRuntimeInput(snapshot, context)
			if err == nil || got.RuntimeRecords != nil || got.UnresolvedOwners != nil {
				t.Fatalf("malformed CONTINUE input=%#v err=%v", got, err)
			}
		})
	}
}

func TestBuildContinueRuntimeInputAllowsInactivePresentationGarbage(t *testing.T) {
	snapshot := validContinueSnapshot()
	snapshot.RuntimeRecords[1].Raw[5] = 1
	snapshot.RuntimeRecords[1].Raw[0] = 0xff
	snapshot.RuntimeRecords[1].Raw[1] = 0xff
	snapshot.RuntimeRecords[1].Raw[3] = 0xff
	snapshot.RuntimeRecords[1].Raw[4] = 0xff
	if _, err := BuildContinueRuntimeInput(snapshot, validContinueContext()); err != nil {
		t.Fatalf("inactive record presentation should stay raw: %v", err)
	}
}
