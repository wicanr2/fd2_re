package main

import (
	"bytes"
	"encoding/binary"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestModernBattlePanelTextFrozenSources(t *testing.T) {
	g, actor, _ := physicalSceneTestGame(t)
	attachOfficialLocale(t, g)
	g.font = loadFont()
	if g.font == nil {
		t.Fatal("Noto font required")
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	scene := &nativePhysicalScene{panelAssets: assets}
	record, err := battle.NativeBattlePanelRecordForUnit(actor)
	if err != nil {
		t.Fatal(err)
	}
	base := make([]byte, 64000)
	if err := g.renderPhysicalBattlePanel(scene, record, base, 0); err != nil {
		t.Fatal(err)
	}
	first := append([]byte(nil), base...)
	// A malformed BG is rejected before attaching otherwise valid typed HUD.
	rejected := &modernHDBattleSources{Base: append([]byte(nil), first...)}
	badScene := &nativePhysicalScene{hdPanelText: scene.hdPanelText, backgroundFrame: fdother.Frame{Width: 320, Height: 100}}
	if err := bindModernPhysicalScenery(badScene, rejected); err == nil || len(rejected.BaseScenery) != 0 || !bytes.Equal(rejected.Base, first) {
		t.Fatal("rejected scenery attached partial panel metadata", err)
	}
	binary.LittleEndian.PutUint16(record[0x40:], 73)
	if err := g.renderPhysicalBattlePanel(scene, record, base, 0); err != nil {
		t.Fatal(err)
	}
	if len(scene.hdPanelText) != 2 || scene.hdPanelText[0].values.HP != 80 || scene.hdPanelText[1].values.HP != 73 {
		t.Fatal("panel snapshots read later HP")
	}
	for i, pixels := range [][]byte{first, base} {
		s := &modernHDBattleSources{Base: pixels}
		bindModernPhysicalPanelText(scene, s)
		bindModernPhysicalPanelText(scene, s)
		if len(s.BaseScenery) != 1 || s.BaseScenery[0].PanelText != scene.hdPanelText[i] {
			t.Fatal("source identity or duplicate layer")
		}
		p := s.BaseScenery[0]
		check := append([]byte(nil), pixels...)
		if err := p.Frame.BlitAt(check, 320); err != nil || !bytes.Equal(check, pixels) {
			t.Fatal("native reconstruction changed")
		}
		plan, err := g.modernHDBattlePlan(s, g.nativeUIPalette, .5)
		if err != nil || len(plan.Layers) != 1 || !plan.Layers[0].PreservePalette || plan.Layers[0].HD == nil {
			t.Fatal("HD panel plan lost source or palette exemption", err)
		}
		for y := 4; y < 21; y++ {
			if plan.Layers[0].HD.At(4*6, y*6) != color.RGBAModel.Convert(g.nativeUIPalette[s.BaseScenery[0].PanelText.background]) {
				t.Fatal("native name left-shadow survived")
			}
		}
		broken := append([]byte(nil), pixels...)
		broken[4*320+171] ^= 1
		bad := &modernHDBattleSources{Base: broken}
		bindModernPhysicalPanelText(scene, bad)
		if len(bad.BaseScenery) != 0 {
			t.Fatal("mutated native source accepted")
		}
	}
	// Overflow remains the real original marker; it is not reformatted.
	binary.LittleEndian.PutUint16(record[0x40:], 1234)
	record[0x21] = 255
	if err := g.renderPhysicalBattlePanel(scene, record, base, 0); err != nil {
		t.Fatal(err)
	}
	p := scene.hdPanelText[len(scene.hdPanelText)-1]
	picture, _, err := p.raster(g.nativeUIPalette)
	if err != nil {
		t.Fatal(err)
	}
	for y := 21; y < 30; y++ {
		for x := 126; x < 145; x++ {
			want := g.nativeUIPalette[p.native.Pixels[y*149+x]]
			if picture.At(x*6, y*6) != color.RGBAModel.Convert(want) {
				t.Fatal("overflow marker was repainted")
			}
		}
	}
	for y := 4; y < 13; y++ {
		for x := 132; x < 147; x++ {
			if picture.At(x*6, y*6) != color.RGBAModel.Convert(g.nativeUIPalette[p.native.Pixels[y*149+x]]) {
				t.Fatal("level overflow marker was repainted")
			}
		}
	}
	bad := *p
	bad.name = string(bytes.Repeat([]byte("W"), 1000))
	bad.picture = nil
	if _, _, err := bad.raster(g.nativeUIPalette); err == nil || bad.picture != nil {
		t.Fatal("overflowing text published a partial panel")
	}
	before := append([]byte(nil), base...)
	g.font = nil
	scene.hdPanelText = nil
	if err := g.renderPhysicalBattlePanel(scene, record, base, 0); err != nil || len(scene.hdPanelText) != 0 || !bytes.Equal(before, base) {
		t.Fatal("native fontless path changed", err)
	}
}

func TestModernBattlePanelTextGPU(t *testing.T) {
	if os.Getenv("FD2_HD_PANEL_GPU") != "1" {
		t.Skip("dedicated GPU process required")
	}
	g, actor, _ := physicalSceneTestGame(t)
	g.font = loadFont()
	if g.font == nil {
		t.Fatal("Noto required")
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	var cases []modernHDBattleGPUCase
	for _, locale := range []string{"zh-Hant", "zh-Hans", "ja", "en"} {
		g.localeID = locale
		g.localeEntities, err = loadOfficialLocaleEntities(locale)
		if err != nil {
			t.Fatal(err)
		}
		for _, side := range []byte{0, 2} {
			record, err := battle.NativeBattlePanelRecordForUnit(actor)
			if err != nil {
				t.Fatal(err)
			}
			record[6] = side
			record[8] = 78
			binary.LittleEndian.PutUint16(record[0x40:], 73)
			scene := &nativePhysicalScene{panelAssets: assets}
			base := make([]byte, 64000)
			if err := g.renderPhysicalBattlePanel(scene, record, base, 0); err != nil {
				t.Fatal(err)
			}
			p := scene.hdPanelText[0]
			picture, key, err := p.raster(g.nativeUIPalette)
			if err != nil {
				t.Fatal(err)
			}
			for _, shift := range []int{-100, 0, 100} {
				f := p.native
				r := image.Rect((f.X+shift)*6, f.Y*6, (f.X+shift+149)*6, (f.Y+42)*6)
				clip := image.Rect(0, 0, 1920, 1200)
				plan := modernHDBattlePlan{Base: make([]byte, 64000), Palette: g.nativeUIPalette, HDSubtract: .75, HDMix: .5,
					Layers: []modernHDBattleLayer{{Native: f, HD: picture, HDKey: key, Rect: r, Fill: -1, Clip: &clip, PreservePalette: true}}}
				want := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
				draw.Draw(want, want.Bounds(), image.NewUniform(g.nativeUIPalette[0]), image.Point{}, draw.Src)
				draw.Draw(want, r, picture, image.Point{}, draw.Over)
				cases = append(cases, modernHDBattleGPUCase{name: locale + string(rune(side+'0')) + r.String(), plan: plan, want: want, tolerance: 1})
			}
		}
	}
	probe := &modernHDBattleGPUProbe{cases: cases}
	defer probe.gpu.close()
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if probe.err != nil || probe.index != 24 {
		t.Fatal("panel GPU", probe.index, probe.err)
	}
}

func TestModernBattlePanelTextFourLocaleInk(t *testing.T) {
	f := loadFont()
	if f == nil {
		t.Fatal("Noto font required")
	}
	for _, locale := range []string{"zh-Hant", "zh-Hans", "ja", "en"} {
		catalog, err := loadOfficialLocaleEntities(locale)
		if err != nil {
			t.Fatal(err)
		}
		regional := f.hdFonts[locale]
		if regional == nil {
			t.Fatal("regional face required")
		}
		count := 0
		for id := 0; id < 94; id++ {
			name, err := catalog.BattleName(id)
			if err != nil {
				continue
			}
			canvas := image.NewRGBA(image.Rect(0, 0, 720, 120))
			rect := image.Rect(20, 10, 674, 106)
			if err := drawModernBattlePanelInk(canvas, regional, name, rect, 96, color.RGBA{255, 255, 255, 255}); err != nil {
				t.Fatalf("%s/%d: %v", locale, id, err)
			}
			ink := 0
			partial := 0
			for y := 0; y < 120; y++ {
				for x := 0; x < 720; x++ {
					a := canvas.RGBAAt(x, y).A
					if a != 0 {
						ink++
						if !image.Pt(x, y).In(rect) {
							t.Fatal("ink escaped safe bounds")
						}
					}
					if a > 0 && a < 255 {
						partial++
					}
				}
			}
			if name != "" && (ink == 0 || partial == 0) {
				t.Fatalf("%s/%d lacks actual antialiased ink", locale, id)
			}
			count++
		}
		if count != 57 {
			t.Fatal("incomplete name coverage", locale, count)
		}
	}
}

func TestModernBattlePanelTextFourLocalePreview(t *testing.T) {
	out := os.Getenv("FD2_HD_PANEL_PREVIEW")
	g, actor, _ := physicalSceneTestGame(t)
	g.font = loadFont()
	if g.font == nil {
		t.Fatal("Noto font required")
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 894, 252*4))
	for i, locale := range []string{"zh-Hant", "zh-Hans", "ja", "en"} {
		g.localeID = locale
		g.localeEntities, err = loadOfficialLocaleEntities(locale)
		if err != nil {
			t.Fatal(err)
		}
		scene := &nativePhysicalScene{panelAssets: assets}
		record, err := battle.NativeBattlePanelRecordForUnit(actor)
		if err != nil {
			t.Fatal(err)
		}
		record[8] = 78
		binary.LittleEndian.PutUint16(record[0x40:], 73)
		if err := g.renderPhysicalBattlePanel(scene, record, make([]byte, 64000), 0); err != nil {
			t.Fatal(err)
		}
		if len(scene.hdPanelText) != 1 {
			t.Fatal("panel missing")
		}
		picture, _, err := scene.hdPanelText[0].raster(g.nativeUIPalette)
		if err != nil {
			t.Fatal(err)
		}
		draw.Draw(canvas, image.Rect(0, i*252, 894, (i+1)*252), picture, image.Point{}, draw.Src)
	}
	if out != "" {
		if !filepath.IsAbs(out) {
			t.Fatal("absolute preview path required")
		}
		file, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, canvas); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
