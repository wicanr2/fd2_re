package fdother

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	loadSlotsSourceSize   = 3382481
	loadSlotsSourceMD5    = "22f56e5027edc7c766ad34ca4e5aca93"
	loadSlotsSourceSHA256 = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
	loadSlotsResourceSize = 53210
)

// LoadSeparatedLoadSlotsFrame loads the exact FDOTHER #13 entry used by the
// original four-slot LOAD screen. Archive decoding belongs to the offline
// importer; the production runtime accepts only the validated standard pack.
func LoadSeparatedLoadSlotsFrame(root string) (LMI1Entry, error) {
	directory := filepath.Join(root, "fdother_013_load_slots")
	raw, err := os.ReadFile(filepath.Join(directory, "resource.json"))
	if err != nil {
		return LMI1Entry{}, fmt.Errorf("fdother: separated load-slots metadata: %w", err)
	}
	var document separatedItemPanelDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return LMI1Entry{}, fmt.Errorf("fdother: separated load-slots JSON: %w", err)
	}
	if document.SchemaVersion != 1 || document.Kind != "fdother_lmi1_load_slots" ||
		document.AssetID != "ui/FDOTHER_013/load_slots" || document.Status != "decoded" ||
		document.Evidence != "confirmed" || document.Source.File != "FDOTHER.DAT" ||
		document.Source.Resource != 13 || document.Source.Size != loadSlotsSourceSize ||
		document.Source.MD5 != loadSlotsSourceMD5 || document.Source.SHA256 != loadSlotsSourceSHA256 ||
		document.Source.RawSize != loadSlotsResourceSize || len(document.Entries) != 1 {
		return LMI1Entry{}, fmt.Errorf("fdother: separated load-slots metadata mismatch")
	}
	entry := document.Entries[0]
	if entry.Index != 16 || entry.Codec != "opaque_high_run" ||
		entry.Width != 310 || entry.Height != 86 ||
		entry.Frame != "entry_016/frame.png" || entry.Mask != "" {
		return LMI1Entry{}, fmt.Errorf("fdother: separated load-slots entry metadata mismatch")
	}
	pixels, err := loadItemPanelIndexedPNG(
		filepath.Join(directory, entry.Frame), entry.Width, entry.Height,
	)
	if err != nil {
		return LMI1Entry{}, fmt.Errorf("fdother: separated load-slots frame: %w", err)
	}
	return LMI1Entry{Width: entry.Width, Height: entry.Height, Pixels: pixels}, nil
}
