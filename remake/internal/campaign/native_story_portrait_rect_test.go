package campaign

import (
	"image"
	"testing"
)

func TestNativeStoryDialoguePortraitRectPreservesAsymmetricAnchors(t *testing.T) {
	tests := map[string]image.Rectangle{
		"FFEF": image.Rect(232, 5, 312, 85),
		"FFED": image.Rect(232, 5, 312, 85),
		"FFEE": image.Rect(8, 115, 88, 195),
		"FFEC": image.Rect(8, 115, 88, 195),
	}
	for control, want := range tests {
		got, err := NativeStoryDialoguePortraitRect(control)
		if err != nil || got != want {
			t.Fatalf("%s rect=%v err=%v, want %v", control, got, err, want)
		}
	}
	if _, err := NativeStoryDialoguePortraitRect("FFFF"); err == nil {
		t.Fatal("unsupported control accepted")
	}
}
