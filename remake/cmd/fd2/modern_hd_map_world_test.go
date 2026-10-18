package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func hdWorldFixture(t *testing.T) (indexedmap.FrameInput, []byte, color.Palette, map[int][]image.Image) {
	t.Helper()
	in, palette := hdForegroundPlanFixture()
	unitInput, _ := hdUnitPlanFixture(t)
	in.UnitBank, in.SelectorCache = unitInput.UnitBank, unitInput.SelectorCache
	in.Units = []fdicon.NativeUnitLayerEntry{{X: 4, Y: 4, Slot: 1}}
	in.TerrainBank = nativeFrameTestBank(12, 10)
	in.RangeBank = nativeFrameTestBank(20, 20)
	in.Controls[0] = 0
	in.ForegroundUnits = []fdicon.NativeForegroundLayerEntry{{X: 4, Y: 4}}
	in.SelectionOverlay = func([]byte, int, int) error { t.Fatal("world candidate consumed UI callback"); return nil }
	seed := bytes.Repeat([]byte{55}, indexedmap.NativeUnitPresentWorkSize)
	frames := map[int][]image.Image{0: make([]image.Image, 12)}
	for frame := range frames[0] {
		img := image.NewRGBA(image.Rect(0, 0, 144, 144))
		draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{200, 100, 60, 255}), image.Point{}, draw.Src)
		img.SetRGBA(7, 8, color.RGBA{100, 50, 30, 128})
		frames[0][frame] = img
	}
	return in, seed, palette, frames
}

func TestModernHDMapWorldLayersAndImmutableSources(t *testing.T) {
	in, seed, palette, frames := hdWorldFixture(t)
	beforeSeed := append([]byte(nil), seed...)
	beforeCells := append([]fdicon.NativeTerrainCell(nil), in.Cells...)
	beforeFrame := append([]byte(nil), frames[0][0].(*image.RGBA).Pix...)
	world, err := composeModernHDMapWorld(in, seed, palette, frames)
	if err != nil {
		t.Fatal(err)
	}
	if world.Bounds() != image.Rect(0, 0, 1920, 1200) || world.RGBAAt(0, 0) != (color.RGBA{A: 255}) || world.RGBAAt(1896, 1176) != (color.RGBA{A: 255}) {
		t.Fatal("HD dimensions or native black border changed")
	}
	const left, top = 52 * 6, 46 * 6
	if world.RGBAAt(left, top) != (color.RGBA{200, 100, 60, 255}) {
		t.Fatal("HD unit missing or old indexed unit drawn over it")
	}
	if pixel := world.RGBAAt(left+7, top+8); pixel.A != 255 || pixel == (color.RGBA{200, 100, 60, 255}) {
		t.Fatal("HD coverage alpha was quantized")
	}
	// A palette-index-zero foreground source must hide the HD figure even
	// when its native color equals the terrain's initial index.
	in.Controls[0] = 0x80
	for i := range in.ForegroundBank.Sprites[1].Mask {
		in.ForegroundBank.Sprites[1].Mask[i] = 0
	}
	in.ForegroundBank.Sprites[1].Mask[18*24] = 1
	in.ForegroundBank.Sprites[1].Pixels[18*24] = 0
	world, err = composeModernHDMapWorld(in, seed, palette, frames)
	if err != nil {
		t.Fatal(err)
	}
	want := color.RGBAModel.Convert(palette[0]).(color.RGBA)
	if world.RGBAAt(left, top) != want || world.RGBAAt(left+6, top) != (color.RGBA{200, 100, 60, 255}) {
		t.Fatal("foreground mask ordering or transparent hole lost")
	}
	if !bytes.Equal(seed, beforeSeed) || !reflect.DeepEqual(in.Cells, beforeCells) || !bytes.Equal(frames[0][0].(*image.RGBA).Pix, beforeFrame) {
		t.Fatal("world composition mutated input state or art")
	}
}

func TestModernHDMapWorldRawFlagAndAtomicRejection(t *testing.T) {
	in, seed, palette, frames := hdWorldFixture(t)
	in.Units[0].Flags = 0x80
	world, err := composeModernHDMapWorld(in, seed, palette, frames)
	if err != nil {
		t.Fatal(err)
	}
	if world.RGBAAt(52*6, 46*6) != (color.RGBA{110, 55, 36, 255}) {
		t.Fatal("existing modern dim policy changed")
	}
	in.Units = append(in.Units, fdicon.NativeUnitLayerEntry{X: 5, Y: 4, Slot: 4})
	if world, err := composeModernHDMapWorld(in, seed, palette, frames); err == nil || world != nil {
		t.Fatal("missing later HD group published partial world")
	}
	in.Units = in.Units[:1]
	frames[0][0] = image.NewRGBA(image.Rect(0, 0, 24, 24))
	if world, err := composeModernHDMapWorld(in, seed, palette, frames); err == nil || world != nil {
		t.Fatal("24px preview admitted as HD source")
	}
	if world, err := composeModernHDMapWorld(in, seed[:20], palette, frames); err == nil || world != nil {
		t.Fatal("incomplete terrain seed admitted")
	}
}
