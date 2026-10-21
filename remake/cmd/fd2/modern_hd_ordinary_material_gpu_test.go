package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// Stream one registered bank at a time. Holding a full expected world for every
// effect frame would turn this source-to-GPU check into a memory stress test.
type ordinaryMaterialGPUProbe struct {
	g         *Game
	resources []int
	resource  int
	frame     int
	native    *figani.Animation
	gpu       modernHDBattleGPU
	pixels    []byte
	want      *image.RGBA
	rows      []map[string]any
	err       error
}

func (p *ordinaryMaterialGPUProbe) Update() error {
	if p.err != nil || p.resource == len(p.resources) {
		return ebiten.Termination
	}
	return nil
}

func (p *ordinaryMaterialGPUProbe) Layout(int, int) (int, int) { return 640, 400 }

func (p *ordinaryMaterialGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.resource == len(p.resources) {
		return
	}
	resource := p.resources[p.resource]
	if p.native == nil {
		p.native, p.err = figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", resource)
		if p.err != nil {
			return
		}
		p.err = p.g.modernStoryPortraits.prepareBattleHDBanks(nil, []int{resource})
		if p.err != nil {
			return
		}
	}
	f := p.native.Frames[p.frame]
	plan, err := p.g.modernHDBattlePlan(&modernHDBattleSources{
		Base:   make([]byte, 64000),
		Layers: []modernHDBattleSource{{Archive: "FDOTHER.DAT", Resource: resource, Index: p.frame, Frame: f, Fill: -1}},
	}, hdBattleCandidatePalette(), 0)
	if err != nil {
		p.err = err
		return
	}
	layer := plan.Layers[0]
	entry := p.g.modernStoryPortraits.battleEffects[resource]
	if layer.HD == nil || layer.HDKey != entry.hashes[p.frame] {
		p.err = fmt.Errorf("FDOTHER%d:%d did not bind the registered HD source", resource, p.frame)
		return
	}
	draw.Draw(p.want, p.want.Bounds(), image.NewUniform(plan.Palette[0]), image.Point{}, draw.Src)
	draw.Draw(p.want, layer.Rect, layer.HD, layer.HD.Bounds().Min, draw.Over)
	p.gpu.resetSources()
	if p.err = p.gpu.draw(plan); p.err != nil {
		return
	}
	p.gpu.World.ReadPixels(p.pixels)
	for i, value := range p.want.Pix {
		delta := int(p.pixels[i]) - int(value)
		if delta < -1 || delta > 1 {
			p.err = fmt.Errorf("FDOTHER%d:%d RGBA byte%d: got%d want%d", resource, p.frame, i, p.pixels[i], value)
			return
		}
	}
	p.rows = append(p.rows, map[string]any{"resource": resource, "frame": p.frame, "sha256": layer.HDKey, "rgba_tolerance": 1})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.0/3, 1.0/3)
	screen.DrawImage(p.gpu.World, op)
	p.frame++
	if p.frame == len(p.native.Frames) {
		p.resource++
		p.frame, p.native = 0, nil
	}
}

func TestModernHDOrdinaryMaterialAllFramesGPU(t *testing.T) {
	if os.Getenv("FD2_HD_ORDINARY_MATERIAL_GPU") != "1" {
		t.Skip("dedicated registered ordinary material GPU process required")
	}
	testModernHDOrdinaryFramesGPU(t, []int{18, 19, 20, 21, 22, 23, 24, 25, 28, 30, 32, 33, 37, 38, 39, 43}, 304, "PASS-ORDINARY-MATERIAL-304-SOURCE-GPU")
}

func TestModernHDDragonAllFramesGPU(t *testing.T) {
	if os.Getenv("FD2_HD_DRAGON_GPU") != "1" {
		t.Skip("dedicated registered fire dragon GPU process required")
	}
	testModernHDOrdinaryFramesGPU(t, []int{26, 27}, 36, "PASS-DRAGON-36-SOURCE-GPU")
}

func testModernHDOrdinaryFramesGPU(t *testing.T, resources []int, frameCount int, status string) {
	t.Helper()
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	p := &ordinaryMaterialGPUProbe{g: &Game{modernStoryPortraits: set},
		resources: resources,
		pixels:    make([]byte, 1920*1200*4), want: image.NewRGBA(image.Rect(0, 0, 1920, 1200))}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	defer p.gpu.close()
	if p.err != nil || p.resource != len(resources) || len(p.rows) != frameCount {
		t.Fatalf("ordinary material GPU: %v resources%d frames%d", p.err, p.resource, len(p.rows))
	}
	if out := os.Getenv("FD2_TEST_MODERN_EFFECT_OUT"); out != "" {
		if err := os.Mkdir(out, 0755); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.MarshalIndent(map[string]any{"status": status, "frames": p.rows,
			"scope": "Registered FDOTHER binding and complete GPU world against independent CPU Over composition; not Game.Draw, original RGB parity or player-path validation."}, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d registered FDOTHER frames: complete GPU world matches CPU Over within one RGBA unit", frameCount)
}
