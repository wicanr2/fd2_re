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
	raw, err := os.ReadFile(filepath.Join(dir, "bank.json"))
	if err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != asset.SourceMetadataSHA256 {
		return errors.New("modern HD map overlay: source metadata sha256 mismatch")
	}
	native, err := fdother.LoadSeparatedLMI1Bank(dir, 5)
	if err != nil {
		return err
	}
	return loadModernMapOverlayHDWithNative(set, asset, root, native)
}

// Decode every declared source before publishing the optional bank. Legacy
// themes without the role retain the original indexed presentation.
func loadModernMapOverlayHDWithNative(set *modernStoryPortraitSet, asset modernThemeAsset, root string, native []fdother.LMI1Entry) error {
	if set == nil || set.mapOverlays != nil || asset.Role != "map_overlay_hd_set" ||
		asset.SourceArchive != "FDOTHER.DAT" || asset.SourceResource != 5 ||
		asset.ConsumerContract != "native_lmi1_rgba_hd6_v1" || asset.AlphaContract != "coverage" ||
		asset.CyclePolicy != "native_lmi1_entries" || asset.LayoutPolicy != "native_lmi1_rect_hd6" ||
		asset.Width != modernHDWidth || asset.Height != modernHDHeight ||
		(asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") ||
		asset.FrameCount != 12 || len(asset.Files) != 12 || len(asset.FrameSHA256) != 12 || len(asset.NativeLMIFrames) != 12 {
		return errors.New("modern HD map overlay: invalid bank contract")
	}
	if err := fdother.ValidateNativeCh28PostLMI(native); err != nil {
		return err
	}
	bank := &modernMapOverlayHDResource{frames: make(map[int]*image.RGBA), native: make(map[int]modernLMIHDDescriptor)}
	for i, descriptor := range asset.NativeLMIFrames {
		index := 68 + i
		if descriptor != modernLMIHDDescription(index, native[index]) || descriptor.Width > 128 || descriptor.Height > 128 {
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
	set.mapOverlays = bank
	return nil
}
