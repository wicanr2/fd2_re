package main

import (
	"image"
	"image/color"
	"testing"
)

func referenceModernMapAlpha(source image.Image) modernMapSpriteAlphaStats {
	result := modernMapSpriteAlphaStats{}
	for y := source.Bounds().Min.Y; y < source.Bounds().Max.Y; y++ {
		for x := source.Bounds().Min.X; x < source.Bounds().Max.X; x++ {
			_, _, _, a := source.At(x, y).RGBA()
			result.transparent = result.transparent || a == 0
			result.opaque = result.opaque || a == 65535
			if a != 0 {
				result.ink = result.ink.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return result
}

func TestModernMapSpriteAlphaPreservesReferenceFormatsAndStride(t *testing.T) {
	bounds := image.Rect(-5, -3, 9, 8)
	for _, kind := range []string{"nrgba", "rgba", "nrgba64", "rgba64", "alpha", "paletted", "gray"} {
		t.Run(kind, func(t *testing.T) {
			var source interface {
				image.Image
				Set(int, int, color.Color)
			}
			switch kind {
			case "nrgba":
				source = image.NewNRGBA(bounds)
			case "rgba":
				source = image.NewRGBA(bounds)
			case "nrgba64":
				source = image.NewNRGBA64(bounds)
			case "rgba64":
				source = image.NewRGBA64(bounds)
			case "alpha":
				source = image.NewAlpha(bounds)
			case "paletted":
				source = image.NewPaletted(bounds, color.Palette{color.NRGBA{A: 0}, color.NRGBA{A: 1}, color.NRGBA{A: 128}, color.White})
			case "gray":
				source = image.NewGray(bounds)
			}
			for y := -2; y < 6; y++ {
				for x := -4; x < 7; x++ {
					a := uint16((x + 5 + (y+3)*14) * 733)
					source.Set(x, y, color.NRGBA64{R: 65535, G: 32000, B: 12000, A: a})
				}
			}
			source.Set(-3, -1, color.White)
			source.Set(0, 0, color.Transparent)
			check := func(im image.Image) {
				t.Helper()
				if got, want := modernMapSpriteAlpha(im), referenceModernMapAlpha(im); got != want {
					t.Fatalf("alpha changed: got%+v want%+v", got, want)
				}
			}
			check(source)
			if sub, ok := source.(interface {
				SubImage(image.Rectangle) image.Image
			}); ok {
				check(sub.SubImage(image.Rect(-2, -1, 5, 4)))
			}
		})
	}
	for _, a := range []byte{0, 1, 128, 254, 255} {
		im := image.NewNRGBA(image.Rect(0, 0, 3, 4))
		for i := 3; i < len(im.Pix); i += 4 {
			im.Pix[i] = a
		}
		if got, want := modernMapSpriteAlpha(im), referenceModernMapAlpha(im); got != want {
			t.Fatalf("uniform alpha%d changed", a)
		}
	}
	if got := modernMapSpriteAlpha(image.NewRGBA(image.Rectangle{})); got != (modernMapSpriteAlphaStats{}) {
		t.Fatal("empty alpha changed")
	}
}

func BenchmarkModernMapSpriteAlpha(b *testing.B) {
	im := image.NewNRGBA(image.Rect(0, 0, 144, 144))
	for y := 4; y < 140; y++ {
		for x := 6; x < 138; x++ {
			im.SetNRGBA(x, y, color.NRGBA{R: 120, G: 80, B: 40, A: byte((x + y) % 3 * 127)})
		}
	}
	for _, name := range []string{"reference", "direct-alpha"} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(144 * 144)
			for i := 0; i < b.N; i++ {
				if name == "reference" {
					_ = referenceModernMapAlpha(im)
				} else {
					_ = modernMapSpriteAlpha(im)
				}
			}
		})
	}
}
