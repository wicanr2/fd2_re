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
	actor.Camp, actor.OnField, actor.MP, actor.NativeRecordByte6 = Enemy, true, 8, 1
	actor.X, actor.Y = 0, 0
	actor.NativeMapPresentation.X, actor.NativeMapPresentation.Y = 0, 0
	target := completeNativeAIScoringUnit()
	target.Camp, target.OnField, target.HP, target.MaxHP = Enemy, true, 20, 100
	target.X, target.Y = 1, 0
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

func TestPlanNativePlayerCommand20UsesConfirmedCursorTargets(t *testing.T) {
	st, actor, target := nativeAI2022State(20)
	st.NativeCommandBook[20].SelectionMode, st.NativeCommandBook[20].EffectMode, st.NativeCommandBook[20].TargetCode = 1, 0, 1
	actor.Camp = Own
	target.Camp = Own
	target.NativeTransient[3] = 3
	plan, err := st.PlanNativePlayerCommand2022(actor, target, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Targets) == 0 || plan.Targets[0] != target || len(plan.Results) == 0 {
		t.Fatalf("unexpected player targets/results: %#v", plan)
	}
	if target.NativeTransient[3] != 3 || actor.MP != 8 {
		t.Fatal("player planning mutated live state")
	}
	if err := PublishNativeAICommand2022(plan); err != nil {
		t.Fatal(err)
	}
	if target.NativeTransient[3] != 0 || actor.MP != 6 || actor.Acted {
		t.Fatalf("publish actor=%#v target=%#v", actor, target)
	}
}

func TestPlanNativePlayerCommand22UsesConfirmedCursorTargets(t *testing.T) {
	st, actor, target := nativeAI2022State(22)
	st.NativeCommandBook[22].SelectionMode, st.NativeCommandBook[22].EffectMode, st.NativeCommandBook[22].TargetCode = 1, 0, 0
	actor.Camp = Own
	target.Camp = Enemy
	plan, err := st.PlanNativePlayerCommand2022(actor, target, 22, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Targets) == 0 || plan.Targets[0] != target || len(plan.Results) == 0 || plan.Results[0].Apply == nil {
		t.Fatalf("unexpected player targets/results: %#v", plan)
	}
	if target.HP != 20 || actor.MP != 8 {
		t.Fatal("player planning mutated live state")
	}
	if err := PublishNativeAICommand2022(plan); err != nil {
		t.Fatal(err)
	}
	if actor.MP != 6 || actor.Acted {
		t.Fatalf("publish actor=%#v", actor)
	}
}

func TestPlanNativePlayerCommand25ClearsOnlyRawByte5AndRollsBack(t *testing.T) {
	st, actor, target := nativeAI2022State(25)
	st.NativeCommandBook[25].SelectionMode, st.NativeCommandBook[25].EffectMode, st.NativeCommandBook[25].TargetCode = 1, 0, 1
	actor.Camp, target.Camp = Own, Own
	target.NativeRecordByte5, target.HasNativeRecordByte5 = 0x80, true
	plan, err := st.PlanNativePlayerCommand2022(actor, target, 25, 0x1234)
	if err != nil || len(plan.Results) != 1 || !plan.Results[0].Command25 || !plan.Results[0].ClearedByte5 {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
	if target.NativeRecordByte5 != 0x80 || actor.MP != 8 || actor.Acted {
		t.Fatal("command25 planning mutated live state")
	}
	if err := PublishNativeAICommand2022(plan); err != nil {
		t.Fatal(err)
	}
	if target.NativeRecordByte5 != 0 || actor.MP != 6 || actor.Acted || plan.RNGState != 0x1234 {
		t.Fatalf("published actor=%#v target=%#v plan=%#v", actor, target, plan)
	}
	if err := AbortNativeAICommand2022(plan); err != nil {
		t.Fatal(err)
	}
	if target.NativeRecordByte5 != 0x80 || actor.MP != 8 || actor.Acted {
		t.Fatal("command25 rollback failed")
	}
}

func TestPlanNativeAICommand26UsesRawSelectorApplication(t *testing.T) {
	st, actor, target := nativeAI2022State(26)
	actor.NativeTransient[3], target.NativeTransient[3] = 0, 0
	plan, err := st.PlanNativeAICommand2022(actor, 26, 0)
	if err != nil || len(plan.Results) == 0 {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
	found := false
	for _, result := range plan.Results {
		if result.Target == target && result.Apply != nil && result.Offset == 0x25 {
			found = true
		}
	}
	if !found || target.NativeTransient[3] != 0 || target.HP != 20 || actor.MP != 8 {
		t.Fatalf("AI26 plan did not preserve raw-selector transaction: %#v", plan)
	}
}
