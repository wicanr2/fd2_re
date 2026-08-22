package battle

import "testing"

func nativeCommand0TestState() (*State, *Unit, *Unit) {
	actor := &Unit{Camp: Own, X: 0, Y: 0, HP: 20, MP: 3, OnField: true}
	target := &Unit{Camp: Enemy, ClassID: 5, X: 1, Y: 0, HP: 100, OnField: true}
	book := make([]NativeCommandRecord, 36)
	for id := range book {
		book[id].ID = id
	}
	book[0] = NativeCommandRecord{ID: 0, Damage: 50, Hit: 100, SelectionMode: 1, EffectMode: 0, MPCost: 2, TargetCode: 0}
	return &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}, actor, target
}

func TestExecuteBoundNativeCommand0UsesTwoStageTargetsAndOneMPDebit(t *testing.T) {
	actor := &Unit{Camp: Own, X: 0, Y: 0, HP: 20, MP: 3, OnField: true}
	confirmed := &Unit{Camp: Enemy, ClassID: 5, X: 1, Y: 0, HP: 100, OnField: true}
	other := &Unit{Camp: Enemy, ClassID: 5, X: 2, Y: 0, HP: 100, OnField: true}
	st := &State{W: 3, H: 1, Units: []*Unit{actor, confirmed, other}, NativeCompositionEventBytes: make([]byte, 3), NativeCommandBook: []NativeCommandRecord{{ID: 0}}}
	// The executor requires the complete verified book rather than an invented
	// partial record; fill unused rows with exact sequential IDs for this unit
	// test, which only dispatches ID 0.
	for id := 1; id < 36; id++ {
		st.NativeCommandBook = append(st.NativeCommandBook, NativeCommandRecord{ID: id})
	}
	st.NativeCommandBook[0] = NativeCommandRecord{ID: 0, Damage: 50, Hit: 100, SelectionMode: 1, EffectMode: 0, MPCost: 2, TargetCode: 0}
	st.NativeCommandResistances = map[int]int{5: 10}
	results, state, err := st.ExecuteBoundNativeCommand0(actor, confirmed, 3)
	if err != nil || len(results) != 1 || results[0].Target != confirmed || !results[0].Hit {
		t.Fatalf("results=%+v state=%#x err=%v", results, state, err)
	}
	if actor.MP != 1 || !actor.Acted || confirmed.HP >= 100 || other.HP != 100 {
		t.Fatalf("mp/acted/hp actor=%d acted=%v confirmed=%d other=%d", actor.MP, actor.Acted, confirmed.HP, other.HP)
	}
}

func TestExecuteBoundNativeCommand0FailsBeforeMPOnMissingResistance(t *testing.T) {
	actor := &Unit{Camp: Own, X: 0, Y: 0, HP: 20, MP: 3, OnField: true}
	target := &Unit{Camp: Enemy, ClassID: 99, X: 1, Y: 0, HP: 100, OnField: true}
	book := make([]NativeCommandRecord, 36)
	for id := range book {
		book[id].ID = id
	}
	book[0] = NativeCommandRecord{ID: 0, Damage: 50, Hit: 100, SelectionMode: 1, EffectMode: 0, MPCost: 2, TargetCode: 0}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	if _, _, err := st.ExecuteBoundNativeCommand0(actor, target, 1); err == nil || actor.MP != 3 || actor.Acted || target.HP != 100 {
		t.Fatalf("missing resistance mutated state: mp=%d acted=%v hp=%d err=%v", actor.MP, actor.Acted, target.HP, err)
	}
}

func TestPlanBoundNativeCommand0PublishesSevenRecoveredHPStages(t *testing.T) {
	st, actor, target := nativeCommand0TestState()
	st.NativeCommandResistances = map[int]int{target.ClassID: 10}
	actor.MP = 8
	beforeActorMP, beforeTargetHP := actor.MP, target.HP
	plan, err := st.PlanBoundNativeCommand0(actor, target, 3)
	if err != nil {
		t.Fatal(err)
	}
	if actor.MP != beforeActorMP || actor.Acted || target.HP != beforeTargetHP {
		t.Fatalf("plan mutated state mp=%d acted=%v hp=%d", actor.MP, actor.Acted, target.HP)
	}
	if err := ApplyNativeCommandDamageMP(plan); err != nil {
		t.Fatal(err)
	}
	for stage := 1; stage <= 7; stage++ {
		if err := ApplyNativeCommandDamageStage(plan, 0, stage); err != nil {
			t.Fatalf("stage %d: %v", stage, err)
		}
		want := plan.Results[0].HPBefore - (plan.Results[0].HPBefore-plan.Results[0].HPAfter)*stage/7
		if target.HP != want {
			t.Fatalf("stage %d hp=%d want=%d", stage, target.HP, want)
		}
	}
	if err := CompleteNativeCommandDamage(plan); err != nil {
		t.Fatal(err)
	}
	if !actor.Acted || target.HP != plan.Results[0].HPAfter || actor.MP != plan.MPAfter {
		t.Fatalf("completion mp=%d acted=%v hp=%d plan=%+v", actor.MP, actor.Acted, target.HP, plan)
	}
}

func TestPlanBoundNativeCommand0RejectsChangedStateBetweenMarkers(t *testing.T) {
	st, actor, target := nativeCommand0TestState()
	st.NativeCommandResistances = map[int]int{target.ClassID: 10}
	plan, err := st.PlanBoundNativeCommand0(actor, target, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyNativeCommandDamageMP(plan); err != nil {
		t.Fatal(err)
	}
	target.HP--
	if err := ApplyNativeCommandDamageStage(plan, 0, 1); err == nil {
		t.Fatal("changed target state was accepted")
	}
}

func TestExecuteNativeCommandDamageAcceptsRecoveredIDOne(t *testing.T) {
	book := make([]NativeCommandRecord, 36)
	for id := range book {
		book[id].ID = id
	}
	book[1] = NativeCommandRecord{ID: 1, Damage: 120, Hit: 100, SelectionMode: 1, EffectMode: 0, MPCost: 1, TargetCode: 0}
	actor := &Unit{Camp: Own, X: 0, MP: 2, HP: 1, OnField: true}
	target := &Unit{Camp: Enemy, ClassID: 5, X: 1, HP: 200, OnField: true}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	if got, _, err := st.ExecuteNativeCommandDamage(actor, target, 1, map[int]int{5: 10}, 1); err != nil || len(got) != 1 || actor.MP != 1 || !actor.Acted || target.HP >= 200 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestExecuteNativeCommandDamageAcceptsRecoveredCompositorIDTen(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, HP: 20, MP: 3, X: 0, Y: 0}
	target := &Unit{Camp: Enemy, ClassID: 5, OnField: true, HP: 200, MaxHP: 200, X: 1, Y: 0}
	book := make([]NativeCommandRecord, 36)
	for id := range book {
		book[id] = NativeCommandRecord{ID: id}
	}
	book[10] = NativeCommandRecord{ID: 10, Damage: 100, Hit: 100, SelectionMode: 1, EffectMode: 0, MPCost: 1, TargetCode: 0}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}

	if got, _, err := st.ExecuteNativeCommandDamage(actor, target, 10, map[int]int{5: 10}, 1); err != nil || len(got) != 1 || !got[0].Hit || actor.MP != 2 || !actor.Acted || target.HP >= 200 {
		t.Fatalf("ID10 numeric route = %#v actor=%#v target=%#v err=%v", got, actor, target, err)
	}
}
