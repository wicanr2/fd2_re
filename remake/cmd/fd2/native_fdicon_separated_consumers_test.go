package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

func TestIntermissionConsumersUseSeparatedFDICONBank(t *testing.T) {
	const original = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(original); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	root := t.TempDir()
	source, err := filepath.Abs(original)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, filepath.Join(root, "FDOTHER.DAT")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(root, "FDOTHER.DAT"))
	t.Setenv("FD2_ASSET_PACK", "../../generated-assets/fd2-original-b97caf22")

	preparation, err := loadNativePreparationUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	church, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	town, err := loadNativeTownUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	for name, bank := range map[string]*fdicon.Bank{
		"preparation": preparation.roster.Units,
		"church":      church.units,
		"town":        {Sprites: town.scene.Pulse[:]},
	} {
		if bank == nil || (name != "town" && len(bank.Sprites) != fdicon.SeparatedSpriteCount) ||
			(name == "town" && len(bank.Sprites) != 3) {
			t.Fatalf("%s separated FDICON bank is incomplete", name)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "FDICON.B24")); !os.IsNotExist(err) {
		t.Fatal("test oracle unexpectedly provided FDICON.B24 beside FDOTHER.DAT")
	}
}
