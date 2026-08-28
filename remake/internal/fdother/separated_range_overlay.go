package fdother

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

type separatedRangeOverlaySource struct {
	File     string `json:"file"`
	Resource int    `json:"resource"`
	Size     int    `json:"size"`
	MD5      string `json:"md5"`
	SHA256   string `json:"sha256"`
	RawSize  int    `json:"raw_size"`
}

type separatedRangeOverlaySprite struct {
	Index     int    `json:"index"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Frame     string `json:"frame"`
	Mask      string `json:"mask"`
	RemapMask string `json:"remap_mask"`
}

type separatedRangeOverlayDocument struct {
	SchemaVersion int                           `json:"schema_version"`
	Kind          string                        `json:"kind"`
	AssetID       string                        `json:"asset_id"`
	Status        string                        `json:"status"`
	Evidence      string                        `json:"evidence"`
	Source        separatedRangeOverlaySource   `json:"source"`
	Sprites       []separatedRangeOverlaySprite `json:"sprites"`
}

// LoadSeparatedRangeOverlayBank loads the complete FDOTHER #1 descriptor
// bank shared by the tactical range overlay and preparation cursor.
func LoadSeparatedRangeOverlayBank(root string) (*fdicon.Bank, error) {
	raw, err := os.ReadFile(filepath.Join(root, "bank.json"))
	if err != nil {
		return nil, fmt.Errorf("fdother: separated range-overlay metadata: %w", err)
	}
	var document separatedRangeOverlayDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("fdother: separated range-overlay JSON: %w", err)
	}
	if document.SchemaVersion != 1 || document.Kind != "fdother_sprite_bank" ||
		document.AssetID != "sprites/FDOTHER_001/range_overlay" ||
		document.Status != "decoded" || document.Evidence != "confirmed" ||
		document.Source.File != "FDOTHER.DAT" || document.Source.Resource != 1 ||
		document.Source.Size != 3382481 ||
		document.Source.MD5 != "22f56e5027edc7c766ad34ca4e5aca93" ||
		document.Source.SHA256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce" ||
		document.Source.RawSize != 2235 || len(document.Sprites) != nativeRangeOverlayTiles {
		return nil, errors.New("fdother: separated range-overlay contract mismatch")
	}
	bank := &fdicon.Bank{Sprites: make([]fdicon.Sprite, nativeRangeOverlayTiles)}
	seen := make([]bool, nativeRangeOverlayTiles)
	for _, entry := range document.Sprites {
		if entry.Index < 0 || entry.Index >= nativeRangeOverlayTiles || seen[entry.Index] ||
			entry.Width != fdicon.NativeSize || entry.Height != fdicon.NativeSize {
			return nil, fmt.Errorf("fdother: invalid separated range-overlay sprite %d", entry.Index)
		}
		prefix := fmt.Sprintf("sprite_%04d/", entry.Index)
		if entry.Frame != prefix+"frame.png" || entry.Mask != prefix+"mask.png" ||
			entry.RemapMask != prefix+"remap_mask.png" {
			return nil, fmt.Errorf("fdother: range-overlay sprite %d path mismatch", entry.Index)
		}
		pixels, err := loadRangeOverlayIndexed(filepath.Join(root, filepath.FromSlash(entry.Frame)))
		if err != nil {
			return nil, fmt.Errorf("fdother: range-overlay sprite %d frame: %w", entry.Index, err)
		}
		mask, err := loadRangeOverlayMask(filepath.Join(root, filepath.FromSlash(entry.Mask)))
		if err != nil {
			return nil, fmt.Errorf("fdother: range-overlay sprite %d mask: %w", entry.Index, err)
		}
		remap, err := loadRangeOverlayMask(filepath.Join(root, filepath.FromSlash(entry.RemapMask)))
		if err != nil {
			return nil, fmt.Errorf("fdother: range-overlay sprite %d remap mask: %w", entry.Index, err)
		}
		bank.Sprites[entry.Index] = fdicon.Sprite{Pixels: pixels, Mask: mask, RemapMask: remap}
		seen[entry.Index] = true
	}
	for index, present := range seen {
		if !present {
			return nil, fmt.Errorf("fdother: range-overlay sprite %d is missing", index)
		}
	}
	return bank, nil
}

func loadRangeOverlayIndexed(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	decoded, _, decodeErr := image.Decode(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return nil, decodeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	paletted, ok := decoded.(*image.Paletted)
	if !ok || paletted.Bounds() != image.Rect(0, 0, fdicon.NativeSize, fdicon.NativeSize) ||
		len(paletted.Palette) != 256 {
		return nil, errors.New("indexed PNG must be 24x24 with 256 palette entries")
	}
	pixels := make([]byte, fdicon.NativeSize*fdicon.NativeSize)
	for row := 0; row < fdicon.NativeSize; row++ {
		copy(pixels[row*fdicon.NativeSize:(row+1)*fdicon.NativeSize],
			paletted.Pix[row*paletted.Stride:row*paletted.Stride+fdicon.NativeSize])
	}
	return pixels, nil
}

func loadRangeOverlayMask(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	decoded, _, decodeErr := image.Decode(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return nil, decodeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	gray, ok := decoded.(*image.Gray)
	if !ok || gray.Bounds() != image.Rect(0, 0, fdicon.NativeSize, fdicon.NativeSize) {
		return nil, errors.New("mask PNG must be 24x24 grayscale")
	}
	mask := make([]byte, fdicon.NativeSize*fdicon.NativeSize)
	for row := 0; row < fdicon.NativeSize; row++ {
		for col := 0; col < fdicon.NativeSize; col++ {
			value := gray.GrayAt(col, row).Y
			if value != 0 && value != 0xff {
				return nil, errors.New("mask PNG contains a non-binary value")
			}
			if value != 0 {
				mask[row*fdicon.NativeSize+col] = 1
			}
		}
	}
	return mask, nil
}
