package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func battleReuseFixture() modernHDBattlePlan {
	picture := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			picture.SetNRGBA(x, y, color.NRGBA{R: byte(20*x + 10), G: 170, B: 70, A: byte(100 + 10*y)})
		}
	}
	return modernHDBattlePlan{Base: make([]byte, 64000), Palette: hdBattleCandidatePalette(),
		Layers: []modernHDBattleLayer{
			{Native: figani.Frame{Width: 2, Height: 2, Pixels: []byte{3, 4, 5, 6}, Mask: []byte{1, 1, 1, 0}}, Rect: image.Rect(60, 60, 72, 72), Fill: -1},
			{HD: picture, HDKey: "reuse-hd", Rect: image.Rect(65, 65, 81, 81), Fill: -1},
		}}
}

var battleReuseMutations = []struct {
	name   string
	change func(*modernHDBattlePlan)
}{
	{"base", func(p *modernHDBattlePlan) { p.Base[0] = 12 }},
	{"palette", func(p *modernHDBattlePlan) { p.Palette[3] = color.RGBA{R: 230, A: 255} }},
	{"native-pixels", func(p *modernHDBattlePlan) { p.Layers[0].Native.Pixels[0] = 50 }},
	{"native-mask", func(p *modernHDBattlePlan) { p.Layers[0].Native.Mask[0] = 0 }},
	{"native-shape", func(p *modernHDBattlePlan) { p.Layers[0].Native.Width, p.Layers[0].Native.Height = 1, 4 }},
	{"hd-key", func(p *modernHDBattlePlan) { p.Layers[1].HDKey = "reuse-hd-second" }},
	{"hd-shape", func(p *modernHDBattlePlan) {
		p.Layers[1].HD, p.Layers[1].HDKey = image.NewNRGBA(image.Rect(0, 0, 7, 8)), "reuse-hd-shape"
	}},
	{"rect-min-x", func(p *modernHDBattlePlan) { p.Layers[1].Rect.Min.X++ }},
	{"rect-min-y", func(p *modernHDBattlePlan) { p.Layers[1].Rect.Min.Y++ }},
	{"rect-max-x", func(p *modernHDBattlePlan) { p.Layers[1].Rect.Max.X++ }},
	{"rect-max-y", func(p *modernHDBattlePlan) { p.Layers[1].Rect.Max.Y++ }},
	{"clip", func(p *modernHDBattlePlan) { c := image.Rect(70, 69, 80, 79); p.Layers[1].Clip = &c }},
	{"empty-clip", func(p *modernHDBattlePlan) { c := image.Rect(10, 10, 20, 20); p.Layers[1].Clip = &c }},
	{"fill", func(p *modernHDBattlePlan) { p.Layers[1].Fill = 33 }},
	{"subtract", func(p *modernHDBattlePlan) { p.HDSubtract = .2 }},
	{"mix", func(p *modernHDBattlePlan) { p.HDMix = .4 }},
	{"tint-r", func(p *modernHDBattlePlan) { p.HDTarget[0] = .7 }},
	{"tint-g", func(p *modernHDBattlePlan) { p.HDTarget[1] = .5 }},
	{"tint-b", func(p *modernHDBattlePlan) { p.HDTarget[2] = .3 }},
	{"preserve-palette", func(p *modernHDBattlePlan) { p.Layers[1].PreservePalette = true }},
	{"layer-order", func(p *modernHDBattlePlan) { p.Layers[0], p.Layers[1] = p.Layers[1], p.Layers[0] }},
	{"layer-count", func(p *modernHDBattlePlan) { p.Layers = p.Layers[:1] }},
	{"packed", func(p *modernHDBattlePlan) {
		l := &p.Layers[1]
		l.Native = figani.Frame{Width: 2, Height: 2, Pixels: []byte{0xb2, 0xb3, 0xb2, 0xb3}, Mask: []byte{1, 1, 1, 1}}
		l.PackedShader, l.PackedColours = 0x2b0, []byte{0, 1}
	}},
}

func TestModernHDBattleFrameKeyMutableInputs(t *testing.T) {
	plan := battleReuseFixture()
	palette, err := modernHDBattlePalette(plan.Palette)
	if err != nil {
		t.Fatal(err)
	}
	want := modernHDBattleFrameKey(plan, palette)
	copyPlan := battleReuseFixture()
	if modernHDBattleFrameKey(copyPlan, palette) != want {
		t.Fatal("equivalent independent slices or images have different identity")
	}
	for _, mutation := range battleReuseMutations {
		t.Run(mutation.name, func(t *testing.T) {
			p := battleReuseFixture()
			mutation.change(&p)
			pal, err := modernHDBattlePalette(p.Palette)
			if err != nil {
				t.Fatal(err)
			}
			if modernHDBattleFrameKey(p, pal) == want {
				t.Fatal("consumed input changed without invalidation")
			}
		})
	}
	packed := battleReuseFixture()
	battleReuseMutations[len(battleReuseMutations)-1].change(&packed)
	key := modernHDBattleFrameKey(packed, palette)
	packed.Layers[1].PackedShader = 0x3b0
	if modernHDBattleFrameKey(packed, palette) == key {
		t.Fatal("packed shader aliases prior key")
	}
	packed.Layers[1].PackedShader = 0x2b0
	packed.Layers[1].PackedColours[1] = 9 // Same mapped low bits, different interpolation basis.
	if modernHDBattleFrameKey(packed, palette) == key {
		t.Fatal("packed interpolation colours alias prior key")
	}
}

type battleReuseGPUProbe struct {
	gpu          modernHDBattleGPU
	index        int
	pixels, want []byte
	err          error
	cases        int
}

func (p *battleReuseGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *battleReuseGPUProbe) Update() error {
	if p.err != nil || p.index >= len(battleReuseMutations)+3 {
		return ebiten.Termination
	}
	return nil
}

func (p *battleReuseGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= len(battleReuseMutations)+3 {
		return
	}
	plan := battleReuseFixture()
	if p.index < len(battleReuseMutations) {
		battleReuseMutations[p.index].change(&plan)
	} else if p.index == len(battleReuseMutations)+1 {
		p.gpu.resetSources()
	} else if p.index == len(battleReuseMutations)+2 {
		p.gpu.close()
	}
	rendered, reused := p.gpu.rendered, p.gpu.reused
	if p.err = p.gpu.draw(plan); p.err != nil {
		return
	}
	if p.gpu.rendered != rendered+1 || p.gpu.reused != reused {
		p.err = fmt.Errorf("case%d did not invalidate changed/reset frame", p.index)
		return
	}
	p.gpu.World.ReadPixels(p.want)
	if p.err = p.gpu.draw(plan); p.err != nil {
		return
	}
	p.gpu.World.ReadPixels(p.pixels)
	if p.gpu.rendered != rendered+1 || p.gpu.reused != reused+1 || !bytes.Equal(p.pixels, p.want) {
		p.err = fmt.Errorf("case%d reuse changed pixels or repeated composition", p.index)
		return
	}
	// The uncached control uses the exact same destination/source bindings.
	// Separate GPU targets can rasterize fractional edges differently, which
	// is unrelated to whether an unchanged published frame can be retained.
	p.gpu.frameValid = false
	if p.err = p.gpu.draw(plan); p.err != nil {
		return
	}
	p.gpu.World.ReadPixels(p.pixels)
	if p.gpu.rendered != rendered+2 || !bytes.Equal(p.pixels, p.want) {
		p.err = fmt.Errorf("case%d forced full redraw differs from reuse", p.index)
		return
	}
	// Even an otherwise cached frame must still reject incomplete input.
	bad := plan
	bad.Base = nil
	if p.gpu.draw(bad) == nil {
		p.err = fmt.Errorf("case%d cached frame bypassed validation", p.index)
		return
	}
	p.gpu.World.ReadPixels(p.pixels)
	if !bytes.Equal(p.pixels, p.want) || p.gpu.reused != reused+1 {
		p.err = fmt.Errorf("case%d rejected frame changed published canvas", p.index)
		return
	}
	if p.index == 0 {
		for _, change := range []func(){
			func() { plan.Base[2] = 80 },
			func() { plan.Palette[3] = color.RGBA{G: 190, A: 255} },
			func() { plan.Layers[0].Native.Pixels[0] = 55 },
			func() { plan.Layers[0].Native.Mask[1] = 0 },
		} {
			change() // Mutate the exact caller slices used for the previous draw.
			before := p.gpu.rendered
			if p.err = p.gpu.draw(plan); p.err != nil {
				return
			}
			p.gpu.World.ReadPixels(p.want)
			p.gpu.frameValid = false
			if p.err = p.gpu.draw(plan); p.err != nil {
				return
			}
			p.gpu.World.ReadPixels(p.pixels)
			if p.gpu.rendered != before+2 || !bytes.Equal(p.pixels, p.want) {
				p.err = fmt.Errorf("in-place slice mutation reused stale frame")
				return
			}
		}
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.0/3, 1.0/3)
	screen.DrawImage(p.gpu.World, op)
	p.index++
	p.cases++
}

func TestModernHDBattleFrameReuseGPU(t *testing.T) {
	if os.Getenv("FD2_HD_BATTLE_REUSE_GPU") != "1" {
		t.Skip("dedicated complete-canvas reuse GPU process required")
	}
	probe := &battleReuseGPUProbe{pixels: make([]byte, 1920*1200*4), want: make([]byte, 1920*1200*4)}
	defer probe.gpu.close()
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if probe.err != nil || probe.cases != len(battleReuseMutations)+3 {
		t.Fatalf("frame reuse GPU incomplete: %v (%d cases)", probe.err, probe.cases)
	}
	t.Logf("%d cases: full composition/reuse RGBA exact; changed inputs, rejected plans, reset and close checked", probe.cases)
}
