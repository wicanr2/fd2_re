package battle

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNativeFieldEventIDAtMatchesSelector(t *testing.T) {
	st := &State{
		W: 2, H: 2,
		NativeFieldEventSlots: []int{-1, 3, -1, -1},
		NativeFieldEvents:     make([]NativeFieldEvent, 16),
	}
	st.NativeFieldEvents[3] = NativeFieldEvent{EventID: 82, Selector: 1}
	if got, ok := NativeFieldEventIDAt(st, 1, 0, 1); !ok || got != 82 {
		t.Fatalf("event = (%d,%v), want (82,true)", got, ok)
	}
	if _, ok := NativeFieldEventIDAt(st, 1, 0, 0); ok {
		t.Fatal("selector mismatch unexpectedly accepted")
	}
}

func TestNativeFieldEventIDAtFailsClosed(t *testing.T) {
	st := &State{W: 1, H: 1, NativeFieldEventSlots: []int{0}}
	if _, ok := NativeFieldEventIDAt(st, 0, 0, 0); ok {
		t.Fatal("missing table unexpectedly accepted")
	}
	st.NativeFieldEvents = make([]NativeFieldEvent, 16)
	st.NativeFieldEvents[0] = NativeFieldEvent{EventID: 0xff}
	if _, ok := NativeFieldEventIDAt(st, 0, 0, 0); ok {
		t.Fatal("0xff event unexpectedly accepted")
	}
}

func TestAllEditableMapsCarryNativeRendererInputs(t *testing.T) {
	paths, err := filepath.Glob("../../assets/maps/map*/map*_units.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 33 {
		t.Fatalf("map unit assets=%d, want 33", len(paths))
	}
	for _, path := range paths {
		st, err := Load(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if len(st.NativeTileBlitModes) != st.W*st.H ||
			len(st.NativeTerrainControl) == 0 ||
			len(st.NativeTerrainControl)%4 != 0 ||
			!st.HasNativeTurnEventControlState {
			t.Fatalf(
				"%s: renderer modes=%d cells=%d controls=%d turn-controls=%v",
				path, len(st.NativeTileBlitModes), st.W*st.H,
				len(st.NativeTerrainControl), st.HasNativeTurnEventControlState,
			)
		}
		if st.NativeRoundCounter != 1 {
			t.Fatalf("%s: native round seed=%d, want 1", path, st.NativeRoundCounter)
		}
	}
}

func TestNativeTurnEventCatalogRejectsTampering(t *testing.T) {
	raw, err := os.ReadFile("../../assets/maps/native_turn_event_controls.json")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		old  string
		new  string
	}{
		{"round writer", `"writer": "0x2066e"`, `"writer": "0x2066f"`},
		{"control row", `"turn": 255`, `"turn": 254`},
		{"map identity", `"map": 0`, `"map": 1`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corrupt := strings.Replace(string(raw), tt.old, tt.new, 1)
			if corrupt == string(raw) {
				t.Fatalf("%q was not found in the catalog fixture", tt.old)
			}
			mapDir := filepath.Join(t.TempDir(), "maps", "map26")
			if err := os.MkdirAll(mapDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(
				filepath.Join(filepath.Dir(mapDir), "native_turn_event_controls.json"),
				[]byte(corrupt),
				0o644,
			); err != nil {
				t.Fatal(err)
			}
			controls, seed, ok := loadNativeTurnEventControls(filepath.Join(mapDir, "map.json"), 29, 20)
			if ok || seed != 0 || controls != ([16]NativeTurnEventControl{}) {
				t.Fatalf("tampered catalog accepted: ok=%v seed=%d controls=%#v", ok, seed, controls)
			}
		})
	}
}

func TestMap25LoadsEditableNativeFieldEventRules(t *testing.T) {
	st, err := Load("../../assets/maps/map25/map25_units.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(st.NativeFieldEventRules) != 3 {
		t.Fatalf("field event rules=%d, want 3", len(st.NativeFieldEventRules))
	}
	if got := st.NativeFieldEventRules[0]; got.EventID != 59 || got.Selector != 0 ||
		got.TriggerGate != "record_byte6_nonzero" ||
		!reflect.DeepEqual(got.SetModeRanges, []NativeFieldModeRange{{Start: 39, End: 44, Mode: 0}}) {
		t.Fatalf("event59 rule=%#v", got)
	}
	got61 := st.NativeFieldEventRules[2]
	if got61.EventID != 61 || got61.Selector != 1 ||
		got61.OnceState == nil || *got61.OnceState != 12 ||
		got61.RequiredItem == nil || *got61.RequiredItem != 0xD0 ||
		got61.SpawnGroup == nil || *got61.SpawnGroup != 1 ||
		got61.JoinCharacter == nil || *got61.JoinCharacter != 31 ||
		got61.TextIndices == nil ||
		*got61.TextIndices != (NativeFieldTextIndices{MissingItem: 2, Success: 3, Final: 4}) ||
		got61.Presentation == nil ||
		*got61.Presentation != (NativeFieldPresentation{
			Archive: "FDOTHER.DAT", Resource: 45, Frames: 59,
			Helper: "0x2935b", DestinationOffset: 48356, Stride: 320,
			Transparent: -1, DelayHelper: "0x17aa9", DelayTicks: 2,
		}) {
		t.Fatalf("event61 rule=%#v", got61)
	}
}

func map26Event62Cell(t *testing.T, st *State) (int, int) {
	t.Helper()
	for cell, slot := range st.NativeFieldEventSlots {
		if slot < 0 || slot >= len(st.NativeFieldEvents) {
			continue
		}
		event := st.NativeFieldEvents[slot]
		if event.EventID == 62 && event.Selector == 0 {
			return cell % st.W, cell / st.W
		}
	}
	t.Fatal("map26 has no event62 selector0 cell")
	return 0, 0
}

func TestMap26LoadsDormantTurnRowsAndActivatesEvent63(t *testing.T) {
	st, err := Load("../../assets/maps/map26/map26_units.json")
	if err != nil {
		t.Fatal(err)
	}
	if !st.HasNativeTurnEventControlState {
		t.Fatal("complete native turn controls were not loaded")
	}
	if got := st.NativeTurnEventControls[0]; got != (NativeTurnEventControl{Turn: 0xff, EventID: 63, RawCamp: 0}) {
		t.Fatalf("map26 dormant row0=%#v", got)
	}
	if got := st.NativeTurnEventControls[1]; got != (NativeTurnEventControl{Turn: 0xff, EventID: 65, RawCamp: 0}) {
		t.Fatalf("map26 dormant row1=%#v", got)
	}
	if len(st.NativeFieldEventRules) != 1 || st.NativeFieldEventRules[0].TurnActivation == nil {
		t.Fatalf("map26 event rules=%#v", st.NativeFieldEventRules)
	}
	x, y := map26Event62Cell(t, st)
	st.NativeRoundCounter = 8
	eventID, err := ApplyNativeFieldTurnActivationEvent(st, x, y, 0)
	if err != nil || eventID != 62 {
		t.Fatalf("event62 activation=(%d,%v)", eventID, err)
	}
	if got := st.NativeTurnEventControls[0]; got != (NativeTurnEventControl{Turn: 9, EventID: 63, RawCamp: 0}) || st.NativeEventState[17] != 1 {
		t.Fatalf("activated row=%#v state17=%d", got, st.NativeEventState[17])
	}
	before := st.NativeTurnEventControls
	if _, err := ApplyNativeFieldTurnActivationEvent(st, x, y, 0); err == nil || st.NativeTurnEventControls != before {
		t.Fatalf("repeated event62 must fail without mutation: err=%v rows=%#v", err, st.NativeTurnEventControls)
	}
}

func TestEvent62RawDisagreementFailsAtomically(t *testing.T) {
	st, err := Load("../../assets/maps/map26/map26_units.json")
	if err != nil {
		t.Fatal(err)
	}
	x, y := map26Event62Cell(t, st)
	st.NativeRoundCounter = 3
	st.HasNativeFieldControlState = true
	st.NativeFieldControlRaw = make([]byte, 6)
	st.NativeFieldControlRaw[3] = 0 // typed row says 0xff
	st.NativeFieldControlRaw[4] = 63
	beforeRows := st.NativeTurnEventControls
	beforeRaw := append([]byte(nil), st.NativeFieldControlRaw...)
	if _, err := ApplyNativeFieldTurnActivationEvent(st, x, y, 0); err == nil {
		t.Fatal("disagreeing raw row unexpectedly activated")
	}
	if st.NativeTurnEventControls != beforeRows || !reflect.DeepEqual(st.NativeFieldControlRaw, beforeRaw) || st.NativeEventState[17] != 0 {
		t.Fatal("failed event62 activation partially mutated state")
	}
}

func TestChapter29Event75PlansAndCommitsExactTurnChain(t *testing.T) {
	st, err := Load("../../assets/maps/map28/map28_units.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := LoadScenario("../../assets/scenarios/ch29.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sc.SetupChecked(st); err != nil {
		t.Fatal(err)
	}
	trigger := &Unit{
		X: 15, Y: 21,
		HasNativeRecordByte6: true, NativeRecordByte6: 1,
		HasNativeRecordByte8: true, NativeRecordByte8: 9,
	}
	st.NativeRoundCounter = 8
	beforeRows := st.NativeTurnEventControls
	beforeRaw := append([]byte(nil), st.NativeFieldControlRaw...)
	plan, err := PlanNativeFieldEvent75(st, trigger, 15, 21)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Activate || plan.Noop || plan.TextIndex != 1 ||
		st.NativeTurnEventControls != beforeRows || !reflect.DeepEqual(st.NativeFieldControlRaw, beforeRaw) {
		t.Fatalf("event75 plan=%#v mutated state", plan)
	}
	if err := CommitNativeFieldEvent75(st, plan); err != nil {
		t.Fatal(err)
	}
	if st.NativeEventState[17] != 1 || st.NativeEventState[16] != 4 ||
		st.NativeTurnEventControls[1] != (NativeTurnEventControl{Turn: 9, EventID: 76, RawCamp: 2}) ||
		st.NativeTurnEventControls[0] != (NativeTurnEventControl{Turn: 8, EventID: 74, RawCamp: 0}) {
		t.Fatalf("event75 committed state16=%d state17=%d rows=%#v", st.NativeEventState[16], st.NativeEventState[17], st.NativeTurnEventControls[:2])
	}
}

func TestChapter29Event75MismatchAndMissingProvenanceDoNotActivate(t *testing.T) {
	st, err := Load("../../assets/maps/map28/map28_units.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := LoadScenario("../../assets/scenarios/ch29.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sc.SetupChecked(st); err != nil {
		t.Fatal(err)
	}
	trigger := &Unit{
		X: 15, Y: 21,
		HasNativeRecordByte6: true, NativeRecordByte6: 1,
		HasNativeRecordByte8: true, NativeRecordByte8: 8,
	}
	plan, err := PlanNativeFieldEvent75(st, trigger, 15, 21)
	if err != nil || plan.Activate || plan.Noop || plan.TextIndex != 0 {
		t.Fatalf("event75 mismatch plan=%#v err=%v", plan, err)
	}
	trigger.HasNativeRecordByte8 = false
	if _, err := PlanNativeFieldEvent75(st, trigger, 15, 21); err == nil {
		t.Fatal("event75 accepted missing raw +8 provenance")
	}
}

func TestChapter26KeepsWoldPendingUntilEvent61(t *testing.T) {
	st, err := Load("../../assets/maps/map25/map25_units.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := LoadScenario("../../assets/scenarios/ch26.json")
	if err != nil {
		t.Fatal(err)
	}
	if !sc.RuntimeAppendGroups || !reflect.DeepEqual(sc.InitialGroups, []int{0}) {
		t.Fatalf("ch26 constructor policy: runtime=%v initial=%v", sc.RuntimeAppendGroups, sc.InitialGroups)
	}
	sc.Setup(st)
	var activeWold, pendingWold int
	for _, unit := range st.Units {
		if unit != nil && unit.Group == 1 && unit.Fig == 31 {
			activeWold++
		}
	}
	for _, unit := range st.Roster {
		if unit != nil && unit.Group == 1 && unit.Fig == 31 {
			pendingWold++
		}
	}
	if activeWold != 0 || pendingWold != 1 {
		t.Fatalf("event61 前渥德 active=%d pending=%d", activeWold, pendingWold)
	}
}

func setupChapter26Event61(t *testing.T, items ...int) (*State, *Unit) {
	t.Helper()
	st, err := Load("../../assets/maps/map25/map25_units.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := LoadScenario("../../assets/scenarios/ch26.json")
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	trigger := st.Units[0]
	trigger.X, trigger.Y = 1, 46
	trigger.Inventory = append([]int(nil), items...)
	trigger.Equipped = make([]bool, len(items))
	trigger.InventorySlots = make([]int, 8)
	trigger.NativeInventoryFlags = make([]int, 8)
	for i := range trigger.InventorySlots {
		trigger.InventorySlots[i] = 0xff
		trigger.NativeInventoryFlags[i] = 0x80
	}
	for i, item := range items {
		trigger.InventorySlots[i] = item
		trigger.NativeInventoryFlags[i] = 0
	}
	return st, trigger
}

func TestEvent61MissingItemPlansOnlyOriginalText(t *testing.T) {
	st, trigger := setupChapter26Event61(t, 0x20)
	plan, err := PlanNativeFieldEvent61(st, trigger, 1, 46)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.MissingItem || plan.TextIndex != 2 || plan.FinalText != 4 {
		t.Fatalf("missing-item plan=%#v", plan)
	}
	if st.NativeEventState[12] != 0 || len(st.Roster) == 0 ||
		len(trigger.Inventory) != 1 || trigger.Inventory[0] != 0x20 {
		t.Fatal("missing-item planning mutated battle state")
	}
	if _, err := CommitNativeFieldEvent61(st, plan, 59); err == nil {
		t.Fatal("missing-item plan unexpectedly committed")
	}
}

func TestEvent61CommitsAfterAllFramesAndReturnsJoin(t *testing.T) {
	st, trigger := setupChapter26Event61(t, 0xD0, 0x20)
	plan, err := PlanNativeFieldEvent61(st, trigger, 1, 46)
	if err != nil {
		t.Fatal(err)
	}
	if plan.MissingItem || plan.TextIndex != 3 || plan.FinalText != 4 ||
		plan.Presentation.Frames != 59 {
		t.Fatalf("success plan=%#v", plan)
	}
	if _, err := CommitNativeFieldEvent61(st, plan, 58); err == nil {
		t.Fatal("incomplete presentation unexpectedly committed")
	}
	if st.NativeEventState[12] != 0 || trigger.Inventory[0] != 0xD0 {
		t.Fatal("failed commit partially mutated state")
	}
	joined, err := CommitNativeFieldEvent61(st, plan, 59)
	if err != nil {
		t.Fatal(err)
	}
	if joined != 31 || st.NativeEventState[12] != 1 ||
		!reflect.DeepEqual(trigger.Inventory, []int{0x20}) ||
		trigger.InventorySlots[0] != 0x20 ||
		trigger.InventorySlots[1] != 0xff {
		t.Fatalf("event61 commit join=%d state=%d inventory=%v slots=%v",
			joined, st.NativeEventState[12], trigger.Inventory, trigger.InventorySlots)
	}
	var activeWold, pendingWold int
	for _, unit := range st.Units {
		if unit != nil && unit.Group == 1 && unit.Fig == 31 && unit.OnField {
			activeWold++
		}
	}
	for _, unit := range st.Roster {
		if unit != nil && unit.Group == 1 && unit.Fig == 31 {
			pendingWold++
		}
	}
	if activeWold != 1 || pendingWold != 0 {
		t.Fatalf("event61 後渥德 active=%d pending=%d", activeWold, pendingWold)
	}
}

func TestEvent61RevalidatesInventoryBeforeCommit(t *testing.T) {
	st, trigger := setupChapter26Event61(t, 0xD0)
	plan, err := PlanNativeFieldEvent61(st, trigger, 1, 46)
	if err != nil {
		t.Fatal(err)
	}
	trigger.Inventory[0], trigger.InventorySlots[0] = 0x20, 0x20
	if _, err := CommitNativeFieldEvent61(st, plan, 59); err == nil {
		t.Fatal("changed inventory unexpectedly committed")
	}
	if st.NativeEventState[12] != 0 || len(st.Roster) == 0 {
		t.Fatal("revalidation failure partially mutated state")
	}
}

func TestMap25Event59AppliesModeRangeAtomically(t *testing.T) {
	st, err := Load("../../assets/maps/map25/map25_units.json")
	if err != nil {
		t.Fatal(err)
	}
	trigger := &Unit{NativeRecordByte6: 1, HasNativeRecordByte6: true}
	for index := 39; index <= 44; index++ {
		st.Units[index].NativeRecordByte34 |= 0xA0
		st.Units[index].HasNativeRecordByte34 = true
	}
	if eventID, ok := ApplyNativeFieldModeEvent(st, trigger, 10, 36, 0); !ok || eventID != 59 {
		t.Fatalf("event59=(%d,%v)", eventID, ok)
	}
	for index := 39; index <= 44; index++ {
		if got := st.Units[index].NativeRecordByte34; got != 0xA0 {
			t.Fatalf("unit%d byte34=%#x, want 0xa0", index, got)
		}
	}
}

func TestNativeFieldModeEventFailsClosedBeforePartialWrite(t *testing.T) {
	st, err := Load("../../assets/maps/map25/map25_units.json")
	if err != nil {
		t.Fatal(err)
	}
	trigger := &Unit{NativeRecordByte6: 1, HasNativeRecordByte6: true}
	before := st.Units[23].NativeRecordByte34
	st.Units[56].HasNativeRecordByte34 = false
	if _, ok := ApplyNativeFieldModeEvent(st, trigger, 10, 22, 0); ok {
		t.Fatal("incomplete event60 target provenance unexpectedly accepted")
	}
	if st.Units[23].NativeRecordByte34 != before {
		t.Fatal("event60 partially mutated before validation completed")
	}
}
