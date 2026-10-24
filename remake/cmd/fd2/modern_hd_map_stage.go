package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernMapStageHD struct {
	native figani.Frame
	art    modernTerrainHDAsset
	hash   string
}

func loadModernMapStageHD(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	if set == nil || asset.Role != "map_stage_hd" || asset.SourceArchive != "FDOTHER.DAT" || asset.SourceResource != 42 ||
		(asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") || asset.ConsumerContract != "native_map_stage_rgba_hd6_v1" ||
		asset.LayoutPolicy != "native_map_stage_hd6" || asset.AlphaContract != "coverage" || asset.Width != 1872 || asset.Height != 1152 ||
		len(asset.NativeFrames) != 1 || asset.NativeFrames[0] != (modernBattleHDDescriptor{Width: 312, Height: 192}) ||
		filepath.Base(asset.File) != asset.File || filepath.Ext(asset.File) != ".png" || set.mapStages[42] != nil {
		return errors.New("modern HD map stage: invalid or duplicate source")
	}
	metadata, err := os.ReadFile(separatedAssetPath("surfaces/FDOTHER_042/resource.json"))
	if err != nil {
		return err
	}
	digest := sha256.Sum256(metadata)
	if hex.EncodeToString(digest[:]) != asset.SourceMetadataSHA256 {
		return errors.New("modern HD map stage: metadata changed")
	}
	source, err := fdother.LoadSeparatedNativeCh23Stage(separatedAssetPath("surfaces"))
	if err != nil {
		return err
	}
	native, err := modernHDBattleSurfaceFrame(source)
	if err != nil {
		return err
	}
	if digest := sha256.Sum256(native.Pixels); hex.EncodeToString(digest[:]) != "1c95b24d091e860b4a91c75ad300e89ebf039b6e9dfc9b1dd7223bb0680a3158" {
		return errors.New("modern HD map stage: native indexed source changed")
	}
	colours := []byte{}
	seen := [256]bool{}
	for i, m := range native.Mask {
		if m != 255 {
			return errors.New("modern HD map stage: native source is not opaque")
		}
		seen[native.Pixels[i]] = true
	}
	for i, yes := range seen {
		if yes {
			colours = append(colours, byte(i))
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, asset.File))
	if err != nil {
		return err
	}
	digest = sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != asset.SHA256 {
		return errors.New("modern HD map stage: image SHA changed")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width != 1872 || config.Height != 1152 {
		return errors.New("modern HD map stage: invalid image geometry")
	}
	im, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil || im.Bounds() != image.Rect(0, 0, 1872, 1152) {
		return errors.New("modern HD map stage: invalid image geometry")
	}
	picture := image.NewRGBA(im.Bounds())
	draw.Draw(picture, picture.Bounds(), im, image.Point{}, draw.Src)
	for i := 3; i < len(picture.Pix); i += 4 {
		if picture.Pix[i] != 255 {
			return errors.New("modern HD map stage: opaque source requires opaque art")
		}
	}
	entry := &modernMapStageHD{native: native, hash: asset.SHA256, art: modernTerrainHDAsset{Image: picture, Colours: colours}}
	if set.mapStages == nil {
		set.mapStages = make(map[int]*modernMapStageHD)
	}
	set.mapStages[42] = entry
	set.hasHD = true
	return nil
}

func (s *modernMapStageHD) validate(state *nativeCh23AdapterState) error {
	if s == nil || state == nil || state.rowShift < 0 || state.rowShift >= 192 || len(state.staging) != 312*192 || len(s.native.Pixels) != 312*192 {
		return errors.New("modern HD map stage: source provenance absent")
	}
	if s.art.Image == nil || s.art.Image.Bounds() != image.Rect(0, 0, 1872, 1152) || s.hash == "" || len(s.art.Colours) == 0 {
		return errors.New("modern HD map stage: HD source absent or wrong size")
	}
	for y := 0; y < 192; y++ {
		src := (y - state.rowShift + 192) % 192 * 312
		if !bytes.Equal(state.staging[y*312:(y+1)*312], s.native.Pixels[src:src+312]) {
			return errors.New("modern HD map stage: indexed row rotation changed")
		}
	}
	return nil
}

// Native literal writes hide the HD underlay. Destination-remap spans select
// the same LUT colour profile, and transparent spans retain the current DAC.
func (g *Game) prepareModernHDMapStage(in indexedmap.FrameInput, palette color.Palette, viewportRow int, state *nativeCh23AdapterState) (*image.RGBA, error) {
	if state == nil || g.modernStoryPortraits == nil || g.nativeMapAssets == nil || g.nativeMapAssets.MapIndex != 23 {
		return nil, nil
	}
	s := g.modernStoryPortraits.mapStages[42]
	if s == nil {
		return nil, nil
	}
	if err := s.validate(state); err != nil {
		return nil, err
	}
	if viewportRow < 0 {
		viewportRow = 0
	}
	ctx, err := terrainColourContext(23, g.nativeMapAssets.Palette, palette, in.LUT)
	if err != nil {
		return nil, err
	}
	raw := s.art.shade(ctx, true)
	var mapped *image.RGBA
	out := image.NewRGBA(image.Rect(0, 0, 1872, 1152))
	for row := 0; row < 8; row++ {
		for col := 0; col < 13; col++ {
			at := (in.CameraY+row)*in.MapWidth + in.CameraX + col
			if at < 0 || at >= len(in.Cells) {
				return nil, errors.New("modern HD map stage: terrain cell absent")
			}
			cell := in.Cells[at]
			tile := int(cell.Tile & 0x3ff)
			if tile*4 >= len(in.Controls) {
				return nil, errors.New("modern HD map stage: terrain control absent")
			}
			frame, err := fdicon.NativeTerrainFrameIndex(tile, in.Controls[tile*4], in.Flip, in.TerrainCycle)
			if err != nil {
				return nil, err
			}
			sprite, err := in.TerrainBank.SpriteFor(frame/12, (frame%12)/3, frame%3)
			if err != nil {
				return nil, err
			}
			if len(sprite.Mask) != 576 || len(sprite.RemapMask) != 576 {
				return nil, errors.New("modern HD map stage: terrain masks absent")
			}
			for y := 0; y < 24; y++ {
				for x := 0; x < 24; x++ {
					i := y*24 + x
					if sprite.Mask[i] != 0 {
						continue
					}
					picture := raw
					if cell.BlitMode != 0xff && sprite.RemapMask[i] != 0 {
						if mapped == nil {
							mapped = s.art.shade(ctx, false)
						}
						picture = mapped
					}
					dx, dy := (col*24+x)*6, (row*24+y-viewportRow)*6
					if dy < 0 || dy+6 > 1152 {
						continue
					}
					sy := ((row*24+y)*6 - state.rowShift*6 + 1152) % 1152
					for yy := 0; yy < 6; yy++ {
						copy(out.Pix[(dy+yy)*out.Stride+dx*4:(dy+yy)*out.Stride+(dx+6)*4], picture.Pix[(sy+yy)*picture.Stride+dx*4:(sy+yy)*picture.Stride+(dx+6)*4])
					}
				}
			}
		}
	}
	return out, nil
}

// Stage palettes also affect HD figures. Keep their coverage alpha and the
// existing inactive-unit band; the four-colour RGB mapping is an art extension.
func shadeModernHDMapFigure(raw image.Image, native fdicon.Sprite, ctx modernTerrainColourContext) (*image.RGBA, error) {
	if raw == nil || raw.Bounds() != image.Rect(0, 0, 144, 144) || len(native.Pixels) != 576 || len(native.Mask) != 576 {
		return nil, errors.New("modern HD map stage: figure source absent")
	}
	seen := [256]bool{}
	colours := []byte{}
	for i, m := range native.Mask {
		if m != 0 {
			seen[native.Pixels[i]] = true
		}
	}
	identity, black := true, true
	for i, yes := range seen {
		if yes {
			colours = append(colours, byte(i))
			identity = identity && ctx.Raw[i] == ctx.Base[i]
			black = black && ctx.Raw[i] == [3]float64{}
		}
	}
	if len(colours) == 0 {
		return nil, errors.New("modern HD map stage: figure has no source writes")
	}
	input := image.NewNRGBA(raw.Bounds())
	draw.Draw(input, input.Bounds(), raw, image.Point{}, draw.Src)
	values := make(map[uint32][3]byte)
	if !identity {
		for i := 0; i < len(input.Pix); i += 4 {
			p := input.Pix[i : i+4]
			if p[3] == 0 {
				continue
			}
			key := uint32(p[0])<<16 | uint32(p[1])<<8 | uint32(p[2])
			value, ok := values[key]
			if !ok {
				if !black {
					w := terrainColourWeights(color.RGBA{p[0], p[1], p[2], 255}, colours, ctx.Base)
					v := [3]float64{float64(p[0]), float64(p[1]), float64(p[2])}
					for slot, weight := range w.Weights {
						index := w.Indices[slot]
						for c := range v {
							v[c] += weight * (ctx.Raw[index][c] - ctx.Base[index][c])
						}
					}
					for c, x := range v {
						value[c] = byte(math.Round(math.Max(0, math.Min(255, x))))
					}
				}
				values[key] = value
			}
			copy(p[:3], value[:])
		}
	}
	out := image.NewRGBA(input.Bounds())
	draw.Draw(out, out.Bounds(), input, image.Point{}, draw.Src)
	return out, nil
}

func (g *Game) colourModernHDStageFigures(frame *modernHDMapFrame, in indexedmap.NativeFrameInput, palette color.Palette) error {
	ctx, err := terrainColourContext(23, g.nativeMapAssets.Palette, palette, in.Frame.LUT)
	if err != nil {
		return err
	}
	prepare := func(unit *modernHDMapUnitDraw) error {
		native, err := in.Frame.UnitBank.SpriteFor(unit.Group, unit.Frame/3, unit.Frame%3)
		if err != nil {
			return err
		}
		unit.ColourImage, err = shadeModernHDMapFigure(g.modernStoryPortraits.mapSpritesHD[unit.Group][unit.Frame], native, ctx)
		return err
	}
	for i := range frame.Units {
		if err := prepare(&frame.Units[i]); err != nil {
			return err
		}
	}
	if frame.HUDUnit != nil {
		if err := prepare(&frame.HUDUnit.Unit); err != nil {
			return err
		}
	}
	return nil
}
