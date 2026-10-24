package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// Each basis and its at most five colour variants belong to one command owner.
// Native source pixels remain unchanged; the exact LUT is applied to fallback.
type modernBattleSurfaceLUTBasis struct {
	native     figani.Frame
	palette    color.Palette
	paletteSHA [32]byte
	colours    []byte
	weights    map[uint32]modernTerrainColourWeights
	profiles   map[int]*modernBattleSurfaceLUT
}

type modernBattleSurfaceLUT struct {
	index     int
	lookup    []byte
	lookupSHA [32]byte
	basis     *modernBattleSurfaceLUTBasis
	cache     map[string]modernBattleSurfaceLUTImage
	activeKey string
}

type modernBattleSurfaceLUTImage struct {
	picture image.Image
	key     string
}

func newModernBattleSurfaceLUTBasis(native figani.Frame, palette color.Palette) (*modernBattleSurfaceLUTBasis, error) {
	raw, err := modernHDBattlePalette(palette)
	if err != nil || native.Width <= 0 || native.Height <= 0 || len(native.Pixels) != native.Width*native.Height || len(native.Mask) != len(native.Pixels) {
		return nil, fmt.Errorf("modern HD surface LUT: incomplete basis")
	}
	b := &modernBattleSurfaceLUTBasis{native: native, paletteSHA: sha256.Sum256(raw),
		weights: make(map[uint32]modernTerrainColourWeights), profiles: make(map[int]*modernBattleSurfaceLUT)}
	b.native.Pixels, b.native.Mask = append([]byte(nil), native.Pixels...), append([]byte(nil), native.Mask...)
	b.palette = make(color.Palette, 256)
	seen := [256]bool{}
	for i, mask := range native.Mask {
		if mask != 0 && mask != 255 {
			return nil, fmt.Errorf("modern HD surface LUT: invalid source mask")
		}
		if mask != 0 {
			seen[native.Pixels[i]] = true
		}
	}
	for i := range b.palette {
		b.palette[i] = color.RGBA{raw[4*i], raw[4*i+1], raw[4*i+2], 255}
		if seen[i] {
			b.colours = append(b.colours, byte(i))
		}
	}
	return b, nil
}

func (b *modernBattleSurfaceLUTBasis) profile(index int, lookup []byte) (*modernBattleSurfaceLUT, error) {
	if b == nil || index < 11 || index > 15 || len(lookup) != 256 {
		return nil, fmt.Errorf("modern HD surface LUT: unsupported lookup")
	}
	if old := b.profiles[index]; old != nil {
		if !bytes.Equal(old.lookup, lookup) {
			return nil, fmt.Errorf("modern HD surface LUT: lookup changed during preparation")
		}
		return old, nil
	}
	p := &modernBattleSurfaceLUT{index: index, lookup: append([]byte(nil), lookup...), lookupSHA: sha256.Sum256(lookup), basis: b}
	b.profiles[index] = p
	return p, nil
}

func (p *modernBattleSurfaceLUT) validate(frame figani.Frame) error {
	if p == nil || p.basis == nil || p.index < 11 || p.index > 15 || len(p.lookup) != 256 || sha256.Sum256(p.lookup) != p.lookupSHA {
		return fmt.Errorf("modern HD surface LUT: frozen lookup differs")
	}
	b := p.basis
	raw, err := modernHDBattlePalette(b.palette)
	if err != nil || sha256.Sum256(raw) != b.paletteSHA || frame.Width != b.native.Width || frame.Height != b.native.Height ||
		!bytes.Equal(frame.Pixels, b.native.Pixels) || !bytes.Equal(frame.Mask, b.native.Mask) {
		return fmt.Errorf("modern HD surface LUT: frozen palette or raw source differs")
	}
	return nil
}

func (p *modernBattleSurfaceLUT) nativeFrame(frame figani.Frame) (figani.Frame, error) {
	if err := p.validate(frame); err != nil {
		return frame, err
	}
	frame.Pixels = append([]byte(nil), frame.Pixels...)
	for i, mask := range frame.Mask {
		if mask != 0 {
			frame.Pixels[i] = p.lookup[frame.Pixels[i]]
		}
	}
	return frame, nil
}

func (p *modernBattleSurfaceLUT) prepare(set *modernStoryPortraitSet, raw image.Image, rawKey string, frame figani.Frame) error {
	if err := p.validate(frame); err != nil {
		return err
	}
	if raw == nil {
		return nil // An entirely unregistered surface keeps exact native fallback.
	}
	if _, exists := p.cache[rawKey]; exists {
		return nil
	}
	b := p.basis
	ctx, err := terrainColourContext(0, b.palette, b.palette, p.lookup)
	if err != nil || len(b.colours) == 0 || raw.Bounds().Dx() != frame.Width*6 || raw.Bounds().Dy() != frame.Height*6 || rawKey == "" {
		return fmt.Errorf("modern HD surface LUT: colour inputs unavailable")
	}
	identity, black, white := true, true, true
	for _, index := range b.colours {
		identity = identity && ctx.Mapped[index] == ctx.Base[index]
		black = black && ctx.Mapped[index] == [3]float64{}
		white = white && ctx.Mapped[index] == [3]float64{255, 255, 255}
	}
	picture, key := raw, rawKey
	if !identity {
		input := image.NewNRGBA(image.Rect(0, 0, raw.Bounds().Dx(), raw.Bounds().Dy()))
		draw.Draw(input, input.Bounds(), raw, raw.Bounds().Min, draw.Src)
		output := image.NewNRGBA(input.Bounds())
		copy(output.Pix, input.Pix)
		mapped := make(map[uint32][3]byte)
		for offset := 0; offset < len(input.Pix); offset += 4 {
			v := input.Pix[offset : offset+4]
			if v[3] == 0 {
				continue
			}
			rgb := uint32(v[0])<<16 | uint32(v[1])<<8 | uint32(v[2])
			value, found := mapped[rgb]
			if !found {
				if black {
					value = [3]byte{}
				} else if white {
					value = [3]byte{255, 255, 255}
				} else {
					weights, exists := b.weights[rgb]
					if !exists {
						weights = terrainColourWeights(color.RGBA{v[0], v[1], v[2], 255}, b.colours, ctx.Base)
						b.weights[rgb] = weights
					}
					channels := [3]float64{float64(v[0]), float64(v[1]), float64(v[2])}
					for slot, weight := range weights.Weights {
						index := weights.Indices[slot]
						for c := range channels {
							channels[c] += weight * (ctx.Mapped[index][c] - ctx.Base[index][c])
						}
					}
					for c, channel := range channels {
						value[c] = byte(math.Round(math.Max(0, math.Min(255, channel))))
					}
				}
				mapped[rgb] = value
			}
			copy(output.Pix[offset:offset+3], value[:])
		}
		picture = output
		key = fmt.Sprintf("surface-lut/%s/%x/%x", rawKey, p.lookupSHA, b.paletteSHA)
	}
	// Keep the last drawn content while a different F2 candidate is checked.
	// Equal source hashes can share content after the normal raw-source guard.
	if p.cache == nil {
		p.cache = make(map[string]modernBattleSurfaceLUTImage)
	}
	if len(p.cache) == 2 {
		for old := range p.cache {
			if old != p.activeKey {
				delete(p.cache, old)
				break
			}
		}
	}
	p.cache[rawKey] = modernBattleSurfaceLUTImage{picture: picture, key: key}
	if p.activeKey == "" {
		p.activeKey = rawKey
	}
	return nil
}

func (p *modernBattleSurfaceLUT) prepared(set *modernStoryPortraitSet, raw image.Image, rawKey string, frame figani.Frame) (image.Image, string, error) {
	if err := p.validate(frame); err != nil {
		return nil, "", err
	}
	if raw == nil {
		return nil, "", nil
	}
	cached, exists := p.cache[rawKey]
	if !exists || cached.picture == nil || cached.key == "" {
		return nil, "", fmt.Errorf("modern HD surface LUT: colour image was not prepared")
	}
	p.activeKey = rawKey
	return cached.picture, cached.key, nil
}

func prepareModernBattleSurfaceSource(set *modernStoryPortraitSet, source modernHDBattleSource) error {
	picture, key, err := set.battleSurfaceHD(source)
	if err != nil || source.SurfaceLUT == nil {
		return err
	}
	return source.SurfaceLUT.prepare(set, picture, key, source.Frame)
}
