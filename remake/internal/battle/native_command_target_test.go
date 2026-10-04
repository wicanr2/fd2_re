package battle

import (
	"reflect"
	"testing"
)

func TestNativeCompositionBaseFlagsMasksArchiveHighBits(t *testing.T) {
	got, err := NativeCompositionBaseFlags(
		3, 1, []byte{0xff, 0x40, 0x85},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 0x1f || got[1] != 0 || got[2] != 5 {
		t.Fatalf("composition base flags=%#v", got)
	}
}

func TestStateNativeCommandBaseFlagsAreRebuiltPerCall(t *testing.T) {
	st := &State{
		W:                           2,
		H:                           1,
		NativeCompositionEventBytes: []byte{0xE5, 0x7F},
	}
	first, err := st.NativeCommandBaseFlags()
	if err != nil {
		t.Fatal(err)
	}
	first[0] = 0xFF
	second, err := st.NativeCommandBaseFlags()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := second, []byte{0x05, 0x1F}; !reflect.DeepEqual(got, want) {
		t.Fatalf("rebuilt flags=%v want %v", got, want)
	}
}

func TestStateNativeCommandBaseFlagsRejectMissingSource(t *testing.T) {
	var missing *State
	if _, err := missing.NativeCommandBaseFlags(); err == nil {
		t.Fatal("expected nil state to fail closed")
	}
	st := &State{W: 1, H: 1}
	if _, err := st.NativeCommandBaseFlags(); err == nil {
		t.Fatal("expected missing composition event bytes to fail closed")
	}
}

func TestNativeCommandRuntimeFlagsMatches145CDWriters(t *testing.T) {
	unit := func(x, y int, selector byte, byte5 byte) *Unit {
		return &Unit{
			NativeMapPresentation: NativeMapPresentationState{
				X: byte(x), Y: byte(y),
			},
			HasNativeMapPresentation: true,
			NativeRecordByte5:        byte5,
			HasNativeRecordByte5:     true,
			NativeRecordByte6:        selector,
			HasNativeRecordByte6:     true,
		}
	}
	units := []*Unit{
		unit(1, 1, 1, 0),
		unit(0, 0, 0, 0),
		unit(2, 2, 1, 1),
	}
	flags, err := NativeCommandRuntimeFlags(
		3, 3,
		[]byte{0xff, 0, 0, 0, 0, 0, 0, 0, 0},
		units,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{
		0x1f, 0x80, 0,
		0x80, 0x40, 0x80,
		0, 0x80, 0,
	}
	for index := range want {
		if flags[index] != want[index] {
			t.Fatalf(
				"selector0 flags[%d]=%#x want %#x; all=%#v",
				index, flags[index], want[index], flags,
			)
		}
	}

	flags, err = NativeCommandRuntimeFlags(
		3, 3, make([]byte, 9), units, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if flags[0] != 0x40 || flags[1] != 0x80 ||
		flags[3] != 0x80 || flags[4] != 0 {
		t.Fatalf("selector1 flags=%#v", flags)
	}
}

func TestNativeCommandRuntimeFlagsFailsClosedWithoutRawProvenance(t *testing.T) {
	if _, err := NativeCommandRuntimeFlags(
		1, 1, []byte{0}, []*Unit{{}}, 0,
	); err == nil {
		t.Fatal("runtime flag writer accepted incomplete roster")
	}
}

func TestNativeCommandRuntimeFlagsAllowsEmptyRuntimeArray(t *testing.T) {
	got, err := NativeCommandRuntimeFlags(
		2, 1, []byte{0xe5, 0x40}, nil, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{5, 0}; !reflect.DeepEqual(got, want) {
		t.Fatalf("empty-runtime flags=%#v want %#v", got, want)
	}
}

func TestNativeFutureGroupPlacementMatches10C50OccupancyAndTieOrder(t *testing.T) {
	unit := func(x, y int, campSelector byte, byte5 byte) *Unit {
		return &Unit{
			NativeMapPresentation: NativeMapPresentationState{
				X: byte(x), Y: byte(y),
			},
			HasNativeMapPresentation: true,
			NativeRecordByte5:        byte5,
			HasNativeRecordByte5:     true,
			NativeRecordByte6:        campSelector,
			HasNativeRecordByte6:     true,
		}
	}
	// Both selector classes are marked because 0x10c50 calls 0x145cd with
	// zero and one. The inactive record at (2,0) must not block its cell.
	units := []*Unit{
		unit(1, 1, 0, 0),
		unit(2, 1, 2, 0),
		unit(2, 0, 1, 1),
	}
	got, err := NativeFutureGroupPlacement(
		3, 3, make([]byte, 9), units,
		NativePositionRecord{XWord: 1, YWord: 1},
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	// Four distance-one candidates surround the occupied origin. Row-major
	// scanning replaces equal distances, so the last one, (1,2), wins.
	if want := (Cell{X: 1, Y: 2}); got != want {
		t.Fatalf("future placement=%v want %v", got, want)
	}
}

func TestNativeFutureGroupPlacementUsesPositionLowBytesAndRawGate(t *testing.T) {
	got, err := NativeFutureGroupPlacement(
		4, 4, make([]byte, 16), nil,
		NativePositionRecord{XWord: 0x1203, YWord: 0x3402},
		7,
	)
	if err != nil {
		t.Fatal(err)
	}
	if want := (Cell{X: 3, Y: 2}); got != want {
		t.Fatalf("direct future placement=%v want %v", got, want)
	}
}

func TestAppendGroupWithNativePlacementConsumesPerCallGate(t *testing.T) {
	active := &Unit{
		X: 1, Y: 1,
		MapSelectorKey:           2,
		HasMapSelectorKey:        true,
		NativeMapPresentation:    NativeMapPresentationState{X: 1, Y: 1},
		HasNativeMapPresentation: true,
		NativeRecordByte5:        0,
		HasNativeRecordByte5:     true,
		NativeRecordByte6:        2,
		HasNativeRecordByte6:     true,
	}
	pending := func() *Unit {
		return &Unit{
			Group: 6, Dir: 0, Lv: 2,
			MapSelectorKey:          1,
			HasMapSelectorKey:       true,
			NativeRecordByte5:       0,
			HasNativeRecordByte5:    true,
			NativeRecordByte6:       1,
			HasNativeRecordByte6:    true,
			NativePositionRecord:    NativePositionRecord{XWord: 1, YWord: 1},
			HasNativePositionRecord: true,
			NativeConstructor: &NativeConstructorTable{
				Branch: "high_class", Index: 0,
				Record: []byte{4, 5, 10, 0, 3, 6, 7, 8, 9, 0},
			},
			Inventory: []int{0}, Equipped: []bool{true},
			InventorySlots:       []int{0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			NativeInventoryFlags: []int{0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
		}
	}
	newState := func(unit *Unit) *State {
		st := &State{
			W: 3, H: 3, Roster: []*Unit{unit},
			NativeCompositionEventBytes: make([]byte, 9),
		}
		if err := st.BindNativeFutureItemRows(make([]byte, NativeItemEffectRowSize)); err != nil {
			t.Fatal(err)
		}
		if err := st.AppendNativeMapSelectorBatch([]*Unit{active}); err != nil {
			t.Fatal(err)
		}
		return st
	}

	search := newState(pending())
	if n, err := search.AppendGroupWithNativePlacement(6, 0); err != nil || n != 1 {
		t.Fatalf("search append n=%d err=%v", n, err)
	}
	if got, want := (Cell{X: search.Units[1].X, Y: search.Units[1].Y}), (Cell{X: 1, Y: 2}); got != want {
		t.Fatalf("gate=0 placement=%v want %v", got, want)
	}

	directPending := pending()
	direct := newState(directPending)
	if n, err := direct.AppendGroupWithNativePlacement(6, 1); err != nil || n != 1 {
		t.Fatalf("direct append n=%d err=%v", n, err)
	}
	if got, want := (Cell{X: direct.Units[1].X, Y: direct.Units[1].Y}), (Cell{X: 1, Y: 1}); got != want {
		t.Fatalf("gate=1 placement=%v want %v", got, want)
	}
	spawned := direct.Units[1]
	if spawned.AP != 12 || spawned.DP != 14 || spawned.HIT != 16 || spawned.EV != 16 ||
		spawned.HP != 20 || spawned.MP != 6 || spawned.MV != 9 ||
		spawned.NativeRecordRace != 4 || spawned.NativeRecordClass != 5 {
		t.Fatalf("constructor projection=%#v", spawned)
	}
	if directPending.AP != 0 || directPending.HasNativeRecordRace {
		t.Fatal("preflight mutated the source roster record")
	}
}

func TestAppendGroupWithNativePlacementFailsBeforeRosterMutation(t *testing.T) {
	pending := &Unit{Group: 2, MapSelectorKey: 1, HasMapSelectorKey: true}
	st := &State{
		W: 1, H: 1, Roster: []*Unit{pending},
		NativeCompositionEventBytes: []byte{0},
	}
	if _, err := st.AppendGroupWithNativePlacement(2, 0); err == nil {
		t.Fatal("native append accepted a row without its six-byte position record")
	}
	if len(st.Units) != 0 || len(st.Roster) != 1 || st.Roster[0] != pending {
		t.Fatalf("failed append mutated state: units=%d roster=%#v", len(st.Units), st.Roster)
	}
}

func TestNativeFutureGroupPlacementFailsClosed(t *testing.T) {
	if _, err := NativeFutureGroupPlacement(
		1, 1, []byte{0}, []*Unit{{
			NativeMapPresentation:    NativeMapPresentationState{},
			HasNativeMapPresentation: true,
			NativeRecordByte5:        0,
			HasNativeRecordByte5:     true,
			NativeRecordByte6:        2,
			HasNativeRecordByte6:     true,
		}},
		NativePositionRecord{},
		0,
	); err == nil {
		t.Fatal("future placement accepted a fully occupied map")
	}
	if _, err := NativeFutureGroupPlacement(
		1, 1, []byte{0}, nil,
		NativePositionRecord{XWord: 1},
		1,
	); err == nil {
		t.Fatal("future placement accepted an out-of-bounds direct coordinate")
	}
	if _, err := NativeFutureGroupPlacement(
		1, 1, []byte{0}, []*Unit{{}},
		NativePositionRecord{},
		0,
	); err == nil {
		t.Fatal("future placement accepted a roster without raw provenance")
	}
}

func TestNativeCommandTargetCellsFloodFillHonorsRawFlags(t *testing.T) {
	flags := make([]byte, 5)
	flags[1] = NativeCommandGridBlocked
	flags[3] = NativeCommandGridZeroBudget
	cells, err := NativeCommandTargetCells(5, 1, Cell{X: 0, Y: 0}, 2, flags)
	if err != nil {
		t.Fatal(err)
	}
	if cells[Cell{X: 1, Y: 0}] || cells[Cell{X: 2, Y: 0}] {
		t.Fatalf("blocked cell leaked through: %v", cells)
	}
	// Bit 0x80 forces the entered cell's remaining byte to zero, so it is
	// reachable but cannot form the previously asserted zero-cost chain.
	flags = make([]byte, 5)
	flags[1], flags[2], flags[3] = NativeCommandGridZeroBudget, NativeCommandGridZeroBudget, NativeCommandGridZeroBudget
	cells, err = NativeCommandTargetCells(5, 1, Cell{}, 1, flags)
	if err != nil || !cells[Cell{X: 1, Y: 0}] || cells[Cell{X: 2, Y: 0}] {
		t.Fatalf("zero-budget terminal cells=%v err=%v", cells, err)
	}
}

func TestNativeCommandTargetFieldBytesMatches14818Lifecycle(t *testing.T) {
	got, err := NativeCommandTargetFieldBytes(5, 1, Cell{X: 2}, 2, 1, make([]byte, 5))
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0, 1, 0xff, 1, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("field[%d]=%#x want %#x; all=%#v", i, got[i], want[i], got)
		}
	}
	cross, err := NativeCommandTargetFieldBytes(5, 1, Cell{X: 2}, 0x11, 0, []byte{
		NativeCommandGridBlocked, NativeCommandGridBlocked, 0, NativeCommandGridBlocked, NativeCommandGridBlocked,
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{0xff, 0, 0, 0, 0xff}; len(cross) != len(want) {
		t.Fatalf("cross len=%d", len(cross))
	} else {
		for i := range want {
			if cross[i] != want[i] {
				t.Fatalf("cross[%d]=%#x want %#x", i, cross[i], want[i])
			}
		}
	}
}

func TestNativeCommandTargetCellsCrossIgnoresRawFlags(t *testing.T) {
	flags := []byte{NativeCommandGridBlocked, NativeCommandGridBlocked, NativeCommandGridBlocked, NativeCommandGridBlocked, NativeCommandGridBlocked}
	cells, err := NativeCommandTargetCells(5, 1, Cell{X: 2, Y: 0}, 0x12, flags)
	if err != nil || len(cells) != 5 {
		t.Fatalf("cross cells=%v err=%v", cells, err)
	}
}

func TestNativeCommandTargetsMatchesRecoveredCampCodes(t *testing.T) {
	enemy := &Unit{Camp: Enemy, X: 0, Y: 0, HP: 1, OnField: true}
	ally := &Unit{Camp: Ally, X: 1, Y: 0, HP: 1, OnField: true}
	own := &Unit{Camp: Own, X: 2, Y: 0, HP: 1, OnField: true}
	for code, want := range map[int][]*Unit{
		0: {enemy},
		1: {ally, own},
		2: {ally},
		3: {own},
	} {
		got, err := NativeCommandTargets(3, 1, Cell{X: 1, Y: 0}, 2, code, make([]byte, 3), []*Unit{enemy, ally, own})
		if err != nil || len(got) != len(want) {
			t.Fatalf("code=%d got=%v err=%v want=%v", code, got, err, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("code=%d got[%d]=%p want=%p", code, i, got[i], want[i])
			}
		}
	}
}

func TestNativeCursorConfirmationMatches115B6And14742(t *testing.T) {
	enemy := &Unit{Camp: Enemy, X: 1, Y: 0, HasNativeRecordByte5: true}
	ally := &Unit{Camp: Ally, X: 0, Y: 0, HasNativeRecordByte5: true}
	inactive := &Unit{Camp: Enemy, X: 0, Y: 0, HasNativeRecordByte5: true, NativeRecordByte5: 1}
	units := []*Unit{enemy, ally, inactive}

	// targetCode 5 and 0xff cells reject before the 0x14742 count. A
	// non-0xff targetCode 4 accepts before any roster scan.
	if ok, err := NativeCursorConfirmationAllowed(Cell{}, 0, 3, 5, nil); err != nil || ok {
		t.Fatalf("target code 5 reject ok=%v err=%v", ok, err)
	}
	if ok, err := NativeCursorConfirmationAllowed(Cell{}, 0xff, 3, 4, nil); err != nil || ok {
		t.Fatalf("0xff reject ok=%v err=%v", ok, err)
	}
	if ok, err := NativeCursorConfirmationAllowed(Cell{}, 0, 3, 4, nil); err != nil || !ok {
		t.Fatalf("target code 4 accept ok=%v err=%v", ok, err)
	}

	// selector 3 becomes strict radius 2, so the enemy at distance one
	// accepts. Selector 2 becomes radius one, where distance one is excluded.
	if ok, err := NativeCursorConfirmationAllowed(Cell{}, 0, 3, 0, units); err != nil || !ok {
		t.Fatalf("enemy inside strict radius rejected: ok=%v err=%v", ok, err)
	}
	if ok, err := NativeCursorConfirmationAllowed(Cell{}, 0, 2, 0, units); err != nil || ok {
		t.Fatalf("strict radius boundary accepted: ok=%v err=%v", ok, err)
	}
	if ok, err := NativeCursorConfirmationAllowed(Cell{}, 0, 2, 2, units); err != nil || !ok {
		t.Fatalf("code 2 failed to match ally exactly: ok=%v err=%v", ok, err)
	}
}

func TestNativeCursorConfirmationFailsClosedWithoutRawRoster(t *testing.T) {
	if _, err := NativeCursorConfirmationAllowed(Cell{}, 0, 3, 0, []*Unit{{Camp: Enemy}}); err == nil {
		t.Fatal("incomplete raw roster accepted")
	}
	if _, err := NativeCursorConfirmationAllowed(Cell{}, 0, 3, 6, nil); err == nil {
		t.Fatal("relocation target code accepted by generic gate")
	}
}

func TestNativeCommandTargetsPrefersRawByte5Predicate(t *testing.T) {
	// A fully materialized roster follows the recovered 0x14818 byte+5 gate;
	// normalized HP/OnField values must not reintroduce a second predicate.
	usableRaw := &Unit{Camp: Enemy, X: 1, Y: 0, HP: 0, OnField: false, HasNativeRecordByte5: true, NativeRecordByte5: 0}
	inactiveRaw := &Unit{Camp: Enemy, X: 2, Y: 0, HP: 10, OnField: true, HasNativeRecordByte5: true, NativeRecordByte5: 1}
	units := []*Unit{usableRaw, inactiveRaw}
	got, err := NativeCommandTargets(3, 1, Cell{X: 0, Y: 0}, 2, 0, make([]byte, 3), units)
	if err != nil || len(got) != 1 || got[0] != usableRaw {
		t.Fatalf("raw target filter got=%v err=%v", got, err)
	}
}

func TestNativeCommandTargetCellsFailsClosedWithoutRawFlags(t *testing.T) {
	if _, err := NativeCommandTargetCells(2, 2, Cell{}, 1, nil); err == nil {
		t.Fatal("missing raw flags must fail closed")
	}
}

func TestNativeCommandEffectTargetsRequiresOriginalTwoStages(t *testing.T) {
	actor := &Unit{Camp: Own, X: 0, Y: 0, HP: 10, OnField: true}
	confirmed := &Unit{Camp: Enemy, X: 1, Y: 0, HP: 10, OnField: true}
	otherEnemy := &Unit{Camp: Enemy, X: 2, Y: 0, HP: 10, OnField: true}
	units := []*Unit{actor, confirmed, otherEnemy}

	// Generic command: +3=1 chooses the adjacent enemy as center; +4=0
	// resolves the final effect only on that confirmed cell.
	effects, err := NativeCommandEffectTargets(3, 1, actor, confirmed, 1, 0, 0, make([]byte, 3), units)
	if err != nil || len(effects) != 1 || effects[0] != confirmed {
		t.Fatalf("effects=%v err=%v", effects, err)
	}
	if _, err := NativeCommandEffectTargets(3, 1, actor, otherEnemy, 1, 0, 0, make([]byte, 3), units); err == nil {
		t.Fatal("a non-selection candidate must not become the effect origin")
	}
}

// #160：正常原版 actor(20,22)、我方中心(22,20)、敵目標(22,18)。
func TestNativeCommand6PlayerCursorCenterOutsideDirectCandidates(t *testing.T) {
	actor := &Unit{Camp: Own, X: 20, Y: 22, HP: 331, MP: 1101, OnField: true, HasNativeRecordByte5: true}
	center := &Unit{Camp: Own, X: 22, Y: 20, HP: 877, OnField: true, HasNativeRecordByte5: true}
	target := &Unit{Camp: Enemy, X: 22, Y: 18, HP: 83, ClassID: 4, OnField: true, HasNativeRecordByte5: true}
	book, err := LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	st := &State{W: 30, H: 30, Units: []*Unit{actor, center, target}, NativeCommandBook: book, NativeCompositionEventBytes: make([]byte, 900)}
	plan, err := st.PlanNativeCommand6DamageAtCursor(actor, Cell{X: center.X, Y: center.Y}, map[int]int{4: 4}, 3473, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Results) != 1 || plan.Results[0].Target != target || plan.MPAfter != 1071 {
		t.Fatalf("plan=%+v", plan)
	}
	if actor.MP != 1101 || actor.Acted || target.HP != 83 || center.HP != 877 {
		t.Fatal("planning mutated state")
	}
}

func TestNativeCommand6CursorAdmissionAndAtomicPreflight(t *testing.T) {
	book, err := LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                 string
		actor, cursor        Cell
		inactive, missingRaw bool
		want                 bool
	}{
		{"own-center", Cell{20, 22}, Cell{22, 20}, false, false, true},
		{"empty-center", Cell{20, 22}, Cell{21, 19}, false, false, true},
		{"direct-enemy", Cell{20, 20}, Cell{22, 18}, false, false, true},
		{"outside-selection", Cell{20, 22}, Cell{22, 17}, false, false, false},
		{"outside-grid", Cell{20, 22}, Cell{30, 20}, false, false, false},
		{"strict-radius-boundary", Cell{20, 22}, Cell{22, 21}, false, false, false},
		{"no-enemy-nearby", Cell{20, 22}, Cell{20, 22}, false, false, false},
		{"inactive-enemy", Cell{20, 22}, Cell{22, 20}, true, false, false},
		{"missing-raw", Cell{20, 22}, Cell{22, 20}, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actor := &Unit{Camp: Own, X: tc.actor.X, Y: tc.actor.Y, HP: 331, MP: 1101, OnField: true, HasNativeRecordByte5: true}
			center := &Unit{Camp: Own, X: 22, Y: 20, HP: 877, OnField: true, HasNativeRecordByte5: true}
			target := &Unit{Camp: Enemy, X: 22, Y: 18, HP: 83, ClassID: 4, OnField: true, HasNativeRecordByte5: !tc.missingRaw}
			if tc.inactive {
				target.NativeRecordByte5 = 1
			}
			st := &State{W: 30, H: 30, Units: []*Unit{actor, center, target}, NativeCommandBook: book, NativeCompositionEventBytes: make([]byte, 900)}
			beforeActor, beforeCenter, beforeTarget := *actor, *center, *target
			plan, err := st.PlanNativeCommand6DamageAtCursor(actor, tc.cursor, map[int]int{4: 4}, 3473, nil)
			if (err == nil) != tc.want {
				t.Fatalf("plan=%+v err=%v want=%v", plan, err, tc.want)
			}
			if !reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*center, beforeCenter) || !reflect.DeepEqual(*target, beforeTarget) {
				t.Fatal("preflight changed units")
			}
			if tc.want && (len(plan.Results) != 1 || plan.Results[0].Target != target) {
				t.Fatalf("targets=%+v", plan.Results)
			}
			if tc.want {
				called := false
				_, err = st.PlanNativeCommand6DamageAtCursor(actor, tc.cursor, nil, 3473, func(int, func(int, uint16) (uint16, bool, error)) (uint16, error) { called = true; return 0, nil })
				if err == nil || called {
					t.Fatal("missing resistance reached RNG walk")
				}
			}
		})
	}
}
