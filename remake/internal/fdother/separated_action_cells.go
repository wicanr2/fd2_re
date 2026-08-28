package fdother

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	actionCellsSourceSize   = 3382481
	actionCellsSourceMD5    = "22f56e5027edc7c766ad34ca4e5aca93"
	actionCellsSourceSHA256 = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
	actionCellsResourceSize = 37680
	actionCellsCount        = 78
)

// LoadSeparatedActionCells returns the complete FDOTHER #2 raw-cell bank.
// Pixel value zero remains a destination-preserving index at blit time.
func LoadSeparatedActionCells(root string) ([]RawCell, error) {
	directory := filepath.Join(root, "action_cells")
	raw, err := os.ReadFile(filepath.Join(directory, "resource.json"))
	if err != nil {
		return nil, fmt.Errorf("fdother: separated action-cell metadata: %w", err)
	}
	var document separatedItemPanelDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("fdother: separated action-cell JSON: %w", err)
	}
	if document.SchemaVersion != 1 || document.Kind != "fdother_raw_cell_bank" ||
		document.AssetID != "ui/FDOTHER_002/action_cells" || document.Status != "decoded" ||
		document.Evidence != "confirmed" || document.Source.File != "FDOTHER.DAT" ||
		document.Source.Resource != 2 || document.Source.Size != actionCellsSourceSize ||
		document.Source.MD5 != actionCellsSourceMD5 || document.Source.SHA256 != actionCellsSourceSHA256 ||
		document.Source.RawSize != actionCellsResourceSize || len(document.Entries) != actionCellsCount {
		return nil, fmt.Errorf("fdother: separated action-cell metadata mismatch")
	}
	cells := make([]RawCell, actionCellsCount)
	seen := make([]bool, actionCellsCount)
	for _, entry := range document.Entries {
		if entry.Index < 0 || entry.Index >= actionCellsCount || seen[entry.Index] ||
			entry.Codec != "raw_indexed_transparent" || entry.Width <= 0 || entry.Height <= 0 ||
			entry.Frame != fmt.Sprintf("cell_%03d.png", entry.Index) || entry.Mask != "" {
			return nil, fmt.Errorf("fdother: action-cell entry %d metadata mismatch", entry.Index)
		}
		pixels, err := loadItemPanelIndexedPNG(
			filepath.Join(directory, entry.Frame), entry.Width, entry.Height,
		)
		if err != nil {
			return nil, fmt.Errorf("fdother: action cell %d: %w", entry.Index, err)
		}
		cells[entry.Index] = RawCell{Width: entry.Width, Height: entry.Height, Pixels: pixels}
		seen[entry.Index] = true
	}
	for index, present := range seen {
		if !present {
			return nil, fmt.Errorf("fdother: action cell %d is missing", index)
		}
	}
	return cells, nil
}
