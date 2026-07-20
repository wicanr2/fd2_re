package main

import "testing"

func TestDefaultChapterStoryScriptOnlyMatchesGenericStoryNodes(t *testing.T) {
	for _, tc := range []struct {
		id, want string
	}{
		{"story_ch04", "assets/story/ch04.json"},
		{"story_ch30", "assets/story/ch30.json"},
		{"story_ch01_pre", ""},
		{"story_ch21_post_sky_key_intro", ""},
		{"story_ch00_handler", ""},
	} {
		if got := defaultChapterStoryScript(tc.id); got != tc.want {
			t.Fatalf("defaultChapterStoryScript(%q)=%q, want %q", tc.id, got, tc.want)
		}
	}
}
