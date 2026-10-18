package main

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func hdForegroundPlanFixture() (indexedmap.FrameInput, color.Palette) {
	bank := &fdicon.Bank{Sprites: make([]fdicon.Sprite, 12)}
	for frame := range bank.Sprites {
		sprite := fdicon.Sprite{Pixels: make([]byte, 576), Mask: make([]byte, 576), RemapMask: make([]byte, 576)}
		for i := range sprite.Pixels {
			sprite.Pixels[i] = byte((frame*37 + i) % 256)
			if i%5 != 0 {
				sprite.Mask[i] = 1
			} else {
				sprite.RemapMask[i] = 1
			}
		}
		// Fixed index zero still covers a same-color underlying pixel.
		sprite.Pixels[0], sprite.Mask[0], sprite.RemapMask[0] = 0, 1, 0
		bank.Sprites[frame] = sprite
	}
	palette := make(color.Palette, 256)
	lut := make([]byte, 256)
	for i := range palette {
		palette[i] = color.RGBA{uint8(i), uint8(255 - i), uint8(i / 2), 255}
		lut[i] = byte((i + 31) % 256)
	}
	cells := make([]fdicon.NativeTerrainCell, 20*20)
	for i := range cells {
		cells[i] = fdicon.NativeTerrainCell{BlitMode: 0xff}
	}
	return indexedmap.FrameInput{ForegroundBank: bank, MapWidth: 20, Cells: cells, Controls: []byte{0x88, 0, 0, 0}, LUT: lut, CameraX: 2, CameraY: 2}, palette
}

func TestModernHDForegroundPlanMatchesNativeWrites(t *testing.T) {
	in, palette := hdForegroundPlanFixture()
	base := indexedmap.NativeUnitPresentStripLayoutFor(0, 0, 0, 0).WorkOffset
	view := modernHDMapViewport()
	cases := 0
	for pose := byte(0); pose < 4; pose++ {
		for _, motion := range []int{0, 3, 6} {
			for flip := 0; flip < 2; flip++ {
				for _, mode := range []byte{0, 0xff} {
					for _, suppressed := range []bool{false, true} {
						in.Flip = flip
						for i := range in.Cells {
							in.Cells[i].BlitMode = mode
						}
						unit := fdicon.NativeForegroundLayerEntry{X: 4, Y: 4, Pose: pose, MotionOffset: motion}
						if suppressed {
							unit.Race = 4
						}
						in.ForegroundUnits = []fdicon.NativeForegroundLayerEntry{unit, {X: 1, Y: 1}, {X: 14, Y: 10}, {X: 16, Y: 12}, {X: -1, Y: -1}, {X: 3, Y: 3, Inactive: true}}
						work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
						for i := range work {
							work[i] = byte(i % 256)
							if i%7 == 0 {
								work[i] = 0
							}
						}
						before := append([]byte(nil), work...)
						if err := in.ForegroundBank.BlitNativeForegroundLayer(work, fdicon.NativeMapStride, in.ForegroundUnits, in.MapWidth, in.Cells, in.Controls, in.CameraX, in.CameraY, 12, 7, flip, in.LUT); err != nil {
							t.Fatal(err)
						}
						plan, err := buildModernHDMapForegroundPlan(in, base, palette)
						if err != nil {
							t.Fatal(err)
						}
						got := image.NewRGBA(image.Rect(0, 0, view.Dx()/6, view.Dy()/6))
						for y := 0; y < got.Bounds().Dy(); y++ {
							for x := 0; x < got.Bounds().Dx(); x++ {
								got.Set(x, y, palette[before[base+y*456+x]])
							}
						}
						for _, command := range plan {
							dx, dy := (command.Rect.Min.X-view.Min.X)/6, (command.Rect.Min.Y-view.Min.Y)/6
							for y := 0; y < 24; y++ {
								for x := 0; x < 24; x++ {
									pixel := command.Image.RGBAAt(x, y)
									if pixel.A != 0 {
										got.SetRGBA(dx+x, dy+y, pixel)
									}
								}
							}
						}
						for y := 0; y < got.Bounds().Dy(); y++ {
							for x := 0; x < got.Bounds().Dx(); x++ {
								want := color.RGBAModel.Convert(palette[work[base+y*456+x]]).(color.RGBA)
								if got.RGBAAt(x, y) != want {
									t.Fatalf("native foreground mismatch pose=%d motion=%d flip=%d mode=%x suppressed=%v at(%d,%d)", pose, motion, flip, mode, suppressed, x, y)
								}
							}
						}
						cases++
					}
				}
			}
		}
	}
	if cases != 96 {
		t.Fatalf("cases=%d", cases)
	}
}

func TestModernHDForegroundPlanCoverageAndAtomicRejection(t *testing.T) {
	in, palette := hdForegroundPlanFixture()
	base := 0x8088
	in.ForegroundUnits = []fdicon.NativeForegroundLayerEntry{{X: 4, Y: 4}}
	beforePixels := append([]byte(nil), in.ForegroundBank.Sprites[1].Pixels...)
	beforeCells := append([]fdicon.NativeTerrainCell(nil), in.Cells...)
	plan, err := buildModernHDMapForegroundPlan(in, base, palette)
	if err != nil || len(plan) != 2 {
		t.Fatalf("plan=%v err=%v", plan, err)
	}
	if plan[0].Image.RGBAAt(0, 0).A != 255 || plan[0].Image.RGBAAt(5, 0).A != 0 {
		t.Fatal("zero-index write or destination-remap transparency lost")
	}
	if !bytes.Equal(beforePixels, in.ForegroundBank.Sprites[1].Pixels) || !reflect.DeepEqual(beforeCells, in.Cells) {
		t.Fatal("plan mutated original sources")
	}
	in.Cells[3*in.MapWidth+4].Tile = 100
	if plan, err := buildModernHDMapForegroundPlan(in, base, palette); err == nil || plan != nil {
		t.Fatal("invalid later command published an earlier command")
	}
	in, palette = hdForegroundPlanFixture()
	in.ForegroundUnits = []fdicon.NativeForegroundLayerEntry{{X: 4, Y: 4}}
	in.Cells[4*in.MapWidth+4].BlitMode = 0
	in.LUT = nil
	if plan, err := buildModernHDMapForegroundPlan(in, base, palette); err == nil || plan != nil {
		t.Fatal("missing LUT admitted")
	}
	if plan, err := buildModernHDMapForegroundPlan(in, -1, palette); err == nil || plan != nil {
		t.Fatal("negative viewport admitted")
	}
}
