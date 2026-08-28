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

// LoadSeparatedEvent61Frames loads the exact FDOTHER #45 frame table consumed
// by sub_356B7. It never opens or falls back to the original archive.
func LoadSeparatedEvent61Frames(root string) ([]Frame, error) {
	raw, err := os.ReadFile(filepath.Join(root, "bank.json"))
	if err != nil {
		return nil, fmt.Errorf("fdother: separated event61 metadata: %w", err)
	}
	var document separatedFrameBankDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("fdother: separated event61 JSON: %w", err)
	}
	if document.SchemaVersion != 1 || document.Kind != "fdother_frame_bank" ||
		document.AssetID != "animation/FDOTHER_045/event61" ||
		document.Status != "decoded" || document.Evidence != "confirmed" ||
		document.Codec != "fd2_2935b_frame_table" ||
		document.Source.File != "FDOTHER.DAT" || document.Source.Resource != 45 ||
		document.Source.Size != 3382481 ||
		document.Source.MD5 != "22f56e5027edc7c766ad34ca4e5aca93" ||
		document.Source.SHA256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce" ||
		document.Source.RawSize != 2073 ||
		document.Source.RawMD5 != "f26b4bc4ce9c8e9c9d12ec2e2f00f5b8" ||
		document.Source.RawSHA256 != "d7a0e73d7dec5403441c33cd84392a7683de329f7b0914a1bf23c47ccc56513a" ||
		len(document.Frames) != nativeEvent61FrameCount {
		return nil, errors.New("fdother: separated event61 contract mismatch")
	}

	frames := make([]Frame, nativeEvent61FrameCount)
	seen := make([]bool, nativeEvent61FrameCount)
	for _, entry := range document.Frames {
		if entry.Index < 0 || entry.Index >= nativeEvent61FrameCount || seen[entry.Index] ||
			entry.X != 0 || entry.Y != 0 || entry.Width != 8 || entry.Height != 2 {
			return nil, fmt.Errorf("fdother: invalid separated event61 frame %d", entry.Index)
		}
		prefix := fmt.Sprintf("frame_%03d/", entry.Index)
		if entry.Frame != prefix+"frame.png" || entry.Mask != prefix+"mask.png" {
			return nil, fmt.Errorf("fdother: event61 frame %d path mismatch", entry.Index)
		}
		indexed, err := readPalettedSurface(
			filepath.Join(root, filepath.FromSlash(entry.Frame)), entry.Width, entry.Height,
		)
		if err != nil {
			return nil, fmt.Errorf("fdother: event61 frame %d pixels: %w", entry.Index, err)
		}
		mask, err := readBinaryMask(
			filepath.Join(root, filepath.FromSlash(entry.Mask)), entry.Width, entry.Height,
		)
		if err != nil {
			return nil, fmt.Errorf("fdother: event61 frame %d mask: %w", entry.Index, err)
		}
		frames[entry.Index] = Frame{
			X: entry.X, Y: entry.Y, Width: entry.Width, Height: entry.Height,
			Indexed: indexed, Mask: mask,
		}
		seen[entry.Index] = true
	}
	for index, present := range seen {
		if !present {
			return nil, fmt.Errorf("fdother: event61 frame %d is missing", index)
		}
	}
	return frames, nil
}
