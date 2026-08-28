package afm

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
)

const (
	ANIFileSize = 2437547
	ANIMD5      = "81315bcbb78764361c5137ab0f714f7e"
	ANISHA256   = "be909c71d0f1121b6632ae931d978e990f6d54c830f4e0509cd6862187c4d963"
)

var expectedFrameCounts = [...]int{96, 51, 26, 28, 12, 35, 12, 17, 12}
var expectedRawSizes = [...]int{1002800, 635952, 97726, 35566, 36113, 411039, 43553, 137859, 36893}

type separatedSource struct {
	File                string `json:"file"`
	Resource            int    `json:"resource"`
	Size                int    `json:"size"`
	MD5                 string `json:"md5"`
	SHA256              string `json:"sha256"`
	RawSize             int    `json:"raw_size"`
	ContainerEntryCount int    `json:"container_entry_count"`
	EmptyTailIndex      int    `json:"empty_tail_index"`
}

type separatedFrame struct {
	FrameID     string `json:"frame_id"`
	Frame       string `json:"frame"`
	Palette     string `json:"palette"`
	SourceFrame int    `json:"source_frame"`
}

type separatedAnimation struct {
	SchemaVersion int              `json:"schema_version"`
	Kind          string           `json:"kind"`
	AssetID       string           `json:"asset_id"`
	Status        string           `json:"status"`
	Evidence      string           `json:"evidence"`
	Codec         string           `json:"codec"`
	Title         string           `json:"title"`
	Width         int              `json:"width"`
	Height        int              `json:"height"`
	FrameCount    int              `json:"frame_count"`
	Source        separatedSource  `json:"source"`
	Frames        []separatedFrame `json:"frames"`
}

type separatedPalette struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Components    []int  `json:"dac_6bit_components"`
}

// LoadSeparatedResource 載入完整 AFM framebuffer 與六位元 DAC snapshots。
// 正式 runtime 不得在此失敗後回退讀 ANI.DAT。
func LoadSeparatedResource(animationRoot string, resource int) (*Clip, error) {
	if animationRoot == "" || resource < 0 || resource >= len(expectedFrameCounts) {
		return nil, fmt.Errorf("afm: invalid separated resource %d", resource)
	}
	directory := filepath.Join(animationRoot, fmt.Sprintf("ANI_%03d", resource))
	raw, err := os.ReadFile(filepath.Join(directory, "animation.json"))
	if err != nil {
		return nil, fmt.Errorf("afm: separated metadata %d: %w", resource, err)
	}
	var document separatedAnimation
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("afm: separated metadata %d: %w", resource, err)
	}
	wantFrames := expectedFrameCounts[resource]
	if document.SchemaVersion != 1 || document.Kind != "afm_indexed_animation" ||
		document.AssetID != fmt.Sprintf("animation/ANI_%03d", resource) ||
		document.Status != "decoded" || document.Evidence != "confirmed" ||
		document.Codec != "fd2_afm_vm_v1" || document.Width != Width || document.Height != Height ||
		document.FrameCount != wantFrames || len(document.Frames) != wantFrames ||
		document.Source.File != "ANI.DAT" || document.Source.Resource != resource ||
		document.Source.Size != ANIFileSize || document.Source.MD5 != ANIMD5 ||
		document.Source.SHA256 != ANISHA256 || document.Source.RawSize != expectedRawSizes[resource] ||
		document.Source.ContainerEntryCount != 10 || document.Source.EmptyTailIndex != 9 {
		return nil, fmt.Errorf("afm: separated metadata %d violates the contract", resource)
	}
	clip := &Clip{Title: document.Title, HeaderFrames: wantFrames,
		Frames: make([]*image.RGBA, wantFrames), IndexedFrames: make([][]byte, wantFrames), Palettes: make([][]byte, wantFrames)}
	for index, item := range document.Frames {
		if item.FrameID != fmt.Sprintf("frame/%03d", index) || item.SourceFrame != index ||
			filepath.Base(item.Frame) != item.Frame || filepath.Base(item.Palette) != item.Palette {
			return nil, fmt.Errorf("afm: separated frame %d contract is invalid", index)
		}
		file, err := os.Open(filepath.Join(directory, item.Frame))
		if err != nil {
			return nil, fmt.Errorf("afm: separated frame %d: %w", index, err)
		}
		decoded, _, decodeErr := image.Decode(file)
		closeErr := file.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("afm: separated frame %d: %w", index, decodeErr)
		}
		if closeErr != nil {
			return nil, closeErr
		}
		indexed, ok := decoded.(*image.Paletted)
		if !ok || indexed.Rect != image.Rect(0, 0, Width, Height) || indexed.Stride != Width || len(indexed.Pix) != Width*Height {
			return nil, fmt.Errorf("afm: separated frame %d is not a packed 320x200 indexed PNG", index)
		}
		paletteRaw, err := os.ReadFile(filepath.Join(directory, item.Palette))
		if err != nil {
			return nil, fmt.Errorf("afm: separated palette %d: %w", index, err)
		}
		var paletteDocument separatedPalette
		if err := json.Unmarshal(paletteRaw, &paletteDocument); err != nil {
			return nil, fmt.Errorf("afm: separated palette %d: %w", index, err)
		}
		if paletteDocument.SchemaVersion != 1 || paletteDocument.Kind != "vga_dac_6bit_snapshot" || len(paletteDocument.Components) != 768 {
			return nil, fmt.Errorf("afm: separated palette %d violates the contract", index)
		}
		palette := make([]byte, 768)
		for component, value := range paletteDocument.Components {
			if value < 0 || value > 63 {
				return nil, fmt.Errorf("afm: separated palette %d has an invalid component", index)
			}
			palette[component] = byte(value)
		}
		frame := append([]byte(nil), indexed.Pix...)
		clip.IndexedFrames[index], clip.Palettes[index] = frame, palette
		clip.Frames[index] = toRGBA(frame, palette)
	}
	return clip, nil
}
