package fdother

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type separatedFrameBankSource struct {
	File      string `json:"file"`
	Resource  int    `json:"resource"`
	Size      int    `json:"size"`
	MD5       string `json:"md5"`
	SHA256    string `json:"sha256"`
	RawSize   int    `json:"raw_size"`
	RawMD5    string `json:"raw_md5"`
	RawSHA256 string `json:"raw_sha256"`
}

type separatedFrameBankEntry struct {
	Index  int    `json:"index"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Frame  string `json:"frame"`
	Mask   string `json:"mask"`
}

type separatedFrameBankDocument struct {
	SchemaVersion int                       `json:"schema_version"`
	Kind          string                    `json:"kind"`
	AssetID       string                    `json:"asset_id"`
	Status        string                    `json:"status"`
	Evidence      string                    `json:"evidence"`
	Codec         string                    `json:"codec"`
	Source        separatedFrameBankSource  `json:"source"`
	Frames        []separatedFrameBankEntry `json:"frames"`
}

const nativeEvent61FrameCount = 59
const nativeEndingPrefixFrameCount = 111

type separatedFrameBankContract struct {
	label      string
	assetID    string
	resource   int
	rawSize    int
	rawMD5     string
	rawSHA256  string
	frameCount int
	geometry   func(separatedFrameBankEntry) bool
}

// LoadSeparatedEvent61Frames loads the exact FDOTHER #45 frame table consumed
// by sub_356B7. It never opens or falls back to the original archive.
func LoadSeparatedEvent61Frames(root string) ([]Frame, error) {
	return loadSeparatedFrameBank(root, separatedFrameBankContract{
		label: "event61", assetID: "animation/FDOTHER_045/event61", resource: 45,
		rawSize: 2073, rawMD5: "f26b4bc4ce9c8e9c9d12ec2e2f00f5b8",
		rawSHA256:  "d7a0e73d7dec5403441c33cd84392a7683de329f7b0914a1bf23c47ccc56513a",
		frameCount: nativeEvent61FrameCount,
		geometry: func(entry separatedFrameBankEntry) bool {
			return entry.X == 0 && entry.Y == 0 && entry.Width == 8 && entry.Height == 2
		},
	})
}

// LoadSeparatedEndingPrefixFrames loads the complete FDOTHER #54 frame table
// consumed by sub_2BCE5. It never opens or falls back to the original archive.
func LoadSeparatedEndingPrefixFrames(root string) ([]Frame, error) {
	return loadSeparatedFrameBank(root, separatedFrameBankContract{
		label: "ending prefix", assetID: "animation/FDOTHER_054/ending_prefix", resource: 54,
		rawSize: 263655, rawMD5: "4a3853ed5570a319e3a7e6024fb8e84e",
		rawSHA256:  "1e6cb912150102df4404a741846305033fe32f55db7d3a694ccb7fcc63496a46",
		frameCount: nativeEndingPrefixFrameCount,
		geometry:   validEndingPrefixGeometry,
	})
}

func validEndingPrefixGeometry(entry separatedFrameBankEntry) bool {
	if entry.X < 0 || entry.Y < 0 || entry.Width <= 0 || entry.Height <= 0 ||
		entry.X+entry.Width > 320 || entry.Y+entry.Height > 200 {
		return false
	}
	want := map[int][4]int{
		0: {0, 23, 320, 132}, 9: {116, 39, 86, 81},
		108: {116, 39, 86, 81}, 110: {0, 0, 320, 200},
	}
	if geometry, ok := want[entry.Index]; ok {
		return entry.X == geometry[0] && entry.Y == geometry[1] &&
			entry.Width == geometry[2] && entry.Height == geometry[3]
	}
	return true
}

func loadSeparatedFrameBank(root string, contract separatedFrameBankContract) ([]Frame, error) {
	raw, err := os.ReadFile(filepath.Join(root, "bank.json"))
	if err != nil {
		return nil, fmt.Errorf("fdother: separated %s metadata: %w", contract.label, err)
	}
	var document separatedFrameBankDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("fdother: separated %s JSON: %w", contract.label, err)
	}
	if document.SchemaVersion != 1 || document.Kind != "fdother_frame_bank" ||
		document.AssetID != contract.assetID ||
		document.Status != "decoded" || document.Evidence != "confirmed" ||
		document.Codec != "fd2_2935b_frame_table" ||
		document.Source.File != "FDOTHER.DAT" || document.Source.Resource != contract.resource ||
		document.Source.Size != 3382481 ||
		document.Source.MD5 != "22f56e5027edc7c766ad34ca4e5aca93" ||
		document.Source.SHA256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce" ||
		document.Source.RawSize != contract.rawSize ||
		document.Source.RawMD5 != contract.rawMD5 ||
		document.Source.RawSHA256 != contract.rawSHA256 ||
		len(document.Frames) != contract.frameCount {
		return nil, errors.New("fdother: separated " + contract.label + " contract mismatch")
	}

	frames := make([]Frame, contract.frameCount)
	seen := make([]bool, contract.frameCount)
	for _, entry := range document.Frames {
		if entry.Index < 0 || entry.Index >= contract.frameCount || seen[entry.Index] ||
			contract.geometry == nil || !contract.geometry(entry) {
			return nil, fmt.Errorf("fdother: invalid separated %s frame %d", contract.label, entry.Index)
		}
		prefix := fmt.Sprintf("frame_%03d/", entry.Index)
		if entry.Frame != prefix+"frame.png" || entry.Mask != prefix+"mask.png" {
			return nil, fmt.Errorf("fdother: %s frame %d path mismatch", contract.label, entry.Index)
		}
		indexed, err := readPalettedSurface(
			filepath.Join(root, filepath.FromSlash(entry.Frame)), entry.Width, entry.Height,
		)
		if err != nil {
			return nil, fmt.Errorf("fdother: %s frame %d pixels: %w", contract.label, entry.Index, err)
		}
		mask, err := readBinaryMask(
			filepath.Join(root, filepath.FromSlash(entry.Mask)), entry.Width, entry.Height,
		)
		if err != nil {
			return nil, fmt.Errorf("fdother: %s frame %d mask: %w", contract.label, entry.Index, err)
		}
		frames[entry.Index] = Frame{
			X: entry.X, Y: entry.Y, Width: entry.Width, Height: entry.Height,
			Indexed: indexed, Mask: mask,
		}
		seen[entry.Index] = true
	}
	for index, present := range seen {
		if !present {
			return nil, fmt.Errorf("fdother: %s frame %d is missing", contract.label, index)
		}
	}
	return frames, nil
}
