package battle

import (
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func event79Fixture(t *testing.T) (*State, NativeTurnEvent) {
	t.Helper()
	sc, err := LoadScenario("../../assets/scenarios/ch29.json")
	if err != nil {
		t.Fatal(err)
	}
	var event NativeTurnEvent
	for _, candidate := range sc.NativeTurnEvents {
		if candidate.EventID == 79 {
			event = candidate
		}
	}
	units := make([]*Unit, 13)
	for i := range units {
		units[i] = &Unit{Group: 8, HasNativeRecordByte5: true}
	}
	for i := 10; i < 13; i++ {
		units[i].Group = 1
	}
	st := &State{Units: units, NativeRoundCounter: 12, HasNativeTurnEventControlState: true}
	for i := range st.NativeTurnEventControls {
		st.NativeTurnEventControls[i].Turn = 0xff
	}
	st.NativeTurnEventControls[2] = NativeTurnEventControl{Turn: 12, EventID: 79, RawCamp: 0}
	st.NativeEventState[21] = 10
	return st, event
}

func TestEvent79ConsumesOneRNGStepAndMarksTwoCyclicGroup1Slots(t *testing.T) {
	st, event := event79Fixture(t)
	seed := uint16(0xffff)
	wantRNG := fdother.NativeRNGStep(seed)
	want := [2]int{10 + int(wantRNG%3), 10 + (int(wantRNG)+1)%3}
	next, targets, err := ApplyNativeTurnEvent79(st, event, seed)
	if err != nil {
		t.Fatal(err)
	}
	if next != wantRNG || targets != want ||
		st.NativeTurnEventControls[2] != (NativeTurnEventControl{Turn: 13, EventID: 79, RawCamp: 0}) {
		t.Fatalf("event79 rng=%#x/%#x targets=%v/%v row=%#v", next, wantRNG, targets, want, st.NativeTurnEventControls[2])
	}
	for index := 10; index < 13; index++ {
		marked := st.Units[index].NativeRecordByte5&0x80 != 0
		if marked != (index == want[0] || index == want[1]) {
			t.Fatalf("event79 slot%d marked=%v targets=%v", index, marked, want)
		}
	}
}

func TestEvent79MissingThirdGroupRecordFailsAtomically(t *testing.T) {
	st, event := event79Fixture(t)
	st.Units[12].HasNativeRecordByte5 = false
	beforeRows := st.NativeTurnEventControls
	beforeUnits := cloneUnitsForEvent79(st.Units)
	seed := uint16(7)
	next, _, err := ApplyNativeTurnEvent79(st, event, seed)
	if err == nil || next != seed || st.NativeTurnEventControls != beforeRows || !reflect.DeepEqual(st.Units, beforeUnits) {
		t.Fatalf("event79 atomic failure err=%v rng=%#x rows=%#v", err, next, st.NativeTurnEventControls[2])
	}
}

func cloneUnitsForEvent79(units []*Unit) []*Unit {
	out := make([]*Unit, len(units))
	for i, unit := range units {
		if unit != nil {
			copy := *unit
			out[i] = &copy
		}
	}
	return out
}
