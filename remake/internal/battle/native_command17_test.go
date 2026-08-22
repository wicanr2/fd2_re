package battle

import "testing"

func nativeCommandModifierBook(id int) []NativeCommandRecord {
	book := make([]NativeCommandRecord, NativeCommandRecordCount)
	for i := range book {
		book[i] = NativeCommandRecord{ID: i}
	}
	book[id] = NativeCommandRecord{ID: id, SelectionMode: 1, EffectMode: 0, MPCost: 99, TargetCode: 1}
	book[18] = NativeCommandRecord{ID: 18, SelectionMode: 1, EffectMode: 0, MPCost: 4, TargetCode: 1}
	if id == 19 {
		book[19] = NativeCommandRecord{ID: 19, SelectionMode: 1, EffectMode: 0, MPCost: 5, TargetCode: 1}
	}
	return book
}

func TestExecuteNativeCommand17UsesRecord18DebitAndPublishesAtomically(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, HP: 10, MaxHP: 10, MP: 10, X: 0, Y: 0}
	target := &Unit{Camp: Own, OnField: true, HP: 10, MaxHP: 10, AP: 100, DP: 20, HIT: 30, EV: 40, X: 1, Y: 0, Lv: 2, NativeRecordClass: 9, HasNativeRecordClass: true}
	st := &State{
		W: 2, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 2),
		NativeCommandBook:           nativeCommandModifierBook(17),
	}

	got, err := st.ExecuteNativeCommandModifier(actor, target, 17, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.RNGState != 0x80a4 || len(got.WordSteps) != 1 || !got.WordSteps[0].Processed {
		t.Fatalf("modifier result=%#v", got)
	}
	if target.NativeTransient[0] != 2 || target.AP != 116 || actor.MP != 6 || !actor.Acted {
		t.Fatalf("actor=%#v target=%#v", actor, target)
	}
}

func TestExecuteNativeCommand19PublishesPairAndOwnDebit(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, HP: 10, MaxHP: 10, MP: 10, X: 0, Y: 0}
	target := &Unit{Camp: Own, OnField: true, HP: 10, MaxHP: 10, AP: 10, DP: 20, HIT: -2, EV: 1, X: 1, Y: 0, Lv: 2, NativeRecordClass: 9, HasNativeRecordClass: true}
	st := &State{
		W: 2, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 2),
		NativeCommandBook:           nativeCommandModifierBook(19),
	}

	got, err := st.ExecuteNativeCommandModifier(actor, target, 19, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.RNGState != 0x80a4 || len(got.PairSteps) != 1 || !got.PairSteps[0].Processed {
		t.Fatalf("modifier result=%#v", got)
	}
	if target.NativeTransient[2] != 2 || target.HIT != 13 || target.EV != 16 || actor.MP != 5 || !actor.Acted {
		t.Fatalf("actor=%#v target=%#v", actor, target)
	}
}

func TestExecuteNativeCommand18PublishesDPAndOwnDebit(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, HP: 10, MaxHP: 10, MP: 10, X: 0, Y: 0}
	target := &Unit{Camp: Own, OnField: true, HP: 10, MaxHP: 10, AP: 10, DP: 100, HIT: 30, EV: 40, X: 1, Y: 0, Lv: 2, NativeRecordClass: 9, HasNativeRecordClass: true}
	st := &State{
		W: 2, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 2),
		NativeCommandBook:           nativeCommandModifierBook(18),
	}

	got, err := st.ExecuteNativeCommandModifier(actor, target, 18, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.WordSteps) != 1 || !got.WordSteps[0].Processed || target.NativeTransient[1] != 2 ||
		target.DP != 116 || actor.MP != 6 || !actor.Acted {
		t.Fatalf("result=%#v actor=%#v target=%#v", got, actor, target)
	}
}

func TestExecuteNativeCommandModifierRejectsBadTargetBeforeMutation(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, HP: 10, MaxHP: 10, MP: 10, X: 0, Y: 0}
	good := &Unit{AP: 100, DP: 20, HIT: 30, EV: 40, Lv: 2, NativeRecordClass: 9, HasNativeRecordClass: true}
	target := &Unit{Camp: Own, OnField: true, HP: 10, MaxHP: 10, AP: 1 << 20, DP: 20, HIT: 30, EV: 40, X: 1, Y: 0, Lv: 2, NativeRecordClass: 9, HasNativeRecordClass: true}
	st := &State{
		W: 2, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 2),
		NativeCommandBook:           nativeCommandModifierBook(17),
	}

	if _, err := st.executeNativeCommandModifierTargets(actor, []*Unit{good, target}, 17, 0); err == nil {
		t.Fatal("out-of-range target accepted")
	}
	if actor.MP != 10 || actor.Acted || good.NativeTransient != [6]byte{} || good.AP != 100 ||
		target.NativeTransient != [6]byte{} || target.AP != 1<<20 {
		t.Fatalf("failed transaction mutated actor=%#v good=%#v target=%#v", actor, good, target)
	}
}

func TestExecuteNativeAICommandModifierUsesRawSelectorTargets(t *testing.T) {
	actor := completeNativeAIScoringUnit()
	actor.NativeMapPresentation.X, actor.NativeMapPresentation.Y = 0, 0
	actor.NativeRecordByte5, actor.NativeRecordByte6 = 0, 0
	actor.NativeTransient = [6]byte{}
	actor.Camp, actor.MP, actor.AP = Enemy, 10, 100
	target := completeNativeAIScoringUnit()
	target.NativeMapPresentation.X, target.NativeMapPresentation.Y = 1, 0
	target.NativeRecordByte5, target.NativeRecordByte6 = 0, 0
	target.NativeTransient = [6]byte{}
	target.Camp, target.AP = Enemy, 200
	book := nativeCommandModifierBook(17)
	book[17].EffectMode = 1
	st := &State{
		W: 2, H: 1, Units: []*Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book,
	}

	got, err := st.ExecuteNativeAICommandModifier(actor, 17, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.WordSteps) != 2 || actor.AP != 116 || target.AP != 231 || actor.MP != 6 || !actor.Acted {
		t.Fatalf("result=%#v actor=%#v target=%#v", got, actor, target)
	}
}
