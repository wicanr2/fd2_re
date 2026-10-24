package main

import (
	"bytes"
	"image"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func hudHDTestInput(t *testing.T) (*Game, indexedmap.NativeFrameInput) {
	t.Helper()
	g := modernHDMapRuntimeFixture(t)
	f, _ := hdUnitPlanFixture(t)
	return g, indexedmap.NativeFrameInput{HUD: indexedmap.NativeMapHUDInput{
		DisplayGateA: true, DisplayGateB: true, AnchorX: 1,
		OptionalUnit: &indexedmap.NativeMapHUDOptionalUnit{SelectorSlot: 2, Current: 5, Maximum: 10},
	}, Frames: nativeFrameTestHUDFrames(), HUDTerrain: g.nativeMapAssets.Terrain, HUDUnits: f.UnitBank, HUDCache: f.SelectorCache}
}

func TestModernHDHUDUnitSelectionAndHP(t *testing.T) {
	g, in := hudHDTestInput(t)
	// Slot 2 resolves to raw group 48, whose HD source has only three poses.
	g.modernStoryPortraits.mapSpritesHD[48] = g.modernStoryPortraits.mapSpritesHD[48][:3]
	for _, anchor := range []int{1, 242} {
		for state, wantFrame := range []int{0, 1, 2, 1} {
			for _, hp := range [][2]uint16{{5, 10}, {10, 10}, {1000, 1000}, {1000, 1200}} {
				in.HUD.AnchorX, in.HUD.OptionalUnit.RawState = anchor, state
				in.HUD.OptionalUnit.Current, in.HUD.OptionalUnit.Maximum = hp[0], hp[1]
				p, err := g.prepareModernHDMapHUDUnit(in, g.nativeMapAssets.Palette)
				if err != nil {
					t.Fatal(err)
				}
				if p == nil || p.Unit.Group != 48 || p.Unit.Frame != wantFrame || p.Unit.PaletteBand || p.Unit.Rect != image.Rect((anchor+10)*6, 996, (anchor+34)*6, 1140) {
					t.Fatalf("wrong HUD selector: %+v", p)
				}
				// Independent native HP raster, then crop using the documented offsets.
				raw := make([]byte, 456*200)
				if err := indexedmap.BlitNativeMapHUDHP(in.Frames, raw, anchor, hp[0], hp[1]); err != nil {
					t.Fatal(err)
				}
				for y := 0; y < 24; y++ {
					for x := 0; x < 24; x++ {
						if p.Base.RGBAAt(x, y).A != 255 {
							t.Fatal("panel restoration is not opaque")
						}
						at := (162+y)*456 + anchor + 6 + x
						v := p.Above.RGBAAt(x, y)
						if raw[at] != 0 && (v.R != raw[at] || v.A != 255) || raw[at] == 0 && v.A != 0 {
							t.Fatal("HP layer lost its native position/colour")
						}
					}
				}
				ui := image.NewRGBA(image.Rect(0, 0, 320, 200))
				for i := range ui.Pix {
					ui.Pix[i] = 255
				}
				clearModernHDHUDUnitPixels(ui, p)
				for y := 0; y < 200; y++ {
					for x := 0; x < 320; x++ {
						inside := x >= anchor+10 && x < anchor+34 && y >= 166 && y < 190
						if (ui.RGBAAt(x, y).A == 0) != inside {
							t.Fatal("HUD replacement erased unrelated UI")
						}
					}
				}
			}
		}
	}
}

func TestModernHDHUDUnitAdmissionAndEventShift(t *testing.T) {
	g, in := hudHDTestInput(t)
	for _, gate := range []int{0, 1, 2} {
		bad := in
		bad.HUDUnits = nil
		if gate == 0 {
			bad.HUD.DisplayGateA = false
		}
		if gate == 1 {
			bad.HUD.DisplayGateB = false
		}
		if gate == 2 {
			bad.HUD.OptionalUnit = nil
		}
		if p, e := g.prepareModernHDMapHUDUnit(bad, nil); e != nil || p != nil {
			t.Fatal("closed HUD touched absent sources")
		}
	}
	in.HUD.OptionalUnit.SelectorSlot = 3 // remap-only group 39 stays native.
	if p, e := g.prepareModernHDMapHUDUnit(in, g.nativeMapAssets.Palette); e != nil || p != nil {
		t.Fatal("remap-only source acquired invented art", e)
	}
	in.HUD.OptionalUnit.SelectorSlot = 2
	p, e := g.prepareModernHDMapHUDUnit(in, g.nativeMapAssets.Palette)
	if e != nil {
		t.Fatal(e)
	}
	frame := &modernHDMapFrame{Background: image.NewRGBA(image.Rect(0, 0, 320, 200)), UI: image.NewRGBA(image.Rect(0, 0, 320, 200)), HUDUnit: p}
	event := modernPlayer9EventFrame(frame, make([]byte, 320*200), g.nativeMapAssets.Palette)
	if event.HUDUnit == p || event.HUDUnit.Unit.Rect != p.Unit.Rect.Add(image.Pt(-24, -24)) || p.Unit.Rect.Min.Y != 996 {
		t.Fatal("event HUD did not shift independently")
	}
	object := commandHDObjectFrame(frame, make([]byte, 320*200), g.nativeMapAssets.Palette)
	if object.HUDUnit != nil || !bytes.Equal(object.UI.Pix, make([]byte, len(object.UI.Pix))) {
		t.Fatal("sampled object frame retained the fixed HUD")
	}
	for _, kind := range []string{"missing", "size", "state", "slot", "panel"} {
		t.Run(kind, func(t *testing.T) {
			g, in := hudHDTestInput(t)
			switch kind {
			case "missing":
				delete(g.modernStoryPortraits.mapSpritesHD, 48)
			case "size":
				g.modernStoryPortraits.mapSpritesHD[48][0] = image.NewRGBA(image.Rect(0, 0, 24, 24))
			case "state":
				in.HUD.OptionalUnit.RawState = 4
			case "slot":
				in.HUD.OptionalUnit.SelectorSlot = 80
			case "panel":
				in.Frames.Panel.Pixels = nil
				in.Frames.Panel.Indexed = make([]byte, 69*34)
				in.Frames.Panel.Mask = make([]byte, 69*34)
				in.HUDTerrain = nativeFrameTestBank(2, 1)
				for i := range in.HUDTerrain.Sprites {
					clear(in.HUDTerrain.Sprites[i].Mask)
				}
			}
			if p, e := g.prepareModernHDMapHUDUnit(in, g.nativeMapAssets.Palette); e == nil || p != nil {
				t.Fatal("invalid HUD published", kind)
			}
		})
	}
	// Preparing a HUD never edits the original source bank.
	before := append([]byte(nil), in.HUDUnits.Sprites[48*12].Pixels...)
	_, _ = g.prepareModernHDMapHUDUnit(in, g.nativeMapAssets.Palette)
	if !bytes.Equal(before, in.HUDUnits.Sprites[48*12].Pixels) {
		t.Fatal("HD preparation mutated original pixels")
	}
}
