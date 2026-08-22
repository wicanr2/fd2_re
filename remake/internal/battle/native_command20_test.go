package battle

import (
	"math/rand"
	"testing"
)

func nativeCommandClearRestoreBook(id int) []NativeCommandRecord {
	book := make([]NativeCommandRecord, 36)
	for i := range book {
		book[i] = NativeCommandRecord{ID: i}
	}
	book[10] = NativeCommandRecord{ID: 10, Damage: 100}
	book[id] = NativeCommandRecord{ID: id, SelectionMode: 1, EffectMode: 0, MPCost: 2, TargetCode: 1}
	return book
}

func TestExecuteNativeCommandClearRestoreUsesRecordTenAndRawFlag(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, HP: 10, MP: 5, X: 0, Y: 0}
	target := &Unit{Camp: Own, OnField: true, HP: 1, MaxHP: 100, X: 1, Y: 0, NativeTransient: [6]byte{0, 0, 0, 3}}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: nativeCommandClearRestoreBook(20)}

	got, err := st.ExecuteNativeCommandClearRestore(actor, target, 20, rand.New(rand.NewSource(2)))
	if err != nil || len(got) != 1 || !got[0].Cleared || got[0].Offset != 0x25 || got[0].Restore.Rolled < 90 || got[0].Restore.Rolled > 99 {
		t.Fatalf("clear/restore = %#v, %v", got, err)
	}
	if target.NativeTransient[3] != 0 || target.HP != 1+got[0].Restore.Actual || actor.MP != 3 || !actor.Acted {
		t.Fatalf("post clear/restore actor=%#v target=%#v", actor, target)
	}
}

func TestExecuteNativeCommandClearRestoreConsumesCommandWhenFlagEmpty(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, HP: 10, MP: 5, X: 0, Y: 0}
	target := &Unit{Camp: Own, OnField: true, HP: 10, MaxHP: 20, X: 1, Y: 0}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: nativeCommandClearRestoreBook(21)}

	got, err := st.ExecuteNativeCommandClearRestore(actor, target, 21, rand.New(rand.NewSource(1)))
	if err != nil || len(got) != 1 || got[0].Cleared || target.HP != 10 || actor.MP != 3 || !actor.Acted {
		t.Fatalf("empty-flag route = %#v actor=%#v target=%#v err=%v", got, actor, target, err)
	}
}

func TestApplyNativeCommandRestoreClampsStateButReportsRolledAmount(t *testing.T) {
	target := &Unit{HP: 95, MaxHP: 100}
	got, err := ApplyNativeCommandRestore(target, 100, rand.New(rand.NewSource(3)))
	if err != nil || got.Rolled < 90 || got.Rolled > 99 || got.Actual != 5 || target.HP != 100 {
		t.Fatalf("restore cap = %#v target=%#v err=%v", got, target, err)
	}
}

func TestExecuteNativeCommandClearRestorePreflightsBeforeMPDebit(t *testing.T) {
	actor := &Unit{Camp: Own, OnField: true, HP: 10, MP: 5, X: 0, Y: 0}
	target := &Unit{Camp: Own, OnField: true, HP: 11, MaxHP: 10, X: 1, Y: 0, NativeTransient: [6]byte{0, 0, 0, 3}}
	st := &State{W: 2, H: 1, Units: []*Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: nativeCommandClearRestoreBook(20)}

	if _, err := st.ExecuteNativeCommandClearRestore(actor, target, 20, rand.New(rand.NewSource(2))); err == nil {
		t.Fatal("invalid target HP state accepted")
	}
	if actor.MP != 5 || actor.Acted || target.NativeTransient[3] != 3 || target.HP != 11 {
		t.Fatalf("failed preflight mutated actor=%#v target=%#v", actor, target)
	}
}
