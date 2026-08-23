package battle

import "testing"

func nativeCompound35Fixture() (*State, *Unit, *Unit) {
	actor := &Unit{
		Camp: Own, OnField: true, X: 0, Y: 0, HP: 100, MaxHP: 100, MP: 40, Lv: 20,
		BattleFig: 4, HasBattleFig: true, NativeRecordClass: 19, HasNativeRecordClass: true,
	}
	target := &Unit{
		Camp: Own, OnField: true, X: 1, Y: 0, HP: 90, MaxHP: 90, Lv: 12,
		BattleFig: 5, HasBattleFig: true, NativeRecordClass: 2, HasNativeRecordClass: true,
	}
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for id := range book {
		book[id] = NativeCommandRecord{ID: id}
	}
	book[35] = NativeCommandRecord{ID: 35, SelectionMode: 5, EffectMode: 3, MPCost: 36, TargetCode: 1}
	return &State{
		W: 2, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book,
	}, actor, target
}

type nativeCompound35UnitState struct {
	hp, mp int
	acted  bool
	raw    [6]byte
}

func nativeCompound35State(unit *Unit) nativeCompound35UnitState {
	return nativeCompound35UnitState{unit.HP, unit.MP, unit.Acted, unit.NativeTransient}
}

func TestExecuteNativeCompoundCommand35PublishesThreeApplicationsAtomically(t *testing.T) {
	st, actor, target := nativeCompound35Fixture()
	result, err := st.ExecuteNativeCompoundCommand35(actor, target, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := []int{26, 22, 27}
	wantOffsets := []int{0x25, 0x27, 0x26}
	if len(result.Stages) != 3 || result.RNGState != 0x5541 {
		t.Fatalf("command35 stages=%#v rng=%#x", result.Stages, result.RNGState)
	}
	for stageIndex, stage := range result.Stages {
		if stage.CommandID != wantIDs[stageIndex] || stage.MarkerOffset != wantOffsets[stageIndex] ||
			len(stage.Results) != 2 || stage.Accumulator != 8*(actor.Lv+target.Lv) {
			t.Fatalf("command35 stage %d=%#v", stageIndex, stage)
		}
		for _, application := range stage.Results {
			if !application.Applied || application.Marker < 2 || application.Marker > 5 {
				t.Fatalf("command35 stage %d application=%#v", stageIndex, application)
			}
		}
	}
	if actor.MP != 40 || !actor.Acted || actor.HP >= 100 || target.HP >= 90 {
		t.Fatalf("command35 published actor=%#v target=%#v", actor, target)
	}
	for index, unit := range []*Unit{actor, target} {
		if unit.NativeTransient[3] == 0 || unit.NativeTransient[4] == 0 || unit.NativeTransient[5] == 0 {
			t.Fatalf("command35 target %d markers=%v", index, unit.NativeTransient)
		}
	}
}

func TestExecuteNativeCompoundCommand35RejectsIncompleteTargetAtomically(t *testing.T) {
	st, actor, target := nativeCompound35Fixture()
	target.HasBattleFig = false
	actorBefore, targetBefore := nativeCompound35State(actor), nativeCompound35State(target)
	if _, err := st.ExecuteNativeCompoundCommand35(actor, target, 1); err == nil {
		t.Fatal("command35 accepted incomplete target provenance")
	}
	if nativeCompound35State(actor) != actorBefore || nativeCompound35State(target) != targetBefore {
		t.Fatalf("failed command35 mutated actor=%#v target=%#v", actor, target)
	}
}

func TestExecuteNativeCompoundCommand35RejectsUnprovenSelectorAndMPGate(t *testing.T) {
	for _, mutate := range []func(*Unit){
		func(actor *Unit) { actor.BattleFig = 18 },
		func(actor *Unit) { actor.MP = 35 },
	} {
		st, actor, target := nativeCompound35Fixture()
		mutate(actor)
		actorBefore, targetBefore := nativeCompound35State(actor), nativeCompound35State(target)
		if _, err := st.ExecuteNativeCompoundCommand35(actor, target, 1); err == nil {
			t.Fatal("command35 accepted an unproven actor gate")
		}
		if nativeCompound35State(actor) != actorBefore || nativeCompound35State(target) != targetBefore {
			t.Fatalf("failed command35 gate mutated actor=%#v target=%#v", actor, target)
		}
	}
}
