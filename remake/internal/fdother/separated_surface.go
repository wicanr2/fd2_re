package fdother

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

type separatedSurfaceSource struct {
	File     string `json:"file"`
	Resource int    `json:"resource"`
	Size     int    `json:"size"`
	MD5      string `json:"md5"`
	SHA256   string `json:"sha256"`
	RawSize  int    `json:"raw_size"`
}

type separatedSurfaceDocument struct {
	SchemaVersion int                    `json:"schema_version"`
	Kind          string                 `json:"kind"`
	AssetID       string                 `json:"asset_id"`
	Status        string                 `json:"status"`
	Codec         string                 `json:"codec"`
	Width         int                    `json:"width"`
	Height        int                    `json:"height"`
	Frame         string                 `json:"frame"`
	Mask          string                 `json:"mask"`
	Evidence      string                 `json:"evidence"`
	ReasonCode    string                 `json:"reason_code,omitempty"`
	Source        separatedSurfaceSource `json:"source"`
}

type separatedSurfaceIdentity struct {
	prefix string
	size   int
	md5    string
	sha256 string
}

var separatedSurfaceIdentities = map[string]separatedSurfaceIdentity{
	"BG.DAT":      {prefix: "BG", size: 624564, md5: "4b5414c92b40ef25ba0ee10c80f9e149", sha256: "b9fc21d019d6256a4bb7e6da1cefcb0bfe331d8ff74a52a8201570afc98b56de"},
	"TAI.DAT":     {prefix: "TAI", size: 94917, md5: "7cfe4b9ad2cbff44b2ebd7ab2f94e4aa", sha256: "d56fea9c43f8bb59aad89ad76698885d7e07f380d12a4547888a0b60ea5e0410"},
	"FDOTHER.DAT": {prefix: "FDOTHER", size: 3382481, md5: "22f56e5027edc7c766ad34ca4e5aca93", sha256: "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"},
}

// LoadSeparatedSingleFrame 載入由固定版本 BG.DAT／TAI.DAT／FDOTHER.DAT 匯出的索引單幀。
// 它不讀取或回退原始 archive。
func LoadSeparatedSingleFrame(surfaceRoot, sourceFile string, resource int) (Frame, error) {
	identity, ok := separatedSurfaceIdentities[sourceFile]
	if !ok || resource < 0 {
		return Frame{}, errors.New("fdother: unsupported separated surface source")
	}
	directoryName := fmt.Sprintf("%s_%03d", identity.prefix, resource)
	directory := filepath.Join(surfaceRoot, directoryName)
	raw, err := os.ReadFile(filepath.Join(directory, "resource.json"))
	if err != nil {
		return Frame{}, fmt.Errorf("fdother: separated surface metadata: %w", err)
	}
	var doc separatedSurfaceDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Frame{}, fmt.Errorf("fdother: separated surface metadata: %w", err)
	}
	wantAssetID := fmt.Sprintf("surface/%s_%03d", identity.prefix, resource)
	if doc.SchemaVersion != 1 || doc.Kind != "indexed_surface" || doc.AssetID != wantAssetID ||
		doc.Status != "decoded" || doc.Codec != "fd2_4e63d_single_frame" || doc.Evidence != "confirmed" ||
		doc.Width <= 0 || doc.Height <= 0 || doc.Frame != "frame.png" || doc.Mask != "mask.png" ||
		doc.Source.File != sourceFile || doc.Source.Resource != resource || doc.Source.Size != identity.size ||
		doc.Source.MD5 != identity.md5 || doc.Source.SHA256 != identity.sha256 || doc.Source.RawSize < 4 {
		return Frame{}, errors.New("fdother: separated surface metadata contract mismatch")
	}
	indexed, err := readPalettedSurface(filepath.Join(directory, doc.Frame), doc.Width, doc.Height)
	if err != nil {
		return Frame{}, err
	}
	mask, err := readBinaryMask(filepath.Join(directory, doc.Mask), doc.Width, doc.Height)
	if err != nil {
		return Frame{}, err
	}
	return Frame{Width: doc.Width, Height: doc.Height, Indexed: indexed, Mask: mask}, nil
}

func readPalettedSurface(path string, width, height int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("fdother: separated surface frame: %w", err)
	}
	defer file.Close()
	decoded, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("fdother: separated surface frame: %w", err)
	}
	paletted, ok := decoded.(*image.Paletted)
	if !ok || paletted.Rect != image.Rect(0, 0, width, height) || paletted.Stride != width || len(paletted.Pix) != width*height {
		return nil, errors.New("fdother: separated surface frame is not tightly packed indexed PNG")
	}
	return append([]byte(nil), paletted.Pix...), nil
}

func readBinaryMask(path string, width, height int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("fdother: separated surface mask: %w", err)
	}
	defer file.Close()
	decoded, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("fdother: separated surface mask: %w", err)
	}
	gray, ok := decoded.(*image.Gray)
	if !ok || gray.Rect != image.Rect(0, 0, width, height) || gray.Stride != width || len(gray.Pix) != width*height {
		return nil, errors.New("fdother: separated surface mask is not tightly packed grayscale PNG")
	}
	mask := append([]byte(nil), gray.Pix...)
	for _, value := range mask {
		if value != 0 && value != 255 {
			return nil, errors.New("fdother: separated surface mask is not binary")
		}
	}
	return mask, nil
}
