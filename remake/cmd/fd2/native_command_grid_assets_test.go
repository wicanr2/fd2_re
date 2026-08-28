package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeCommandGridPlayerAssetGate(t *testing.T) {
	pack := filepath.Clean("../../generated-assets/fd2-original-b97caf22")
	if _, err := os.Stat(filepath.Join(pack, "palette", "fdother_000.json")); err != nil {
		t.Skip("player-generated separated asset pack is absent")
	}
	t.Setenv("FD2_ASSET_PACK", pack)
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(t.TempDir(), "must-not-be-read.dat"))
	if palette := loadNativeUIPalette(); len(palette) != 256 {
		t.Fatalf("native UI palette entries=%d, want 256", len(palette))
	}
	if labels := loadNativeCommandLabels(); labels[0] == "" {
		t.Fatal("native command label 0 is unavailable")
	}
	if font := loadFont(); font == nil {
		t.Fatal("native command grid font is unavailable")
	}
}
