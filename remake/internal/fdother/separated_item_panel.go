package fdother

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

const (
	itemPanelSourceSize   = 3382481
	itemPanelSourceMD5    = "22f56e5027edc7c766ad34ca4e5aca93"
	itemPanelSourceSHA256 = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
	itemPanelResourceSize = 44181
)

var (
	itemPanelOpaqueEntries = []int{22}
	itemPanelRawEntries    = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 23, 24, 25, 26, 27, 28, 29, 30, 53, 54, 55, 56, 57, 59, 60, 61, 62, 63, 64, 65, 66, 67, 92}
	itemPanelFrameEntries  = []int{31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 93, 119, 120, 121, 122, 123, 124, 125, 126, 127, 128, 129}
)

type separatedItemPanelSource struct {
	File     string `json:"file"`
	Resource int    `json:"resource"`
	Size     int    `json:"size"`
	MD5      string `json:"md5"`
	SHA256   string `json:"sha256"`
	RawSize  int    `json:"raw_size"`
}

type separatedItemPanelEntry struct {
	Index  int    `json:"index"`
	Codec  string `json:"codec"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Frame  string `json:"frame"`
	Mask   string `json:"mask,omitempty"`
}

type separatedItemPanelDocument struct {
	SchemaVersion int                       `json:"schema_version"`
	Kind          string                    `json:"kind"`
	AssetID       string                    `json:"asset_id"`
	Status        string                    `json:"status"`
	Evidence      string                    `json:"evidence"`
	Source        separatedItemPanelSource  `json:"source"`
	Entries       []separatedItemPanelEntry `json:"entries"`
}

// SeparatedItemPanelEntries is the exact mixed-codec subset consumed by the
// shared item/status/transfer renderer.
type SeparatedItemPanelEntries struct {
	Opaque map[int]LMI1Entry
	Raw    map[int]RawCell
	Frames map[int]Frame
}

func expectedItemPanelCodec(index int) string {
	for _, value := range itemPanelOpaqueEntries {
		if index == value {
			return "opaque_high_run"
		}
	}
	for _, value := range itemPanelRawEntries {
		if index == value {
			return "raw_indexed_opaque"
		}
	}
	for _, value := range itemPanelFrameEntries {
		if index == value {
			return "four_mode_frame"
		}
	}
	return ""
}

func LoadSeparatedItemPanelEntries(root string) (SeparatedItemPanelEntries, error) {
	directory := filepath.Join(root, "fdother_005_item_panel")
	raw, err := os.ReadFile(filepath.Join(directory, "resource.json"))
	if err != nil {
		return SeparatedItemPanelEntries{}, fmt.Errorf("fdother: separated item panel metadata: %w", err)
	}
	var document separatedItemPanelDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return SeparatedItemPanelEntries{}, fmt.Errorf("fdother: separated item panel JSON: %w", err)
	}
	wantCount := len(itemPanelOpaqueEntries) + len(itemPanelRawEntries) + len(itemPanelFrameEntries)
	if document.SchemaVersion != 1 || document.Kind != "fdother_lmi1_item_panel" ||
		document.AssetID != "ui/FDOTHER_005/item_panel" || document.Status != "decoded" ||
		document.Evidence != "confirmed" || document.Source.File != "FDOTHER.DAT" ||
		document.Source.Resource != 5 || document.Source.Size != itemPanelSourceSize ||
		document.Source.MD5 != itemPanelSourceMD5 || document.Source.SHA256 != itemPanelSourceSHA256 ||
		document.Source.RawSize != itemPanelResourceSize || len(document.Entries) != wantCount {
		return SeparatedItemPanelEntries{}, fmt.Errorf("fdother: separated item panel metadata mismatch")
	}
	result := SeparatedItemPanelEntries{Opaque: make(map[int]LMI1Entry), Raw: make(map[int]RawCell), Frames: make(map[int]Frame)}
	seen := make(map[int]bool, wantCount)
	for _, entry := range document.Entries {
		codec := expectedItemPanelCodec(entry.Index)
		wantFrame := fmt.Sprintf("entry_%03d/frame.png", entry.Index)
		if codec == "" || seen[entry.Index] || entry.Codec != codec || entry.Width <= 0 || entry.Height <= 0 || entry.Frame != wantFrame {
			return SeparatedItemPanelEntries{}, fmt.Errorf("fdother: separated item panel entry %d metadata mismatch", entry.Index)
		}
		seen[entry.Index] = true
		pixels, err := loadItemPanelIndexedPNG(filepath.Join(directory, entry.Frame), entry.Width, entry.Height)
		if err != nil {
			return SeparatedItemPanelEntries{}, fmt.Errorf("fdother: separated item panel entry %d: %w", entry.Index, err)
		}
		switch codec {
		case "opaque_high_run":
			if entry.Mask != "" {
				return SeparatedItemPanelEntries{}, fmt.Errorf("fdother: opaque item panel entry %d has a mask", entry.Index)
			}
			result.Opaque[entry.Index] = LMI1Entry{Width: entry.Width, Height: entry.Height, Pixels: pixels}
		case "raw_indexed_opaque":
			if entry.Mask != "" {
				return SeparatedItemPanelEntries{}, fmt.Errorf("fdother: raw item panel entry %d has a mask", entry.Index)
			}
			result.Raw[entry.Index] = RawCell{Width: entry.Width, Height: entry.Height, Pixels: pixels}
		case "four_mode_frame":
			wantMask := fmt.Sprintf("entry_%03d/mask.png", entry.Index)
			if entry.Mask != wantMask {
				return SeparatedItemPanelEntries{}, fmt.Errorf("fdother: frame item panel entry %d lacks its mask", entry.Index)
			}
			mask, err := loadItemPanelMaskPNG(filepath.Join(directory, entry.Mask), entry.Width, entry.Height)
			if err != nil {
				return SeparatedItemPanelEntries{}, fmt.Errorf("fdother: separated item panel entry %d mask: %w", entry.Index, err)
			}
			result.Frames[entry.Index] = Frame{Width: entry.Width, Height: entry.Height, Indexed: pixels, Mask: mask}
		}
	}
	return result, nil
}

func loadItemPanelIndexedPNG(path string, width, height int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	decoded, decodeErr := png.Decode(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return nil, decodeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	paletted, ok := decoded.(*image.Paletted)
	if !ok || paletted.Rect.Dx() != width || paletted.Rect.Dy() != height || paletted.Stride != width {
		return nil, fmt.Errorf("indexed PNG mode or geometry mismatch")
	}
	return append([]byte(nil), paletted.Pix...), nil
}

func loadItemPanelMaskPNG(path string, width, height int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	decoded, decodeErr := png.Decode(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return nil, decodeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	gray, ok := decoded.(*image.Gray)
	if !ok || gray.Rect.Dx() != width || gray.Rect.Dy() != height || gray.Stride != width {
		return nil, fmt.Errorf("mask PNG mode or geometry mismatch")
	}
	mask := append([]byte(nil), gray.Pix...)
	for _, value := range mask {
		if value != 0 && value != 0xff {
			return nil, fmt.Errorf("mask PNG is not binary")
		}
	}
	return mask, nil
}
