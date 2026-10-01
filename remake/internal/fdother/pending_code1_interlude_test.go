package fdother

import "testing"

func TestNativePendingCode1PresentationPlanPreservesRawOrder(t *testing.T) {
	got := NativePendingCode1PresentationPlan()
	wantKinds := []PendingCode1PresentationKind{
		PendingCode1StopBGM, PendingCode1WaitTick, PendingCode1PreparePalette, PendingCode1ClearScreen,
		PendingCode1DrawFrame, PendingCode1FadeIn, PendingCode1WaitTick,
		PendingCode1DrawFrame, PendingCode1WaitTick, PendingCode1Release,
	}
	if len(got) != len(wantKinds) {
		t.Fatalf("steps=%d want=%d", len(got), len(wantKinds))
	}
	for index, kind := range wantKinds {
		if got[index].Kind != kind {
			t.Fatalf("step %d kind=%q want=%q", index, got[index].Kind, kind)
		}
	}
	if got[1].Count != 1 || got[4].Frame != 0 || got[5].Count != 65 || got[5].DurationMS != 2 ||
		got[6].Count != 9 || got[7].Frame != 1 || got[8].Count != 36 {
		t.Fatalf("raw schedule drifted: %+v", got)
	}
}
