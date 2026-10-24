package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type modernLMIHDDescriptor struct {
	Index        int    `json:"index"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	PixelsSHA256 string `json:"pixels_sha256"`
}

type modernMapOverlayHDResource struct {
	frames map[int]*image.RGBA
	native map[int]modernLMIHDDescriptor
}

func modernLMIHDDescription(index int, entry fdother.LMI1Entry) modernLMIHDDescriptor {
	digest := sha256.Sum256(entry.Pixels)
	return modernLMIHDDescriptor{Index: index, Width: entry.Width, Height: entry.Height, PixelsSHA256: hex.EncodeToString(digest[:])}
}

func loadModernMapOverlayHDAsset(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	dir := separatedAssetPath("ui/fdother_005_lmi1_opaque")
	if asset.SourceResource == 6 {
		dir = separatedAssetPath("effects/fdother_006_lmi1_opaque")
	} else if asset.SourceResource != 5 {
		return errors.New("modern HD map overlay: unproved source resource")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "bank.json"))
	if err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != asset.SourceMetadataSHA256 {
		return errors.New("modern HD map overlay: source metadata sha256 mismatch")
	}
	native, err := fdother.LoadSeparatedLMI1Bank(dir, asset.SourceResource)
	if err != nil {
		return err
	}
	return loadModernMapOverlayHDWithNative(set, asset, root, native)
}

// Decode every declared source before publishing the optional bank. Legacy
// themes without the role retain the original indexed presentation.
func modernMapOverlayHDIndices(resource, count int) []int {
	var indices []int
	switch {
	case resource == 5 && count == 12:
		for i := 68; i <= 79; i++ {
			indices = append(indices, i)
		}
	case resource == 5 && count == 13:
		for i := 94; i <= 103; i++ {
			indices = append(indices, i)
		}
		indices = append(indices, 116, 117, 118)
	case resource == 6 && count == 27:
		for i := 87; i <= 113; i++ {
			indices = append(indices, i)
		}
	case resource == 5 && count == 10:
		for i := 105; i <= 114; i++ {
			indices = append(indices, i)
		}
	case resource == 6 && count == 7:
		for i := 57; i <= 63; i++ {
			indices = append(indices, i)
		}
	case resource == 6 && count == 12:
		for i := 64; i <= 75; i++ {
			indices = append(indices, i)
		}
	case resource == 6 && count == 65:
		for i := 126; i <= 190; i++ {
			indices = append(indices, i)
		}
	case resource == 6 && count == 39:
		for i := 191; i <= 229; i++ {
			indices = append(indices, i)
		}
	case resource == 6 && count == 11:
		for i := 114; i <= 124; i++ {
			indices = append(indices, i)
		}
	}
	return indices
}

func hasModernHDDeathBank(set *modernStoryPortraitSet) bool {
	if set == nil || set.mapOverlays == nil {
		return false
	}
	for index := 68; index <= 79; index++ {
		_, declared := set.mapOverlays.native[index]
		if declared || set.mapOverlays.frames[index] != nil {
			return true
		}
	}
	return false
}

func loadModernMapOverlayHDWithNative(set *modernStoryPortraitSet, asset modernThemeAsset, root string, native []fdother.LMI1Entry) error {
	indices := modernMapOverlayHDIndices(asset.SourceResource, asset.FrameCount)
	if set == nil || len(indices) == 0 || asset.Role != "map_overlay_hd_set" ||
		asset.SourceArchive != "FDOTHER.DAT" ||
		asset.ConsumerContract != "native_lmi1_rgba_hd6_v1" || asset.AlphaContract != "coverage" ||
		asset.CyclePolicy != "native_lmi1_entries" || asset.LayoutPolicy != "native_lmi1_rect_hd6" ||
		asset.Width != modernHDWidth || asset.Height != modernHDHeight ||
		(asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") ||
		len(asset.Files) != len(indices) || len(asset.FrameSHA256) != len(indices) || len(asset.NativeLMIFrames) != len(indices) {
		return errors.New("modern HD map overlay: invalid bank contract")
	}
	if asset.SourceResource == 5 && asset.FrameCount == 12 {
		if err := fdother.ValidateNativeCh28PostLMI(native); err != nil {
			return err
		}
	}
	bank := &modernMapOverlayHDResource{frames: make(map[int]*image.RGBA), native: make(map[int]modernLMIHDDescriptor)}
	previous := set.mapOverlays
	if asset.SourceResource == 6 {
		previous = set.mapEffects
	}
	if previous != nil {
		for index, picture := range previous.frames {
			bank.frames[index] = picture
		}
		for index, descriptor := range previous.native {
			bank.native[index] = descriptor
		}
	}
	for i, descriptor := range asset.NativeLMIFrames {
		index := indices[i]
		if index >= len(native) || bank.frames[index] != nil ||
			descriptor != modernLMIHDDescription(index, native[index]) ||
			descriptor.Width < 1 || descriptor.Height < 1 || descriptor.Width > 128 || descriptor.Height > 128 {
			return fmt.Errorf("modern HD map overlay: native entry %d mismatch", index)
		}
		name := asset.Files[i]
		if filepath.Base(name) != name || filepath.Ext(name) != ".png" {
			return errors.New("modern HD map overlay: unsafe PNG path")
		}
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) != asset.FrameSHA256[i] {
			return fmt.Errorf("modern HD map overlay: entry %d sha256 mismatch", index)
		}
		config, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil || config.Width != descriptor.Width*6 || config.Height != descriptor.Height*6 {
			return fmt.Errorf("modern HD map overlay: entry %d geometry mismatch", index)
		}
		decoded, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return err
		}
		picture := image.NewRGBA(decoded.Bounds())
		draw.Draw(picture, picture.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
		transparent, visible := false, false
		for y := 0; y < picture.Bounds().Dy(); y++ {
			for x := 0; x < picture.Bounds().Dx(); x++ {
				a := picture.RGBAAt(x, y).A
				transparent = transparent || a == 0
				visible = visible || a != 0
			}
		}
		if !transparent || !visible {
			return fmt.Errorf("modern HD map overlay: entry %d missing transparency or ink", index)
		}
		bank.frames[index], bank.native[index] = picture, descriptor
	}
	if asset.SourceResource == 6 {
		set.mapEffects = bank
	} else {
		set.mapOverlays = bank
	}
	return nil
}
