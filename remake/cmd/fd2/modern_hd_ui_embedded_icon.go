package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type modernUIEmbeddedIcon struct {
	native  modernLMIHDDescriptor
	source  image.Rectangle
	picture *image.RGBA
	digest  [32]byte
}

func preparationCountIconDescriptor() modernLMIHDDescriptor {
	return modernLMIHDDescriptor{Index: 137, Width: 86, Height: 86, PixelsSHA256: preparationCountFrameSHA}
}

func loadModernUIEmbeddedIcon(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	d := preparationCountIconDescriptor()
	if set == nil || asset.Role != "native_ui_embedded_icon_hd" ||
		asset.Status != "runtime_candidate" && asset.Status != "runtime_ready" ||
		asset.SourceArchive != "FDOTHER.DAT" || asset.SourceResource != 5 ||
		asset.SourceMetadataSHA256 != modernUIPanelIconMetadataSHA ||
		asset.ConsumerContract != "native_embedded_ui_icon_hd6_v1" ||
		asset.LayoutPolicy != "native_frame_source_rect_hd6" || asset.AlphaContract != "true_alpha" ||
		asset.Width != 144 || asset.Height != 108 || len(asset.NativeLMIFrames) != 1 || asset.NativeLMIFrames[0] != d ||
		len(asset.NativeSourceRect) != 4 || asset.NativeSourceRect[0] != 20 || asset.NativeSourceRect[1] != 22 ||
		asset.NativeSourceRect[2] != 44 || asset.NativeSourceRect[3] != 40 {
		return errors.New("modern HD embedded UI icon: unproved source contract")
	}
	if set.uiEmbeddedIcons[137] != nil {
		return errors.New("modern HD embedded UI icon: duplicate source")
	}
	if filepath.Base(asset.File) != asset.File || filepath.Ext(asset.File) != ".png" ||
		filepath.Base(asset.MasterFile) != asset.MasterFile || filepath.Ext(asset.MasterFile) != ".png" ||
		len(asset.SourceFiles) != 1 || asset.SourceFiles[0] != asset.MasterFile || len(asset.SourceSHA256) != 1 {
		return errors.New("modern HD embedded UI icon: unsafe PNG provenance")
	}
	if h, err := hex.DecodeString(asset.SourceSHA256[0]); err != nil || len(h) != 32 {
		return errors.New("modern HD embedded UI icon: invalid master provenance")
	}
	dir := separatedAssetPath("ui/fdother_005_item_panel")
	metadata, err := os.ReadFile(filepath.Join(dir, "resource.json"))
	if err != nil {
		return err
	}
	hash := sha256.Sum256(metadata)
	if hex.EncodeToString(hash[:]) != modernUIPanelIconMetadataSHA {
		return errors.New("modern HD embedded UI icon: original metadata differs")
	}
	entries, err := fdother.LoadSeparatedItemPanelEntries(filepath.Dir(dir))
	if err != nil {
		return err
	}
	if !preparationCountSourceMatches(entries.Frames[137]) {
		return errors.New("modern HD embedded UI icon: original frame or mask differs")
	}
	raw, err := os.ReadFile(filepath.Join(root, asset.File))
	if err != nil {
		return err
	}
	hash = sha256.Sum256(raw)
	if hex.EncodeToString(hash[:]) != asset.SHA256 {
		return errors.New("modern HD embedded UI icon: PNG SHA differs")
	}
	config, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width != 144 || config.Height != 108 {
		return errors.New("modern HD embedded UI icon: PNG geometry differs")
	}
	pic, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	rgba := image.NewRGBA(image.Rect(0, 0, 144, 108))
	draw.Draw(rgba, rgba.Bounds(), pic, image.Point{}, draw.Src)
	min, max := byte(255), byte(0)
	for at := 3; at < len(rgba.Pix); at += 4 {
		v := rgba.Pix[at]
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if min != 0 || max != 255 {
		return errors.New("modern HD embedded UI icon: real alpha unavailable")
	}
	if set.uiEmbeddedIcons == nil {
		set.uiEmbeddedIcons = make(map[int]*modernUIEmbeddedIcon)
	}
	set.uiEmbeddedIcons[137] = &modernUIEmbeddedIcon{native: d, source: image.Rect(20, 22, 44, 40), picture: rgba, digest: sha256.Sum256(rgba.Pix)}
	return nil
}

// Missing optional artwork retains the original count symbol. A changed
// registered source must never reuse the previous material from the cache.
func modernPreparationCountIcon(ui *nativePreparationUIAssets, set *modernStoryPortraitSet) (*modernUIEmbeddedIcon, error) {
	if set == nil || set.uiEmbeddedIcons[137] == nil {
		return nil, nil
	}
	icon := set.uiEmbeddedIcons[137]
	if ui == nil || ui.roster == nil || !preparationCountSourceMatches(ui.roster.UpperLeft) ||
		icon.native != preparationCountIconDescriptor() || icon.source != image.Rect(20, 22, 44, 40) ||
		icon.picture == nil || icon.picture.Bounds() != image.Rect(0, 0, 144, 108) || sha256.Sum256(icon.picture.Pix) != icon.digest {
		return nil, errors.New("modern HD embedded UI icon: changed count-symbol source")
	}
	return icon, nil
}
