package fdother

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
)

type separatedPaletteSource struct {
	File     string `json:"file"`
	Resource int    `json:"resource"`
	Size     int    `json:"size"`
	MD5      string `json:"md5"`
	SHA256   string `json:"sha256"`
	RawSize  int    `json:"raw_size"`
}

type separatedPaletteDocument struct {
	SchemaVersion int                    `json:"schema_version"`
	AssetID       string                 `json:"asset_id"`
	Source        separatedPaletteSource `json:"source"`
	Components    []int                  `json:"dac_6bit_components"`
}

// LoadSeparatedFDOTHERPalette 載入固定版本 FDOTHER.DAT 的具名 VGA DAC。
// 它不讀取或回退原始 archive。
func LoadSeparatedFDOTHERPalette(paletteRoot string, resource int) ([]byte, color.Palette, error) {
	if paletteRoot == "" || (resource != 0 && resource != 8 && resource != 57 && resource != 76 && resource != 99 && resource != 101) {
		return nil, nil, fmt.Errorf("fdother: unsupported separated palette request %d", resource)
	}
	path := filepath.Join(paletteRoot, fmt.Sprintf("fdother_%03d.json", resource))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("fdother: separated palette metadata: %w", err)
	}
	var doc separatedPaletteDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, nil, fmt.Errorf("fdother: separated palette metadata: %w", err)
	}
	if doc.SchemaVersion != 1 || doc.AssetID != fmt.Sprintf("palette/fdother_%03d", resource) ||
		doc.Source.File != "FDOTHER.DAT" || doc.Source.Resource != resource || doc.Source.Size != 3382481 ||
		doc.Source.MD5 != "22f56e5027edc7c766ad34ca4e5aca93" ||
		doc.Source.SHA256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce" ||
		doc.Source.RawSize != 768 || len(doc.Components) != 768 {
		return nil, nil, fmt.Errorf("fdother: separated palette %d contract mismatch", resource)
	}
	dac := make([]byte, len(doc.Components))
	for index, component := range doc.Components {
		if component < 0 || component > 63 {
			return nil, nil, fmt.Errorf("fdother: separated palette %d component %d outside 0..63", resource, index)
		}
		dac[index] = byte(component)
	}
	palette, err := ParseVGAPalette(dac)
	if err != nil {
		return nil, nil, err
	}
	return dac, palette, nil
}
