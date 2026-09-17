package battle

import (
	"reflect"
	"testing"
)

func TestScoreNativeAI1598ARejectsAllZeroWinner(t *testing.T) {
	unit := completeNativeAIScoringUnit()
	unit.NativeCommandMask = [5]byte{0, 0, 0, 1, 0}
	unit.NativeMapPresentation.X = 0
	unit.NativeMapPresentation.Y = 0
	unit.MP = 255
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for id := range book {
		book[id] = NativeCommandRecord{ID: id}
	}
	book[24] = NativeCommandRecord{
		ID: 24, SelectionMode: 0, EffectMode: 0, TargetCode: 1,
	}
	records, err := NativeAIScoringRecords([]*Unit{unit})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ScoreNativeAI1598A(
		1, 1, records, 1, 0, 1, unit, book,
		[]byte{0}, []byte{0}, make([]byte, NativeMovementCostRowSize), nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.MaxScore != 0 || got.HasPositiveWinner {
		t.Fatalf("all-zero result invented winner: %+v", got)
	}
}

func TestScoreNativeAI1598AFailsClosedWhenActorRecordDisagrees(t *testing.T) {
	unit := completeNativeAIScoringUnit()
	unit.NativeMapPresentation.X = 0
	unit.NativeMapPresentation.Y = 0
	records, err := NativeAIScoringRecords([]*Unit{unit})
	if err != nil {
		t.Fatal(err)
	}
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for id := range book {
		book[id] = NativeCommandRecord{ID: id}
	}
	unit.MP++
	if _, err := ScoreNativeAI1598A(
		1, 1, records, 1, 0, 0, unit, book,
		[]byte{0}, []byte{0}, make([]byte, NativeMovementCostRowSize), nil,
	); err == nil {
		t.Fatal("actor MP/record mismatch was accepted")
	}
}

func TestMap0AssetsProducePositiveNativeAI1598AScoreForCommand0(t *testing.T) {
	st, err := Load("../../assets/maps/map0/map0_units.json")
	if err != nil {
		t.Fatal(err)
	}
	actor := 23
	if st.Units[actor].NativeRecordByte8 != 103 ||
		!st.Units[actor].HasNativeRecordByte8 ||
		st.Units[actor].HasNativeIdentity {
		t.Fatalf("map0 actor raw +8=%d want 103", st.Units[actor].NativeRecordByte8)
	}
	st.Units[actor].NativeCommandMask = [5]byte{1, 0, 0, 0, 0}
	st.Units[actor].MP = 255
	for _, unit := range st.Units {
		if err := unit.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
	}
	records, err := NativeAIScoringRecords(st.Units)
	if err != nil {
		t.Fatal(err)
	}
	book, err := LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := LoadNativeMovementCostRows("../../assets/data/native_movement_cost_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ScoreNativeAI1598A(
		st.W, st.H, records, len(st.Units), actor, 0, st.Units[actor], book,
		nativeCompositionBaseFlagsForTest(t, st),
		st.NativeTerrainMoveCodes, rows[0], nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasPositiveWinner || got.MaxScore != 96 {
		t.Fatalf("map0 command0 native score=%+v", got)
	}
	if got.PositiveWinner.CommandID != 0 ||
		got.PositiveWinner.X != 23 || got.PositiveWinner.Y != 14 {
		t.Fatalf("map0 command0 native winner=%+v", got)
	}
	// 0x159A5 push 0 → 0x4E555：施法落點固定用成本列 0，行動者自己的移動成本列
	// （這裡用全部是 10 的列代表崎嶇地形）不能讓落點變少。
	heavy := make([]byte, NativeMovementCostRowSize)
	for i := range heavy {
		heavy[i] = 10
	}
	again, err := ScoreNativeAI1598A(
		st.W, st.H, records, len(st.Units), actor, 0, st.Units[actor], book,
		nativeCompositionBaseFlagsForTest(t, st),
		st.NativeTerrainMoveCodes, heavy, nil,
	)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("行動者成本列影響了 0x1598a 落點：%+v／%v，要 %+v", again, err, got)
	}
}
