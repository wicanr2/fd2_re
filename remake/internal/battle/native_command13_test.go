package battle

import (
	"math/rand"
	"testing"
)

func nativeCommandHealBook(id int) []NativeCommandRecord {
	book := make([]NativeCommandRecord, 36)
	for i := range book {
		book[i] = NativeCommandRecord{ID: i}
	}
	book[id] = NativeCommandRecord{ID: id, Damage: 70, SelectionMode: 1, EffectMode: 0, MPCost: 3, TargetCode: 1}
	return book
}

func TestExecuteNativeCommandHealUsesSelectedRecordAndCapsHP(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, HP: 20, MP: 5, X: 0, Y: 0}
	target := &Unit{Camp: Own, OnField: true, HP: 40, MaxHP: 100, X: 1, Y: 0}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: nativeCommandHealBook(13)}

	got, err := st.ExecuteNativeCommandHeal(actor, target, 13, rand.New(rand.NewSource(2)))
	if err != nil || len(got) != 1 || got[0].Target != target || got[0].Restore.Rolled < 63 || got[0].Restore.Rolled > 69 {
		t.Fatalf("heal = %#v, %v", got, err)
	}
	if target.HP != 100 || got[0].Restore.Actual != 60 || actor.MP != 2 || !actor.Acted {
		t.Fatalf("post heal actor=%#v target=%#v result=%#v", actor, target, got[0])
	}
}

func TestExecuteNativeCommandHealRejectsFamilyBoundaryBeforeMutation(t *testing.T) {
	actor := &Unit{MP: 5}
	if _, err := (&State{}).ExecuteNativeCommandHeal(actor, nil, 12, rand.New(rand.NewSource(1))); err == nil || actor.MP != 5 || actor.Acted {
		t.Fatalf("non-heal ID must fail closed: actor=%#v err=%v", actor, err)
	}
}

func TestExecuteNativeAICommandHealRebuildsTargetsFromRawSelector(t *testing.T) {
	actor := completeNativeAIScoringUnit()
	actor.NativeMapPresentation.X, actor.NativeMapPresentation.Y = 0, 0
	actor.NativeRecordByte5, actor.NativeRecordByte6 = 0, 0
	actor.HP, actor.MaxHP, actor.MP = 100, 100, 5
	target := completeNativeAIScoringUnit()
	target.NativeMapPresentation.X, target.NativeMapPresentation.Y = 1, 0
	target.NativeRecordByte5, target.NativeRecordByte6 = 0, 0
	target.HP, target.MaxHP = 40, 100
	book := nativeCommandHealBook(13)
	book[13].EffectMode = 1
	st := &State{
		W: 2, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book,
	}

	got, err := st.ExecuteNativeAICommandHeal(actor, 13, rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Target != actor || got[1].Target != target {
		t.Fatalf("AI target array=%#v", got)
	}
	if actor.MP != 2 || target.HP != 100 || !actor.Acted {
		t.Fatalf("AI heal actor=%#v target=%#v", actor, target)
	}
}

func TestExecuteNativeAICommandHealRejectsMissingSelectorBeforeMutation(t *testing.T) {
	actor := completeNativeAIScoringUnit()
	actor.HasNativeRecordByte6 = false
	actor.MP = 5
	st := &State{NativeCommandBook: nativeCommandHealBook(13), Units: []*Unit{actor}}
	if _, err := st.ExecuteNativeAICommandHeal(actor, 13, rand.New(rand.NewSource(1))); err == nil || actor.MP != 5 || actor.Acted {
		t.Fatalf("missing selector crossed AI heal gate: actor=%#v err=%v", actor, err)
	}
}
