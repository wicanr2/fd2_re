package battle

import (
	"encoding/binary"
	"testing"
)

func nativeAI1567EFixture() ([]byte, []byte, []byte) {
	records := make([]byte, 3*nativeRecordSize)
	records[0] = 0
	records[1] = 0
	records[6] = 2
	records[0x0a] = 0
	records[0x0b] = 0
	for slot := 1; slot < 8; slot++ {
		records[0x0a+slot*2] = 0x80
		records[0x0b+slot*2] = 0xff
	}
	records[nativeRecordSize] = 1
	records[nativeRecordSize+1] = 0
	records[nativeRecordSize+6] = 0
	binary.LittleEndian.PutUint16(records[nativeRecordSize+0x40:], 30)
	binary.LittleEndian.PutUint16(records[nativeRecordSize+0x42:], 90)
	records[2*nativeRecordSize] = 2
	records[2*nativeRecordSize+1] = 0
	records[2*nativeRecordSize+6] = 1

	itemRows := make([]byte, NativeItemEffectRowSize)
	itemRows[0x0d] = 5
	itemRows[0x10] = 1
	itemRows[0x11] = 2
	itemRows[0x12] = 0
	return records, itemRows, make([]byte, 3)
}

func TestScoreNativeAI1567ELowCommandKeepsFirstStrictWinner(t *testing.T) {
	records, itemRows, flags := nativeAI1567EFixture()
	records[0x0c] = 0
	records[0x0d] = 0
	got, err := ScoreNativeAI1567E(
		3, 1, records, 3, 0, 0, itemRows, nil, flags,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasPositiveWinner || got.MaxScore != 8 ||
		got.X != 1 || got.Y != 0 || got.InventorySlot != 0 || got.ItemID != 0 {
		t.Fatalf("low-command result=%+v", got)
	}
}

func TestScoreNativeAI1567EHighCommandUsesActorToDestinationLine(t *testing.T) {
	records, _, flags := nativeAI1567EFixture()
	records[nativeRecordSize] = 2
	records[nativeRecordSize+6] = 2
	binary.LittleEndian.PutUint16(records[nativeRecordSize+0x40:], 20)
	itemRows := make([]byte, NativeItemEffectRowSize)
	itemRows[0x0d] = 0x18
	itemRows[0x0e] = 20
	itemRows[0x10] = 0x12
	got, err := ScoreNativeAI1567E(
		3, 1, records, 3, 0, 0, itemRows, nil, flags,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasPositiveWinner || got.MaxScore != 0x12 ||
		got.X != 1 || got.Y != 0 {
		t.Fatalf("high-command result=%+v", got)
	}
}

func TestScoreNativeAI1567EHighCommandRejectsRawZeroAndBlockedDirections(t *testing.T) {
	records, _, flags := nativeAI1567EFixture()
	records[nativeRecordSize] = 2
	records[nativeRecordSize+6] = 0
	records[2*nativeRecordSize+5] = 1
	itemRows := make([]byte, NativeItemEffectRowSize)
	itemRows[0x0d], itemRows[0x0e], itemRows[0x10] = 0x18, 20, 0x12
	for _, targetCamp := range []byte{0, 1, 2} {
		records[nativeRecordSize+6] = targetCamp
		flags[1] = NativeCommandGridBlocked
		got, err := ScoreNativeAI1567E(3, 1, records, 3, 0, 0, itemRows, nil, flags)
		if err != nil || got.HasPositiveWinner {
			t.Fatalf("blocked direction camp=%d got=%+v err=%v", targetCamp, got, err)
		}
		flags[1] = 0
		got, err = ScoreNativeAI1567E(3, 1, records, 3, 0, 0, itemRows, nil, flags)
		if err != nil || got.HasPositiveWinner != (targetCamp != 0) {
			t.Fatalf("selector zero camp=%d got=%+v err=%v", targetCamp, got, err)
		}
	}
}

func TestScoreNativeAI1567ESelectorOneKeepsRowTargetCode(t *testing.T) {
	records, itemRows, flags := nativeAI1567EFixture()
	itemRows[0x10] = 2
	itemRows[0x11] = 2
	binary.LittleEndian.PutUint16(records[2*nativeRecordSize+0x40:], 30)
	binary.LittleEndian.PutUint16(records[2*nativeRecordSize+0x42:], 90)
	got, err := ScoreNativeAI1567E(
		3, 1, records, 3, 0, 1, itemRows, nil, flags,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasPositiveWinner || got.MaxScore != 8 ||
		got.X != 2 || got.Y != 0 {
		t.Fatalf("selector-one result=%+v", got)
	}
}

func TestScoreNativeAI1567EFollowsCountSizedRawPrefix(t *testing.T) {
	records, itemRows, flags := nativeAI1567EFixture()
	records[0x0a] = 0x80
	records[0x0b] = 0xff
	records[0x0c] = 0
	records[0x0d] = 0
	if _, err := ScoreNativeAI1567E(
		3, 1, records, 3, 0, 0, itemRows, nil, flags,
	); err == nil {
		t.Fatal("native count-sized scan accepted unavailable stale item row")
	}
}

func TestMap0AssetsAnchorNativeAI1567EWithTrackedItem79(t *testing.T) {
	st, err := Load("../../assets/maps/map0/map0_units.json")
	if err != nil {
		t.Fatal(err)
	}
	actor := 23
	st.Units[actor].InventorySlots = []int{79, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	st.Units[actor].NativeInventoryFlags = []int{0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	for _, unit := range st.Units {
		if err := unit.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
	}
	records, err := NativeAIScoringRecords(st.Units)
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	book, err := LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ScoreNativeAI1567E(
		st.W, st.H, records, len(st.Units), actor, 0,
		itemRows, book, nativeCompositionBaseFlagsForTest(t, st),
	)
	if err != nil {
		t.Fatal(err)
	}
	// The former score8/(19,15) assertion selected a raw-zero ally.
	// 0x14AE4..0x14AF4 excludes that record for selector zero.
	if got.HasPositiveWinner || got.MaxScore != 0 || len(got.TargetIndices) != 0 {
		t.Fatalf("map0 item79 result=%+v", got)
	}
}
