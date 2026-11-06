package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type modernHDFlameGPUProbe struct {
	gpu   modernHDBattleGPU
	index int
	err   error
}

func (p *modernHDFlameGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *modernHDFlameGPUProbe) Update() error {
	if p.err != nil || p.index >= 18 {
		return ebiten.Termination
	}
	return nil
}
func (p *modernHDFlameGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= 18 {
		return
	}
	quality := hdTextureQuality(p.index / 6)
	p.gpu.setQuality(quality)
	zoom := 1.0
	if p.index%6 >= 3 {
		zoom = 1.07
	}
	alpha := []float32{1, 0.5, 0}[p.index%3]
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{A: 255}
	}
	sprite := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	hud := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			sprite.SetNRGBA(x, y, color.NRGBA{R: 200, G: 100, B: 20, A: 255})
			hud.SetNRGBA(x, y, color.NRGBA{R: 30, G: 60, B: 200, A: 255})
		}
	}
	plan := modernHDBattlePlan{Base: make([]byte, 64000), Palette: palette, CameraZoom: zoom,
		Layers:    []modernHDBattleLayer{{HD: hud, HDKey: "fixed-hud-probe", Rect: image.Rect(600, 600, 630, 650), Fill: -1, ScreenFixed: true}},
		Particles: []modernHDBattleLayer{{HD: sprite, HDKey: "opacity-probe", Rect: image.Rect(600, 600, 660, 660), Fill: -1, Opacity: &alpha}}}
	rendered := p.gpu.rendered
	if err := p.gpu.draw(plan); err != nil {
		p.err = err
		return
	}
	if p.gpu.rendered != rendered+1 || len(p.gpu.sources) != 2 {
		p.err = fmt.Errorf("alpha or camera change reused a stale frame or texture")
		return
	}
	pixels := make([]byte, p.gpu.World.Bounds().Dx()*p.gpu.World.Bounds().Dy()*4)
	p.gpu.World.ReadPixels(pixels)
	x := int((654*zoom + (1-zoom)*960) * quality.density())
	y := int((630*zoom + (1-zoom)*600) * quality.density())
	at := (y*p.gpu.World.Bounds().Dx() + x) * 4
	want := []byte{byte(200 * alpha), byte(100 * alpha), byte(20 * alpha), 255}
	for i, value := range want {
		if delta := int(pixels[at+i]) - int(value); delta < -1 || delta > 1 {
			p.err = fmt.Errorf("quality%d zoom%f alpha%f pixel%v want%v", quality, zoom, alpha, pixels[at:at+4], want)
			return
		}
	}
	hudX, hudY := int(615*quality.density()), int(615*quality.density())
	hudAt := (hudY*p.gpu.World.Bounds().Dx() + hudX) * 4
	if !bytes.Equal(pixels[hudAt:hudAt+4], []byte{30, 60, 200, 255}) {
		p.err = fmt.Errorf("particle or camera painted over fixed HUD")
		return
	}
	reused := p.gpu.reused
	if err := p.gpu.draw(plan); err != nil || p.gpu.reused != reused+1 {
		p.err = fmt.Errorf("identical particle plan did not reuse frame: %v", err)
		return
	}
	for _, invalid := range []float32{-1, 2, float32(math.NaN()), float32(math.Inf(1))} {
		bad := plan
		bad.Particles = append([]modernHDBattleLayer(nil), plan.Particles...)
		bad.Particles[0].Opacity = &invalid
		if err := p.gpu.draw(bad); err == nil {
			p.err = fmt.Errorf("invalid particle opacity accepted")
			return
		}
		after := make([]byte, len(pixels))
		p.gpu.World.ReadPixels(after)
		if !bytes.Equal(after, pixels) {
			p.err = fmt.Errorf("invalid particle plan erased published world")
			return
		}
	}
	p.index++
}

func TestModernHDFlameEmberGPU(t *testing.T) {
	if os.Getenv("FD2_HD_FLAME_GPU") != "1" {
		t.Skip("dedicated flame GPU process required")
	}
	p := &modernHDFlameGPUProbe{}
	defer p.gpu.close()
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 18 {
		t.Fatalf("particle GPU %d/18: %v", p.index, p.err)
	}
	data, _ := json.MarshalIndent(map[string]any{"status": "PASS-FLAME-PARTICLE-GPU", "cases": p.index,
		"qualities": 3, "camera_scales": []float64{1, 1.07}, "opacity_values": []float32{1, 0.5, 0},
		"invalid_opacity_rejected": 72, "identical_frame_reused": true, "one_cached_particle_texture": true,
		"fixed_hud_protected": true}, "", "  ")
	if err := os.WriteFile(os.Getenv("FD2_HD_FLAME_GPU_REPORT"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
