package fdother

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type separatedChapterAuxSource struct {
	File      string `json:"file"`
	Resource  int    `json:"resource"`
	Size      int    `json:"size"`
	MD5       string `json:"md5"`
	SHA256    string `json:"sha256"`
	RawSize   int    `json:"raw_size"`
	RawMD5    string `json:"raw_md5"`
	RawSHA256 string `json:"raw_sha256"`
}

type separatedChapterAuxDocument struct {
	SchemaVersion int                       `json:"schema_version"`
	Kind          string                    `json:"kind"`
	AssetID       string                    `json:"asset_id"`
	Status        string                    `json:"status"`
	Evidence      string                    `json:"evidence"`
	Codec         string                    `json:"codec"`
	Width         int                       `json:"width"`
	Height        int                       `json:"height"`
	Frame         string                    `json:"frame"`
	Source        separatedChapterAuxSource `json:"source"`
}

// LoadSeparatedChapterAuxSurface loads FDOTHER #55's raw opaque 320x200
// indexed surface. It does not accept a four-mode RLE surface or archive fallback.
func LoadSeparatedChapterAuxSurface(root string) (*NativeChapterAuxSurface, error) {
	directory := filepath.Join(root, "FDOTHER_055")
	raw, err := os.ReadFile(filepath.Join(directory, "resource.json"))
	if err != nil {
		return nil, fmt.Errorf("fdother: separated chapter auxiliary metadata: %w", err)
	}
	var document separatedChapterAuxDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("fdother: separated chapter auxiliary JSON: %w", err)
	}
	if document.SchemaVersion != 1 || document.Kind != "indexed_surface" ||
		document.AssetID != "surface/FDOTHER_055" || document.Status != "decoded" ||
		document.Evidence != "confirmed" || document.Codec != "raw_indexed_opaque" ||
		document.Width != 320 || document.Height != 200 || document.Frame != "frame.png" ||
		document.Source.File != "FDOTHER.DAT" || document.Source.Resource != 55 ||
		document.Source.Size != 3382481 || document.Source.MD5 != "22f56e5027edc7c766ad34ca4e5aca93" ||
		document.Source.SHA256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce" ||
		document.Source.RawSize != 64004 || document.Source.RawMD5 != "710ce98d109298ff0110b1a4fb8fec53" ||
		document.Source.RawSHA256 != "a1999b7547bc4eabfb79049ae7cd7d08b12fd4402132e9e6e67b1fb56c981e65" {
		return nil, errors.New("fdother: separated chapter auxiliary contract mismatch")
	}
	pixels, err := readPalettedSurface(filepath.Join(directory, document.Frame), 320, 200)
	if err != nil {
		return nil, err
	}
	return &NativeChapterAuxSurface{Pixels: pixels}, nil
}
