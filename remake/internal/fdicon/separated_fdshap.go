package fdicon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	SeparatedFDSHAPMapCount = 33
	fdshapSourceSize        = 3557794
	fdshapSourceMD5         = "9b0d356074f57cc27aebf3bb89aae247"
	fdshapSourceSHA256      = "901b70ea82d5d977192759fad510921ffe16a0ab6af6ab7c32757de03e30aa3c"
)

type separatedFDSHAPDocument struct {
	SchemaVersion   int               `json:"schema_version"`
	Kind            string            `json:"kind"`
	AssetID         string            `json:"asset_id"`
	Status          string            `json:"status"`
	Evidence        string            `json:"evidence"`
	Source          separatedSource   `json:"source"`
	MapIndex        int               `json:"map_index"`
	ImageResource   int               `json:"image_resource"`
	ControlResource int               `json:"control_resource"`
	Controls        []int             `json:"controls"`
	Sprites         []separatedSprite `json:"sprites"`
}

// LoadSeparatedFDSHAPBank loads one complete tactical terrain bank and its
// adjacent four-byte control table. It has no FDSHAP.DAT fallback.
func LoadSeparatedFDSHAPBank(root string, mapIndex int) (*Bank, []byte, error) {
	if root == "" || mapIndex < 0 || mapIndex >= SeparatedFDSHAPMapCount {
		return nil, nil, errors.New("fdicon: invalid separated FDSHAP request")
	}
	directory := filepath.Join(root, fmt.Sprintf("map_%02d", mapIndex))
	raw, err := os.ReadFile(filepath.Join(directory, "bank.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("fdicon: separated FDSHAP metadata: %w", err)
	}
	var document separatedFDSHAPDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, nil, fmt.Errorf("fdicon: separated FDSHAP metadata: %w", err)
	}
	wantID := fmt.Sprintf("tilesets/fdshap/map_%02d", mapIndex)
	if document.SchemaVersion != 1 || document.Kind != "fdshap_terrain_bank" ||
		document.AssetID != wantID || document.Status != "decoded" || document.Evidence != "confirmed" ||
		document.Source.File != "FDSHAP.DAT" || document.Source.Size != fdshapSourceSize ||
		document.Source.MD5 != fdshapSourceMD5 || document.Source.SHA256 != fdshapSourceSHA256 ||
		document.MapIndex != mapIndex || document.ImageResource != mapIndex*2 ||
		document.ControlResource != mapIndex*2+1 || len(document.Controls) == 0 || len(document.Controls)%4 != 0 ||
		len(document.Sprites) == 0 {
		return nil, nil, errors.New("fdicon: separated FDSHAP contract mismatch")
	}
	controls := make([]byte, len(document.Controls))
	for index, value := range document.Controls {
		if value < 0 || value > 0xff {
			return nil, nil, fmt.Errorf("fdicon: separated FDSHAP control %d=%d", index, value)
		}
		controls[index] = byte(value)
	}
	bank := &Bank{Sprites: make([]Sprite, len(document.Sprites))}
	seen := make([]bool, len(document.Sprites))
	for _, entry := range document.Sprites {
		if entry.Index < 0 || entry.Index >= len(document.Sprites) || seen[entry.Index] ||
			entry.Width != NativeSize || entry.Height != NativeSize {
			return nil, nil, fmt.Errorf("fdicon: invalid separated FDSHAP sprite index %d", entry.Index)
		}
		prefix := fmt.Sprintf("tile_%04d/", entry.Index)
		if entry.Frame != prefix+"frame.png" || entry.Mask != prefix+"mask.png" ||
			entry.RemapMask != prefix+"remap_mask.png" {
			return nil, nil, fmt.Errorf("fdicon: separated FDSHAP sprite %d path mismatch", entry.Index)
		}
		pixels, err := loadIndexedLayer(filepath.Join(directory, filepath.FromSlash(entry.Frame)))
		if err != nil {
			return nil, nil, fmt.Errorf("fdicon: separated FDSHAP sprite %d frame: %w", entry.Index, err)
		}
		mask, err := loadBinaryMask(filepath.Join(directory, filepath.FromSlash(entry.Mask)))
		if err != nil {
			return nil, nil, fmt.Errorf("fdicon: separated FDSHAP sprite %d mask: %w", entry.Index, err)
		}
		remap, err := loadBinaryMask(filepath.Join(directory, filepath.FromSlash(entry.RemapMask)))
		if err != nil {
			return nil, nil, fmt.Errorf("fdicon: separated FDSHAP sprite %d remap mask: %w", entry.Index, err)
		}
		bank.Sprites[entry.Index] = Sprite{Pixels: pixels, Mask: mask, RemapMask: remap}
		seen[entry.Index] = true
	}
	for index, ok := range seen {
		if !ok {
			return nil, nil, fmt.Errorf("fdicon: separated FDSHAP sprite %d is missing", index)
		}
	}
	return bank, controls, nil
}
