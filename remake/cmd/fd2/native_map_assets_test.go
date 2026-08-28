package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestLoadNativeMapAssetsUsesSeparatedFDSHAPAndFDICONBanks(t *testing.T) {
	const original = "../../../org_game/炎龍騎士團/FLAME2"
	for _, name := range []string{"FDOTHER.DAT"} {
		if _, err := os.Stat(filepath.Join(original, name)); err != nil {
			t.Skip("player-provided original archives are absent")
		}
	}
	root := t.TempDir()
	for _, name := range []string{"FDOTHER.DAT"} {
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
	for _, sample := range []struct {
		index, sprites, controls int
	}{{0, 288, 1200}, {23, 96, 400}, {32, 192, 1200}} {
		assets, err := loadNativeMapAssets(assetPath(fmt.Sprintf("assets/maps/map%d", sample.index)))
		if err != nil {
			t.Fatalf("map%d: %v", sample.index, err)
		}
		if assets.Units == nil || len(assets.Units.Sprites) != fdicon.SeparatedSpriteCount {
			t.Fatalf("map%d separated FDICON sprites unavailable", sample.index)
		}
		if assets.Terrain == nil || len(assets.Terrain.Sprites) != sample.sprites || len(assets.Controls) != sample.controls {
			t.Fatalf("map%d separated FDSHAP sprites=%d controls=%d", sample.index, len(assets.Terrain.Sprites), len(assets.Controls))
		}
	}
	if _, err := os.Stat(filepath.Join(root, "FDICON.B24")); !os.IsNotExist(err) {
		t.Fatal("test oracle unexpectedly provided FDICON.B24 beside runtime archives")
	}
	if _, err := os.Stat(filepath.Join(root, "FDSHAP.DAT")); !os.IsNotExist(err) {
		t.Fatal("test oracle unexpectedly provided FDSHAP.DAT beside runtime archives")
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
