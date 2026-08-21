package battle

import "testing"

func nativeCh28PostPresenterFixture() *State {
	return &State{Units: []*Unit{
		{HP: 0, NativeMapPresentation: NativeMapPresentationState{X: 9, Y: 7}, HasNativeMapPresentation: true, HasNativeRecordByte5: true},
		{HP: 0, NativeMapPresentation: NativeMapPresentationState{X: 23, Y: 17}, HasNativeMapPresentation: true, HasNativeRecordByte5: true},
		{HP: 1, NativeMapPresentation: NativeMapPresentationState{X: 10, Y: 8}, HasNativeMapPresentation: true, HasNativeRecordByte5: true},
		{HP: 0, NativeMapPresentation: NativeMapPresentationState{X: 10, Y: 8}, HasNativeMapPresentation: true, NativeRecordByte5: 1, HasNativeRecordByte5: true},
	}}
}

func TestPlanNativeCh28PostPresentationUsesExactViewportGate(t *testing.T) {
	st := nativeCh28PostPresenterFixture()
	plan, err := PlanNativeCh28PostPresentation(st, 10, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Targets) != 2 || plan.Targets[0].Slot != 0 || plan.Targets[1].Slot != 1 {
		t.Fatalf("targets=%+v want boundary slots 0,1", plan.Targets)
	}
	st.Units[1].NativeMapPresentation.X++
	plan, err = PlanNativeCh28PostPresentation(st, 10, 8)
	if err != nil || len(plan.Targets) != 1 || plan.Targets[0].Slot != 0 {
		t.Fatalf("outside right boundary targets=%+v err=%v", plan.Targets, err)
	}
}

func TestNativeCh28PostPoseAndInactiveWritesFollowRawGates(t *testing.T) {
	st := nativeCh28PostPresenterFixture()
	if err := ApplyNativeCh28PostPoseFrame(st, 11); err != nil {
		t.Fatal(err)
	}
	if st.Units[0].NativeMapPresentation.Pose != 3 || st.Units[1].NativeMapPresentation.Pose != 3 ||
		st.Units[2].NativeMapPresentation.Pose != 0 || st.Units[3].NativeMapPresentation.Pose != 0 {
		t.Fatalf("pose write mismatch: %+v", st.Units)
	}
	if err := ApplyNativeCh28PostInactiveMark(st); err != nil {
		t.Fatal(err)
	}
	if st.Units[0].NativeRecordByte5 != 1 || st.Units[1].NativeRecordByte5 != 1 ||
		st.Units[2].NativeRecordByte5 != 0 || st.Units[3].NativeRecordByte5 != 1 {
		t.Fatalf("inactive writes mismatch: %+v", st.Units)
	}
}

func TestNativeCh28PostPresenterRejectsMissingProvenanceAtomically(t *testing.T) {
	st := nativeCh28PostPresenterFixture()
	st.Units[2].HasNativeMapPresentation = false
	if _, err := PlanNativeCh28PostPresentation(st, 10, 8); err == nil {
		t.Fatal("missing consumed raw fields were accepted")
	}
	before := st.Units[0].NativeMapPresentation.Pose
	st.Units[2].HasNativeMapPresentation = true
	st.Units[1].HasNativeRecordByte5 = false
	if err := ApplyNativeCh28PostPoseFrame(st, 2); err == nil {
		t.Fatal("missing pose provenance was accepted")
	}
	if st.Units[0].NativeMapPresentation.Pose != before {
		t.Fatal("failed pose frame partially mutated state")
	}
}
