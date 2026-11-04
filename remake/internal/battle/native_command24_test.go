package battle

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

func TestPlanNativeCommand30MatchesStateOnlyClampAndDeath(t *testing.T) {
	for _, c := range []struct{ ap, dp, hp int }{{1, 100, 20}, {1000, 0, 5}} {
		t.Run(fmt.Sprintf("%d/%d/%d", c.ap, c.dp, c.hp), func(t *testing.T) {
			makeState := func() (*State, *Unit, *Unit) {
				a := &Unit{Camp: Own, OnField: true, X: 0, Y: 0, HP: 40, AP: c.ap, MP: 29}
				b := &Unit{Camp: Enemy, OnField: true, X: 1, Y: 0, HP: c.hp, MaxHP: c.hp, DP: c.dp, HasNativeRecordByte5: true}
				book := nativeCommand24Book()
				book[30] = NativeCommandRecord{ID: 30, SelectionMode: 0x14, MPCost: 24}
				return &State{W: 2, H: 1, Units: []*Unit{a, b}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}, a, b
			}
			st, a, b := makeState()
			control, ca, cb := makeState()
			rng, crng := rand.New(rand.NewSource(30)), rand.New(rand.NewSource(30))
			plan, err := st.PlanNativeCommand30(a, Cell{}, Cell{X: 1}, rng)
			if err != nil || len(plan.Results) != 1 || a.MP != 29 || a.Acted || b.HP != c.hp {
				t.Fatalf("planning mutation: %v", err)
			}
			want, err := control.ExecuteNativeCommand30(ca, Cell{}, Cell{X: 1}, crng)
			if err != nil || plan.Results[0].Damage != want[0].Damage || plan.Results[0].Amount != want[0].Amount {
				t.Fatalf("old state-only result changed: %v", err)
			}
			if err := ApplyNativeCommandDerivedStrikeMP(plan); err != nil {
				t.Fatal(err)
			}
			if err := ApplyNativeCommandDerivedStrikeTarget(plan, 0); err != nil {
				t.Fatal(err)
			}
			if err := CompleteNativeCommandDerivedStrike(plan); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(a, ca) || !reflect.DeepEqual(b, cb) || rng.Int63() != crng.Int63() {
				t.Fatal("planned final HP/MP/death or RNG differs from state-only control")
			}
		})
	}
}

func nativeCommand24Book() []NativeCommandRecord {
	book := make([]NativeCommandRecord, 36)
	for id := range book {
		book[id] = NativeCommandRecord{ID: id}
	}
	// Raw record24 is 00 00 00 05 01 16 00: selection=5, effect=1,
	// MP=22, target-code=0.  Damage/hit are intentionally unused here.
	book[24] = NativeCommandRecord{ID: 24, SelectionMode: 5, EffectMode: 1, MPCost: 22, TargetCode: 0}
	return book
}

func TestNativeCommand29FormalSelfCenterKeepsEnemyListAndTransaction(t *testing.T) {
	book, err := LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	if book[29].SelectionMode != 0 || book[29].EffectMode != 2 || book[29].MPCost != 26 {
		t.Fatal("formal command29 record changed")
	}
	actor := &Unit{Camp: Own, X: 2, Y: 2, HP: 180, MP: 40, AP: 120, OnField: true, HasNativeRecordByte5: true}
	near := &Unit{Camp: Enemy, X: 3, Y: 2, HP: 500, DP: 20, OnField: true, HasNativeRecordByte5: true}
	edge := &Unit{Camp: Enemy, X: 4, Y: 2, HP: 500, DP: 25, OnField: true, HasNativeRecordByte5: true}
	far := &Unit{Camp: Enemy, X: 5, Y: 2, HP: 500, OnField: true, HasNativeRecordByte5: true}
	friend := &Unit{Camp: Own, X: 2, Y: 3, HP: 180, OnField: true, HasNativeRecordByte5: true}
	inactive := &Unit{Camp: Enemy, X: 2, Y: 1, HP: 500, OnField: true, HasNativeRecordByte5: true, NativeRecordByte5: 1}
	st := &State{W: 7, H: 5, Units: []*Unit{actor, near, edge, far, friend, inactive}, NativeCommandBook: book, NativeCompositionEventBytes: make([]byte, 35)}
	var plan *NativeCommandDerivedStrikePlan
	for _, cursor := range []Cell{{2, 2}, {3, 2}, {2, 3}} {
		plan, err = st.PlanNativeCommand29AtCursor(actor, cursor, rand.New(rand.NewSource(7)))
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.Results) != 2 || plan.Results[0].Target != near || plan.Results[1].Target != edge || plan.MPAfter != 14 {
			t.Fatalf("confirmation cursor%v changed fixed list=%+v MP=%d", cursor, plan.Results, plan.MPAfter)
		}
	}
	if actor.MP != 40 || actor.Acted || near.HP != 500 || edge.HP != 500 {
		t.Fatal("planning changed live units")
	}
	if err := ApplyNativeCommandDerivedStrikeMP(plan); err != nil {
		t.Fatal(err)
	}
	for i := range plan.Results {
		if err := ApplyNativeCommandDerivedStrikeTarget(plan, i); err != nil {
			t.Fatal(err)
		}
	}
	if err := CompleteNativeCommandDerivedStrike(plan); err != nil {
		t.Fatal(err)
	}
	if actor.MP != 14 || !actor.Acted || near.HP >= 500 || edge.HP >= 500 || far.HP != 500 || friend.HP != 180 || inactive.HP != 500 {
		t.Fatal("self-center publication changed excluded units or missed final transaction")
	}
}

func TestNativeCommand29InvalidCursorKeepsUnitsAndRNG(t *testing.T) {
	book, err := LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name         string
		cursor       Cell
		missingRaw   bool
		inactive     bool
		insufficient bool
	}{
		{name: "outside-effect-field", cursor: Cell{3, 0}},
		{name: "outside-grid", cursor: Cell{-1, 0}},
		{name: "missing-raw", missingRaw: true},
		{name: "inactive-enemy", inactive: true},
		{name: "insufficient-mp", insufficient: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			actor := &Unit{Camp: Own, X: 0, Y: 0, HP: 180, MP: 40, AP: 120, OnField: true, HasNativeRecordByte5: true}
			target := &Unit{Camp: Enemy, X: 1, Y: 0, HP: 500, OnField: true, HasNativeRecordByte5: !c.missingRaw}
			if c.inactive {
				target.NativeRecordByte5 = 1
			}
			if c.insufficient {
				actor.MP = 25
			}
			beforeActor, beforeTarget := *actor, *target
			st := &State{W: 4, H: 1, Units: []*Unit{actor, target}, NativeCommandBook: book, NativeCompositionEventBytes: make([]byte, 4)}
			rng, control := rand.New(rand.NewSource(7)), rand.New(rand.NewSource(7))
			if _, err := st.PlanNativeCommand29AtCursor(actor, c.cursor, rng); err == nil {
				t.Fatal("invalid self-center admission accepted")
			}
			if !reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) || rng.Int63() != control.Int63() {
				t.Fatal("rejected admission changed units or RNG")
			}
		})
	}
	for _, id := range []int{-1, 36, 255} {
		st := &State{NativeCommandBook: book}
		if _, err := st.PlanNativeCommandDerivedStrike(nil, nil, id, rand.New(rand.NewSource(7))); err == nil {
			t.Fatalf("invalid command accepted: %d", id)
		}
	}
}

func TestExecuteNativeCommand24UsesDerivedStatsAndOneMPDebit(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, X: 0, Y: 0, AP: 100, MP: 30}
	target := &Unit{Camp: Enemy, OnField: true, X: 1, Y: 0, DP: 20, HP: 200}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: nativeCommand24Book()}

	got, err := st.ExecuteNativeCommand24(actor, target, rand.New(rand.NewSource(2)))
	if err != nil || len(got) != 1 || got[0].Target != target || got[0].Amount != 130 || got[0].Damage < 117 || got[0].Damage > 129 {
		t.Fatalf("ID24 result=%#v err=%v", got, err)
	}
	if target.HP != 200-got[0].Damage || actor.MP != 8 || !actor.Acted {
		t.Fatalf("ID24 state actor=%#v target=%#v result=%#v", actor, target, got[0])
	}
}

func TestExecuteNativeCommand24RejectsInvalidBookBeforeMutation(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, MP: 30}
	target := &Unit{Camp: Enemy, OnField: true, HP: 100}
	book := nativeCommand24Book()
	book[24].ID = 0
	st := &State{W: 1, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: []byte{0}, NativeCommandBook: book}
	if _, err := st.ExecuteNativeCommand24(actor, target, rand.New(rand.NewSource(1))); err == nil || actor.MP != 30 || actor.Acted || target.HP != 100 {
		t.Fatalf("invalid ID24 book mutated state actor=%#v target=%#v err=%v", actor, target, err)
	}
}

func TestPlanNativeCommand24PublishesMPDamageAndCompletionAtSeparateMarkers(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, X: 0, Y: 0, AP: 100, MP: 30}
	target := &Unit{Camp: Enemy, OnField: true, X: 1, Y: 0, DP: 20, HP: 200}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: nativeCommand24Book()}

	plan, err := st.PlanNativeCommandDerivedStrike(actor, target, 24, rand.New(rand.NewSource(2)))
	if err != nil || len(plan.Results) != 1 {
		t.Fatalf("ID24 plan=%#v err=%v", plan, err)
	}
	if actor.MP != 30 || actor.Acted || target.HP != 200 {
		t.Fatalf("planning mutated public state actor=%#v target=%#v", actor, target)
	}
	if err := ApplyNativeCommandDerivedStrikeMP(plan); err != nil {
		t.Fatal(err)
	}
	if actor.MP != 8 || actor.Acted || target.HP != 200 {
		t.Fatalf("actor marker published wrong state actor=%#v target=%#v", actor, target)
	}
	if err := ApplyNativeCommandDerivedStrikeTarget(plan, 0); err != nil {
		t.Fatal(err)
	}
	if target.HP != plan.Results[0].HPAfter || actor.Acted {
		t.Fatalf("target marker published wrong state actor=%#v target=%#v", actor, target)
	}
	if err := CompleteNativeCommandDerivedStrike(plan); err != nil || !actor.Acted {
		t.Fatalf("completion err=%v actor=%#v", err, actor)
	}
}

func TestPlanNativeCommand24PreservesDamageClampAndDeathProjection(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, X: 0, Y: 0, AP: 100, MP: 30}
	target := &Unit{Camp: Enemy, OnField: true, X: 1, Y: 0, HP: 5, MaxHP: 5,
		HasNativeRecordByte5: true, NativeRecordByte5: 0}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: nativeCommand24Book()}

	plan, err := st.PlanNativeCommandDerivedStrike(actor, target, 24, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Results[0].Damage; got != 5 {
		t.Fatalf("clamped damage=%d want 5", got)
	}
	if err := ApplyNativeCommandDerivedStrikeMP(plan); err != nil {
		t.Fatal(err)
	}
	if err := ApplyNativeCommandDerivedStrikeTarget(plan, 0); err != nil {
		t.Fatal(err)
	}
	if target.HP != 0 || target.NativeRecordByte5 != 1 {
		t.Fatalf("death projection HP=%d raw+5=%02x", target.HP, target.NativeRecordByte5)
	}
}

func TestPlanNativeCommand24DoesNotTurnNegativeDamageIntoHealing(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, X: 0, Y: 0, AP: 1, MP: 30}
	target := &Unit{Camp: Enemy, OnField: true, X: 1, Y: 0, DP: 100, HP: 20, MaxHP: 20}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: nativeCommand24Book()}

	plan, err := st.PlanNativeCommandDerivedStrike(actor, target, 24, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	if result := plan.Results[0]; result.Damage != 0 || result.HPAfter != 20 {
		t.Fatalf("negative raw damage changed HP: %#v", result)
	}
}

func TestExecuteNativeCommandDerivedStrikeUsesRecoveredID28MultiplierAndDenominator(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, X: 0, Y: 0, AP: 100, MP: 30, BattleFig: 19, HasBattleFig: true}
	target := &Unit{Camp: Enemy, OnField: true, X: 1, Y: 0, DP: 20, HP: 300}
	book := nativeCommand24Book()
	book[28] = NativeCommandRecord{ID: 28, SelectionMode: 1, EffectMode: 0, MPCost: 22, TargetCode: 0}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}

	got, err := st.ExecuteNativeCommandDerivedStrike(actor, target, 28, rand.New(rand.NewSource(2)))
	if err != nil || len(got) != 1 || got[0].Amount != 180 || got[0].Damage < 162 || got[0].Damage > 179 {
		t.Fatalf("ID28 result=%#v err=%v", got, err)
	}
	if actor.MP != 8 || !actor.Acted || target.HP != 300-got[0].Damage {
		t.Fatalf("ID28 state actor=%#v target=%#v result=%#v", actor, target, got[0])
	}
}

func TestNativeCommand28EightStagesClampBeforeDivision(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, X: 0, Y: 0, AP: 100, MP: 30, BattleFig: 19, HasBattleFig: true, HasNativeRecordByte5: true}
	target := &Unit{Camp: Enemy, OnField: true, X: 1, Y: 0, DP: 20, HP: 7, HasNativeRecordByte5: true}
	book := nativeCommand24Book()
	book[28] = NativeCommandRecord{ID: 28, SelectionMode: 1, EffectMode: 0, MPCost: 22, TargetCode: 0}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	plan, err := st.PlanNativeCommandDerivedStrike(actor, target, 28, rand.New(rand.NewSource(2)))
	if err != nil || plan.TargetImpactCount != 8 || plan.Results[0].Damage != 7 || target.HP != 7 || actor.MP != 30 {
		t.Fatalf("capped stage plan=%+v err=%v", plan, err)
	}
	if err := ApplyNativeCommand28TargetStage(plan, 0, 1); err == nil || target.HP != 7 {
		t.Fatal("target impact preceded MP")
	}
	if err := ApplyNativeCommandDerivedStrikeMP(plan); err != nil {
		t.Fatal(err)
	}
	if err := ApplyNativeCommand28TargetStage(plan, 0, 2); err == nil || target.HP != 7 {
		t.Fatal("target impact skipped a stage")
	}
	for stage := 1; stage <= 8; stage++ {
		if err := ApplyNativeCommand28TargetStage(plan, 0, stage); err != nil {
			t.Fatal(err)
		}
		if target.HP != 7-7*stage/8 || actor.MP != 8 || actor.Acted {
			t.Fatalf("stage%d: HP%d MP%d acted%v", stage, target.HP, actor.MP, actor.Acted)
		}
		if err := ApplyNativeCommand28TargetStage(plan, 0, stage); err == nil {
			t.Fatal("duplicate marker published twice")
		}
		if stage < 8 {
			if target.NativeRecordByte5 != 0 || CompleteNativeCommandDerivedStrike(plan) == nil {
				t.Fatal("command28 completed or killed target before eight impacts")
			}
		}
	}
	if err := CompleteNativeCommandDerivedStrike(plan); err != nil || !actor.Acted || target.HP != 0 || target.NativeRecordByte5 != 1 {
		t.Fatalf("final stage did not complete: %v", err)
	}
}

func TestNativeCommand28UnprovenSelectorRejectsBeforeRNG(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, AP: 100, MP: 30, BattleFig: 20, HasBattleFig: true}
	target := &Unit{Camp: Enemy, OnField: true, X: 1, HP: 300}
	book := nativeCommand24Book()
	book[28] = NativeCommandRecord{ID: 28, SelectionMode: 1, MPCost: 22}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	rng, control := rand.New(rand.NewSource(2)), rand.New(rand.NewSource(2))
	if _, err := st.PlanNativeCommandDerivedStrike(actor, target, 28, rng); err == nil || actor.MP != 30 || actor.Acted || target.HP != 300 || rng.Int63() != control.Int63() {
		t.Fatal("unproven command28 source reached RNG or state publication")
	}
}

func TestExecuteNativeCommandDerivedStrikeUsesRecoveredID29And31Multipliers(t *testing.T) {
	for _, tc := range []struct {
		id         int
		multiplier int
	}{
		{id: 29, multiplier: 12},
		{id: 31, multiplier: 18},
	} {
		t.Run(fmt.Sprintf("id%d", tc.id), func(t *testing.T) {
			actor := &Unit{Camp: Own, OnField: true, X: 0, Y: 0, AP: 100, MP: 40}
			target := &Unit{Camp: Enemy, OnField: true, X: 1, Y: 0, DP: 20, HP: 300}
			book := nativeCommand24Book()
			book[tc.id] = NativeCommandRecord{ID: tc.id, SelectionMode: 1, EffectMode: 0, MPCost: 26, TargetCode: 0}
			st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}

			got, err := st.ExecuteNativeCommandDerivedStrike(actor, target, tc.id, rand.New(rand.NewSource(2)))
			wantAmount := actor.AP*tc.multiplier/10 - target.DP
			if err != nil || len(got) != 1 || got[0].Amount != wantAmount || got[0].Damage <= 0 {
				t.Fatalf("result=%#v err=%v want amount=%d", got, err, wantAmount)
			}
			if actor.MP != 14 || !actor.Acted || target.HP != 300-got[0].Damage {
				t.Fatalf("state actor=%#v target=%#v result=%#v", actor, target, got[0])
			}
		})
	}
}

func TestExecuteNativeCommandDerivedStrikeRejectsSpecialSelectorID30(t *testing.T) {
	actor := &Unit{MP: 30}
	if _, err := (&State{}).ExecuteNativeCommandDerivedStrike(actor, nil, 30, rand.New(rand.NewSource(1))); err == nil || actor.MP != 30 || actor.Acted {
		t.Fatalf("generic derived-strike executor must reject ID30 special selector: actor=%#v err=%v", actor, err)
	}
}

func TestNativeCommand30TargetsUsesSavedCursorAndXPriority(t *testing.T) {
	xLine := &Unit{Camp: Enemy, OnField: true, HP: 1, X: 1, Y: 0}
	yDiagonal := &Unit{Camp: Enemy, OnField: true, HP: 1, X: 0, Y: 1}
	got, err := NativeCommand30Targets(5, 3, Cell{X: 0, Y: 0}, Cell{X: 2, Y: 2}, 4, []*Unit{xLine, yDiagonal})
	if err != nil || len(got) != 1 || got[0] != xLine {
		t.Fatalf("ID30 X-priority targets=%#v err=%v", got, err)
	}
	plusY := &Unit{Camp: Enemy, OnField: true, HP: 1, X: 1, Y: 2}
	got, err = NativeCommand30Targets(3, 4, Cell{X: 1, Y: 1}, Cell{X: 1, Y: 1}, 1, []*Unit{plusY})
	if err != nil || len(got) != 1 || got[0] != plusY {
		t.Fatalf("ID30 equal-cursor must follow native +Y branch: targets=%#v err=%v", got, err)
	}
}

func TestExecuteNativeCommand30UsesSpecialLineAndOneMPDebit(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, X: 0, Y: 0, AP: 100, MP: 30}
	between := &Unit{Camp: Enemy, OnField: true, X: 2, Y: 0, DP: 20, HP: 300}
	confirmed := &Unit{Camp: Enemy, OnField: true, X: 4, Y: 0, DP: 20, HP: 300}
	friendly := &Unit{Camp: Own, OnField: true, X: 3, Y: 0, DP: 20, HP: 300}
	book := nativeCommand24Book()
	book[30] = NativeCommandRecord{ID: 30, SelectionMode: 20, EffectMode: 0, MPCost: 24, TargetCode: 0}
	st := &State{W: 5, H: 1, Units: []*Unit{actor, between, confirmed, friendly}, NativeCompositionEventBytes: make([]byte, 5), NativeCommandBook: book}

	got, err := st.ExecuteNativeCommand30(actor, Cell{X: 0, Y: 0}, Cell{X: 4, Y: 0}, rand.New(rand.NewSource(2)))
	if err != nil || len(got) != 2 || got[0].Target != between || got[1].Target != confirmed || got[0].Amount != 160 || got[1].Amount != 160 {
		t.Fatalf("ID30 result=%#v err=%v", got, err)
	}
	if actor.MP != 6 || !actor.Acted || friendly.HP != 300 || between.HP >= 300 || confirmed.HP >= 300 {
		t.Fatalf("ID30 mutation actor=%#v between=%#v confirmed=%#v friendly=%#v", actor, between, confirmed, friendly)
	}
}

func TestExecuteNativeCommand30RejectsNonCandidateBeforeMP(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, X: 0, Y: 0, AP: 100, MP: 30}
	target := &Unit{Camp: Enemy, OnField: true, X: 4, Y: 0, DP: 20, HP: 300}
	book := nativeCommand24Book()
	book[30] = NativeCommandRecord{ID: 30, SelectionMode: 20, MPCost: 24, TargetCode: 0}
	st := &State{W: 5, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 5), NativeCommandBook: book}
	if _, err := st.ExecuteNativeCommand30(actor, Cell{X: 0, Y: 0}, Cell{X: 1, Y: 0}, rand.New(rand.NewSource(2))); err == nil || actor.MP != 30 || actor.Acted || target.HP != 300 {
		t.Fatalf("invalid ID30 confirmation mutated state actor=%#v target=%#v err=%v", actor, target, err)
	}
}
