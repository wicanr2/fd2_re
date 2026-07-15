package campaign

import "testing"

func TestLoadCountAlignedStoryIndexMap(t *testing.T) {
	index, err := LoadStoryIndexMap("../../assets/cutscenes/dialogue-index/count-aligned.json")
	if err != nil {
		t.Fatal(err)
	}
	if index.MappingKind != "count_aligned_only" || len(index.Diagnostics) != 10 {
		t.Fatalf("manifest identity = %#v", index)
	}

	palaceFirst, ok := index.Lookup("FDTXT_033", "ch00_palace.json", 0)
	if !ok || len(palaceFirst) != 1 || palaceFirst[0].Scene == nil || *palaceFirst[0].Scene != "王座廳,傳位" {
		t.Fatalf("palace FDTXT #0 = %#v, want throne lines", palaceFirst)
	}
	if got := palaceFirst[0].Lines; len(got) != 6 || got[0] != 0 || got[5] != 5 {
		t.Fatalf("palace FDTXT #0 lines = %#v, want 0..5", got)
	}

	palaceLast, ok := index.Lookup("FDTXT_033", "ch00_palace.json", 5)
	if !ok || len(palaceLast) != 1 || palaceLast[0].Scene == nil || *palaceLast[0].Scene != "王城一隅,亞雷斯撞見" {
		t.Fatalf("palace FDTXT #5 = %#v, want grass tail", palaceLast)
	}
	if got := palaceLast[0].Lines; len(got) != 12 || got[0] != 10 || got[11] != 21 {
		t.Fatalf("palace FDTXT #5 lines = %#v, want 10..21", got)
	}

	if _, ok := index.Lookup("FDTXT_033", "ch33.json", 0); ok {
		t.Fatal("unmapped ch33 context unexpectedly resolved")
	}
	if _, ok := index.Lookup("FDTXT_002", "ch02.json", 0); ok {
		t.Fatal("count-mismatched FDTXT_002 unexpectedly resolved")
	}
}

func TestStoryIndexMapRequiresExactCounts(t *testing.T) {
	resource := StoryIndexResource{SourceDAT: "FDTXT_001", RawStringCount: 1, RawUtteranceCount: 2}
	mapping := StoryIndexScriptMapping{
		Script: "ch01.json", SourceDAT: "FDTXT_001", Status: "count_aligned",
		RawUtteranceCount: 2, StoryLineCount: 1,
		Mappings: []StoryIndexStringMap{{StringIndex: 0, UtteranceCount: 2, Targets: []StoryIndexTarget{{SceneIndex: 0, Lines: []int{0, 1}}}}},
	}
	if err := validateStoryIndexMapping(resource, mapping); err == nil {
		t.Fatal("mismatched story count was accepted")
	}
}
