package main

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// 目的矩形由已驗證的呈現計畫提供；GPU不替美術決定占寬或重新取錨點。
type modernHDBattleLayer struct {
	PreservePalette bool
	Native          figani.Frame
	HD              image.Image
	HDKey           string
	Rect            image.Rectangle
	Fill            int
	Clip            *image.Rectangle
	PackedShader    int
	PackedColours   []byte
}

type modernHDBattlePlan struct {
	Base       []byte
	Palette    color.Palette
	Layers     []modernHDBattleLayer
	HDSubtract float32
	HDMix      float32
	HDTarget   [3]float32
}

type modernHDBattleGPU struct {
	World, base *ebiten.Image
	shader      *ebiten.Shader
	sources     map[string]*ebiten.Image
	owner       any
	theme       *modernStoryPortraitSet
}

const modernHDBattleShader = `//kage:unit pixels
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

func (r *modernHDBattleGPU) resetSources() {
	for _, texture := range r.sources {
		texture.Dispose()
	}
	r.sources = make(map[string]*ebiten.Image)
}

func (r *modernHDBattleGPU) close() {
	r.resetSources()
	if r.World != nil {
		r.World.Dispose()
		r.World = nil
	}
	if r.base != nil {
		r.base.Dispose()
		r.base = nil
	}
	if r.shader != nil {
		r.shader.Dispose()
		r.shader = nil
	}
	r.owner, r.theme = nil, nil
}

func modernHDBattlePalette(palette color.Palette) ([]byte, error) {
	if len(palette) != 256 {
		return nil, errors.New("modern HD battle: palette unavailable")
	}
	raw := make([]byte, 1024)
	for i, value := range palette {
		if value == nil {
			return nil, errors.New("modern HD battle: palette color unavailable")
		}
		r, g, b, a := value.RGBA()
		if a != 0xffff {
			return nil, errors.New("modern HD battle: native palette must be opaque")
		}
		copy(raw[i*4:], []byte{byte(r >> 8), byte(g >> 8), byte(b >> 8), 255})
	}
	return raw, nil
}

func modernHDBattleNativeImage(frame figani.Frame, palette []byte) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, frame.Width, frame.Height))
	for i, pixel := range frame.Pixels {
		if frame.Mask[i] != 0 {
			copy(out.Pix[i*4:i*4+4], palette[int(pixel)*4:int(pixel)*4+4])
		}
	}
	return out
}

func (r *modernHDBattleGPU) draw(plan modernHDBattlePlan) error {
	palette, err := modernHDBattlePalette(plan.Palette)
	if err != nil {
		return err
	}
	if len(plan.Base) != 64000 || plan.HDSubtract < 0 || plan.HDSubtract > 1 || math.IsNaN(float64(plan.HDSubtract)) ||
		plan.HDMix < 0 || plan.HDMix > 1 || math.IsNaN(float64(plan.HDMix)) {
		return errors.New("modern HD battle: invalid base or shade")
	}
	for _, value := range plan.HDTarget {
		if value < 0 || value > 1 || math.IsNaN(float64(value)) {
			return errors.New("modern HD battle: invalid tint")
		}
	}
	// 先驗完整計畫；錯誤不清除上一張世界。
	for _, layer := range plan.Layers {
		f := layer.Native
		if err := validateModernHDPackedGuide(f, layer.PackedShader, layer.PackedColours); err != nil {
			return err
		}
		if layer.Rect.Empty() || layer.Rect.Min.X < -1<<20 || layer.Rect.Min.Y < -1<<20 ||
			layer.Rect.Max.X > 1<<20 || layer.Rect.Max.Y > 1<<20 || (layer.Fill != -1 && layer.Fill != 33) {
			return errors.New("modern HD battle: invalid layer rectangle or fill")
		}
		if c := layer.Clip; c != nil && (c.Min.X > c.Max.X || c.Min.Y > c.Max.Y || c.Min.X < -1<<20 || c.Min.Y < -1<<20 || c.Max.X > 1<<20 || c.Max.Y > 1<<20) {
			return errors.New("modern HD battle: invalid layer clip")
		}
		if layer.HD == nil {
			if f.Width <= 0 || f.Height <= 0 || f.Width > 1024 || f.Height > 1024 || len(f.Pixels) != f.Width*f.Height || len(f.Mask) != len(f.Pixels) {
				return errors.New("modern HD battle: native layer unavailable")
			}
		} else if layer.HDKey == "" || layer.HD.Bounds().Min != (image.Point{}) || layer.HD.Bounds().Empty() ||
			layer.HD.Bounds().Dx() > 6144 || layer.HD.Bounds().Dy() > 6144 {
			return errors.New("modern HD battle: HD layer unavailable")
		}
	}
	if r.shader == nil {
		r.shader, err = ebiten.NewShader([]byte(modernHDBattleShader))
		if err != nil {
			return err
		}
	}
	if r.sources == nil {
		r.sources = make(map[string]*ebiten.Image)
	}
	if r.World == nil {
		r.World = ebiten.NewImage(modernHDWidth, modernHDHeight)
		r.base = ebiten.NewImage(320, 200)
	}
	r.World.Clear()
	base := make([]byte, 64000*4)
	for i, pixel := range plan.Base {
		copy(base[i*4:i*4+4], palette[int(pixel)*4:int(pixel)*4+4])
	}
	r.base.WritePixels(base)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(6, 6)
	r.World.DrawImage(r.base, op)
	for _, layer := range plan.Layers {
		destination := r.World
		if layer.Clip != nil {
			clip := r.World.Bounds().Intersect(*layer.Clip)
			if clip.Empty() {
				continue
			}
			destination = r.World.SubImage(clip).(*ebiten.Image)
		}
		var picture image.Image
		key := "hd/" + layer.HDKey
		if layer.HD != nil {
			picture = layer.HD
			if layer.PackedShader != 0 {
				key = modernHDPackedTextureKey(key, palette, layer.PackedColours, layer.PackedShader)
				if r.sources[key] == nil {
					picture, err = modernHDPackedPicture(picture, palette, layer.PackedColours, layer.PackedShader)
					if err != nil {
						return err
					}
				}
			}
		} else {
			f := layer.Native
			h := sha256.New()
			var shape [8]byte
			binary.LittleEndian.PutUint32(shape[:4], uint32(f.Width))
			binary.LittleEndian.PutUint32(shape[4:], uint32(f.Height))
			h.Write(shape[:])
			h.Write(f.Pixels)
			h.Write(f.Mask)
			h.Write(palette)
			key = fmt.Sprintf("native/%x", h.Sum(nil))
			if r.sources[key] == nil {
				picture = modernHDBattleNativeImage(f, palette)
			}
		}
		texture := r.sources[key]
		if texture == nil {
			texture = ebiten.NewImageFromImage(picture)
			r.sources[key] = texture
		}
		draw := &ebiten.DrawRectShaderOptions{}
		draw.Images[0] = texture
		draw.GeoM.Scale(float64(layer.Rect.Dx())/float64(texture.Bounds().Dx()), float64(layer.Rect.Dy())/float64(texture.Bounds().Dy()))
		draw.GeoM.Translate(float64(layer.Rect.Min.X), float64(layer.Rect.Min.Y))
		fill := []float32{0, 0, 0, 0}
		if layer.Fill == 33 {
			fill = []float32{float32(palette[132]) / 255, float32(palette[133]) / 255, float32(palette[134]) / 255, 1}
		}
		subtract := float32(0)
		mix := float32(0)
		if layer.HD != nil && !layer.PreservePalette {
			subtract = plan.HDSubtract
			mix = plan.HDMix
		}
		draw.Uniforms = map[string]interface{}{"FillColor": fill, "Subtract": subtract, "Tint": plan.HDTarget[:], "Mix": mix}
		destination.DrawRectShader(texture.Bounds().Dx(), texture.Bounds().Dy(), r.shader, draw)
	}
	return nil
}
