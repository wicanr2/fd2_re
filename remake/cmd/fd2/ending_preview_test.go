package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/ending"
)

func TestNativeEndingDialogLinesUseNativePortraitIDs(t *testing.T) {
	lines, err := nativeEndingDialogLines([]ending.DialogueBlock{{PortraitID: 37, Script: "ch30.json", SceneIndex: 1, Line: 0, Count: 6}, {PortraitID: 21, Script: "ch30.json", SceneIndex: 1, Line: 6, Count: 2}})
	if err != nil || len(lines) != 8 {
		t.Fatalf("lines=%d err=%v", len(lines), err)
	}
	for i, line := range lines {
		want := 37
		if i >= 6 {
			want = 21
		}
		if line.Speaker != want || line.Text == "" {
			t.Fatalf("line %d=%#v want speaker %d", i, line, want)
		}
	}
}
