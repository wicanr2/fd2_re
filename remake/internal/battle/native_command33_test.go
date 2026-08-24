package battle

import "testing"

func nativeCompound33Fixture() (*State, *Unit, *Unit) {
	actor := &Unit{
		Camp: Own, OnField: true, X: 0, Y: 0, HP: 40, MaxHP: 100, MP: 60, Lv: 20,
		BattleFig: 4, HasBattleFig: true, NativeRecordClass: 19, HasNativeRecordClass: true,
		NativeTransient: [6]byte{1, 2, 3, 4, 5, 6},
	}
	target := &Unit{
		Camp: Own, OnField: true, X: 1, Y: 0, HP: 10, MaxHP: 120, Lv: 12,
		BattleFig: 5, HasBattleFig: true, NativeRecordClass: 2, HasNativeRecordClass: true,
		NativeTransient: [6]byte{7, 8, 9, 10, 11, 12},
	}
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for id := range book {
		book[id] = NativeCommandRecord{ID: id}
	}
	book[33] = NativeCommandRecord{ID: 33, SelectionMode: 5, EffectMode: 3, MPCost: 52, TargetCode: 1}
	return &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}, actor, target
}

func TestExecuteNativeCompoundCommand33ClearsRawMarkersAndRestoresAtomically(t *testing.T) {
	st, actor, target := nativeCompound33Fixture()
	result, err := st.ExecuteNativeCompoundCommand33(actor, target, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Restore.Results) != 2 || result.Restore.RNGState == 1 || actor.MP != 60 || !actor.Acted {
		t.Fatalf("command33 result=%#v actor=%#v", result, actor)
	}
	for index, unit := range []*Unit{actor, target} {
		if unit.HP != unit.MaxHP || unit.NativeTransient[3] != 0 || unit.NativeTransient[4] != 0 || unit.NativeTransient[5] != 0 {
			t.Fatalf("target %d did not publish restore/clear: %#v", index, unit)
		}
	}
}

func TestExecuteNativeCompoundCommand33RejectsIncompleteTargetWithoutMutation(t *testing.T) {
	st, actor, target := nativeCompound33Fixture()
	target.HasBattleFig = false
	actorHP, actorMP, actorActed, actorRaw := actor.HP, actor.MP, actor.Acted, actor.NativeTransient
	targetHP, targetRaw := target.HP, target.NativeTransient
	if _, err := st.ExecuteNativeCompoundCommand33(actor, target, 1); err == nil {
		t.Fatal("incomplete target provenance was accepted")
	}
	if actor.HP != actorHP || actor.MP != actorMP || actor.Acted != actorActed || actor.NativeTransient != actorRaw ||
		target.HP != targetHP || target.NativeTransient != targetRaw {
		t.Fatalf("failed preflight mutated actor=%#v target=%#v", actor, target)
	}
}

func TestExecuteNativeCompoundCommand33RejectsUnprovenPlayerSelector(t *testing.T) {
	st, actor, target := nativeCompound33Fixture()
	actor.BattleFig = 18
	if _, err := st.ExecuteNativeCompoundCommand33(actor, target, 1); err == nil || actor.Acted || actor.MP != 60 {
		t.Fatalf("unproven selector transaction actor=%#v err=%v", actor, err)
	}
}

func TestNativeCompoundCommand33PlanPublishesOnlyAtExplicitBoundary(t *testing.T) {
	st, actor, target := nativeCompound33Fixture()
	actorHP, actorRaw, targetHP, targetRaw := actor.HP, actor.NativeTransient, target.HP, target.NativeTransient
	plan, err := st.PlanNativeCompoundCommand33(actor, target, 1)
	if err != nil {
		t.Fatal(err)
	}
	if actor.HP != actorHP || actor.NativeTransient != actorRaw || target.HP != targetHP || target.NativeTransient != targetRaw || actor.Acted {
		t.Fatalf("plan mutated live state actor=%#v target=%#v", actor, target)
	}
	if err := PublishNativeCompoundCommand33(plan); err != nil {
		t.Fatal(err)
	}
	if actor.HP != actor.MaxHP || target.HP != target.MaxHP || actor.NativeTransient[3] != 0 || target.NativeTransient[5] != 0 || actor.Acted {
		t.Fatalf("publication boundary mismatch actor=%#v target=%#v", actor, target)
	}
	if err := CompleteNativeCompoundCommand33(plan); err != nil {
		t.Fatal(err)
	}
	if !actor.Acted {
		t.Fatal("completion did not publish Acted")
	}
}

func TestNativeCompoundCommand33AbortRestoresPublishedBatch(t *testing.T) {
	st, actor, target := nativeCompound33Fixture()
	actorHP, actorRaw, targetHP, targetRaw := actor.HP, actor.NativeTransient, target.HP, target.NativeTransient
	plan, err := st.PlanNativeCompoundCommand33(actor, target, 7)
	if err != nil {
		t.Fatal(err)
	}
	if err := PublishNativeCompoundCommand33(plan); err != nil {
		t.Fatal(err)
	}
	if err := AbortNativeCompoundCommand33(plan); err != nil {
		t.Fatal(err)
	}
	if actor.HP != actorHP || actor.NativeTransient != actorRaw || target.HP != targetHP || target.NativeTransient != targetRaw || actor.Acted {
		t.Fatalf("abort did not restore batch actor=%#v target=%#v", actor, target)
	}
}

func TestNativeCompoundCommand33ChangedTargetFailsBeforePublication(t *testing.T) {
	st, actor, target := nativeCompound33Fixture()
	plan, err := st.PlanNativeCompoundCommand33(actor, target, 9)
	if err != nil {
		t.Fatal(err)
	}
	target.HP++
	if err := PublishNativeCompoundCommand33(plan); err == nil || actor.HP != 40 || actor.Acted {
		t.Fatalf("changed target publication actor=%#v target=%#v err=%v", actor, target, err)
	}
}
