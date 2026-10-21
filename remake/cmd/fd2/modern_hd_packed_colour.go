package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func validModernHDPackedShader(shader int) bool {
	return shader == 0 || (shader > 0 && shader&0xff == 0xb0 && shader>>8 >= 2 && shader>>8 <= 8)
}

func validateModernHDPackedGuide(frame figani.Frame, shader int, colours []byte) error {
	if !validModernHDPackedShader(shader) || (shader == 0 && len(colours) != 0) {
		return fmt.Errorf("modern HD packed: invalid shader")
	}
	if shader == 0 {
		return nil
	}
	if frame.Width <= 0 || frame.Height <= 0 || frame.Width > 1024 || frame.Height > 1024 ||
		len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) || len(colours) > 256 {
		return fmt.Errorf("modern HD packed: invalid native guide")
	}
	for index := 1; index < len(colours); index++ {
		if colours[index] <= colours[index-1] {
			return fmt.Errorf("modern HD packed: unordered native colours")
		}
	}
	var mappedColours [256]bool
	for _, index := range colours {
		mappedColours[((int(index)+(shader>>8))&7)+0xb0] = true
	}
	for index, mask := range frame.Mask {
		if mask != 0 && !mappedColours[frame.Pixels[index]] {
			return fmt.Errorf("modern HD packed: native guide is not shaded or complete")
		}
	}
	if len(colours) == 0 {
		for _, mask := range frame.Mask {
			if mask != 0 {
				return fmt.Errorf("modern HD packed: visible source has no colours")
			}
		}
	}
	return nil
}

// 原indexed拷貝保持現行command6公式；HD另保留未變色的可見索引作色差基底。
func modernHDPackedTargetSource(source modernHDBattleSource, shader int) (modernHDBattleSource, error) {
	if shader == 0 {
		return source, nil
	}
	frame := source.Frame
	if !validModernHDPackedShader(shader) || frame.Width <= 0 || frame.Height <= 0 ||
		frame.Width > 1024 || frame.Height > 1024 || len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
		return modernHDBattleSource{}, fmt.Errorf("modern HD packed: target source unavailable")
	}
	var visible [256]bool
	source.Frame.Pixels = append([]byte(nil), frame.Pixels...)
	for index, value := range frame.Pixels {
		if frame.Mask[index] != 0 {
			visible[value] = true
		}
		source.Frame.Pixels[index] = byte((int(value)+(shader>>8))&7) + 0xb0
	}
	source.PackedShader = shader
	for index, used := range visible {
		if used {
			source.PackedColours = append(source.PackedColours, byte(index))
		}
	}
	return source, nil
}

func modernHDPackedTextureKey(key string, palette, colours []byte, shader int) string {
	h := sha256.New()
	h.Write([]byte{byte(shader), byte(shader >> 8)})
	h.Write(palette)
	h.Write(colours)
	return fmt.Sprintf("%s/packed/%x", key, h.Sum(nil))
}

// 沿既有地形的四近鄰RGB色差插值；這是現代美術近似，沒有恢復原版索引。
// 使用直通RGB計算，保留來源alpha；只在新增GPU texture時呼叫。
func modernHDPackedPicture(picture image.Image, palette, colours []byte, shader int) (image.Image, error) {
	if shader == 0 {
		return picture, nil
	}
	if picture == nil || picture.Bounds().Min != (image.Point{}) || picture.Bounds().Empty() ||
		picture.Bounds().Dx() > 6144 || picture.Bounds().Dy() > 6144 || len(palette) != 1024 ||
		!validModernHDPackedShader(shader) || len(colours) > 256 {
		return nil, fmt.Errorf("modern HD packed: colour inputs unavailable")
	}
	for index := 1; index < len(colours); index++ {
		if colours[index] <= colours[index-1] {
			return nil, fmt.Errorf("modern HD packed: unordered colour inputs")
		}
	}
	if len(colours) == 0 {
		return picture, nil
	}
	var base, mapped [256][3]float64
	for index := 0; index < 256; index++ {
		if palette[index*4+3] != 255 {
			return nil, fmt.Errorf("modern HD packed: palette must be opaque")
		}
		to := ((index + (shader >> 8)) & 7) + 0xb0
		for channel := 0; channel < 3; channel++ {
			base[index][channel] = float64(palette[index*4+channel])
			mapped[index][channel] = float64(palette[to*4+channel])
		}
	}
	out := image.NewNRGBA(picture.Bounds())
	cache := make(map[uint32][3]uint8)
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			value := color.NRGBAModel.Convert(picture.At(x, y)).(color.NRGBA)
			if value.A == 0 {
				continue
			}
			key := uint32(value.R)<<16 | uint32(value.G)<<8 | uint32(value.B)
			rgb, found := cache[key]
			if !found {
				weights := terrainColourWeights(color.RGBA{R: value.R, G: value.G, B: value.B, A: 255}, colours, base)
				channels := [3]float64{float64(value.R), float64(value.G), float64(value.B)}
				for slot, weight := range weights.Weights {
					index := weights.Indices[slot]
					for channel := range channels {
						channels[channel] += weight * (mapped[index][channel] - base[index][channel])
					}
				}
				for channel, result := range channels {
					rgb[channel] = uint8(math.Round(math.Max(0, math.Min(255, result))))
				}
				cache[key] = rgb
			}
			out.SetNRGBA(x, y, color.NRGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: value.A})
		}
	}
	return out, nil
}
