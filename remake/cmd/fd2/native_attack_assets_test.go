package main

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureNativeAttackPresentationLoadsLateRawResources(t *testing.T) {
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	figaniPath := filepath.Join(base, "FIGANI.DAT")
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(figaniPath); err != nil {
		t.Skip("player-provided FIGANI.DAT is absent")
	}
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	t.Setenv("FD2_ORIGINAL_FIGANI", figaniPath)
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
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
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{R: byte(i), G: byte(i), B: byte(i), A: 0xff}
	}
	g := &Game{nativeUIPalette: palette}
	if err := g.ensureNativeAttackPresentation(126, 56); err == nil {
		t.Fatal("missing FIGANI archive was accepted")
	}
	if len(g.figani) != 0 || len(g.figaniDelays) != 0 || len(g.figMeta) != 0 {
		t.Fatal("failed preload partially published presentation maps")
	}
}
