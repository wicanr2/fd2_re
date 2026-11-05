package main

import (
	"image"
	"image/draw"

	xdraw "golang.org/x/image/draw"
)

type hdTextureQuality uint8

const (
	hdTextureFull hdTextureQuality = iota
	hdTextureBalanced
	hdTextureLight
)

func parseHDTextureQuality(name string) hdTextureQuality {
	switch name {
	case "full":
		return hdTextureFull
	case "light":
		return hdTextureLight
	default:
		return hdTextureBalanced
	}
}

func (q hdTextureQuality) name() string {
	switch q {
	case hdTextureFull:
		return "full"
	case hdTextureLight:
		return "light"
	default:
		return "balanced"
	}
}

func (q hdTextureQuality) numerator() int {
	switch q {
	case hdTextureFull:
		return 6
	case hdTextureLight:
		return 3
	default:
		return 4
	}
}

func (q hdTextureQuality) density() float64 { return float64(q.numerator()) / 6 }

func (q hdTextureQuality) rasterBounds() image.Rectangle {
	return image.Rect(0, 0, modernHDWidth*q.numerator()/6, modernHDHeight*q.numerator()/6)
}

// Plans keep their HD6 coordinates. Integer floor/ceil avoids floating point
// clipping loss at viewport boundaries; native multiples of six stay exact.
func (q hdTextureQuality) rasterRect(r image.Rectangle) image.Rectangle {
	floor := func(v int) int {
		v *= q.numerator()
		result := v / 6
		if v < 0 && v%6 != 0 {
			result--
		}
		return result
	}
	ceil := func(v int) int {
		v *= q.numerator()
		result := v / 6
		if v > 0 && v%6 != 0 {
			result++
		}
		return result
	}
	return image.Rect(floor(r.Min.X), floor(r.Min.Y), ceil(r.Max.X), ceil(r.Max.Y))
}

// Source validation and native geometry precede this GPU-only derivative.
// Drawing keeps the original world rectangles; UI and source art stay intact.
func (q hdTextureQuality) textureSource(source image.Image) image.Image {
	if q == hdTextureFull {
		return source
	}
	numerator := q.numerator()
	b := source.Bounds()
	w, h := b.Dx()*numerator/6, b.Dy()*numerator/6
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	picture := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(picture, picture.Bounds(), source, b, draw.Src, nil)
	return picture
}

func hdTextureQualityLabels(locale string) [3]string {
	switch locale {
	case "en":
		return [3]string{"Full", "Balanced", "Light"}
	case "ja":
		return [3]string{"最高", "標準", "軽量"}
	case "zh-Hans":
		return [3]string{"完整", "均衡", "轻量"}
	default:
		return [3]string{"完整", "均衡", "輕量"}
	}
}

func (g *Game) setHDTextureQuality(q hdTextureQuality) {
	g.playerHDQuality = q
	configured := loadSettings()
	configured.HDTextureQuality = q.name()
	saveSettings(configured)
}
