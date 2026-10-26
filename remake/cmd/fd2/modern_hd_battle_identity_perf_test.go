package main

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Baseline shader from 7e4cdd70:modern_hd_battle_gpu.go. This probe compares
// GPU primitives in one process; it is not a normal-player FPS benchmark.
const legacyBattlePerfShader = `//kage:unit pixels
package main
var FillColor vec4
var Subtract float
var Tint vec3
var Mix float
func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
    src:=imageSrc0At(texCoord)
    if src.a==0 { return vec4(0) }
    rgb:=src.rgb/src.a
    if FillColor.a>0 { rgb=FillColor.rgb }
    rgb=max(rgb-vec3(Subtract),vec3(0))
    rgb=mix(rgb,Tint,Mix)
    return vec4(rgb*src.a,src.a)*color
}`

type battleIdentityPerfProbe struct {
	gpu       modernHDBattleGPU
	plan      modernHDBattlePlan
	legacy    *ebiten.Shader
	texture   *ebiten.Image
	pixels    [4][]byte
	index     int
	maxDiff   int
	durations map[string][]float64
	err       error
}

func (p *battleIdentityPerfProbe) Update() error {
	if p.err != nil || p.index >= 60 {
		return ebiten.Termination
	}
	return nil
}
func (p *battleIdentityPerfProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *battleIdentityPerfProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= 60 {
		return
	}
	variant := p.index % 4
	legacy, colour := variant%2 == 0, variant/2
	plan := p.plan
	name := "identity"
	if colour == 1 {
		plan.HDSubtract, plan.HDMix, plan.HDTarget = 0.2, 0.25, [3]float32{0.2, 0.4, 0.6}
		name = "colour"
	}
	started := time.Now()
	if legacy {
		// The same base preparation, then the original shader on one quad.
		// Baseline omits layer validation/cache lookup, favouring the baseline.
		base := plan
		base.Layers = nil
		p.err = p.gpu.draw(base)
		if p.err == nil {
			op := &ebiten.DrawRectShaderOptions{}
			op.Images[0] = p.texture
			op.Uniforms = map[string]interface{}{"FillColor": []float32{0, 0, 0, 0},
				"Subtract": plan.HDSubtract, "Mix": plan.HDMix, "Tint": plan.HDTarget[:]}
			p.gpu.World.DrawRectShader(modernHDWidth, modernHDHeight, p.legacy, op)
			p.gpu.frameValid = false // This test-only manual write bypasses draw's frame identity.
		}
		name = "legacy-" + name
	} else {
		p.err = p.gpu.draw(plan)
		name = "candidate-" + name
	}
	if p.err != nil {
		return
	}
	p.gpu.World.ReadPixels(p.pixels[variant]) // Include GPU completion in both paths.
	seconds := time.Since(started).Seconds()
	if p.index >= 12 { // Three warm cycles, then 12 interleaved samples each.
		p.durations[name] = append(p.durations[name], seconds)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.0/3, 1.0/3)
	screen.DrawImage(p.gpu.World, op)
	p.index++
}

func TestModernHDBattleIdentityPerformanceGPU(t *testing.T) {
	if os.Getenv("FD2_HD_GPU_IDENTITY_PERF") != "1" {
		t.Skip("dedicated primitive GPU comparison required")
	}
	out := os.Getenv("FD2_HD_GPU_IDENTITY_REPORT")
	if out == "" {
		t.Fatal("new report path required")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("report already exists or cannot be checked")
	}
	picture := image.NewNRGBA(image.Rect(0, 0, modernHDWidth, modernHDHeight))
	for y := 0; y < modernHDHeight; y++ {
		for x := 0; x < modernHDWidth; x++ {
			at := y*picture.Stride + x*4
			alpha := byte(255)
			if (x+y)%3 == 0 {
				alpha = 128
			}
			if y%17 == 0 {
				alpha = 0
			}
			copy(picture.Pix[at:at+4], []byte{byte(x), byte(y), byte(x + y), alpha})
		}
	}
	shader, err := ebiten.NewShader([]byte(legacyBattlePerfShader))
	if err != nil {
		t.Fatal(err)
	}
	probe := &battleIdentityPerfProbe{legacy: shader, texture: ebiten.NewImageFromImage(picture),
		durations: make(map[string][]float64),
		plan: modernHDBattlePlan{Base: make([]byte, 64000), Palette: hdBattleCandidatePalette(),
			Layers: []modernHDBattleLayer{{HD: picture, HDKey: "identity-perf", Rect: picture.Bounds(), Fill: -1}}}}
	defer probe.gpu.close()
	defer probe.texture.Dispose()
	defer shader.Dispose()
	for i := range probe.pixels {
		probe.pixels[i] = make([]byte, modernHDWidth*modernHDHeight*4)
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if probe.err != nil || probe.index != 60 {
		t.Fatalf("primitive comparison incomplete: %v (%d/60)", probe.err, probe.index)
	}
	// Validate only after the timed sequence. Otherwise untimed comparison
	// work on one side consumes the CPU quota of the following measurement.
	for colour := 0; colour < 2; colour++ {
		for i, value := range probe.pixels[colour*2+1] {
			diff := int(value) - int(probe.pixels[colour*2][i])
			if diff < 0 {
				diff = -diff
			}
			if diff > probe.maxDiff {
				probe.maxDiff = diff
			}
			if diff > 1 {
				t.Fatal(fmt.Errorf("primitive RGBA byte%d differs by%d", i, diff))
			}
		}
	}
	medians := map[string]float64{}
	for name, values := range probe.durations {
		if len(values) != 12 {
			t.Fatal("missing interleaved samples", name, len(values))
		}
		sorted := append([]float64(nil), values...)
		sort.Float64s(sorted)
		medians[name] = (sorted[5] + sorted[6]) / 2
	}
	report := map[string]any{"status": "PASS-INTERLEAVED-GPU-PRIMITIVES", "samples_seconds": probe.durations,
		"median_seconds": medians, "max_rgba_difference": probe.maxDiff,
		"current_shader_equals_legacy_reference": modernHDBattleShader == legacyBattlePerfShader,
		"scope":                                  "1920x1200合成RGBA；同一程序交錯比較原著色器及目前繪製，皆含GPU讀回。基準省略圖層驗證／快取查找，較有利於基準。不代表正常玩家FPS或平台驗收。"}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("GPU primitive medians (seconds): %v; max RGBA difference%d", medians, probe.maxDiff)
}
