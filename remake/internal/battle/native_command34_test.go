package battle

import "testing"

func nativeCompound34Fixture() (*State, *Unit, *Unit) {
	actor := &Unit{
		Camp: Own, OnField: true, X: 0, Y: 0, HP: 100, MaxHP: 100, MP: 30, AP: 100, DP: 80, HIT: 60, EV: 50, Lv: 20,
		BattleFig: 4, HasBattleFig: true, NativeRecordClass: 19, HasNativeRecordClass: true,
	}
	target := &Unit{
		Camp: Own, OnField: true, X: 1, Y: 0, HP: 90, MaxHP: 90, MP: 0, AP: 70, DP: 60, HIT: 50, EV: 40, Lv: 12,
		BattleFig: 5, HasBattleFig: true, NativeRecordClass: 2, HasNativeRecordClass: true,
	}
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for id := range book {
		book[id] = NativeCommandRecord{ID: id}
	}
	book[34] = NativeCommandRecord{ID: 34, SelectionMode: 5, EffectMode: 3, MPCost: 28, TargetCode: 1}
	return &State{
		W: 2, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book,
	}, actor, target
}

type nativeCompound34UnitState struct {
	mp, ap, dp, hit, ev int
	acted               bool
	transient           [6]byte
}

func nativeCompound34State(unit *Unit) nativeCompound34UnitState {
	return nativeCompound34UnitState{unit.MP, unit.AP, unit.DP, unit.HIT, unit.EV, unit.Acted, unit.NativeTransient}
}

func TestExecuteNativeCompoundCommand34PublishesThreeStagesAtomically(t *testing.T) {
	st, actor, target := nativeCompound34Fixture()
	result, err := st.ExecuteNativeCompoundCommand34(actor, target, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stages) != 3 || result.Stages[0].CommandID != 17 ||
		result.Stages[1].CommandID != 18 || result.Stages[2].CommandID != 19 || result.RNGState == 1 {
		t.Fatalf("compound stages=%#v rng=%#x", result.Stages, result.RNGState)
	}
	if actor.MP != 30 || !actor.Acted {
		t.Fatalf("reachable class-19 MP contract actor=%#v", actor)
	}
	for index, unit := range []*Unit{actor, target} {
		if unit.NativeTransient[0] == 0 || unit.NativeTransient[1] == 0 || unit.NativeTransient[2] == 0 ||
			unit.AP <= []int{100, 70}[index] || unit.DP <= []int{80, 60}[index] ||
			unit.HIT <= []int{60, 50}[index] || unit.EV <= []int{50, 40}[index] {
			t.Fatalf("target %d did not receive all raw stages: %#v", index, unit)
		}
	}
}

func TestExecuteNativeCompoundCommand34RejectsUnprovenSelectorWithoutMutation(t *testing.T) {
	st, actor, target := nativeCompound34Fixture()
	actor.BattleFig = 18
	actorBefore, targetBefore := nativeCompound34State(actor), nativeCompound34State(target)
	if _, err := st.ExecuteNativeCompoundCommand34(actor, target, 1); err == nil {
		t.Fatal("unproven compound selector was accepted")
	}
	if nativeCompound34State(actor) != actorBefore || nativeCompound34State(target) != targetBefore {
		t.Fatalf("failed selector gate mutated state actor=%#v target=%#v", actor, target)
	}
}

func TestExecuteNativeCompoundCommand34PreflightsEveryTargetBeforeMutation(t *testing.T) {
	st, actor, target := nativeCompound34Fixture()
	target.HasNativeRecordClass = false
	actorBefore, targetBefore := nativeCompound34State(actor), nativeCompound34State(target)
	if _, err := st.ExecuteNativeCompoundCommand34(actor, target, 1); err == nil {
		t.Fatal("incomplete final target provenance was accepted")
	}
	if nativeCompound34State(actor) != actorBefore || nativeCompound34State(target) != targetBefore {
		t.Fatalf("failed target preflight mutated state actor=%#v target=%#v", actor, target)
	}
}
