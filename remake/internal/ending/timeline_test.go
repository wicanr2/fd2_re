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
	if len(timeline.Segments) != 17 {
		t.Fatalf("segment count = %d, want 17", len(timeline.Segments))
	}
	if timeline.Segments[12].Op != "blit_frame_sequence" || timeline.Segments[15].Op != "native_composite_loop_opaque" || timeline.Segments[16].Op != "native_composite_loop_opaque" {
		t.Fatalf("frame schedule landmarks = %#v", timeline.Segments)
	}
	blocks := timeline.Segments[9].ElseDialogue
	if len(blocks) != 5 || blocks[0] != (DialogueBlock{VisualResourceIndex: 37, SourceDAT: "FDTXT_030", Script: "ch30.json", StringIndex: 2, SceneIndex: 1, Line: 0, Count: 6}) || blocks[4].StringIndex != 6 || blocks[4].Line != 12 || blocks[4].Count != 1 {
		t.Fatalf("first ending dialogue branch = %#v", blocks)
	}
	blocks = timeline.Segments[13].ElseDialogue
	if len(blocks) != 1 || blocks[0] != (DialogueBlock{VisualResourceIndex: 45, SourceDAT: "FDTXT_030", Script: "ch30.json", StringIndex: 7, SceneIndex: 1, Line: 13, Count: 1}) {
		t.Fatalf("second ending dialogue branch = %#v", blocks)
	}
}
