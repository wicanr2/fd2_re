package battle

import "testing"

func nativeAI2022Book(id int) []NativeCommandRecord {
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for i := range book {
		book[i] = NativeCommandRecord{ID: i}
	}
	book[10] = NativeCommandRecord{ID: 10, Damage: 10}
	book[id] = NativeCommandRecord{ID: id, EffectMode: 1, MPCost: 2, TargetCode: 1}
	return book
}

func nativeAI2022State(id int) (*State, *Unit, *Unit) {
	actor := completeNativeAIScoringUnit()
	actor.Camp, actor.MP, actor.NativeRecordByte6 = Enemy, 8, 1
	actor.NativeMapPresentation.X, actor.NativeMapPresentation.Y = 0, 0
	target := completeNativeAIScoringUnit()
	target.Camp, target.HP, target.MaxHP = Enemy, 20, 100
	target.NativeMapPresentation.X, target.NativeMapPresentation.Y = 1, 0
	return &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: nativeAI2022Book(id)}, actor, target
}

func TestPlanNativeAICommand20UsesRawSelectorAndRollback(t *testing.T) {
	st, actor, target := nativeAI2022State(20)
	target.NativeTransient[3] = 3
	plan, err := st.PlanNativeAICommand2022(actor, 20, 0)
	if err != nil || len(plan.Targets) != 2 || len(plan.Results) != 2 || plan.Results[1].Restore == nil {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
	if target.HP != 20 || target.NativeTransient[3] != 3 || actor.MP != 8 {
		t.Fatal("planning mutated live state")
	}
	if err := PublishNativeAICommand2022(plan); err != nil {
		t.Fatal(err)
	}
	if target.NativeTransient[3] != 0 || actor.MP != 6 || actor.Acted {
		t.Fatalf("publish actor=%#v target=%#v", actor, target)
	}
	if err := AbortNativeAICommand2022(plan); err != nil {
		t.Fatal(err)
	}
	if target.HP != 20 || target.NativeTransient[3] != 3 || actor.MP != 8 {
		t.Fatal("rollback failed")
	}
}

func TestPlanNativeAICommand22UsesNativeApplicationAndCompletes(t *testing.T) {
	st, actor, target := nativeAI2022State(22)
	plan, err := st.PlanNativeAICommand2022(actor, 22, 0)
	if err != nil || len(plan.Results) == 0 || plan.Results[0].Apply == nil {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
	if err := PublishNativeAICommand2022(plan); err != nil {
		t.Fatal(err)
	}
	if err := CompleteNativeAICommand2022(plan); err != nil {
		t.Fatal(err)
	}
	if actor.MP != 6 || !actor.Acted {
		t.Fatalf("actor=%#v", actor)
	}
	applied := false
	for _, result := range plan.Results {
		if result.Apply != nil && result.Apply.Applied {
			applied = true
		}
	}
	if !applied || target.HP > 20 {
		t.Fatalf("application did not publish plan=%#v target=%#v", plan, target)
	}
}
