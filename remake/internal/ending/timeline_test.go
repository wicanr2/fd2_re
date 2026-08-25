package ending

import "testing"

func TestNative2BCE5TimelineIsRecoveredButNotPlayable(t *testing.T) {
	timeline, err := LoadTimeline("../../assets/endings/native_2bce5.json")
	if err != nil {
		t.Fatal(err)
	}
	if timeline.NativeHandler != "0x2bce5" || timeline.Resource.Archive != "FDOTHER.DAT" || timeline.Resource.Index != 0x36 {
		t.Fatalf("timeline header = %#v", timeline)
	}
	if timeline.Ready() {
		t.Fatal("opaque native ending timeline must remain fail-closed")
	}
	if len(timeline.Segments) != 18 {
		t.Fatalf("segment count = %d, want 18", len(timeline.Segments))
	}
	if timeline.Segments[17].Op != "native_post_composite_opaque" {
		t.Fatalf("post-composite gate = %#v", timeline.Segments[17])
	}
	if timeline.Segments[15].FirstFrameFormula != "(i%4)+1" || timeline.Segments[15].SecondFrameFormula != "(i%4)+5" || timeline.Segments[16].FirstFrameFormula != "(i%4)+1" || timeline.Segments[16].SecondFrameFormula != "(i%4)+5" {
		t.Fatalf("composite frame formulas = %#v / %#v", timeline.Segments[15], timeline.Segments[16])
	}
	if timeline.Segments[12].Op != "blit_frame_sequence" || timeline.Segments[15].Op != "native_composite_loop_opaque" || timeline.Segments[16].Op != "native_composite_loop_baseline" {
		t.Fatalf("frame schedule landmarks = %#v", timeline.Segments)
	}
	if frame := timeline.Segments[0].Frame; frame == nil || *frame != 0 || timeline.Segments[0].Target != "offscreen" || timeline.Segments[0].Stride != 320 || timeline.Segments[0].Transparent == nil || *timeline.Segments[0].Transparent != -1 {
		t.Fatalf("first native blit = %#v", timeline.Segments[0])
	}
	ani := timeline.Segments[3]
	if ani.Op != "ani_play" || ani.ANIResource == nil || *ani.ANIResource != 2 || ani.FrameDelayMs != 100 || ani.Skippable == nil || *ani.Skippable {
		t.Fatalf("ANI prefix = %#v", ani)
	}
	blocks := timeline.Segments[9].ElseDialogue
	if len(blocks) != 5 || blocks[0].PortraitID != 37 || blocks[0].SourceDAT != "FDTXT_030" || blocks[0].Script != "ch30.json" || blocks[0].StringIndex != 2 || blocks[0].SceneIndex != 1 || blocks[0].Line != 0 || blocks[0].Count != 6 || blocks[4].StringIndex != 6 || blocks[4].Line != 12 || blocks[4].Count != 1 {
		t.Fatalf("first ending dialogue branch = %#v", blocks)
	}
	if got := []int{blocks[0].NativeUtterances[0].Operand, blocks[0].NativeUtterances[1].Operand, blocks[0].NativeUtterances[2].Operand, blocks[0].NativeUtterances[3].Operand, blocks[0].NativeUtterances[4].Operand, blocks[0].NativeUtterances[5].Operand}; !equalInts(got, []int{4, 24, 126, 0, 126, 122}) {
		t.Fatalf("first ending native speakers = %v", got)
	}
	if got := blocks[0].NativeUtterances[2]; got.ControlSource != "fdtxt" || got.Control != "FFEF" || len(got.Pages) != 2 {
		t.Fatalf("paged ending utterance = %#v", got)
	}
	if got := blocks[4].NativeUtterances[0].Pages; len(got) != 1 || len(got[0]) != 4 {
		t.Fatalf("four-row ending page = %#v", got)
	}
	blocks = timeline.Segments[9].ThenDialogue
	if len(blocks) != 1 || blocks[0].PortraitID != 4 || blocks[0].SourceDAT != "FDTXT_027" || blocks[0].Script != "ch27.json" || blocks[0].StringIndex != 17 || blocks[0].SceneIndex != 3 || blocks[0].Line != 1 || blocks[0].Count != 1 || blocks[0].NativeUtterances[0].ControlSource != "caller_2c39b" || blocks[0].NativeUtterances[0].Operand != 4 {
		t.Fatalf("first bad-ending dialogue branch = %#v", blocks)
	}
	blocks = timeline.Segments[13].ElseDialogue
	if len(blocks) != 1 || blocks[0].PortraitID != 45 || blocks[0].SourceDAT != "FDTXT_030" || blocks[0].Script != "ch30.json" || blocks[0].StringIndex != 7 || blocks[0].SceneIndex != 1 || blocks[0].Line != 13 || blocks[0].Count != 1 || blocks[0].NativeUtterances[0].Operand != 9 {
		t.Fatalf("second ending dialogue branch = %#v", blocks)
	}
	blocks = timeline.Segments[13].ThenDialogue
	if len(blocks) != 3 || blocks[0].StringIndex != 18 || blocks[0].Line != 2 || blocks[2].StringIndex != 20 || blocks[2].Line != 4 {
		t.Fatalf("second bad-ending dialogue branch = %#v", blocks)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
