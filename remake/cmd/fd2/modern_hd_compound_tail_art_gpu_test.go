package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type compoundTailArtGPUProbe struct {
	g          *Game
	indices    []int
	index      int
	err        error
	background *image.RGBA
	ui         *image.RGBA
	pixels     []byte
	rows       []map[string]any
}

func (p *compoundTailArtGPUProbe) Update() error {
	if p.err != nil || p.index == len(p.indices)*2 {
		return ebiten.Termination
	}
	return nil
}
func (p *compoundTailArtGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *compoundTailArtGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= len(p.indices)*2 {
		return
	}
	index := p.indices[p.index/2]
	picture := p.g.modernStoryPortraits.mapEffects.frames[index]
	if picture == nil {
		p.err = fmt.Errorf("missing source%d", index)
		return
	}
	origin := image.Pt(600, 480)
	if p.index%2 == 1 {
		origin = image.Pt(-3, -3)
	}
	rect := picture.Bounds().Add(origin)
	frame := &modernHDMapFrame{Background: p.background, UI: p.ui, Viewport: image.Rect(0, 0, 1920, 1200), OverlaysAboveUI: true,
		Overlays: []modernHDMapOverlayDraw{{Entry: index, Image: picture, Rect: rect}}}
	if !p.g.drawPreparedModernHDMapFrame(screen, frame, false) {
		p.err = fmt.Errorf("source GPU refused%d", index)
		return
	}
	p.g.modernHDMapGPU.World.ReadPixels(p.pixels)
	visible, partial := 0, 0
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1920; x++ {
			point := image.Pt(x, y)
			at := (y*1920 + x) * 4
			want := [4]byte{20, 30, 40, 255}
			if point.In(rect) {
				v := picture.RGBAAt(x-rect.Min.X, y-rect.Min.Y)
				if v.A != 0 {
					visible++
					if v.A != 255 {
						partial++
					}
				}
				for c, value := range []byte{v.R, v.G, v.B} {
					want[c] = byte(math.Min(255, math.Round(float64(value)+float64(want[c])*(1-float64(v.A)/255))))
				}
			}
			for c, value := range want {
				diff := int(p.pixels[at+c]) - int(value)
				if diff < -2 || diff > 2 {
					p.err = fmt.Errorf("source%d placement%d pixel(%d,%d) channel%d got%d want%d", index, p.index%2, x, y, c, p.pixels[at+c], value)
					return
				}
			}
		}
	}
	if visible == 0 || partial == 0 {
		p.err = fmt.Errorf("source%d has no visible partial-alpha samples", index)
		return
	}
	p.rows = append(p.rows, map[string]any{"index": index, "placement": p.index % 2, "visible_pixels": visible, "partial_alpha_pixels": partial, "readback_sha256": fmt.Sprintf("%x", sha256.Sum256(p.pixels))})
	p.index++
}

func TestModernCompoundTailArtSourceGPU(t *testing.T) {
	if os.Getenv("FD2_HD_COMPOUND_TAIL_ART_GPU") != "1" {
		t.Skip("dedicated77-source GPU process required")
	}
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	indices := append(modernMapOverlayHDIndices(6, 12), modernMapOverlayHDIndices(6, 65)...)
	if len(indices) != 77 {
		t.Fatal("wrong source domain")
	}
	p := &compoundTailArtGPUProbe{g: &Game{modernStoryPortraits: set}, indices: indices, background: image.NewRGBA(image.Rect(0, 0, 320, 200)), ui: image.NewRGBA(image.Rect(0, 0, 320, 200)), pixels: make([]byte, 1920*1200*4)}
	draw.Draw(p.background, p.background.Bounds(), &image.Uniform{C: color.RGBA{20, 30, 40, 255}}, image.Point{}, draw.Src)
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || len(p.rows) != 154 {
		t.Fatal("source GPU", p.err, len(p.rows))
	}
	report := map[string]any{"status": "PASS-COMPOUND-TAIL-ART-SOURCE-GPU", "sources": 77, "worlds": len(p.rows), "rgba_bytes_per_world": 1920 * 1200 * 4, "tolerance": 2, "cases": p.rows, "scope": "isolated prepared-map GPU source-over and clipping; actual Game.Draw verified separately"}
	if out := os.Getenv("FD2_HD_COMPOUND_TAIL_ART_REPORT"); out != "" {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(out, append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
