package fdother

import "testing"

func TestNativePendingCode1PresentationPlanPreservesRawOrder(t *testing.T) {
	got := NativePendingCode1PresentationPlan()
	wantKinds := []PendingCode1PresentationKind{
		PendingCode1StopBGM, PendingCode1PreparePalette, PendingCode1ClearScreen,
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
	if got[3].Frame != 0 || got[4].Count != 65 || got[4].DurationMS != 2 ||
		got[5].Count != 9 || got[6].Frame != 1 || got[7].Count != 36 {
		t.Fatalf("raw schedule drifted: %+v", got)
	}
}
