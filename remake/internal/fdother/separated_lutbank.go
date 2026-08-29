package fdother

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type separatedLUTBankSource struct {
	File      string `json:"file"`
	Resource  int    `json:"resource"`
	Size      int    `json:"size"`
	MD5       string `json:"md5"`
	SHA256    string `json:"sha256"`
	RawSize   int    `json:"raw_size"`
	RawMD5    string `json:"raw_md5"`
	RawSHA256 string `json:"raw_sha256"`
}

type separatedLUTEntry struct {
	Index      int   `json:"index"`
	Components []int `json:"components"`
}

type separatedLUTBankDocument struct {
	SchemaVersion int                    `json:"schema_version"`
	Kind          string                 `json:"kind"`
	AssetID       string                 `json:"asset_id"`
	Status        string                 `json:"status"`
	Evidence      string                 `json:"evidence"`
	Source        separatedLUTBankSource `json:"source"`
	LUTs          []separatedLUTEntry    `json:"luts"`
}

// LoadSeparatedLUTBank loads FDOTHER #3's complete 23x256 remap table. It
// preserves raw indices and values and never opens the original archive.
func LoadSeparatedLUTBank(root string) ([][]byte, error) {
	raw, err := os.ReadFile(filepath.Join(root, "fdother_003_luts.json"))
	if err != nil {
		return nil, fmt.Errorf("fdother: separated LUT bank metadata: %w", err)
	}
	var document separatedLUTBankDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("fdother: separated LUT bank JSON: %w", err)
	}
	if document.SchemaVersion != 1 || document.Kind != "fdother_lut_bank" ||
		document.AssetID != "lut/FDOTHER_003" || document.Status != "decoded" ||
		document.Evidence != "confirmed" || document.Source.File != "FDOTHER.DAT" ||
		document.Source.Resource != 3 || document.Source.Size != 3382481 ||
		document.Source.MD5 != "22f56e5027edc7c766ad34ca4e5aca93" ||
		document.Source.SHA256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce" ||
		document.Source.RawSize != 5990 || document.Source.RawMD5 != "f37194b6d933187d86711c6701732b5f" ||
		document.Source.RawSHA256 != "fddfdbd91b048081ea621b5a4be7d29fb24234a62e19e2b21837df52cbb02ccc" ||
		len(document.LUTs) != 23 {
		return nil, errors.New("fdother: separated LUT bank contract mismatch")
	}
	luts := make([][]byte, 23)
	seen := make([]bool, 23)
	for _, entry := range document.LUTs {
		if entry.Index < 0 || entry.Index >= len(luts) || seen[entry.Index] || len(entry.Components) != 256 {
			return nil, fmt.Errorf("fdother: separated LUT %d metadata mismatch", entry.Index)
		}
		lut := make([]byte, 256)
		for offset, value := range entry.Components {
			if value < 0 || value > 255 {
				return nil, fmt.Errorf("fdother: separated LUT %d component %d out of range", entry.Index, offset)
			}
			lut[offset] = byte(value)
		}
		luts[entry.Index], seen[entry.Index] = lut, true
	}
	for index, present := range seen {
		if !present {
			return nil, fmt.Errorf("fdother: separated LUT %d is missing", index)
		}
	}
	return luts, nil
}
