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
}
