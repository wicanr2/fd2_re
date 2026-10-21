package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func packedColourControlPalette() color.Palette {
	p := make(color.Palette, 256)
	for i := range p {
		p[i] = color.RGBA{A: 255}
	}
	p[1], p[2], p[3], p[4] = color.RGBA{A: 255}, color.RGBA{R: 100, A: 255}, color.RGBA{G: 100, A: 255}, color.RGBA{R: 100, G: 100, A: 255}
	for i := 0; i < 8; i++ {
		p[0xb0+i] = color.RGBA{R: byte(10 + i*10), G: byte(20 + i*10), B: byte(30 + i*10), A: 255}
	}
	p[0xb3], p[0xb4], p[0xb5], p[0xb6] = color.RGBA{60, 20, 80, 255}, color.RGBA{120, 40, 100, 255}, color.RGBA{80, 140, 120, 255}, color.RGBA{160, 160, 140, 255}
	return p
}

func TestModernHDPackedTargetSourcePreservesOriginalAndGuide(t *testing.T) {
	frame := figani.Frame{Width: 4, Height: 1, Pixels: []byte{1, 5, 9, 1}, Mask: []byte{1, 0, 1, 1}}
	before := append([]byte(nil), frame.Pixels...)
	source, err := modernHDPackedTargetSource(modernHDBattleSource{Frame: frame}, 0x2b0)
	if err != nil || !reflect.DeepEqual(source.Frame.Pixels, []byte{0xb3, 0xb7, 0xb3, 0xb3}) ||
		!reflect.DeepEqual(source.PackedColours, []byte{1, 9}) || !reflect.DeepEqual(frame.Pixels, before) ||
		!reflect.DeepEqual(source.Frame.Mask, frame.Mask) {
		t.Fatalf("packed native source changed: %+v %v", source, err)
	}
	if err := validateModernHDPackedGuide(source.Frame, source.PackedShader, source.PackedColours); err != nil {
		t.Fatal(err)
	}
	for _, shader := range []int{-1, 0xb0, 0x1b0, 0x9b0, 0x2b1} {
		if _, err := modernHDPackedTargetSource(modernHDBattleSource{Frame: frame}, shader); err == nil {
			t.Fatalf("invalid shader%x accepted", shader)
		}
	}
	if err := validateModernHDPackedGuide(source.Frame, source.PackedShader, nil); err == nil {
		t.Fatal("missing opaque colour guide accepted")
	}
	if err := validateModernHDPackedGuide(source.Frame, source.PackedShader, []byte{9, 1}); err == nil {
		t.Fatal("unordered colour guide accepted")
	}
}

func TestModernHDPackedColourControlsAndAlpha(t *testing.T) {
	palette, err := modernHDBattlePalette(packedColourControlPalette())
	if err != nil {
		t.Fatal(err)
	}
	picture := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	picture.SetNRGBA(0, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 254})
	picture.SetNRGBA(1, 0, color.NRGBA{R: 50, G: 50, A: 37})
	picture.SetNRGBA(2, 0, color.NRGBA{R: 250, G: 250, B: 255, A: 64})
	picture.SetNRGBA(3, 0, color.NRGBA{R: 200, G: 100, B: 60, A: 0})
	before := append([]byte(nil), picture.Pix...)
	for shade := 2; shade <= 8; shade++ {
		got, err := modernHDPackedPicture(picture, palette, []byte{1, 2, 3, 4}, shade<<8|0xb0)
		if err != nil {
			t.Fatal(err)
		}
		first := color.NRGBAModel.Convert(got.At(0, 0)).(color.NRGBA)
		at := (((1 + shade) & 7) + 0xb0) * 4
		if first != (color.NRGBA{R: palette[at], G: palette[at+1], B: palette[at+2], A: 254}) {
			t.Fatalf("exact control shade%d: %v", shade, first)
		}
		for x := 0; x < 4; x++ {
			if color.NRGBAModel.Convert(got.At(x, 0)).(color.NRGBA).A != picture.NRGBAAt(x, 0).A {
				t.Fatal("alpha changed")
			}
		}
		if shade == 2 {
			// Four equally distant native corners have equal weights. Their mapped mean is independently known.
			if color.NRGBAModel.Convert(got.At(1, 0)).(color.NRGBA) != (color.NRGBA{105, 90, 110, 37}) {
				t.Fatal("four-corner mean differs")
			}
			if color.NRGBAModel.Convert(got.At(2, 0)).(color.NRGBA) != (color.NRGBA{255, 255, 255, 64}) {
				t.Fatal("RGB clamp differs")
			}
		}
	}
	if !reflect.DeepEqual(before, picture.Pix) {
		t.Fatal("source painting changed")
	}
	if identity, err := modernHDPackedPicture(picture, palette, nil, 0); err != nil || identity != picture {
		t.Fatal("identity did not preserve original")
	}
	key := modernHDPackedTextureKey("painting", palette, []byte{1, 2, 3, 4}, 0x2b0)
	changed := append([]byte(nil), palette...)
	changed[0xb3*4]++
	for _, other := range []string{modernHDPackedTextureKey("painting", changed, []byte{1, 2, 3, 4}, 0x2b0), modernHDPackedTextureKey("painting", palette, []byte{1, 2, 3}, 0x2b0), modernHDPackedTextureKey("painting", palette, []byte{1, 2, 3, 4}, 0x3b0), modernHDPackedTextureKey("other", palette, []byte{1, 2, 3, 4}, 0x2b0)} {
		if key == other {
			t.Fatal("texture cache identities collided")
		}
	}
}

func appendModernPackedColourGPUCases(t *testing.T, cases []modernHDBattleGPUCase) []modernHDBattleGPUCase {
	t.Helper()
	palette := packedColourControlPalette()
	actor := image.NewNRGBA(image.Rect(0, 0, 12, 12))
	target := image.NewNRGBA(actor.Bounds())
	for y := 0; y < 12; y++ {
		for x := 0; x < 12; x++ {
			actor.SetNRGBA(x, y, color.NRGBA{40, 80, 120, 255})
			alpha := []byte{0, 64, 128, 255}[(y/6)*2+x/6]
			target.SetNRGBA(x, y, color.NRGBA{50, 50, 0, alpha})
		}
	}
	for variant := 0; variant < 8; variant++ {
		shade := variant + 2
		if variant == 7 {
			shade = 2
		}
		p := append(color.Palette(nil), palette...)
		if variant == 7 {
			c := p[0xb3].(color.RGBA)
			c.R += 20
			p[0xb3] = c
		}
		frame := figani.Frame{X: 20, Y: 15, Width: 2, Height: 2, Pixels: []byte{1, 2, 3, 4}, Mask: []byte{1, 1, 1, 1}}
		source, err := modernHDPackedTargetSource(modernHDBattleSource{Frame: frame}, shade<<8|0xb0)
		if err != nil {
			t.Fatal(err)
		}
		plan := modernHDBattlePlan{Base: make([]byte, 64000), Palette: p, Layers: []modernHDBattleLayer{
			{HD: actor, HDKey: "packed-matrix-actor", Rect: actor.Bounds().Add(image.Pt(30, 40)), Fill: -1},
			{Native: source.Frame, HD: target, HDKey: "packed-matrix-target", Rect: target.Bounds().Add(image.Pt(120, 90)), Fill: -1, PackedShader: source.PackedShader, PackedColours: source.PackedColours},
		}}
		var mean [3]float64
		for index := 1; index <= 4; index++ {
			v := color.NRGBAModel.Convert(p[((index+shade)&7)+0xb0]).(color.NRGBA)
			mean[0] += float64(v.R) / 4
			mean[1] += float64(v.G) / 4
			mean[2] += float64(v.B) / 4
		}
		paint := image.NewNRGBA(target.Bounds())
		for y := 0; y < 12; y++ {
			for x := 0; x < 12; x++ {
				paint.SetNRGBA(x, y, color.NRGBA{byte(math.Round(mean[0])), byte(math.Round(mean[1])), byte(math.Round(mean[2])), target.NRGBAAt(x, y).A})
			}
		}
		want := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		draw.Draw(want, want.Bounds(), image.NewUniform(p[0]), image.Point{}, draw.Src)
		draw.Draw(want, plan.Layers[0].Rect, actor, image.Point{}, draw.Over)
		draw.Draw(want, plan.Layers[1].Rect, paint, image.Point{}, draw.Over)
		cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("packed-control/%d/%d", shade, variant), plan: plan, want: want, tolerance: 1})
	}
	return cases
}
