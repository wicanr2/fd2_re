package main

import (
	"image/color"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

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
