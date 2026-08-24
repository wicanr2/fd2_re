package battle

import "testing"

func nativeCompound32Fixture() (*State, *Unit, *Unit) {
	actor := &Unit{
		Camp: Own, OnField: true, X: 0, Y: 0, HP: 100, MaxHP: 100, MP: 80, Lv: 20,
		ClassID: 5, BattleFig: 4, HasBattleFig: true, NativeRecordClass: 19, HasNativeRecordClass: true,
	}
	target := &Unit{
		Camp: Enemy, OnField: true, X: 1, Y: 0, HP: 100, MaxHP: 100, Lv: 12,
		ClassID: 5, BattleFig: 5, HasBattleFig: true, NativeRecordClass: 2, HasNativeRecordClass: true,
	}
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for id := range book {
		book[id] = NativeCommandRecord{ID: id}
	}
	book[32] = NativeCommandRecord{ID: 32, Damage: 50, Hit: 100, SelectionMode: 1, EffectMode: 0, MPCost: 76, TargetCode: 0}
	return &State{
		W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2),
		NativeCommandBook: book, NativeCommandResistances: map[int]int{5: 10},
	}, actor, target
}

func TestExecuteNativeCompoundCommand32PublishesDamageAtomically(t *testing.T) {
	st, actor, target := nativeCompound32Fixture()
	result, err := st.ExecuteNativeCompoundCommand32(actor, target, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Targets) != 1 || !result.Targets[0].Damage.Hit || result.Targets[0].HPBefore != 100 ||
		result.Targets[0].HPAfter >= 100 || target.HP != result.Targets[0].HPAfter ||
		result.RNGState == 7 || actor.MP != 80 || !actor.Acted {
		t.Fatalf("command32 result=%+v actor=%#v target=%#v", result, actor, target)
	}
	if target.NativeTransient != [6]byte{} {
		t.Fatalf("command32 invented raw markers: %v", target.NativeTransient)
	}
}

func TestExecuteNativeCompoundCommand32MissConsumesOnlyHitRoll(t *testing.T) {
	st, actor, target := nativeCompound32Fixture()
	st.NativeCommandBook[32].Hit = 0
	result, err := st.ExecuteNativeCompoundCommand32(actor, target, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, wantState, err := ResolveNativeCommandDamage(50, 0, 10, 1)
	if err != nil || len(result.Targets) != 1 || result.Targets[0].Damage.Hit || target.HP != 100 || result.RNGState != wantState {
		t.Fatalf("command32 miss=%+v target=%#v want_rng=%#x", result, target, wantState)
	}
}

func TestExecuteNativeCompoundCommand32RejectsMissingProvenanceAtomically(t *testing.T) {
	for _, mutate := range []func(*State, *Unit, *Unit){
		func(_ *State, actor, _ *Unit) { actor.BattleFig = 18 },
		func(_ *State, actor, _ *Unit) { actor.MP = 75 },
		func(st *State, _, target *Unit) { delete(st.NativeCommandResistances, target.ClassID) },
		func(_ *State, _, target *Unit) { target.HasNativeRecordClass = false },
	} {
		st, actor, target := nativeCompound32Fixture()
		mutate(st, actor, target)
		actorHP, actorMP, actorActed, targetHP := actor.HP, actor.MP, actor.Acted, target.HP
		if _, err := st.ExecuteNativeCompoundCommand32(actor, target, 1); err == nil {
			t.Fatal("command32 accepted an unproven input")
		}
		if actor.HP != actorHP || actor.MP != actorMP || actor.Acted != actorActed || target.HP != targetHP {
			t.Fatalf("failed command32 mutated actor=%#v target=%#v", actor, target)
		}
	}
}

func TestPlanNativeCompoundCommand32DoesNotPublishState(t *testing.T) {
	st, actor, target := nativeCompound32Fixture()
	plan, err := st.PlanNativeCompoundCommand32(actor, target, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Result.Targets) != 1 || target.HP != 100 || actor.Acted || actor.MP != 80 {
		t.Fatalf("command32 plan published state: plan=%+v actor=%#v target=%#v", plan, actor, target)
	}
	if err := ApplyNativeCompoundCommand32Target(plan, 0); err != nil {
		t.Fatal(err)
	}
	if target.HP != plan.Result.Targets[0].HPAfter || actor.Acted {
		t.Fatalf("command32 target boundary mismatch: plan=%+v actor=%#v target=%#v", plan, actor, target)
	}
	if err := CompleteNativeCompoundCommand32(plan); err != nil {
		t.Fatal(err)
	}
	if !actor.Acted || actor.MP != 80 {
		t.Fatalf("command32 completion mismatch: actor=%#v", actor)
	}
}

func TestNativeCompoundCommand32PlanFailsClosedAcrossBoundaries(t *testing.T) {
	t.Run("complete before target", func(t *testing.T) {
		st, actor, target := nativeCompound32Fixture()
		plan, err := st.PlanNativeCompoundCommand32(actor, target, 7)
		if err != nil {
			t.Fatal(err)
		}
		if err := CompleteNativeCompoundCommand32(plan); err == nil {
			t.Fatal("command32 completed before publishing its target")
		}
		if target.HP != 100 || actor.Acted {
			t.Fatalf("failed completion mutated actor=%#v target=%#v", actor, target)
		}
	})

	t.Run("target changed", func(t *testing.T) {
		st, actor, target := nativeCompound32Fixture()
		plan, err := st.PlanNativeCompoundCommand32(actor, target, 7)
		if err != nil {
			t.Fatal(err)
		}
		target.HP--
		if err := ApplyNativeCompoundCommand32Target(plan, 0); err == nil {
			t.Fatal("command32 published over changed target state")
		}
		if target.HP != 99 || actor.Acted {
			t.Fatalf("failed publication changed actor=%#v target=%#v", actor, target)
		}
	})

	t.Run("abort restores published target", func(t *testing.T) {
		st, actor, target := nativeCompound32Fixture()
		plan, err := st.PlanNativeCompoundCommand32(actor, target, 7)
		if err != nil {
			t.Fatal(err)
		}
		plan.Result.Targets[0].HPAfter = 1 // Public report is not transaction authority.
		if err := ApplyNativeCompoundCommand32Target(plan, 0); err != nil {
			t.Fatal(err)
		}
		if target.HP == 1 {
			t.Fatal("command32 trusted a caller-mutated public result")
		}
		if err := AbortNativeCompoundCommand32(plan); err != nil {
			t.Fatal(err)
		}
		if target.HP != 100 || actor.Acted {
			t.Fatalf("command32 rollback mismatch: actor=%#v target=%#v", actor, target)
		}
	})

	t.Run("repeat and out of range publication", func(t *testing.T) {
		st, actor, target := nativeCompound32Fixture()
		plan, err := st.PlanNativeCompoundCommand32(actor, target, 7)
		if err != nil {
			t.Fatal(err)
		}
		if err := ApplyNativeCompoundCommand32Target(plan, -1); err == nil {
			t.Fatal("command32 accepted negative target index")
		}
		if err := ApplyNativeCompoundCommand32Target(plan, 0); err != nil {
			t.Fatal(err)
		}
		publishedHP := target.HP
		if err := ApplyNativeCompoundCommand32Target(plan, 0); err == nil {
			t.Fatal("command32 published a target twice")
		}
		if target.HP != publishedHP || actor.Acted {
			t.Fatalf("repeated publication changed actor=%#v target=%#v", actor, target)
		}
	})
}
