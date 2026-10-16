package battle

import (
	"bytes"
	"testing"
)

func TestBuildNativeAIPhaseDiagnosticPlanRequiresExactEligibleInputs(t *testing.T) {
	st := nativeAIPhaseDiagnosticFixture(t)
	book := nativeAIPhaseDiagnosticBook()
	itemRows := make([]byte, NativeItemEffectRowSize)
	costRow := make([]byte, NativeMovementCostRowSize)

	if _, _, err := BuildNativeAIPhaseDiagnosticPlan(
		st, book, itemRows, nil,
	); err == nil {
		t.Fatal("missing eligible selector-zero input was accepted")
	}
	if _, _, err := BuildNativeAIPhaseDiagnosticPlan(
		st, book, itemRows,
		[]NativeAIPhaseZeroInput{{UnitIndex: 0, CostRow: costRow}},
	); err == nil {
		t.Fatal("selector-one input was accepted as selector-zero provenance")
	}
	if _, _, err := BuildNativeAIPhaseDiagnosticPlan(
		st, book, itemRows,
		[]NativeAIPhaseZeroInput{
			{UnitIndex: 1, CostRow: costRow},
			{UnitIndex: 1, CostRow: costRow},
		},
	); err == nil {
		t.Fatal("duplicate selector-zero input was accepted")
	}
}

func TestBuildNativeAIPhaseDiagnosticPlanIsReadOnlyAndKeepsThreePasses(t *testing.T) {
	st := nativeAIPhaseDiagnosticFixture(t)
	before, err := NativeAIScoringRecords(st.Units)
	if err != nil {
		t.Fatal(err)
	}
	plan, diagnostics, err := BuildNativeAIPhaseDiagnosticPlan(
		st, nativeAIPhaseDiagnosticBook(),
		make([]byte, NativeItemEffectRowSize),
		[]NativeAIPhaseZeroInput{{
			UnitIndex: 1, CostRow: make([]byte, NativeMovementCostRowSize),
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	after, err := NativeAIScoringRecords(st.Units)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("native AI diagnostic plan mutated battle state")
	}
	if len(plan.SelectorOne) != 1 || plan.SelectorOne[0].UnitIndex != 0 ||
		len(plan.ZeroPreselection) != 1 ||
		plan.ZeroPreselection[0].UnitIndex != 1 ||
		plan.ZeroPreselection[0].RunUnitAction ||
		len(plan.ZeroSecondPass) != 1 ||
		plan.ZeroSecondPass[0].UnitIndex != 1 {
		t.Fatalf("phase plan=%+v", plan)
	}
	if len(diagnostics) != 1 || diagnostics[0].UnitIndex != 1 ||
		diagnostics[0].CommandMaskResult.MaxScore != 0 ||
		diagnostics[0].ItemCommandResult.MaxScore != 0 {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

func TestMap0CrossFixtureBuildsNativeAIPhaseDiagnosticGate(t *testing.T) {
	st, err := Load("../../assets/maps/map0/map0_units.json")
	if err != nil {
		t.Fatal(err)
	}
	actor := 23
	for index, unit := range st.Units {
		if err := unit.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
		if unit.NativeRecordByte6 == 0 && index != actor {
			unit.NativeRecordByte5 |= 0x80
		}
	}
	unit := st.Units[actor]
	unit.NativeCommandMask = [5]byte{1, 0, 0, 0, 0}
	unit.MP = 255
	unit.InventorySlots = []int{79, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	unit.NativeInventoryFlags = []int{0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}

	book, err := LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := LoadNativeItemEffectRowPrefix(
		"../../assets/data/native_item_effect_rows.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	costRows, err := LoadNativeMovementCostRows(
		"../../assets/data/native_movement_cost_rows.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	before, err := NativeAIScoringRecords(st.Units)
	if err != nil {
		t.Fatal(err)
	}
	plan, diagnostics, err := BuildNativeAIPhaseDiagnosticPlan(
		st, book, itemRows,
		[]NativeAIPhaseZeroInput{{UnitIndex: actor, CostRow: costRows[0]}},
	)
	if err != nil {
		t.Fatal(err)
	}
	after, err := NativeAIScoringRecords(st.Units)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("map0 diagnostic plan mutated battle state")
	}
	if len(plan.ZeroPreselection) != 1 ||
		plan.ZeroPreselection[0].UnitIndex != actor ||
		!plan.ZeroPreselection[0].RunUnitAction ||
		len(plan.ZeroSecondPass) != 1 ||
		plan.ZeroSecondPass[0].UnitIndex != actor {
		t.Fatalf("map0 phase plan=%+v", plan)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("map0 diagnostics=%+v", diagnostics)
	}
	got := diagnostics[0]
	if got.CommandMaskResult.MaxScore != 96 ||
		!got.CommandMaskResult.HasPositiveWinner ||
		got.CommandMaskResult.PositiveWinner.CommandID != 0 ||
		got.CommandMaskResult.PositiveWinner.X != 23 ||
		got.CommandMaskResult.PositiveWinner.Y != 14 {
		t.Fatalf("map0 command-mask diagnostic=%+v", got.CommandMaskResult)
	}
	if got.ItemCommandResult.MaxScore != 0 || got.ItemCommandResult.HasPositiveWinner {
		t.Fatalf("map0 item-command diagnostic=%+v", got.ItemCommandResult)
	}
	if got.PhaseGateScorePair.SpellCandidateScore53C23 != 96 ||
		got.PhaseGateScorePair.ItemCommandScore53C33 != 0 {
		t.Fatalf("map0 score pair=%+v", got.PhaseGateScorePair)
	}
}

func TestMap19RealAssetInputsProduceZeroAIScores(t *testing.T) {
	st, err := Load("../../assets/maps/map19/map19_units.json")
	if err != nil {
		t.Fatal(err)
	}
	nonzeroFlags := 0
	for _, flag := range st.NativeCompositionEventBytes {
		if flag != 0 {
			nonzeroFlags++
		}
	}
	if len(st.NativeCompositionEventBytes) != 40*40 ||
		nonzeroFlags != 7 ||
		st.NativeCompositionEventBytes[122] != 5 ||
		st.NativeCompositionEventBytes[310] != 7 ||
		st.NativeCompositionEventBytes[711] != 6 {
		t.Fatalf("map19 composition event bytes lack FDFIELD provenance")
	}
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
	itemRows, err := LoadNativeItemEffectRowPrefix(
		"../../assets/data/native_item_effect_rows.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	costRows, err := LoadNativeMovementCostRows(
		"../../assets/data/native_movement_cost_rows.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	const actor = 55
	if st.Units[actor].NativeRecordByte8 != 92 ||
		!st.Units[actor].HasNativeRecordByte8 ||
		st.Units[actor].HasNativeIdentity ||
		st.Units[actor].NativeCommandMask != [5]byte{4, 0, 0, 8, 0} ||
		st.Units[actor].MP != 288 {
		t.Fatalf("map19 actor55=%+v", st.Units[actor])
	}
	commandMask, err := ScoreNativeAI1598A(
		st.W, st.H, records, len(st.Units), actor, 0, st.Units[actor],
		book, nativeCompositionBaseFlagsForTest(t, st),
		st.NativeTerrainMoveCodes,
		costRows[0], nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	itemCommand, err := ScoreNativeAI1567E(
		st.W, st.H, records, len(st.Units), actor, 0,
		itemRows, book, nativeCompositionBaseFlagsForTest(t, st),
	)
	if err != nil {
		t.Fatal(err)
	}
	if commandMask.MaxScore != 0 || commandMask.HasPositiveWinner ||
		itemCommand.MaxScore != 0 || itemCommand.HasPositiveWinner {
		t.Fatalf(
			"map19 real scores command-mask=%+v item-command=%+v",
			commandMask, itemCommand,
		)
	}
}

func nativeCompositionBaseFlagsForTest(t *testing.T, st *State) []byte {
	t.Helper()
	flags, err := NativeCompositionBaseFlags(
		st.W, st.H, st.NativeCompositionEventBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	return flags
}

func nativeAIPhaseDiagnosticFixture(t *testing.T) *State {
	t.Helper()
	units := []*Unit{
		completeNativeAIScoringUnit(),
		completeNativeAIScoringUnit(),
	}
	for index, unit := range units {
		unit.NativeMapPresentation.X = byte(index)
		unit.NativeMapPresentation.Y = 0
		unit.NativeRecordByte6 = byte(1 - index)
		unit.NativeTransient[4] = 0
		unit.NativeCommandMask = [5]byte{}
		unit.NativeInventoryFlags = []int{
			0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
		}
		unit.InventorySlots = []int{
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		}
	}
	return &State{
		W: 2, H: 1, Units: units,
		NativeCompositionEventBytes: make([]byte, 2),
		NativeTerrainMoveCodes:      make([]byte, 2),
	}
}

func nativeAIPhaseDiagnosticBook() []NativeCommandRecord {
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for id := range book {
		book[id] = NativeCommandRecord{ID: id}
	}
	return book
}
