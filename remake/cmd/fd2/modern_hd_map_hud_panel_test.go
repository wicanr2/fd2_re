package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func hudPanelTestSource(t *testing.T) (*Game, indexedmap.NativeFrameInput) {
	t.Helper()
	g, in := hudHDTestInput(t)
	g.font = loadFont()
	if g.font == nil {
		t.Fatal("Noto font required")
	}
	var err error
	in.Frames, err = indexedmap.LoadSeparatedNativeMapHUDFrames(separatedAssetPath("ui"))
	if err != nil {
		t.Fatal(err)
	}
	return g, in
}

// The expectation draws from explicit source ink and literal native formatting;
// it does not call the production HUD text renderer or formatter.
func hudPanelExpectedText(t *testing.T, f *Font, text string, ink color.RGBA, signHeight int) *image.RGBA {
	t.Helper()
	out := image.NewRGBA(image.Rect(0, 0, len(text)*36, 48))
	for i, ch := range text {
		h := 48
		if i == 0 && signHeight != 0 {
			h = signHeight * 6
		}
		for size := 48; size >= 24; size-- {
			face, _ := f.faceFor(size)
			bound, _ := font.BoundString(face, string(ch))
			if bound.Max.X.Ceil()-bound.Min.X.Floor() > 36 || bound.Max.Y.Ceil()-bound.Min.Y.Floor() > h {
				continue
			}
			d := font.Drawer{Dst: out, Src: image.NewUniform(ink), Face: face,
				Dot: fixed.P(i*36-bound.Min.X.Floor(), -bound.Min.Y.Floor())}
			d.DrawString(string(ch))
			break
		}
	}
	return out
}

func TestModernHDHUDPanelFormattingAndFrozenCache(t *testing.T) {
	g, in := hudPanelTestSource(t)
	palette := g.nativeMapAssets.Palette // distinct native index colours.
	original, _ := json.Marshal(in.Frames)
	var previous *modernHDMapHUDArt
	for control, want := range [][2]string{{"+05", "+00"}, {"+00", "+00"}, {"-05", "+10"}, {"-05", "+10"}, {"-05", "-05"}, {"+00", "+00"}} {
		in.HUD.TerrainControl = byte(control)
		art, err := prepareModernHUDArt(in, palette, g.font, previous)
		if err != nil {
			t.Fatal(err)
		}
		for row, text := range want {
			// Derive ink from the native source rather than assign a new colour.
			for place, ch := range text {
				glyph := in.Frames.PositiveSign
				x, h := 43, 7
				if ch == '-' {
					glyph, h = in.Frames.NegativeSign, 5
				} else if place > 0 {
					glyph, x, h = in.Frames.Digits[int(ch-'0')], 51+(place-1)*6, 8
				}
				ink, err := modernHUDGlyphInk(glyph, palette)
				if err != nil {
					t.Fatal(err)
				}
				expected := hudPanelExpectedText(t, g.font, string(ch), ink, h)
				for y := 0; y < 48; y++ {
					for xx := 0; xx < 36; xx++ {
						if c := expected.RGBAAt(xx, y); c.A != 0 {
							got := art.Base.RGBAAt(x*6+xx, (8+11*row)*6+y)
							bg := art.Base.RGBAAt(40*6, 3*6)
							// Compare premultiplied alpha over the native body.
							for _, pair := range [][3]byte{{got.R, c.R, bg.R}, {got.G, c.G, bg.G}, {got.B, c.B, bg.B}} {
								v := int(pair[1]) + (int(pair[2])*(255-int(c.A))+127)/255
								if delta := int(pair[0]) - v; delta < -1 || delta > 1 {
									t.Fatal("wrong sign/digit", control, row, place, xx, y)
								}
							}
						}
					}
				}
			}
		}
		frozen := append([]byte(nil), art.Base.Pix...)
		same, err := prepareModernHUDArt(in, palette, g.font, art)
		if err != nil || same != art {
			t.Fatal("stable HUD did not reuse art", err)
		}
		in.HUD.OptionalUnit.Current++
		changed, err := prepareModernHUDArt(in, palette, g.font, art)
		if err != nil || changed == art || !bytes.Equal(art.Base.Pix, frozen) {
			t.Fatal("frozen HUD mutated", err)
		}
		in.HUD.OptionalUnit.Current--
		previous = art
	}
	for _, hp := range []struct {
		current, maximum uint16
		text             string
	}{{0, 0, "000"}, {5, 10, "005"}, {999, 999, "999"}, {1000, 1000, "???"}, {65535, 65534, "???"}} {
		in.HUD.OptionalUnit.Current, in.HUD.OptionalUnit.Maximum = hp.current, hp.maximum
		art, err := prepareModernHUDArt(in, palette, g.font, nil)
		if err != nil {
			t.Fatal(err)
		}
		for i, ch := range hp.text {
			digits, overflow := in.Frames.Digits, in.Frames.HPEqualOverflow
			if hp.current != hp.maximum {
				digits, overflow = in.Frames.HPMismatchDigits, in.Frames.HPMismatchOverflow
			}
			glyph := overflow
			if ch != '?' {
				glyph = digits[int(ch-'0')]
			}
			ink, err := modernHUDGlyphInk(glyph, palette)
			if err != nil {
				t.Fatal(err)
			}
			want := hudPanelExpectedText(t, g.font, string(ch), ink, 0)
			for y := 0; y < 48; y++ {
				for x := 0; x < 36; x++ {
					if art.Above.RGBAAt((9+i*6)*6+x, 126+y) != want.RGBAAt(x, y) {
						t.Fatal("HP format/colour", hp, i, x, y)
					}
				}
			}
		}
		for y := 0; y < 204; y++ {
			for x := 0; x < 414; x++ {
				if !image.Pt(x, y).In(image.Rect(54, 126, 162, 174)) && art.Above.RGBAAt(x, y).A != 0 {
					t.Fatal("HP escaped native slot")
				}
			}
		}
	}
	after, _ := json.Marshal(in.Frames)
	if !bytes.Equal(original, after) {
		t.Fatal("native frame mutated")
	}
}

func TestModernHDHUDPanelAdmissionGeometryAndEvents(t *testing.T) {
	g, in := hudPanelTestSource(t)
	palette := g.nativeMapAssets.Palette
	for _, anchor := range []int{1, 242} {
		in.HUD.AnchorX = anchor
		for _, unit := range []bool{false, true} {
			x := in
			if !unit {
				x.HUD.OptionalUnit = nil
			}
			p, err := g.prepareModernHDMapHUDPanel(x, palette)
			if err != nil || p == nil || p.Rect != image.Rect((anchor+4)*6, 966, (anchor+73)*6, 1170) || (p.NativeUnit != nil) != unit {
				t.Fatal("panel geometry/admission", p, err)
			}
			ui := image.NewRGBA(image.Rect(0, 0, 320, 200))
			for i := range ui.Pix {
				ui.Pix[i] = 255
			}
			clearModernHDHUDPanelPixels(ui, p)
			for y := 0; y < 200; y++ {
				for xx := 0; xx < 320; xx++ {
					inside := image.Pt(xx*6, y*6).In(p.Rect)
					if (ui.RGBAAt(xx, y).A == 0) != inside {
						t.Fatal("cleared unrelated UI")
					}
				}
			}
			f := &modernHDMapFrame{Background: image.NewRGBA(ui.Bounds()), UI: ui, HUDPanel: p}
			if _, err := rasterUnitPresentHDRegion(f, p.Rect); err == nil {
				t.Fatal("HUD accepted as UI-free strip")
			}
			event := modernPlayer9EventFrame(f, make([]byte, 64000), palette)
			if event.HUDPanel == p || event.HUDPanel.Rect != p.Rect.Add(image.Pt(-24, -24)) || p.Rect.Min.Y != 966 {
				t.Fatal("event mutated frozen geometry")
			}
			if commandHDObjectFrame(f, make([]byte, 64000), palette).HUDPanel != nil {
				t.Fatal("object retained HUD")
			}
		}
	}
	for _, kind := range []string{"gate-a", "gate-b", "no-font", "palette", "panel", "translucent", "terrain", "state", "digits", "control"} {
		t.Run(kind, func(t *testing.T) {
			g, in := hudPanelTestSource(t)
			palette := append(color.Palette(nil), g.nativeMapAssets.Palette...)
			closed := false
			switch kind {
			case "gate-a":
				in.HUD.DisplayGateA = false
				closed = true
			case "gate-b":
				in.HUD.DisplayGateB = false
				closed = true
			case "no-font":
				g.font = nil
				closed = true
			case "palette":
				palette[0] = nil
			case "translucent":
				palette[0] = color.RGBA{}
			case "panel":
				in.Frames.Panel.Width--
			case "terrain":
				in.HUD.TerrainDescriptor = 1024
			case "state":
				in.HUD.OptionalUnit.RawState = 4
			case "digits":
				in.Frames.Digits[0].Width = 0
			case "control":
				in.HUD.TerrainControl = 6
			}
			if closed {
				in.HUDTerrain = nil
				palette = nil
			}
			p, err := g.prepareModernHDMapHUDPanel(in, palette)
			if closed && (err != nil || p != nil) || !closed && (err == nil || p != nil) {
				t.Fatal(fmt.Sprint(kind, ": ", p, err))
			}
		})
	}
}

func TestModernHDHUDPanelNativeTransactionAndTerrain(t *testing.T) {
	hd, in := hudPanelTestSource(t)
	native := modernHDMapRuntimeFixture(t)
	hd.st.Units[0].OnField, native.st.Units[0].OnField = true, true
	hd.nativeMapAssets.Frames, native.nativeMapAssets.Frames = in.Frames, in.Frames
	native.modernHDMapComposing = false
	asset := &modernTerrainHDAsset{Image: terrainTestImage(color.RGBA{30, 100, 70, 255}), Role: "terrain"}
	hd.modernStoryPortraits.terrainHD = map[int]map[int]*modernTerrainHDAsset{0: {0: asset}}
	for _, now := range []time.Time{time.Unix(500, 0), time.Unix(500, 300000000)} {
		if err := native.composeNativeMapFrameAt(now); err != nil {
			t.Fatal(err)
		}
		if err := hd.composeNativeMapFrameAt(now); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(hd.nativeMapWork, native.nativeMapWork) || !bytes.Equal(hd.nativeMapVGA, native.nativeMapVGA) ||
			!reflect.DeepEqual(hd.nativeMapClock, native.nativeMapClock) || hd.st.NativeMapCycleState != native.st.NativeMapCycleState {
			t.Fatal("HD HUD changed native transaction")
		}
		panel := hd.modernHDMapFrame.HUDPanel
		if panel == nil || !panel.TerrainHD || panel.Terrain == nil || hd.modernHDMapFrame.HUDUnit == nil {
			t.Fatal("HUD did not consume registered terrain and unit")
		}
	}
	// The HUD raw-blits terrain independently of the world LUT.
	palette := append(color.Palette(nil), hd.nativeMapAssets.Palette...)
	palette[1] = color.RGBA{90, 20, 10, 255}
	input := *hd.nativeMapFrameSource
	p, err := hd.prepareModernHDMapHUDPanel(input, palette)
	// Existing material mapping adds target(90,20,10)-base(1,1,1) to
	// painting(30,100,70), independently of the all-zero world LUT.
	if err != nil || p.Terrain.RGBAAt(0, 0) != (color.RGBA{119, 119, 79, 255}) {
		t.Fatal("HUD lost raw palette path", err)
	}
	delete(hd.modernStoryPortraits.terrainHD[0], 0)
	p, err = hd.prepareModernHDMapHUDPanel(input, hd.nativeMapAssets.Palette)
	if err != nil || p.TerrainHD || p.Terrain.RGBAAt(0, 0) != (color.RGBA{1, 1, 1, 255}) {
		t.Fatal("native terrain fallback changed", err)
	}
	for _, row := range []int{0, 1} {
		// 0x24B4D stages nine rows; the steady fixture supplies only eight.
		staging := input
		staging.Frame.Cells = append(append([]fdicon.NativeTerrainCell(nil), input.Frame.Cells...), input.Frame.Cells[:13]...)
		frame, err := hd.prepareModernHDMapFrameLayout(staging, hd.modernHDMapFrame.TerrainUnderlay, hd.nativeMapAssets.Palette, true, row)
		if err != nil {
			t.Fatal("row projection", row, err)
		}
		if frame.HUDPanel == nil || frame.HUDPanel.Rect.Min.Y != 966-row*6 || frame.HUDUnit.Unit.Rect.Min.Y != 996-row*6 {
			t.Fatal("HUD row shift", row)
		}
	}
	work, vga, before := append([]byte(nil), hd.nativeMapWork...), append([]byte(nil), hd.nativeMapVGA...), hd.modernHDMapFrame
	hd.font = &Font{}
	if err := hd.composeNativeMapFrameAt(time.Unix(501, 0)); err == nil {
		t.Fatal("bad font published")
	}
	if hd.modernHDMapFrame != before || !bytes.Equal(work, hd.nativeMapWork) || !bytes.Equal(vga, hd.nativeMapVGA) {
		t.Fatal("bad HUD changed published state")
	}
	hd.font = loadFont()
	hd.bannerT = 1
	if err := hd.composeNativeMapFrameAt(time.Unix(501, 0)); err != nil {
		t.Fatal(err)
	}
	if hd.modernHDMapFrame.HUDPanel != nil || hd.modernHDMapFrame.HUDUnit != nil {
		t.Fatal("step owner retained HUD")
	}
}
