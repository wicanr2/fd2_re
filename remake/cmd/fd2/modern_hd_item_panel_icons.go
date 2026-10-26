package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

var modernUIPanelIconIndices = []int{53, 54, 55, 56, 57, 59, 60, 61, 62, 63}

const modernUIPanelIconMetadataSHA = "a547f2f593b980fd9c1b269d1ba95be8727562d7ed31b44015615775d991efd3"

type modernUIPanelIcon struct {
	native  modernLMIHDDescriptor
	picture *image.RGBA
	digest  [32]byte
}

type modernHDItemPanelIcons struct {
	seal         [32]byte
	upper, items *modernHDItemPanelValues
}

func modernUIRawDescription(index int, cell fdother.RawCell) modernLMIHDDescriptor {
	digest := sha256.Sum256(cell.Pixels)
	return modernLMIHDDescriptor{Index: index, Width: cell.Width, Height: cell.Height, PixelsSHA256: hex.EncodeToString(digest[:])}
}

func modernUIPanelIconSize(index int) image.Point {
	for _, supported := range modernUIPanelIconIndices {
		if index == supported {
			if index == 53 || index == 54 {
				return image.Pt(29, 24)
			}
			return image.Pt(24, 20)
		}
	}
	return image.Point{}
}

func loadModernUIPanelIcon(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	d := asset.NativeRawCell
	size := modernUIPanelIconSize(d.Index)
	if set == nil || size == (image.Point{}) || asset.Role != "native_ui_icon_hd" ||
		asset.SourceArchive != "FDOTHER.DAT" || asset.SourceResource != 5 || asset.SourceMetadataSHA256 != modernUIPanelIconMetadataSHA ||
		asset.ConsumerContract != "native_raw_ui_icon_hd6_v1" || asset.LayoutPolicy != "native_raw_cell_rect_hd6" || asset.AlphaContract != "opaque" ||
		(asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") ||
		d.Width != size.X || d.Height != size.Y || asset.Width != size.X*6 || asset.Height != size.Y*6 ||
		len(asset.SourceFiles) != 1 || len(asset.SourceSHA256) != 1 || asset.SourceFiles[0] != asset.MasterFile {
		return errors.New("modern HD panel icon: invalid contract")
	}
	if set.uiPanelIcons[d.Index] != nil {
		return errors.New("modern HD panel icon: duplicate index")
	}
	if filepath.Base(asset.File) != asset.File || filepath.Ext(asset.File) != ".png" ||
		filepath.Base(asset.MasterFile) != asset.MasterFile || filepath.Ext(asset.MasterFile) != ".png" {
		return errors.New("modern HD panel icon: unsafe path")
	}
	if h, err := hex.DecodeString(asset.SourceSHA256[0]); err != nil || len(h) != 32 {
		return errors.New("modern HD panel icon: invalid master provenance")
	}
	dir := separatedAssetPath("ui/fdother_005_item_panel")
	metadata, err := os.ReadFile(filepath.Join(dir, "resource.json"))
	if err != nil {
		return err
	}
	hash := sha256.Sum256(metadata)
	if hex.EncodeToString(hash[:]) != modernUIPanelIconMetadataSHA {
		return errors.New("modern HD panel icon: original metadata differs")
	}
	entries, err := fdother.LoadSeparatedItemPanelEntries(filepath.Dir(dir))
	if err != nil {
		return err
	}
	cell, ok := entries.Raw[d.Index]
	if !ok || d != modernUIRawDescription(d.Index, cell) {
		return errors.New("modern HD panel icon: original raw cell differs")
	}
	raw, err := os.ReadFile(filepath.Join(root, asset.File))
	if err != nil {
		return err
	}
	hash = sha256.Sum256(raw)
	if hex.EncodeToString(hash[:]) != asset.SHA256 {
		return errors.New("modern HD panel icon: PNG sha256 differs")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width != asset.Width || config.Height != asset.Height {
		return errors.New("modern HD panel icon: PNG geometry differs")
	}
	pic, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	rgba := image.NewRGBA(image.Rect(0, 0, asset.Width, asset.Height))
	draw.Draw(rgba, rgba.Bounds(), pic, pic.Bounds().Min, draw.Src)
	for at := 3; at < len(rgba.Pix); at += 4 {
		if rgba.Pix[at] != 255 {
			return errors.New("modern HD panel icon: PNG is not opaque")
		}
	}
	if set.uiPanelIcons == nil {
		set.uiPanelIcons = make(map[int]*modernUIPanelIcon)
	}
	set.uiPanelIcons[d.Index] = &modernUIPanelIcon{native: d, picture: rgba, digest: sha256.Sum256(rgba.Pix)}
	return nil
}

func prepareModernHDItemPanelIcons(d *nativeUnitStatusData, palette color.Palette, set *modernStoryPortraitSet, values *modernHDItemPanelValues, rows *modernHDItemRowLabels) (*modernHDItemPanelIcons, error) {
	if set == nil || len(set.uiPanelIcons) == 0 {
		return nil, nil
	}
	if d == nil || len(d.record) != 80 || len(set.uiPanelIcons) != 10 || values == nil || values.picture == nil || rows == nil ||
		values.picture.Bounds() != image.Rect(0, 0, 223*6, 86*6) || rows.items != nil && (rows.items.picture == nil || rows.items.picture.Bounds() != image.Rect(0, 0, 310*6, 99*6)) {
		return nil, errors.New("modern HD panel icons: source unavailable")
	}
	var sources []any
	for _, index := range modernUIPanelIconIndices {
		icon := set.uiPanelIcons[index]
		cell := d.assets.RawCells[index]
		if icon == nil || icon.picture == nil || icon.native != modernUIRawDescription(index, cell) ||
			icon.picture.Bounds() != image.Rect(0, 0, cell.Width*6, cell.Height*6) || sha256.Sum256(icon.picture.Pix) != icon.digest {
			return nil, fmt.Errorf("modern HD panel icons: source %d differs", index)
		}
		sources = append(sources, icon.native, sha256.Sum256(icon.picture.Pix))
	}
	raw, _ := json.Marshal([]any{d.record, d.itemRows, palette, values.seal, rows.seal, sources})
	seal := sha256.Sum256(raw)
	if d.itemIcons != nil {
		if d.itemIcons.seal != seal {
			return nil, errors.New("modern HD panel icons: frozen source changed")
		}
		return d.itemIcons, nil
	}
	copyArt := func(old *modernHDItemPanelValues, width, height int) *image.RGBA {
		pic := image.NewRGBA(image.Rect(0, 0, width*6, height*6))
		if old != nil {
			draw.Draw(pic, pic.Bounds(), old.picture, old.picture.Bounds().Min, draw.Src)
		}
		return pic
	}
	upper := copyArt(values, 223, 86)
	items := copyArt(rows.items, 310, 99)
	paint := func(pic *image.RGBA, origin image.Point, index, x, y int) error {
		icon := set.uiPanelIcons[index]
		if icon == nil {
			return nil // Unregistered indices retain the native producer.
		}
		at := image.Pt((x-origin.X)*6, (y-origin.Y)*6)
		rect := icon.picture.Bounds().Add(at)
		if !rect.In(pic.Bounds()) {
			return errors.New("modern HD panel icons: placement outside original panel")
		}
		draw.Draw(pic, rect, icon.picture, image.Point{}, draw.Src)
		return nil
	}
	plan := battle.NativeItemPanelDataPlanFor()
	for _, call := range append([]battle.NativeItemPanelIconCall{plan.BaseIcon}, plan.FlagIcons...) {
		if call.DrawWhenFlagNonzero && d.record[call.FlagOffset] == 0 {
			continue
		}
		index := call.FDOTHEREntry
		if call.UseAlternateWhenFlagZero && d.record[call.FlagOffset] == 0 {
			index = call.AlternateFDOTHEREntry
		}
		if err := paint(upper, image.Pt(92, 7), index, call.Destination.X, call.Destination.Y); err != nil {
			return nil, err
		}
	}
	cells, err := battle.NativeItemSelectorCells(d.record, 0, -1, d.itemRows)
	if err != nil {
		return nil, err
	}
	for _, cell := range cells {
		if err := paint(items, image.Pt(5, 94), cell.CategoryIcon, cell.LabelX-29, cell.LabelY-2); err != nil {
			return nil, err
		}
	}
	return &modernHDItemPanelIcons{seal: seal, upper: &modernHDItemPanelValues{seal: seal, picture: upper}, items: &modernHDItemPanelValues{seal: seal, picture: items}}, nil
}
