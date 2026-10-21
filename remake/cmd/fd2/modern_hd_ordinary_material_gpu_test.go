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
	archive   string
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
		p.native, p.err = figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), p.archive, resource)
		if p.err != nil {
			return
		}
		if p.archive == "FIGANI.DAT" {
			p.err = p.g.modernStoryPortraits.prepareBattleHDBanks([]int{resource}, nil)
		} else {
			p.err = p.g.modernStoryPortraits.prepareBattleHDBanks(nil, []int{resource})
		}
		if p.err != nil {
			return
		}
	}
	f := p.native.Frames[p.frame]
	plan, err := p.g.modernHDBattlePlan(&modernHDBattleSources{
		Base:   make([]byte, 64000),
		Layers: []modernHDBattleSource{{Archive: p.archive, Resource: resource, Index: p.frame, Frame: f, Fill: -1}},
	}, hdBattleCandidatePalette(), 0)
	if err != nil {
		p.err = err
		return
	}
	layer := plan.Layers[0]
	entry := p.g.modernStoryPortraits.battleEffects[resource]
	if p.archive == "FIGANI.DAT" {
		entry = p.g.modernStoryPortraits.battleAnimations[resource]
	}
	if layer.HD == nil || layer.HDKey != entry.hashes[p.frame] {
		p.err = fmt.Errorf("%s%d:%d did not bind the registered HD source", p.archive, resource, p.frame)
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
			p.err = fmt.Errorf("%s%d:%d RGBA byte%d: got%d want%d", p.archive, resource, p.frame, i, p.pixels[i], value)
			return
		}
	}
	p.rows = append(p.rows, map[string]any{"archive": p.archive, "resource": resource, "frame": p.frame, "sha256": layer.HDKey, "rgba_tolerance": 1,
		"delay_native": f.Delay, "raw_byte_4": f.RawByte4, "raw_byte_5": f.RawByte5, "raw_byte_7": f.RawByte7})
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
	testModernHDRegisteredFramesGPU(t, "FDOTHER.DAT", resources, frameCount, status)
}

func TestModernHDArmoredAllFramesGPU(t *testing.T) {
	if os.Getenv("FD2_HD_ARMORED_GPU") != "1" {
		t.Skip("dedicated registered armored source GPU process required")
	}
	testModernHDRegisteredFramesGPU(t, "FIGANI.DAT", []int{54, 55, 56, 57, 58, 59}, 60, "PASS-ARMORED-60-SOURCE-GPU")
}

func TestModernHDMountedArcherAllFramesGPU(t *testing.T) {
	if os.Getenv("FD2_HD_MOUNTED_ARCHER_GPU") != "1" {
		t.Skip("dedicated registered mounted fighter and archer GPU process required")
	}
	testModernHDRegisteredFramesGPU(t, "FIGANI.DAT", []int{21, 22, 69, 70}, 50, "PASS-MOUNTED-ARCHER-50-SOURCE-GPU")
}

func TestModernHDNinjaWingedAllFramesGPU(t *testing.T) {
	if os.Getenv("FD2_HD_NINJA_WINGED_GPU") != "1" {
		t.Skip("dedicated registered ninja and winged lancer GPU process required")
	}
	testModernHDRegisteredFramesGPU(t, "FIGANI.DAT", []int{75, 76, 77, 81, 82}, 44, "PASS-NINJA-WINGED-44-SOURCE-GPU")
}

func TestModernHDMonksWandAllFramesGPU(t *testing.T) {
	if os.Getenv("FD2_HD_MONKS_WAND_GPU") != "1" {
		t.Skip("dedicated registered monks and wand fighter GPU process required")
	}
	testModernHDRegisteredFramesGPU(t, "FIGANI.DAT", []int{84, 85, 87, 88, 93, 94}, 44, "PASS-MONKS-WAND-44-SOURCE-GPU")
}

func TestModernHDKnightsAxemanAllFramesGPU(t *testing.T) {
	if os.Getenv("FD2_HD_KNIGHTS_AXEMAN_GPU") != "1" {
		t.Skip("dedicated registered knights and axeman GPU process required")
	}
	testModernHDRegisteredFramesGPU(t, "FIGANI.DAT", []int{96, 97, 98, 99, 100, 102, 103, 104}, 77, "PASS-KNIGHTS-AXEMAN-77-SOURCE-GPU")
}

func testModernHDRegisteredFramesGPU(t *testing.T, archive string, resources []int, frameCount int, status string) {
	t.Helper()
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	p := &ordinaryMaterialGPUProbe{g: &Game{modernStoryPortraits: set},
		archive:   archive,
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
			"scope": "Registered archive binding and complete GPU world against independent CPU Over composition; not Game.Draw, scheduling, original RGB parity or player-path validation."}, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d registered %s frames: complete GPU world matches CPU Over within one RGBA unit", frameCount, archive)
}
