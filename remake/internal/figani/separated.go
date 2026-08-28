package figani

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
)

type separatedResourceStatusDocument struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	ResourceID    string `json:"resource_id"`
	Source        struct {
		File     string `json:"file"`
		Resource int    `json:"resource"`
	} `json:"source"`
	RawSize      int    `json:"raw_size"`
	HeaderWordLE *int   `json:"header_word_le"`
	Status       string `json:"status"`
	ReasonCode   string `json:"reason_code"`
	Evidence     string `json:"evidence"`
}

type separatedAnimationDocument struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	AnimationID   string `json:"animation_id"`
	Source        struct {
		File     string `json:"file"`
		Resource int    `json:"resource"`
	} `json:"source"`
	NativeHeader struct {
		Byte1 byte `json:"byte_1"`
		Byte2 byte `json:"byte_2"`
		Byte4 byte `json:"byte_4"`
	} `json:"native_header"`
	Frames []struct {
		FrameID  string `json:"frame_id"`
		Path     string `json:"path"`
		MaskPath string `json:"mask_path"`
		Delay    int    `json:"delay_native"`
		X        int    `json:"x"`
		Y        int    `json:"y"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		RawByte4 byte   `json:"raw_byte_4"`
		RawByte5 byte   `json:"raw_byte_5"`
		RawByte7 byte   `json:"raw_byte_7"`
	} `json:"frames"`
}

func loadPNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoded, _, err := image.Decode(file)
	return decoded, err
}

// LoadSeparatedResource 重建 FIGANI 的原始 indexed pixels、透明 mask 與
// descriptor metadata。正式 runtime 不可在此失敗後回退讀 FIGANI.DAT。
func LoadSeparatedResource(animationRoot string, resource int) (*Animation, error) {
	if animationRoot == "" || resource < 0 || resource > 999 {
		return nil, fmt.Errorf("figani: invalid separated animation request")
	}
	directory := filepath.Join(animationRoot, fmt.Sprintf("FIGANI_%03d", resource))
	raw, err := os.ReadFile(filepath.Join(directory, "animation.json"))
	if err != nil {
		return nil, fmt.Errorf("figani: separated metadata %d: %w", resource, err)
	}
	var document separatedAnimationDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("figani: separated metadata %d: %w", resource, err)
	}
	wantID := fmt.Sprintf("animation/figani_%03d", resource)
	if document.SchemaVersion != 1 || document.Kind != "animation" || document.AnimationID != wantID ||
		document.Source.File != "FIGANI.DAT" || document.Source.Resource != resource || len(document.Frames) == 0 {
		return nil, fmt.Errorf("figani: separated metadata %d violates the animation contract", resource)
	}
	animation := &Animation{
		HeaderByte1: document.NativeHeader.Byte1,
		HeaderByte2: document.NativeHeader.Byte2,
		HeaderByte4: document.NativeHeader.Byte4,
		Frames:      make([]Frame, len(document.Frames)),
	}
	for index, item := range document.Frames {
		if item.FrameID != fmt.Sprintf("frame/%03d", index) || filepath.Base(item.Path) != item.Path ||
			filepath.Base(item.MaskPath) != item.MaskPath || item.Width <= 0 || item.Height <= 0 || item.Delay < 0 || item.Delay > 255 {
			return nil, fmt.Errorf("figani: separated frame %d contract is invalid", index)
		}
		pixelsImage, err := loadPNG(filepath.Join(directory, item.Path))
		if err != nil {
			return nil, fmt.Errorf("figani: separated frame %d pixels: %w", index, err)
		}
		maskImage, err := loadPNG(filepath.Join(directory, item.MaskPath))
		if err != nil {
			return nil, fmt.Errorf("figani: separated frame %d mask: %w", index, err)
		}
		pixels, ok := pixelsImage.(*image.Paletted)
		if !ok || pixels.Rect != image.Rect(0, 0, item.Width, item.Height) || pixels.Stride != item.Width {
			return nil, fmt.Errorf("figani: separated frame %d is not a packed indexed PNG", index)
		}
		mask, ok := maskImage.(*image.Gray)
		if !ok || mask.Rect != pixels.Rect || mask.Stride != item.Width {
			return nil, fmt.Errorf("figani: separated frame %d mask is not a packed grayscale PNG", index)
		}
		binaryMask := make([]byte, len(mask.Pix))
		for offset, value := range mask.Pix {
			if value != 0 {
				binaryMask[offset] = 1
			}
		}
		animation.Frames[index] = Frame{
			X: item.X, Y: item.Y, Width: item.Width, Height: item.Height,
			Pixels: append([]byte(nil), pixels.Pix...), Mask: binaryMask,
			Delay: item.Delay, RawByte4: item.RawByte4, RawByte5: item.RawByte5, RawByte7: item.RawByte7,
		}
	}
	return animation, nil
}

func separatedResourceHasZeroHeader(animationRoot string, resource int) (bool, error) {
	if animationRoot == "" || resource < 0 || resource > 999 {
		return false, fmt.Errorf("figani: invalid separated resource status request")
	}
	directory := filepath.Join(animationRoot, fmt.Sprintf("FIGANI_%03d", resource))
	raw, err := os.ReadFile(filepath.Join(directory, "resource.json"))
	if err != nil {
		return false, fmt.Errorf("figani: separated resource status %d: %w", resource, err)
	}
	var document separatedResourceStatusDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return false, fmt.Errorf("figani: separated resource status %d: %w", resource, err)
	}
	wantID := fmt.Sprintf("animation-resource/figani_%03d", resource)
	if document.SchemaVersion != 1 || document.Kind != "animation_resource_status" ||
		document.ResourceID != wantID || document.Source.File != "FIGANI.DAT" ||
		document.Source.Resource != resource || document.RawSize < 2 {
		return false, fmt.Errorf("figani: separated resource status %d violates the contract", resource)
	}
	return document.Status == "empty_header_zero" && document.ReasonCode == "zero_header_word" &&
		document.Evidence == "confirmed" && document.HeaderWordLE != nil && *document.HeaderWordLE == 0, nil
}

// LoadSeparatedResourceWithZeroHeaderFallback 保存原版 caller 已證實的
// selector*3+2 首 word 為零時退到前一資源規則。缺少或矛盾的狀態文件不可
// 被誤當成零標頭，亦不可回退讀 FIGANI.DAT。
func LoadSeparatedResourceWithZeroHeaderFallback(animationRoot string, resource int) (*Animation, error) {
	animation, primaryErr := LoadSeparatedResource(animationRoot, resource)
	if primaryErr == nil {
		return animation, nil
	}
	zero, statusErr := separatedResourceHasZeroHeader(animationRoot, resource)
	if statusErr != nil {
		return nil, fmt.Errorf("figani: resource %d unavailable (%v); status: %w", resource, primaryErr, statusErr)
	}
	if !zero || resource == 0 {
		return nil, primaryErr
	}
	return LoadSeparatedResource(animationRoot, resource-1)
}
