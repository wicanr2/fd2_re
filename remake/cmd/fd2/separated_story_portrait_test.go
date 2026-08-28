package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeStoryPortraitLoadsWithoutOriginalArchive(t *testing.T) {
	pack := filepath.Clean("../../generated-assets/fd2-original-b97caf22")
	if _, err := os.Stat(filepath.Join(pack, "portraits", "DATO_026_m3.png")); err != nil {
		t.Skipf("player-generated separated portrait pack is absent: %v", err)
	}
	t.Setenv("FD2_ASSET_PACK", pack)
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(t.TempDir(), "must-not-be-read.dat"))
	frames, err := loadNativeStoryPortrait(26)
	if err != nil || len(frames) != 4 {
		t.Fatalf("frames=%d err=%v", len(frames), err)
	}
	for index, frame := range frames {
		if frame.Width != 80 || frame.Height != 80 || len(frame.Pixels) != 80*80 {
			t.Fatalf("frame %d geometry=%dx%d pixels=%d", index, frame.Width, frame.Height, len(frame.Pixels))
		}
	}
}

func TestNativeStoryPortraitFailsClosedWithoutSeparatedPack(t *testing.T) {
	t.Setenv("FD2_ASSET_PACK", t.TempDir())
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Clean("../../../org_game/炎龍騎士團/FLAME2/DATO.DAT"))
	if _, err := loadNativeStoryPortrait(26); err == nil {
		t.Fatal("story portrait fell back to original DATO.DAT")
	}
}
