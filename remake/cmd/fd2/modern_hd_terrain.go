package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

const modernHDTerrainSize = 144

type modernTerrainColourWeights struct {
	Indices [4]byte
	Weights [4]float64
}

type modernTerrainHDAsset struct {
	Image        *image.RGBA
	Role         string
	Colours      []byte
	Weights      map[uint32]modernTerrainColourWeights
	Shades       map[[33]byte]*image.RGBA
	ShadeOrder   [][33]byte
	WeightsBasis [32]byte
}

func (a *modernTerrainHDAsset) supports(role string) bool {
	return a != nil && (a.Role == role || a.Role == "both")
}

func loadModernTerrainHDAsset(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	count := asset.FrameCount
	if (asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") ||
		asset.ConsumerContract != "fdshap_native_hd_sparse_v1" || asset.MapID < 0 || asset.MapID >= 33 ||
		asset.Width != 144 || asset.Height != 144 || asset.AlphaContract != "native_mask_nearest_6" ||
		asset.CyclePolicy != "native_selector" || count <= 0 || count > 1024 ||
		len(asset.FrameIDs) != count || len(asset.Files) != count || len(asset.FrameSHA256) != count || len(asset.Semantics) != count {
		return errors.New("modern HD terrain: invalid catalog contract")
	}
	frames := make(map[int]*modernTerrainHDAsset, count)
	for i, frame := range asset.FrameIDs {
		role := asset.Semantics[i]
		if frame < 0 || frame >= 1024 || (role != "terrain" && role != "foreground" && role != "both") ||
			frames[frame] != nil || (set.terrainHD[asset.MapID] != nil && set.terrainHD[asset.MapID][frame] != nil) {
			return errors.New("modern HD terrain: invalid or duplicate frame identity")
		}
		name := asset.Files[i]
		if filepath.Base(name) != name || filepath.Ext(name) != ".png" {
			return errors.New("modern HD terrain: unsafe file path")
		}
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return fmt.Errorf("modern HD terrain frame %d: %w", frame, err)
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) != asset.FrameSHA256[i] {
			return fmt.Errorf("modern HD terrain frame %d: sha256 mismatch", frame)
		}
		decoded, format, err := image.Decode(bytes.NewReader(raw))
		if err != nil || format != "png" || decoded.Bounds() != image.Rect(0, 0, 144, 144) {
			return fmt.Errorf("modern HD terrain frame %d: invalid PNG geometry", frame)
		}
		picture := image.NewRGBA(decoded.Bounds())
		draw.Draw(picture, picture.Bounds(), decoded, image.Point{}, draw.Src)
		for offset := 3; offset < len(picture.Pix); offset += 4 {
			if picture.Pix[offset] != 0 && picture.Pix[offset] != 255 {
				return fmt.Errorf("modern HD terrain frame %d: non-binary alpha", frame)
			}
		}
		frames[frame] = &modernTerrainHDAsset{Image: picture, Role: role}
	}
	if set.terrainHD == nil {
		set.terrainHD = make(map[int]map[int]*modernTerrainHDAsset)
	}
	if set.terrainHD[asset.MapID] == nil {
		set.terrainHD[asset.MapID] = make(map[int]*modernTerrainHDAsset)
	}
	for frame, source := range frames {
		set.terrainHD[asset.MapID][frame] = source
	}
	return nil
}

func (s *modernStoryPortraitSet) validateTerrainHD(mapID int, bank *fdicon.Bank) error {
	frames := s.terrainHD[mapID]
	if len(frames) == 0 || s.terrainHDBanks[mapID] == bank && bank != nil {
		return nil
	}
	if bank == nil {
		return errors.New("modern HD terrain: native bank missing")
	}
	for frame, asset := range frames {
		if frame >= len(bank.Sprites) {
			return errors.New("modern HD terrain: frame exceeds actual native bank")
		}
		source := bank.Sprites[frame]
		if len(source.Pixels) != 576 || len(source.Mask) != 576 || len(source.RemapMask) != 576 {
			return errors.New("modern HD terrain: native layers incomplete")
		}
		seen := [256]bool{}
		for i, mask := range source.Mask {
			if mask > 1 || source.RemapMask[i] > 1 || mask != 0 && source.RemapMask[i] != 0 {
				return errors.New("modern HD terrain: source and destination masks must be binary and disjoint")
			}
			if mask != 0 {
				seen[source.Pixels[i]] = true
			}
			for y := i / 24 * 6; y < (i/24+1)*6; y++ {
				for x := i % 24 * 6; x < (i%24+1)*6; x++ {
					want := uint8(0)
					if mask != 0 {
						want = 255
					}
					if asset.Image.RGBAAt(x, y).A != want {
						return fmt.Errorf("modern HD terrain frame %d: native alpha mismatch", frame)
					}
				}
			}
		}
		asset.Colours = nil
		for i, present := range seen {
			if present {
				asset.Colours = append(asset.Colours, byte(i))
			}
		}
		asset.Weights, asset.Shades, asset.ShadeOrder = nil, nil, nil
	}
	if s.terrainHDBanks == nil {
		s.terrainHDBanks = make(map[int]*fdicon.Bank)
	}
	s.terrainHDBanks[mapID] = bank
	return nil
}

type modernTerrainColourContext struct {
	Base, Mapped, Raw [256][3]float64
	Key, Basis        [32]byte
}

func terrainColourContext(mapID int, base, current color.Palette, lut []byte) (modernTerrainColourContext, error) {
	var result modernTerrainColourContext
	if len(base) != 256 || len(current) != 256 || len(lut) != 256 {
		return result, errors.New("modern HD terrain: incomplete palette or LUT")
	}
	data, basis := []byte{byte(mapID)}, make([]byte, 0, 768)
	for i := 0; i < 256; i++ {
		if base[i] == nil || current[i] == nil || current[lut[i]] == nil {
			return result, errors.New("modern HD terrain: palette colour missing")
		}
		for j, colour := range []color.Color{base[i], current[lut[i]], current[i]} {
			r, g, b, _ := colour.RGBA()
			v := [3]float64{float64(r >> 8), float64(g >> 8), float64(b >> 8)}
			switch j {
			case 0:
				result.Base[i] = v
				basis = append(basis, byte(r>>8), byte(g>>8), byte(b>>8))
			case 1:
				result.Mapped[i] = v
			case 2:
				result.Raw[i] = v
			}
			data = append(data, byte(r>>8), byte(g>>8), byte(b>>8))
		}
	}
	result.Key, result.Basis = sha256.Sum256(data), sha256.Sum256(basis)
	return result, nil
}

func terrainColourWeights(value color.RGBA, colours []byte, base [256][3]float64) modernTerrainColourWeights {
	var result modernTerrainColourWeights
	distances := [4]float64{math.Inf(1), math.Inf(1), math.Inf(1), math.Inf(1)}
	for _, index := range colours {
		v := base[index]
		r, g, b := float64(value.R)-v[0], float64(value.G)-v[1], float64(value.B)-v[2]
		distance := r*r + g*g + b*b
		for slot := 0; slot < 4; slot++ {
			if distance < distances[slot] {
				for k := 3; k > slot; k-- {
					distances[k], result.Indices[k] = distances[k-1], result.Indices[k-1]
				}
				distances[slot], result.Indices[slot] = distance, index
				break
			}
		}
	}
	total := 0.0
	for slot, distance := range distances {
		if distances[0] == 0 {
			if distance == 0 {
				result.Weights[slot] = 1
			}
		} else if !math.IsInf(distance, 1) {
			result.Weights[slot] = 1 / distance
		}
		total += result.Weights[slot]
	}
	for i := range result.Weights {
		if total != 0 {
			result.Weights[i] /= total
		}
	}
	return result
}

// The palette controls remain native; interpolation between their RGB colours
// is an explicit modern-art extension, not recovery of an original index.
func (a *modernTerrainHDAsset) shade(ctx modernTerrainColourContext, raw bool) *image.RGBA {
	target := ctx.Mapped
	if raw {
		target = ctx.Raw
	}
	identity := true
	for _, index := range a.Colours {
		identity = identity && target[index] == ctx.Base[index]
	}
	if identity {
		return a.Image
	}
	var key [33]byte
	copy(key[:32], ctx.Key[:])
	if raw {
		key[32] = 1
	}
	if picture := a.Shades[key]; picture != nil {
		return picture
	}
	black, white := true, true
	for _, index := range a.Colours {
		black = black && target[index] == [3]float64{}
		white = white && target[index] == [3]float64{255, 255, 255}
	}
	if a.Weights == nil || a.WeightsBasis != ctx.Basis {
		a.Weights, a.WeightsBasis = make(map[uint32]modernTerrainColourWeights), ctx.Basis
	}
	colours := make(map[uint32]color.RGBA)
	picture := image.NewRGBA(a.Image.Bounds())
	for offset := 0; offset < len(a.Image.Pix); offset += 4 {
		v := a.Image.Pix[offset : offset+4]
		if v[3] == 0 {
			continue
		}
		rgb := uint32(v[0])<<16 | uint32(v[1])<<8 | uint32(v[2])
		mapped, found := colours[rgb]
		if !found {
			if black || white {
				value := uint8(0)
				if white {
					value = 255
				}
				mapped = color.RGBA{value, value, value, 255}
				colours[rgb] = mapped
				copy(picture.Pix[offset:offset+4], []byte{mapped.R, mapped.G, mapped.B, mapped.A})
				continue
			}
			weights, found := a.Weights[rgb]
			if !found {
				weights = terrainColourWeights(color.RGBA{v[0], v[1], v[2], 255}, a.Colours, ctx.Base)
				a.Weights[rgb] = weights
			}
			values := [3]float64{float64(v[0]), float64(v[1]), float64(v[2])}
			for slot, weight := range weights.Weights {
				index := weights.Indices[slot]
				for channel := range values {
					values[channel] += weight * (target[index][channel] - ctx.Base[index][channel])
				}
			}
			clamp := func(value float64) uint8 { return uint8(math.Round(math.Max(0, math.Min(255, value)))) }
			mapped = color.RGBA{clamp(values[0]), clamp(values[1]), clamp(values[2]), 255}
			colours[rgb] = mapped
		}
		copy(picture.Pix[offset:offset+4], []byte{mapped.R, mapped.G, mapped.B, mapped.A})
	}
	if a.Shades == nil {
		a.Shades = make(map[[33]byte]*image.RGBA)
	}
	if len(a.ShadeOrder) == 3 {
		delete(a.Shades, a.ShadeOrder[0])
		a.ShadeOrder = a.ShadeOrder[1:]
	}
	a.Shades[key] = picture
	a.ShadeOrder = append(a.ShadeOrder, key)
	return picture
}

func modernHDIdentityLUT() []byte {
	lut := make([]byte, 256)
	for i := range lut {
		lut[i] = byte(i)
	}
	return lut
}
