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
	if len(blocks) != 5 || blocks[0] != (DialogueBlock{PortraitID: 37, SourceDAT: "FDTXT_030", Script: "ch30.json", StringIndex: 2, SceneIndex: 1, Line: 0, Count: 6}) || blocks[4].StringIndex != 6 || blocks[4].Line != 12 || blocks[4].Count != 1 {
		t.Fatalf("first ending dialogue branch = %#v", blocks)
	}
	blocks = timeline.Segments[9].ThenDialogue
	if len(blocks) != 1 || blocks[0] != (DialogueBlock{PortraitID: 4, SourceDAT: "FDTXT_027", Script: "ch27.json", StringIndex: 17, SceneIndex: 3, Line: 1, Count: 1}) {
		t.Fatalf("first bad-ending dialogue branch = %#v", blocks)
	}
	blocks = timeline.Segments[13].ElseDialogue
	if len(blocks) != 1 || blocks[0] != (DialogueBlock{PortraitID: 45, SourceDAT: "FDTXT_030", Script: "ch30.json", StringIndex: 7, SceneIndex: 1, Line: 13, Count: 1}) {
		t.Fatalf("second ending dialogue branch = %#v", blocks)
	}
	blocks = timeline.Segments[13].ThenDialogue
	if len(blocks) != 3 || blocks[0].StringIndex != 18 || blocks[0].Line != 2 || blocks[2].StringIndex != 20 || blocks[2].Line != 4 {
		t.Fatalf("second bad-ending dialogue branch = %#v", blocks)
	}
}
