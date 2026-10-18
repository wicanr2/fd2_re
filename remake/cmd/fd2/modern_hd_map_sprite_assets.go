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
		asset.SourceGroup < 0 || asset.SourceGroup >= 96 || asset.SourceGroup == 39 || asset.SourceGroup == 57 {
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
		transparent, opaque := false, false
		bounds := decoded.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, alpha := decoded.At(x, y).RGBA()
				transparent = transparent || alpha == 0
				opaque = opaque || alpha == 0xffff
			}
		}
		if !transparent || !opaque {
			return fmt.Errorf("modern HD map sprite %d frame %d lacks transparent background or opaque core", asset.SourceGroup, index)
		}
		ink := modernMapSpriteInk(decoded)
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
	if len(set.mapSpritesHD) != 94 || len(set.mapSprites) != 94 {
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
	for group := 0; group < 96; group++ {
		if group == 39 || group == 57 {
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
	bounds := source.Bounds()
	ink := image.Rectangle{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, alpha := source.At(x, y).RGBA()
			if alpha != 0 {
				pixel := image.Rect(x, y, x+1, y+1)
				ink = ink.Union(pixel)
			}
		}
	}
	return ink
}
