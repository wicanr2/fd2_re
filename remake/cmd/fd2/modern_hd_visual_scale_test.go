package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestModernHDVisualScaleJSON(t *testing.T) {
	for _, raw := range []string{"null", "0", "114", "116", "125", "-1", "115.0", "true", `"115"`, "{}", "[]"} {
		var scale modernHDVisualScale
		if json.Unmarshal([]byte(raw), &scale) == nil {
			t.Fatalf("accepted invalid scale %s", raw)
		}
	}
	for _, raw := range []string{"{}", `{"visual_scale_percent":100}`, `{"visual_scale_percent":115}`} {
		var asset modernThemeAsset
		if err := json.Unmarshal([]byte(raw), &asset); err != nil {
			t.Fatal(err)
		}
	}
}

func TestModernHDVisualScaleFootAndLegacy(t *testing.T) {
	for _, rect := range []image.Rectangle{image.Rect(300, 300, 444, 444), image.Rect(-120, -60, 24, 84)} {
		for _, foot := range []int{6, 120, 132, 138, 144} {
			for _, scale := range []modernHDVisualScale{0, 100} {
				if scale.rect(rect, foot) != rect {
					t.Fatal("legacy geometry changed")
				}
			}
			out := modernHDVisualScale(115).rect(rect, foot)
			if out.Size() != image.Pt(166, 166) {
				t.Fatal("wrong enlarged world extent", out)
			}
			// Compare the transformed foot as a rational, without using the renderer's rounding.
			delta := (out.Min.Y-rect.Min.Y-foot)*144 + foot*out.Dy()
			centreDelta := (out.Min.X + out.Max.X) - (rect.Min.X + rect.Max.X)
			if delta < -72 || delta > 72 || centreDelta < -1 || centreDelta > 1 {
				t.Fatal("registered feet or horizontal centre moved", rect, out, foot, delta)
			}
		}
	}
}

func TestModernHDVisualScaleMapAdmissionAndRole(t *testing.T) {
	path, root, document := writeModernHDMapFixture(t)
	for _, row := range document["assets"].([]any) {
		a := row.(map[string]any)
		if a["role"] == "map_sprite_hd_set" && a["source_group"] == 16 {
			a["visual_scale_percent"] = 115
		}
	}
	writeModernHDCatalog(t, path, document)
	set, err := loadModernStoryPortraitSet(path, root)
	if err != nil {
		t.Fatal(err)
	}
	if set.mapSpriteVisualScale[16] != 115 || len(set.mapSpriteFeet[16]) != 12 || set.mapSpriteFeet[16][0] != 138 {
		t.Fatal("source foot or declared scale missing")
	}
	if set.mapSpritesHD[16][0].Bounds() != image.Rect(0, 0, 144, 144) {
		t.Fatal("source texture was resized")
	}
	for _, row := range document["assets"].([]any) {
		a := row.(map[string]any)
		if a["role"] == "map_sprite_set" {
			a["visual_scale_percent"] = 115
			break
		}
	}
	writeModernHDCatalog(t, path, document)
	if partial, err := loadModernStoryPortraitSet(path, root); err == nil || partial != nil {
		t.Fatal("native projection accepted HD scale or published partial theme")
	}
}

type visualScaleGPUProbe struct {
	mapGPU modernHDMapGPU
	battle modernHDBattleGPU
	plans  [4]modernHDBattlePlan
	unit   modernHDMapUnitDraw
	image  *image.RGBA
	index  int
	err    error
	mask   []byte
}

func (p *visualScaleGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *visualScaleGPUProbe) Update() error {
	if p.err != nil || p.index == 24 {
		return ebiten.Termination
	}
	return nil
}
func (p *visualScaleGPUProbe) Draw(*ebiten.Image) {
	if p.err != nil || p.index == 24 {
		return
	}
	q := []hdTextureQuality{hdTextureFull, hdTextureBalanced, hdTextureLight}[p.index/8]
	mode := p.index % 8
	var world *ebiten.Image
	if mode < 4 {
		p.mapGPU.setQuality(q)
		p.mapGPU.World.Clear()
		p.mapGPU.Theme.mapSpriteVisualScale[16] = 100
		if mode != 0 {
			p.mapGPU.Theme.mapSpriteVisualScale[16] = 115
		}
		clip := modernHDMapViewport()
		if mode == 2 {
			clip = p.unit.Rect
		}
		if mode == 3 {
			rect := p.mapGPU.Theme.mapSpriteVisualRect(16, 0, p.unit.Rect)
			p.mapGPU.drawTerrainRegion(p.image, rect, clip, true)
		} else {
			p.mapGPU.drawUnitClipped(p.unit, clip)
		}
		world = p.mapGPU.World
	} else {
		p.battle.setQuality(q)
		if p.err = p.battle.draw(p.plans[mode-4]); p.err != nil {
			return
		}
		world = p.battle.World
	}
	pixels := make([]byte, world.Bounds().Dx()*world.Bounds().Dy()*4)
	world.ReadPixels(pixels)
	x, y := 296*world.Bounds().Dx()/1920, 290*world.Bounds().Dy()/1200
	i := (y*world.Bounds().Dx() + x) * 4
	red := pixels[i] > 240 && pixels[i+1] < 10 && pixels[i+2] < 10
	if red != (mode == 1 || mode == 3 || mode == 5) {
		p.err = fmt.Errorf("%s mode%d outside ink/caller clip/classic mismatch: %v", q.name(), mode, pixels[i:i+4])
		return
	}
	if mode == 1 {
		p.mask = append(p.mask[:0], pixels...)
	} else if mode == 3 && !bytes.Equal(p.mask, pixels) {
		p.err = fmt.Errorf("%s world body and recolour mask geometry differ", q.name())
		return
	}
	p.index++
}

func TestModernHDVisualScaleGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SCALE_GPU") != "1" {
		t.Skip("dedicated visual-scale GPU process")
	}
	picture := image.NewRGBA(image.Rect(0, 0, 144, 144))
	for y := 1; y < 143; y++ {
		for x := 1; x < 143; x++ {
			picture.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	set := &modernStoryPortraitSet{
		mapSpritesHD: map[int][]image.Image{16: {picture}}, mapSpriteVisualScale: map[int]modernHDVisualScale{16: 115},
		mapSpriteFeet:    map[int][]int{16: {138}},
		battleAnimations: map[int]*modernBattleHDResource{48: {Frames: []image.Image{picture}, hashes: []string{"scale-fixture"}}},
	}
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{0, 0, 0, 255}
	}
	palette[1] = color.RGBA{0, 0, 255, 255}
	frame := figani.Frame{X: 50, Y: 50, Width: 24, Height: 24, Pixels: bytes.Repeat([]byte{1}, 576), Mask: bytes.Repeat([]byte{255}, 576)}
	sources := &modernHDBattleSources{Base: make([]byte, 64000), Layers: []modernHDBattleSource{{Frame: frame, Resource: 48, Fill: -1}}}
	g := &Game{modernStoryPortraits: set}
	p := &visualScaleGPUProbe{image: picture, unit: modernHDMapUnitDraw{Group: 16, Rect: image.Rect(300, 300, 444, 444)},
		mapGPU: modernHDMapGPU{World: ebiten.NewImage(1920, 1200), Theme: set, Units: make(map[int]map[int]*ebiten.Image)}}
	for i := range p.plans {
		set.battleAnimations[48].visualScale = 100
		if i == 1 || i == 2 {
			set.battleAnimations[48].visualScale = 115
		}
		if i == 2 {
			clip := image.Rect(50, 50, 74, 74)
			sources.Layers[0].Clip = &clip
		} else {
			sources.Layers[0].Clip = nil
		}
		if i == 3 {
			g.modernStoryPortraits = nil
		}
		var err error
		p.plans[i], err = g.modernHDBattlePlan(sources, palette, 0)
		if err != nil {
			t.Fatal(err)
		}
	}
	defer p.battle.close()
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 24 {
		t.Fatal(p.index, p.err)
	}
}
