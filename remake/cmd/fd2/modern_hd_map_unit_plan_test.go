package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func hdUnitPlanFixture(t *testing.T) (indexedmap.FrameInput, map[int]int) {
	t.Helper()
	bank := &fdicon.Bank{Sprites: make([]fdicon.Sprite, 96*12)}
	for index := range bank.Sprites {
		bank.Sprites[index] = fdicon.Sprite{Pixels: make([]byte, 576), Mask: make([]byte, 576), RemapMask: make([]byte, 576)}
		bank.Sprites[index].Pixels[0] = byte(index%12 + 1)
		bank.Sprites[index].Mask[0] = 1
	}
	for frame := 0; frame < 12; frame++ {
		bank.Sprites[39*12+frame].Mask[0] = 0
		for i := range bank.Sprites[39*12+frame].RemapMask {
			bank.Sprites[39*12+frame].RemapMask[i] = 1
		}
		if frame >= 3 {
			bank.Sprites[48*12+frame].Mask[0] = 0
		}
	}
	cache := &fdicon.NativeSelectorCache{}
	for _, key := range []int{76, 0, 48, 39, 1} {
		if _, err := cache.SlotFor(key); err != nil {
			t.Fatal(err)
		}
	}
	return indexedmap.FrameInput{UnitBank: bank, SelectorCache: cache, CameraX: 2, CameraY: 2, IdleCycle: 3, MovingCycle: 2}, map[int]int{0: 12, 48: 3}
}

func TestModernHDMapUnitPlanMatchesNativeWrites(t *testing.T) {
	in, counts := hdUnitPlanFixture(t)
	steadyBase := indexedmap.NativeUnitPresentStripLayoutFor(0, 0, 0, 0).WorkOffset
	work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	view := modernHDMapViewport()
	cases := 0
	for _, base := range []int{steadyBase, steadyBase - 12*fdicon.NativeMapStride} {
		for pose := 0; pose < 4; pose++ {
			for motion := 0; motion <= 6; motion++ {
				for _, force := range []bool{false, true} {
					for shift := 0; shift < 2; shift++ {
						for _, position := range [][2]int{{-1, -1}, {0, 0}, {12, 8}, {13, 0}, {0, 9}, {-2, 0}, {4, 3}} {
							unit := fdicon.NativeUnitLayerEntry{X: in.CameraX + position[0], Y: in.CameraY + position[1], Slot: 1, Pose: pose, MotionOffset: motion, ForceBase: force, Flags: 0x80}
							in.Units = []fdicon.NativeUnitLayerEntry{unit}
							in.PixelShift = shift
							clear(work)
							if err := in.UnitBank.BlitNativeUnitLayer(work, fdicon.NativeMapStride, in.SelectorCache, in.Units, in.CameraX, in.CameraY, 12, 7, in.IdleCycle, in.MovingCycle, shift); err != nil {
								t.Fatal(err)
							}
							plan, err := buildModernHDMapUnitPlan(in, base, counts)
							if err != nil {
								t.Fatal(err)
							}
							x, y := -10000, -10000
							if len(plan) == 1 {
								command := plan[0]
								x = (command.Rect.Min.X - view.Min.X) / 6
								y = (command.Rect.Min.Y - view.Min.Y) / 6
								wantFrame := pose*3 + 2
								if motion == 0 {
									wantFrame = pose*3 + 1
								}
								if force {
									wantFrame = pose * 3
								}
								if command.Group != 0 || command.Frame != wantFrame || !command.PaletteBand || command.Rect.Dx() != 144 || command.Rect.Dy() != 144 {
									t.Fatalf("wrong raw selector/geometry: %+v", command)
								}
							} else if len(plan) != 0 {
								t.Fatalf("plan count=%d", len(plan))
							}
							for row := 0; row < view.Dy()/6; row++ {
								for column := 0; column < view.Dx()/6; column++ {
									native := work[base+row*fdicon.NativeMapStride+column] != 0
									if native != (column == x && row == y) {
										t.Fatalf("native write mismatch pose=%d motion=%d force=%v shift=%d pos=%v base=%x at(%d,%d) plan(%d,%d)", pose, motion, force, shift, position, base, column, row, x, y)
									}
								}
							}
							cases++
						}
					}
				}
			}
		}
	}
	if cases != 1568 {
		t.Fatalf("cases=%d", cases)
	}
}

func TestModernHDMapUnitPlanKeepsBlankNativeSlotsAndRejectsMissingSources(t *testing.T) {
	in, counts := hdUnitPlanFixture(t)
	base := indexedmap.NativeUnitPresentStripLayoutFor(0, 0, 0, 0).WorkOffset
	in.Units = []fdicon.NativeUnitLayerEntry{{X: 2, Y: 2, Slot: 3}, {X: 3, Y: 3, Slot: 2, Pose: 1}, {X: 4, Y: 3, Slot: 1, Inactive: true}}
	plan, err := buildModernHDMapUnitPlan(in, base, counts)
	if err != nil || len(plan) != 0 {
		t.Fatalf("blank/remap/inactive slots drawn: %+v %v", plan, err)
	}
	in.Units = []fdicon.NativeUnitLayerEntry{{X: 2, Y: 2, Slot: 1}, {X: 3, Y: 3, Slot: 4}}
	if plan, err := buildModernHDMapUnitPlan(in, base, counts); err == nil || plan != nil {
		t.Fatal("missing second source published a partial plan")
	}
	in.Units = []fdicon.NativeUnitLayerEntry{{X: 2, Y: 2, Slot: 1, Pose: 4}}
	if plan, err := buildModernHDMapUnitPlan(in, base, counts); err == nil || plan != nil {
		t.Fatal("invalid raw pose admitted")
	}
	in.Units = []fdicon.NativeUnitLayerEntry{{X: 2, Y: 2, Slot: 1}}
	if plan, err := buildModernHDMapUnitPlan(in, indexedmap.NativeUnitPresentWorkSize, counts); err == nil || plan != nil {
		t.Fatal("invalid viewport admitted")
	}
}
