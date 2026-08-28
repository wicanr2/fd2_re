package main

import (
	"image/color"
	"path/filepath"
	"testing"
)

func TestEnsureNativeAttackPresentationLoadsSeparatedResourcesWithoutArchive(t *testing.T) {
	animationRoot := separatedAssetPath("animations")
	for _, name := range []string{"FIGANI_168", "FIGANI_379"} {
		if !fileExists(filepath.Join(animationRoot, name, "animation.json")) {
			t.Skip("player-generated separated FIGANI pack is absent")
		}
	}
	t.Setenv("FD2_ORIGINAL_FIGANI", filepath.Join(t.TempDir(), "missing-FIGANI.DAT"))
	g := &Game{nativeUIPalette: loadNativeUIPalette()}
	if err := g.ensureNativeAttackPresentation(126, 56); err != nil {
		t.Fatal(err)
	}
	if !g.nativeAttackPresentationAvailable(126, 56) {
		t.Fatal("late FIGANI pair is not available after exact raw load")
	}
	for _, id := range []int{379, 168} {
		if len(g.figani[id]) == 0 || len(g.figaniDelays[id]) != len(g.figani[id]) ||
			len(g.figMeta[id]) != len(g.figani[id]) {
			t.Fatalf("resource %d was not published as a complete frame/delay/position set", id)
		}
	}
}

func TestEnsureNativeAttackPresentationRejectsWithoutPartialPublish(t *testing.T) {
	t.Setenv("FD2_ORIGINAL_FIGANI", filepath.Join(t.TempDir(), "missing.dat"))
	t.Setenv("FD2_ASSET_PACK", t.TempDir())
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{R: byte(i), G: byte(i), B: byte(i), A: 0xff}
	}
	g := &Game{nativeUIPalette: palette}
	if err := g.ensureNativeAttackPresentation(126, 56); err == nil {
		t.Fatal("missing separated FIGANI resources were accepted")
	}
	if len(g.figani) != 0 || len(g.figaniDelays) != 0 || len(g.figMeta) != 0 {
		t.Fatal("failed preload partially published presentation maps")
	}
}
