package fdicon

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
)

const (
	SeparatedSpriteCount = 1680
	fdiconSourceSize     = 624010
	fdiconSourceMD5      = "46f793540209a063ea73a5373ca14bf4"
	fdiconSourceSHA256   = "7efb4448d05f19c1e17ebd53f3e3afead235f5c008d5167548d834c3686b1e44"
)

type separatedSource struct {
	File   string `json:"file"`
	Size   int    `json:"size"`
	MD5    string `json:"md5"`
	SHA256 string `json:"sha256"`
}

type separatedSprite struct {
	Index     int    `json:"index"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Frame     string `json:"frame"`
	Mask      string `json:"mask"`
	RemapMask string `json:"remap_mask"`
}

type separatedBankDocument struct {
	SchemaVersion int               `json:"schema_version"`
	Kind          string            `json:"kind"`
	AssetID       string            `json:"asset_id"`
	Status        string            `json:"status"`
	Evidence      string            `json:"evidence"`
	Source        separatedSource   `json:"source"`
	Sprites       []separatedSprite `json:"sprites"`
}

// LoadSeparatedBank loads the complete FDICON.B24 sprite bank from standard
// PNG layers. It deliberately has no archive fallback: all 1680 sprites and
// their source-write and destination-remap masks must validate first.
func LoadSeparatedBank(root string) (*Bank, error) {
	if root == "" {
		return nil, errors.New("fdicon: separated bank root is empty")
	}
	metadataPath := filepath.Join(root, "bank.json")
	raw, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("fdicon: separated bank metadata: %w", err)
	}
	var document separatedBankDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("fdicon: separated bank metadata: %w", err)
	}
	if document.SchemaVersion != 1 || document.Kind != "fdicon_sprite_bank" ||
		document.AssetID != "sprites/fdicon" || document.Status != "decoded" ||
		document.Evidence != "confirmed" || document.Source.File != "FDICON.B24" ||
		document.Source.Size != fdiconSourceSize || document.Source.MD5 != fdiconSourceMD5 ||
		document.Source.SHA256 != fdiconSourceSHA256 || len(document.Sprites) != SeparatedSpriteCount {
		return nil, errors.New("fdicon: separated bank contract mismatch")
	}
	bank := &Bank{Sprites: make([]Sprite, SeparatedSpriteCount)}
	seen := make([]bool, SeparatedSpriteCount)
	for _, entry := range document.Sprites {
		if entry.Index < 0 || entry.Index >= SeparatedSpriteCount || seen[entry.Index] ||
			entry.Width != NativeSize || entry.Height != NativeSize {
			return nil, fmt.Errorf("fdicon: invalid separated sprite index %d", entry.Index)
		}
		prefix := fmt.Sprintf("sprite_%04d/", entry.Index)
		if entry.Frame != prefix+"frame.png" || entry.Mask != prefix+"mask.png" ||
			entry.RemapMask != prefix+"remap_mask.png" {
			return nil, fmt.Errorf("fdicon: separated sprite %d path contract mismatch", entry.Index)
		}
		pixels, err := loadIndexedLayer(filepath.Join(root, filepath.FromSlash(entry.Frame)))
		if err != nil {
			return nil, fmt.Errorf("fdicon: separated sprite %d frame: %w", entry.Index, err)
		}
		mask, err := loadBinaryMask(filepath.Join(root, filepath.FromSlash(entry.Mask)))
		if err != nil {
			return nil, fmt.Errorf("fdicon: separated sprite %d mask: %w", entry.Index, err)
		}
		remap, err := loadBinaryMask(filepath.Join(root, filepath.FromSlash(entry.RemapMask)))
		if err != nil {
			return nil, fmt.Errorf("fdicon: separated sprite %d remap mask: %w", entry.Index, err)
		}
		bank.Sprites[entry.Index] = Sprite{Pixels: pixels, Mask: mask, RemapMask: remap}
		seen[entry.Index] = true
	}
	for index, ok := range seen {
		if !ok {
			return nil, fmt.Errorf("fdicon: separated sprite %d is missing", index)
		}
	}
	return bank, nil
}

func loadIndexedLayer(path string) ([]byte, error) {
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
	if !ok || paletted.Bounds() != image.Rect(0, 0, NativeSize, NativeSize) || len(paletted.Palette) != 256 {
		return nil, errors.New("indexed PNG must be 24x24 with 256 palette entries")
	}
	pixels := make([]byte, NativeSize*NativeSize)
	for row := 0; row < NativeSize; row++ {
		copy(pixels[row*NativeSize:(row+1)*NativeSize], paletted.Pix[row*paletted.Stride:row*paletted.Stride+NativeSize])
	}
	return pixels, nil
}

func loadBinaryMask(path string) ([]byte, error) {
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
	if !ok || gray.Bounds() != image.Rect(0, 0, NativeSize, NativeSize) {
		return nil, errors.New("mask PNG must be 24x24 grayscale")
	}
	mask := make([]byte, NativeSize*NativeSize)
	for row := 0; row < NativeSize; row++ {
		for col := 0; col < NativeSize; col++ {
			value := gray.GrayAt(col, row).Y
			if value != 0 && value != 0xff {
				return nil, errors.New("mask PNG contains a non-binary value")
			}
			if value != 0 {
				mask[row*NativeSize+col] = 1
			}
		}
	}
	return mask, nil
}
