package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type modernHDBattleGPUCase struct {
	name      string
	plan      modernHDBattlePlan
	indexed   []byte
	want      *image.RGBA
	tolerance int
}

type modernHDBattleGPUProbe struct {
	cases  []modernHDBattleGPUCase
	index  int
	gpu    modernHDBattleGPU
	pixels []byte
	repeat []byte
	err    error
}

func (p *modernHDBattleGPUProbe) Update() error {
	if p.err != nil || p.index >= len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *modernHDBattleGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *modernHDBattleGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= len(p.cases) {
		return
	}
	c := p.cases[p.index]
	if err := p.gpu.draw(c.plan); err != nil {
		p.err = fmt.Errorf("%s: %w", c.name, err)
		return
	}
	if p.pixels == nil {
		p.pixels = make([]byte, 1920*1200*4)
		p.repeat = make([]byte, len(p.pixels))
	}
	p.gpu.World.ReadPixels(p.pixels)
	if c.want != nil {
		for i, v := range c.want.Pix {
			diff := int(p.pixels[i]) - int(v)
			if diff < -c.tolerance || diff > c.tolerance {
				p.err = fmt.Errorf("%s: RGBA byte%d got%d want%d tolerance%d", c.name, i, p.pixels[i], v, c.tolerance)
				return
			}
		}
	} else {
		palette, _ := modernHDBattlePalette(c.plan.Palette)
		for y := 0; y < 1200; y++ {
			for x := 0; x < 1920; x++ {
				at := (y*1920 + x) * 4
				colorAt := int(c.indexed[(y/6)*320+x/6]) * 4
				if !bytes.Equal(p.pixels[at:at+4], palette[colorAt:colorAt+4]) {
					p.err = fmt.Errorf("%s: native GPU differs at %d,%d: %v vs %v", c.name, x, y, p.pixels[at:at+4], palette[colorAt:colorAt+4])
					return
				}
			}
		}
	}
	// 錯誤計畫不得清除已發布世界。
	if p.index == 0 {
		before := append([]byte(nil), p.pixels...)
		bad := c.plan
		bad.Base = nil
		if err := p.gpu.draw(bad); err == nil {
			p.err = fmt.Errorf("invalid plan accepted")
			return
		}
		p.gpu.World.ReadPixels(p.pixels)
		if !bytes.Equal(before, p.pixels) {
			p.err = fmt.Errorf("invalid plan changed previous world")
			return
		}
		bad = c.plan
		bad.Layers = append([]modernHDBattleLayer(nil), bad.Layers...)
		invalidClip := image.Rectangle{Min: image.Pt(10, 0), Max: image.Pt(5, 20)}
		bad.Layers[0].Clip = &invalidClip
		if err := p.gpu.draw(bad); err == nil {
			p.err = fmt.Errorf("invalid clip accepted")
			return
		}
		p.gpu.World.ReadPixels(p.pixels)
		if !bytes.Equal(before, p.pixels) {
			p.err = fmt.Errorf("invalid clip changed previous world")
			return
		}
		for _, shader := range []int{-1, 0x1b0, 0x9b0, 0x2b1, 0x2b0} {
			bad = c.plan
			bad.Layers = append([]modernHDBattleLayer(nil), bad.Layers...)
			bad.Layers[len(bad.Layers)-1].PackedShader = shader
			if err := p.gpu.draw(bad); err == nil {
				p.err = fmt.Errorf("invalid late packed plan accepted")
				return
			}
			p.gpu.World.ReadPixels(p.pixels)
			if !bytes.Equal(before, p.pixels) {
				p.err = fmt.Errorf("invalid late packed plan changed previous world")
				return
			}
		}
	}
	rendered, reused := p.gpu.rendered, p.gpu.reused
	if err := p.gpu.draw(c.plan); err != nil {
		p.err = fmt.Errorf("%s repeated plan: %w", c.name, err)
		return
	}
	p.gpu.World.ReadPixels(p.repeat)
	if p.gpu.rendered != rendered || p.gpu.reused != reused+1 || !bytes.Equal(p.pixels, p.repeat) {
		p.err = fmt.Errorf("%s repeated plan changed published RGBA or recomposited", c.name)
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.0/3, 1.0/3)
	screen.DrawImage(p.gpu.World, op)
	p.index++
}

func TestModernHDBattleGPUBridge(t *testing.T) {
	if os.Getenv("FD2_HD_BATTLE_GPU") != "1" {
		t.Skip("dedicated GPU lifecycle required")
	}
	requirePhysicalScenePack(t)
	var cases []modernHDBattleGPUCase
	palette := hdBattleCandidatePalette()
	base := make([]byte, 64000)
	for i := range base {
		base[i] = byte(i % 251)
	}
	attack := figani.Frame{Width: 4, Height: 3, Pixels: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, Mask: []byte{1, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1}}
	idle := figani.Frame{Width: 3, Height: 2, Pixels: []byte{0, 81, 82, 83, 84, 85}, Mask: []byte{1, 1, 1, 1, 0, 1}}
	g := &Game{}
	for _, pos := range []image.Point{image.Pt(-2, -1), image.Pt(318, 198), image.Pt(35, 75)} {
		attack.X, attack.Y, idle.X, idle.Y = pos.X, pos.Y, pos.X+1, pos.Y+1
		for _, side := range []byte{0, 1, 255} {
			for _, flag := range []byte{0, 1} {
				for _, fill := range []int{-1, 33} {
					for _, shift := range []image.Point{{}, {X: -3, Y: 2}} {
						attack.RawByte7 = flag
						present := battlepresent.NativePhysicalBodyPresent{DX: shift.X, DY: shift.Y, OpaqueFill: fill}
						want, err := battlepresent.ComposeNativePhysicalBodyPresent(base, attack, idle, side, present)
						if err != nil {
							t.Fatal(err)
						}
						plan, err := g.modernHDBattlePlan(modernHDBattleBodySources(base, attack, idle, side, present, 1, 0, 0, 0), palette, 0)
						if err != nil {
							t.Fatal(err)
						}
						cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("body/%v/%d/%d/%d/%v", pos, side, flag, fill, shift), plan: plan, indexed: want})
					}
				}
			}
		}
	}
	// 真正原始圖源與當幀DAC的兩側九階入場。
	for _, side := range []byte{0, 2} {
		g, actor, target := physicalSceneTestGame(t)
		actor.NativeRecordByte6 = side
		scene, err := g.prepareNativePhysicalScene(actor, target)
		if err != nil {
			t.Fatal(err)
		}
		for _, frame := range scene.prelude {
			sources, err := g.modernHDBattlePreludeSources(scene, frame.Stage)
			if err != nil {
				t.Fatal(err)
			}
			palette, err := fdother.ParseVGAPalette(frame.DAC)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := g.modernHDBattlePlan(sources, palette, float32(frame.Stage*6)/63)
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("prelude/%d/%d", side, frame.Stage), plan: plan, indexed: frame.Pixels})
		}
		// 明示Roll fixture，不宣稱傷害或RNG對拍。
		result := battle.NativePhysicalAttackResult{AttackStrikes: []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 7, Crit: true, Status: true}, DefenderHP: 73}}}
		if err := g.attachNativePhysicalBody(scene, result); err != nil {
			t.Fatal(err)
		}
		for index, job := range scene.body.jobs {
			if job.hdSources == nil {
				continue
			}
			palette := append(color.Palette(nil), g.nativeUIPalette...)
			if job.palette0 != nil {
				palette[0] = *job.palette0
			}
			plan, err := g.modernHDBattlePlan(job.hdSources, palette, 0)
			if err != nil {
				t.Fatal(err)
			}
			want, err := job.indexed()
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("original-body/%d/%d", side, index), plan: plan, indexed: want})
		}
	}
	// HD細節、覆蓋alpha、fill33及RGB淡入。目的矩形是通用GPU輸入，不替蓋亞定版。
	hd := image.NewNRGBA(image.Rect(0, 0, 12, 12))
	for y := 1; y < 11; y++ {
		for x := 1; x < 11; x++ {
			a := byte(255)
			if x%2 == 0 {
				a = 128
			}
			hd.SetNRGBA(x, y, color.NRGBA{R: 230, G: 70, B: 10, A: a})
		}
	}
	for _, pos := range []image.Point{{X: -5, Y: -3}, {X: 40, Y: 60}, {X: 1916, Y: 1196}} {
		for _, fill := range []int{-1, 33} {
			for _, shade := range []float32{0, 0.2, 0.7} {
				base := make([]byte, 64000)
				plan := modernHDBattlePlan{Base: base, Palette: palette, HDSubtract: shade, Layers: []modernHDBattleLayer{{HD: hd, HDKey: "hd-details", Rect: hd.Bounds().Add(pos), Fill: fill}}}
				want := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
				draw.Draw(want, want.Bounds(), &image.Uniform{C: palette[0]}, image.Point{}, draw.Src)
				art := image.NewNRGBA(hd.Bounds())
				for y := 0; y < 12; y++ {
					for x := 0; x < 12; x++ {
						value := hd.NRGBAAt(x, y)
						if fill == 33 {
							f := color.NRGBAModel.Convert(palette[33]).(color.NRGBA)
							value.R, value.G, value.B = f.R, f.G, f.B
						}
						sub := func(v byte) byte { return byte(math.Round(math.Max(0, float64(v)-float64(shade)*255))) }
						value.R, value.G, value.B = sub(value.R), sub(value.G), sub(value.B)
						art.SetNRGBA(x, y, value)
					}
				}
				draw.Draw(want, hd.Bounds().Add(pos), art, image.Point{}, draw.Over)
				cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("hd/%v/%d/%v", pos, fill, shade), plan: plan, want: want, tolerance: 1})
			}
		}
	}
	cases = appendModernCompoundGPUCases(t, cases, palette)
	cases = appendModernCompoundActorGPUCases(t, cases)
	cases = appendModernCommand0GPUCases(t, cases)
	cases = appendModernCommand1GPUCases(t, cases)
	cases = appendModernOrdinaryRemainingGPUCases(t, cases)
	cases = appendModernPackedColourGPUCases(t, cases)
	cases = appendModernCommand5GPUCases(t, cases)
	cases = appendModernScrollGPUCases(t, cases)
	probe := &modernHDBattleGPUProbe{cases: cases}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	defer probe.gpu.close()
	if probe.err != nil || probe.index != len(cases) {
		t.Fatalf("GPU bridge incomplete: %v (%d/%d)", probe.err, probe.index, len(cases))
	}
	packedTextures := 0
	for key := range probe.gpu.sources {
		if strings.HasPrefix(key, "hd/packed-matrix-target/packed/") {
			packedTextures++
		}
	}
	if packedTextures != 8 {
		t.Fatalf("seven shades and changed palette need eight separate textures, got%d", packedTextures)
	}
	t.Logf("%d GPU cases: native complete-canvas exact; HD alpha/RGB shade tolerance1; every repeated plan byte-exact without recomposition; no original-executable or player-path claim", len(cases))
}

func TestModernHDBattleEmptyFrameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_BATTLE_EMPTY_GPU") != "1" {
		t.Skip("dedicated empty-frame GPU process required")
	}
	palette := hdBattleCandidatePalette()
	back := image.NewNRGBA(image.Rect(0, 0, 12, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 12; x++ {
			back.SetNRGBA(x, y, color.NRGBA{R: 20, G: 80, B: 220, A: 255})
		}
	}
	want := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	draw.Draw(want, want.Bounds(), image.NewUniform(palette[0]), image.Point{}, draw.Src)
	draw.Draw(want, back.Bounds().Add(image.Pt(40, 60)), back, image.Point{}, draw.Over)
	plan := modernHDBattlePlan{Base: make([]byte, 64000), Palette: palette, Layers: []modernHDBattleLayer{
		{HD: back, HDKey: "empty-frame-behind", Rect: back.Bounds().Add(image.Pt(40, 60)), Fill: -1},
		{HD: image.NewNRGBA(image.Rect(0, 0, 1920, 1200)), HDKey: "verified-native-empty", Rect: image.Rect(0, 0, 1920, 1200), Fill: -1},
	}}
	probe := &modernHDBattleGPUProbe{cases: []modernHDBattleGPUCase{{name: "empty-full-frame-keeps-other-layer", plan: plan, want: want}}}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	defer probe.gpu.close()
	if probe.err != nil || probe.index != 1 {
		t.Fatalf("empty-frame GPU: %v", probe.err)
	}
}
