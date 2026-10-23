package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

// HD frame sources remain separate from the registered 24px projection.
// Admission does not activate or change world rendering.
func loadModernHDMapSpriteAsset(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	walk := asset.ConsumerContract == "fdicon_hd_map_sprite_12x144_v1" && asset.FrameCount == 12 &&
		(asset.CyclePolicy == "three_distinct_cycles" || asset.CyclePolicy == "source_exact_repeats")
	static := asset.ConsumerContract == "fdicon_hd_static_sprite_3x144_v1" && asset.FrameCount == 3 &&
		asset.CyclePolicy == "static_three_phase"
	if (asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") ||
		(!walk && !static) || asset.Width != 144 || asset.Height != 144 ||
		len(asset.Files) != asset.FrameCount || len(asset.FrameSHA256) != asset.FrameCount ||
		asset.AlphaContract != "coverage" || asset.LayoutPolicy != "contain_22px_cell_feet_v1" ||
		!fdicon.NativeSourceGroupHasPixels(asset.SourceGroup) {
		return fmt.Errorf("modern HD map sprite %d violates the frame contract", asset.SourceGroup)
	}
	if _, exists := set.mapSpritesHD[asset.SourceGroup]; exists {
		return fmt.Errorf("modern HD map sprite %d is duplicated", asset.SourceGroup)
	}
	frames := make([]image.Image, 0, asset.FrameCount)
	for index, name := range asset.Files {
		if filepath.Base(name) != name || filepath.Ext(name) != ".png" {
			return fmt.Errorf("modern HD map sprite %d frame %d has an unsafe path", asset.SourceGroup, index)
		}
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return fmt.Errorf("modern HD map sprite %d frame %d: %w", asset.SourceGroup, index, err)
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) != asset.FrameSHA256[index] {
			return fmt.Errorf("modern HD map sprite %d frame %d has a sha256 mismatch", asset.SourceGroup, index)
		}
		decoded, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil || decoded.Bounds().Dx() != 144 || decoded.Bounds().Dy() != 144 {
			return fmt.Errorf("modern HD map sprite %d frame %d has invalid PNG geometry", asset.SourceGroup, index)
		}
		alpha := modernMapSpriteAlpha(decoded)
		if !alpha.transparent || !alpha.opaque {
			return fmt.Errorf("modern HD map sprite %d frame %d lacks transparent background or opaque core", asset.SourceGroup, index)
		}
		ink := alpha.ink
		if ink.Empty() || ink.Dx() > 132 || ink.Dy() > 132 || ink.Min.X+ink.Max.X < 142 || ink.Min.X+ink.Max.X > 146 {
			return fmt.Errorf("modern HD map sprite %d frame %d violates its proportional cell", asset.SourceGroup, index)
		}
		frames = append(frames, decoded)
	}
	set.mapSpritesHD[asset.SourceGroup] = frames
	return nil
}

func validateModernHDMapSpriteParents(set *modernStoryPortraitSet, assets []modernThemeAsset) error {
	if len(set.mapSpritesHD) == 0 {
		return nil
	}
	visibleGroups := 0
	for group := 0; group < fdicon.NativeSourceGroupCount; group++ {
		if fdicon.NativeSourceGroupHasPixels(group) {
			visibleGroups++
		}
	}
	if len(set.mapSpritesHD) != visibleGroups || len(set.mapSprites) != visibleGroups {
		return errors.New("modern HD map sprite collection or 24px parent collection is incomplete")
	}
	parents := make(map[int]modernThemeAsset)
	children := make(map[int]modernThemeAsset)
	for _, asset := range assets {
		if asset.Role == "map_sprite_set" {
			parents[asset.SourceGroup] = asset
		} else if asset.Role == "map_sprite_hd_set" {
			children[asset.SourceGroup] = asset
		}
	}
	for group := 0; group < fdicon.NativeSourceGroupCount; group++ {
		if !fdicon.NativeSourceGroupHasPixels(group) {
			continue
		}
		parent, parentOK := parents[group]
		child, childOK := children[group]
		if !parentOK || !childOK || parent.FrameCount != child.FrameCount || parent.CyclePolicy != child.CyclePolicy {
			return fmt.Errorf("modern HD map sprite %d does not match its 24px parent", group)
		}
		for i := range child.FrameSHA256 {
			parentInk := modernMapSpriteInk(set.mapSprites[group][i])
			childInk := modernMapSpriteInk(set.mapSpritesHD[group][i])
			if parentInk.Empty() || childInk.Max.Y > parentInk.Max.Y*6 || parentInk.Max.Y*6-childInk.Max.Y > 1 {
				return fmt.Errorf("modern HD map sprite %d frame %d does not preserve its registered feet", group, i)
			}
			for j := 0; j < i; j++ {
				if (child.FrameSHA256[i] == child.FrameSHA256[j]) != (parent.FrameSHA256[i] == parent.FrameSHA256[j]) {
					return fmt.Errorf("modern HD map sprite %d has a different phase repetition at %d/%d", group, i, j)
				}
			}
		}
	}
	return nil
}

func modernMapSpriteInk(source image.Image) image.Rectangle {
	return modernMapSpriteAlpha(source).ink
}

type modernMapSpriteAlphaStats struct {
	ink                 image.Rectangle
	transparent, opaque bool
}

// The PNG decoder's 8-bit RGBA/NRGBA buffers expose alpha directly. This
// preserves the exact 16-bit At().RGBA() predicates without allocating a color
// interface for every pixel. Other image formats retain the reference path.
func modernMapSpriteAlpha(source image.Image) modernMapSpriteAlphaStats {
	bounds := source.Bounds()
	result := modernMapSpriteAlphaStats{}
	minX, minY, maxX, maxY := bounds.Max.X, bounds.Max.Y, bounds.Min.X, bounds.Min.Y
	observe := func(x, y int, alpha uint32) {
		result.transparent = result.transparent || alpha == 0
		result.opaque = result.opaque || alpha == 0xffff
		if alpha != 0 {
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x+1 > maxX {
				maxX = x + 1
			}
			if y+1 > maxY {
				maxY = y + 1
			}
		}
	}
	var pixels []byte
	var stride int
	switch picture := source.(type) {
	case *image.NRGBA:
		pixels, stride = picture.Pix, picture.Stride
	case *image.RGBA:
		pixels, stride = picture.Pix, picture.Stride
	}
	if pixels != nil {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			row := (y - bounds.Min.Y) * stride
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				observe(x, y, uint32(pixels[row+(x-bounds.Min.X)*4+3])*257)
			}
		}
	} else {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, alpha := source.At(x, y).RGBA()
				observe(x, y, alpha)
			}
		}
	}
	if minX < maxX && minY < maxY {
		result.ink = image.Rect(minX, minY, maxX, maxY)
	}
	return result
}
