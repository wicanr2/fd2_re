package main

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestLoadNativeMapAssetsUsesSeparatedFDICONBank(t *testing.T) {
	const original = "../../../org_game/炎龍騎士團/FLAME2"
	for _, name := range []string{"FDOTHER.DAT", "FDSHAP.DAT"} {
		if _, err := os.Stat(filepath.Join(original, name)); err != nil {
			t.Skip("player-provided original archives are absent")
		}
	}
	root := t.TempDir()
	for _, name := range []string{"FDOTHER.DAT", "FDSHAP.DAT"} {
		source, err := filepath.Abs(filepath.Join(original, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(source, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(root, "FDOTHER.DAT"))
	t.Setenv("FD2_ASSET_PACK", "../../generated-assets/fd2-original-b97caf22")
	assets, err := loadNativeMapAssets(assetPath("assets/maps/map00"))
	if err != nil {
		t.Fatal(err)
	}
	if assets.Units == nil {
		t.Fatal("separated FDICON bank is nil")
	}
	if len(assets.Units.Sprites) != fdicon.SeparatedSpriteCount {
		t.Fatalf("separated FDICON sprites=%d", len(assets.Units.Sprites))
	}
	if _, err := os.Stat(filepath.Join(root, "FDICON.B24")); !os.IsNotExist(err) {
		t.Fatal("test oracle unexpectedly provided FDICON.B24 beside runtime archives")
	}
}

func TestNativeMapAssetsRequireRangeOverlayBank(t *testing.T) {
	a := &nativeMapAssets{
		Terrain: &fdicon.Bank{}, Units: &fdicon.Bank{},
		Controls: []byte{0}, LUTs: make([][]byte, 10),
		Palette: make(color.Palette, 256), PaletteDAC: make([]byte, 256*3),
	}
	for i := 1; i <= 9; i++ {
		a.LUTs[i] = make([]byte, 256)
	}
	if nativeMapAssetsAvailable(a) {
		t.Fatal("accepted native map bundle without FDOTHER #1 range bank")
	}
	a.Range = &fdicon.Bank{}
	if !nativeMapAssetsAvailable(a) {
		t.Fatal("complete native map bundle rejected")
	}
	a.LUTs[8] = a.LUTs[8][:255]
	if nativeMapAssetsAvailable(a) {
		t.Fatal("bundle with malformed later transition LUT accepted")
	}
}

func TestNativeMapAssetsRequireChapterAuxOnlyForRawMaps28And29(t *testing.T) {
	makeAssets := func(mapIndex int) *nativeMapAssets {
		a := &nativeMapAssets{
			MapIndex: mapIndex, Terrain: &fdicon.Bank{}, Range: &fdicon.Bank{}, Units: &fdicon.Bank{},
			Controls: []byte{0}, LUTs: make([][]byte, 10),
			Palette: make(color.Palette, 256), PaletteDAC: make([]byte, 256*3),
		}
		for i := 1; i <= 9; i++ {
			a.LUTs[i] = make([]byte, 256)
		}
		return a
	}
	if !nativeMapAssetsAvailable(makeAssets(27)) {
		t.Fatal("unrelated map unexpectedly requires chapter auxiliary surface")
	}
	for _, index := range []int{28, 29} {
		a := makeAssets(index)
		if nativeMapAssetsAvailable(a) {
			t.Fatalf("map %d accepted without FDOTHER #55", index)
		}
		a.ChapterAux = &fdother.NativeChapterAuxSurface{Pixels: make([]byte, 320*200)}
		if !nativeMapAssetsAvailable(a) {
			t.Fatalf("map %d rejected complete FDOTHER #55", index)
		}
	}
}
