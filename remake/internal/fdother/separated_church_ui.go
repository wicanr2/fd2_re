package fdother

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	churchUISourceSize   = 3382481
	churchUISourceMD5    = "22f56e5027edc7c766ad34ca4e5aca93"
	churchUISourceSHA256 = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
	churchUIResourceSize = 51157
)

var (
	churchUIOpaqueEntries = []int{1, 3, 4, 5, 6, 7, 8, 9, 10, 16}
	churchUIRawEntries    = []int{15}
	churchUIFrameEntries  = []int{0, 23, 24, 25, 26, 27, 28, 29, 30, 31}
)

// SeparatedChurchUIAssets is the exact FDOTHER #14 subset consumed by the
// church menu, roster, class-change and revive presentation paths.
type SeparatedChurchUIAssets struct {
	Background Frame
	Entries    []LMI1Entry
	PriceCell  RawCell
	ReviveFX   []Frame
}

func expectedChurchUICodec(index int) string {
	for _, value := range churchUIOpaqueEntries {
		if index == value {
			return "opaque_high_run"
		}
	}
	for _, value := range churchUIRawEntries {
		if index == value {
			return "raw_indexed_opaque"
		}
	}
	for _, value := range churchUIFrameEntries {
		if index == value {
			return "four_mode_frame"
		}
	}
	return ""
}

func LoadSeparatedChurchUIAssets(root string) (SeparatedChurchUIAssets, error) {
	directory := filepath.Join(root, "fdother_014_church")
	raw, err := os.ReadFile(filepath.Join(directory, "resource.json"))
	if err != nil {
		return SeparatedChurchUIAssets{}, fmt.Errorf("fdother: separated church UI metadata: %w", err)
	}
	var document separatedItemPanelDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return SeparatedChurchUIAssets{}, fmt.Errorf("fdother: separated church UI JSON: %w", err)
	}
	wantCount := len(churchUIOpaqueEntries) + len(churchUIRawEntries) + len(churchUIFrameEntries)
	if document.SchemaVersion != 1 || document.Kind != "fdother_lmi1_church_ui" ||
		document.AssetID != "ui/FDOTHER_014/church" || document.Status != "decoded" ||
		document.Evidence != "confirmed" || document.Source.File != "FDOTHER.DAT" ||
		document.Source.Resource != 14 || document.Source.Size != churchUISourceSize ||
		document.Source.MD5 != churchUISourceMD5 || document.Source.SHA256 != churchUISourceSHA256 ||
		document.Source.RawSize != churchUIResourceSize || len(document.Entries) != wantCount {
		return SeparatedChurchUIAssets{}, fmt.Errorf("fdother: separated church UI metadata mismatch")
	}
	result := SeparatedChurchUIAssets{
		Entries:  make([]LMI1Entry, 17),
		ReviveFX: make([]Frame, 9),
	}
	seen := make(map[int]bool, wantCount)
	for _, entry := range document.Entries {
		codec := expectedChurchUICodec(entry.Index)
		wantFrame := fmt.Sprintf("entry_%03d/frame.png", entry.Index)
		if codec == "" || seen[entry.Index] || entry.Codec != codec ||
			entry.Width <= 0 || entry.Height <= 0 || entry.Frame != wantFrame {
			return SeparatedChurchUIAssets{}, fmt.Errorf(
				"fdother: separated church UI entry %d metadata mismatch", entry.Index,
			)
		}
		seen[entry.Index] = true
		pixels, err := loadItemPanelIndexedPNG(
			filepath.Join(directory, entry.Frame), entry.Width, entry.Height,
		)
		if err != nil {
			return SeparatedChurchUIAssets{}, fmt.Errorf(
				"fdother: separated church UI entry %d: %w", entry.Index, err,
			)
		}
		switch codec {
		case "opaque_high_run":
			if entry.Mask != "" || entry.Index >= len(result.Entries) {
				return SeparatedChurchUIAssets{}, fmt.Errorf("fdother: invalid opaque church UI entry %d", entry.Index)
			}
			result.Entries[entry.Index] = LMI1Entry{
				Width: entry.Width, Height: entry.Height, Pixels: pixels,
			}
		case "raw_indexed_opaque":
			if entry.Mask != "" || entry.Index != 15 {
				return SeparatedChurchUIAssets{}, fmt.Errorf("fdother: invalid raw church UI entry %d", entry.Index)
			}
			result.PriceCell = RawCell{Width: entry.Width, Height: entry.Height, Pixels: pixels}
		case "four_mode_frame":
			wantMask := fmt.Sprintf("entry_%03d/mask.png", entry.Index)
			if entry.Mask != wantMask {
				return SeparatedChurchUIAssets{}, fmt.Errorf("fdother: church UI frame %d lacks mask", entry.Index)
			}
			mask, err := loadItemPanelMaskPNG(
				filepath.Join(directory, entry.Mask), entry.Width, entry.Height,
			)
			if err != nil {
				return SeparatedChurchUIAssets{}, fmt.Errorf("fdother: church UI frame %d mask: %w", entry.Index, err)
			}
			frame := Frame{Width: entry.Width, Height: entry.Height, Indexed: pixels, Mask: mask}
			if entry.Index == 0 {
				if entry.Width != 320 || entry.Height != 200 {
					return SeparatedChurchUIAssets{}, fmt.Errorf("fdother: church UI background is not 320x200")
				}
				result.Background = frame
			} else {
				result.ReviveFX[entry.Index-23] = frame
			}
		}
	}
	for _, index := range churchUIOpaqueEntries {
		if !seen[index] {
			return SeparatedChurchUIAssets{}, fmt.Errorf("fdother: missing church UI entry %d", index)
		}
	}
	return result, nil
}
