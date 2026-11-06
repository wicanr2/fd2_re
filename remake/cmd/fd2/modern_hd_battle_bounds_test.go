package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestModernHDBattleExplicitBoundsAtomic(t *testing.T) {
	for _, kind := range []string{"valid", "missing", "negative", "empty", "overflow", "png-size", "v1-rectangles", "native-descriptor"} {
		t.Run(kind, func(t *testing.T) {
			a, native, root := battleHDAssetFixture(t)
			a.ConsumerContract, a.LayoutPolicy = "native_figani_rgba_hd6_v2", "explicit_hd_frame_rects_v2"
			a.HDFrames = []modernBattleHDRectangle{{X: 108, Y: 168, Width: 24, Height: 18}, {X: 100, Y: 170, Width: 12, Height: 12}}
			picture := image.NewNRGBA(image.Rect(0, 0, 24, 18))
			picture.SetNRGBA(2, 2, color.NRGBA{R: 240, A: 255}) // 原圖框以外仍有實際墨跡。
			file, err := os.Create(filepath.Join(root, a.Files[0]))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(file, picture); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(root, a.Files[0]))
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(raw)
			a.FrameSHA256[0] = hex.EncodeToString(digest[:])
			switch kind {
			case "missing":
				a.HDFrames = a.HDFrames[:1]
			case "negative":
				a.HDFrames[1].X = -1
			case "empty":
				a.HDFrames[1].Width = 0
			case "overflow":
				a.HDFrames[1].X = int(^uint(0) >> 1)
			case "png-size":
				a.HDFrames[1].Height++
			case "v1-rectangles":
				a.ConsumerContract, a.LayoutPolicy = "native_figani_rgba_hd6_v1", "contain_native_descriptor_hd6_candidate"
			case "native-descriptor":
				a.NativeFrames[1].X++
			}
			set := &modernStoryPortraitSet{}
			err = loadModernBattleHDAssetWithNative(set, a, root, native)
			if kind != "valid" {
				if err == nil || len(set.battleAnimations) != 0 {
					t.Fatal("invalid complete bank was published", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := set.prepareBattleHDResources(91); err != nil {
				t.Fatal(err)
			}
			entry := set.battleAnimations[91]
			if entry.hdRects[0] != image.Rect(108, 168, 132, 186) || entry.Frames[0].Bounds().Size() != image.Pt(24, 18) {
				t.Fatal("HD rectangle or PNG changed")
			}
			_, _, _, alpha := entry.Frames[0].At(2, 2).RGBA()
			if alpha != 65535 {
				t.Fatal("ink outside native box was lost")
			}
			// 已載入的完整資源不因下一個缺件交易而遺失。
			other, otherNative, otherRoot := battleHDAssetFixture(t)
			other.SourceResource = 90
			if err := loadModernBattleHDAssetWithNative(set, other, otherRoot, otherNative); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(otherRoot, other.Files[1])); err != nil {
				t.Fatal(err)
			}
			if err := set.prepareBattleHDResources(90); err == nil || entry.Frames[0] == nil || set.battleAnimations[90].Frames[0] != nil {
				t.Fatal("failed transaction changed either bank")
			}
		})
	}
}

type explicitBattleBoundsProbe struct {
	gpu    modernHDBattleGPU
	plans  [3]modernHDBattlePlan
	index  int
	err    error
	pixels []byte
}

func (p *explicitBattleBoundsProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *explicitBattleBoundsProbe) Update() error {
	if p.err != nil || p.index == 9 {
		return ebiten.Termination
	}
	return nil
}
func (p *explicitBattleBoundsProbe) Draw(*ebiten.Image) {
	if p.err != nil || p.index == 9 {
		return
	}
	q := []hdTextureQuality{hdTextureFull, hdTextureBalanced, hdTextureLight}[p.index/3]
	p.gpu.setQuality(q)
	mode := p.index % 3
	if p.err = p.gpu.draw(p.plans[mode]); p.err != nil {
		return
	}
	readHDQualityWorldPixels(p.gpu.World, p.pixels)
	// 世界(132,162)位於舊人物框外。只有完整HD能畫出紅色。
	b := p.gpu.World.Bounds()
	x, y := 132*b.Dx()/1920, 162*b.Dy()/1200
	i := (y*b.Dx() + x) * 4
	red := p.pixels[i] > 240 && p.pixels[i+1] < 10 && p.pixels[i+2] < 10
	if red != (mode == 0) {
		p.err = fmt.Errorf("%s mode%d lost outside ink or ignored caller clip/classic: %v", q.name(), mode, p.pixels[i:i+4])
		return
	}
	p.index++
}

func TestModernHDBattleExplicitBoundsGPU(t *testing.T) {
	if os.Getenv("FD2_HD_BOUNDS_GPU") != "1" {
		t.Skip("dedicated explicit HD bounds GPU process")
	}
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{0, 0, 0, 255}
	}
	palette[1] = color.RGBA{0, 0, 255, 255}
	frame := figani.Frame{X: 20, Y: 30, Width: 2, Height: 2, Pixels: []byte{1, 1, 1, 1}, Mask: []byte{255, 255, 255, 255}}
	picture := image.NewRGBA(image.Rect(0, 0, 36, 36))
	for y := 1; y < 35; y++ {
		for x := 1; x < 35; x++ {
			picture.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	set := &modernStoryPortraitSet{battleAnimations: map[int]*modernBattleHDResource{91: {Frames: []image.Image{picture}, hashes: []string{"bounds-fixture"}, hdRects: []image.Rectangle{image.Rect(108, 168, 144, 204)}}}}
	g := &Game{modernStoryPortraits: set}
	sources := &modernHDBattleSources{Base: make([]byte, 64000), Layers: []modernHDBattleSource{{Frame: frame, Resource: 91, DX: 3, DY: -2, Fill: -1}}}
	p := &explicitBattleBoundsProbe{pixels: make([]byte, 1920*1200*4)}
	var err error
	p.plans[0], err = g.modernHDBattlePlan(sources, palette, 0)
	if err != nil {
		t.Fatal(err)
	}
	clip := image.Rect(23, 28, 25, 30)
	sources.Layers[0].Clip = &clip
	p.plans[1], err = g.modernHDBattlePlan(sources, palette, 0)
	if err != nil {
		t.Fatal(err)
	}
	sources.Layers[0].Clip = nil
	g.modernStoryPortraits = nil
	p.plans[2], err = g.modernHDBattlePlan(sources, palette, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer p.gpu.close()
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 9 {
		t.Fatal(p.index, p.err)
	}
}

func TestModernHDSerifKeepsClassicAndHeavyScale(t *testing.T) {
	if os.Getenv("FD2_HD_SERIF_TEST") != "1" {
		t.Skip("requires both approved Noto Serif TTC weights")
	}
	f := loadFont()
	if f == nil {
		t.Fatal("maintained font required")
	}
	classic, err := f.sf.Name(nil, 1)
	if err != nil || classic != "Noto Sans CJK JP" {
		t.Fatal("classic font changed", classic, err)
	}
	for locale, region := range map[string]string{"zh-Hant": "TC", "zh-Hans": "SC", "ja": "JP", "en": "JP"} {
		face := f.hdFonts[locale]
		if face == nil || face.hdHeavy == nil {
			t.Fatal("explicit serif test requires both portable families", locale)
		}
		for _, row := range []struct {
			face   *Font
			weight string
		}{{face, "SemiBold"}, {face.hdHeavy, "Black"}} {
			family, err := row.face.sf.Name(nil, 1)
			if err != nil || family != "Noto Serif CJK "+region+" "+row.weight {
				t.Fatal("wrong regional serif", locale, family, err)
			}
		}
	}
	g := &Game{font: f, fontNm: loadFont()}
	g.applyFontScale(1.6)
	for _, face := range f.hdFonts {
		if face.userScale != g.fontScale || face.hdHeavy.userScale != g.fontScale {
			t.Fatal("heavy font missed user scale")
		}
	}
}

func TestModernHDBattleExplicitBoundsJSON(t *testing.T) {
	for _, raw := range []string{`{}`, `{"x":0,"y":0,"width":12}`, `{"x":null,"y":0,"width":12,"height":12}`, `{"x":true,"y":0,"width":12,"height":12}`, `{"x":0,"y":0,"width":12.5,"height":12}`, `{"x":0,"y":0,"width":12,"height":12,"extra":0}`} {
		var r modernBattleHDRectangle
		if json.Unmarshal([]byte(raw), &r) == nil {
			t.Fatal("incomplete or noninteger rectangle accepted", raw)
		}
	}
}

func TestModernHDBattleExplicitBoundsPlanAndClassic(t *testing.T) {
	frame := figani.Frame{X: 20, Y: 30, Width: 2, Height: 2, Pixels: make([]byte, 4), Mask: []byte{255, 255, 255, 255}}
	rect := image.Rect(108, 168, 132, 186)
	set := &modernStoryPortraitSet{battleAnimations: map[int]*modernBattleHDResource{91: {Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 24, 18))}, hashes: []string{"fixture"}, hdRects: []image.Rectangle{rect}}}}
	clip := image.Rect(0, 0, 320, 200)
	source := &modernHDBattleSources{Base: make([]byte, 64000), Layers: []modernHDBattleSource{{Frame: frame, Resource: 91, DX: 3, DY: -2, Fill: 7, Clip: &clip}}}
	g := &Game{modernStoryPortraits: set}
	plan, err := g.modernHDBattlePlan(source, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	layer := plan.Layers[0]
	if layer.Rect != rect.Add(image.Pt(18, -12)) || layer.Native.X != 20 || layer.Native.Y != 30 || layer.Native.Width != 2 || layer.Native.Height != 2 || layer.Fill != 7 || *layer.Clip != image.Rect(0, 0, 1920, 1200) {
		t.Fatal("HD translation, clip or native source changed", layer.Rect, err)
	}
	g.modernStoryPortraits = nil
	plan, err = g.modernHDBattlePlan(source, nil, 0)
	if err != nil || plan.Layers[0].HD != nil || plan.Layers[0].Rect != image.Rect(138, 168, 150, 180) {
		t.Fatal("classic inherited HD geometry", err)
	}
}
