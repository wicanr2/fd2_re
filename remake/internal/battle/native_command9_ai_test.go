package battle

import "testing"

func TestPlanNativeAICommand9UsesProducerTargetAndTwentyStages(t *testing.T) {
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[9] = NativeCommandRecord{ID: 9, Damage: 999, Hit: 100, SelectionMode: 3, EffectMode: 0, MPCost: 30, TargetCode: 0}
	actor := &Unit{Camp: Enemy, HP: 50, MP: 40, OnField: true}
	target := &Unit{Camp: Own, ClassID: 5, HP: 500, OnField: true}
	st := &State{NativeCommandBook: book, Units: []*Unit{actor, target}}
	plan, err := st.PlanNativeAICommandDamageSingleTarget(actor, target, 9, map[int]int{5: 10}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if plan.DamageStages != 20 || len(plan.Results) != 1 || plan.Results[0].Target != target || actor.MP != 40 || target.HP != 500 {
		t.Fatalf("AI command9 plan=%#v actorMP=%d targetHP=%d", plan, actor.MP, target.HP)
	}
}

func TestPlanNativeAICommand9RejectsPlayerAndAreaRecords(t *testing.T) {
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[9] = NativeCommandRecord{ID: 9, Damage: 999, Hit: 100, EffectMode: 1, MPCost: 30, TargetCode: 0}
	actor := &Unit{Camp: Enemy, HP: 50, MP: 40, OnField: true}
	target := &Unit{Camp: Own, ClassID: 5, HP: 500, OnField: true}
	st := &State{NativeCommandBook: book}
	if _, err := st.PlanNativeAICommandDamageSingleTarget(actor, target, 9, map[int]int{5: 10}, 3); err == nil {
		t.Fatal("unproven area record was accepted")
	}
	actor.Camp = Own
	book[9].EffectMode = 0
	if _, err := st.PlanNativeAICommandDamageSingleTarget(actor, target, 9, map[int]int{5: 10}, 3); err == nil {
		t.Fatal("player actor was accepted by AI planner")
	}
}
