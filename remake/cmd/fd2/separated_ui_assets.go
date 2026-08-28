package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type separatedPaletteDocument struct {
	SchemaVersion int    `json:"schema_version"`
	AssetID       string `json:"asset_id"`
	Components    []int  `json:"dac_6bit_components"`
}

func separatedAssetPath(relative string) string {
	if root := os.Getenv("FD2_ASSET_PACK"); root != "" {
		return filepath.Join(root, filepath.FromSlash(relative))
	}
	return assetPath(filepath.ToSlash(filepath.Join("assets", relative)))
}

func loadNativeUIPalette() color.Palette {
	raw, err := os.ReadFile(separatedAssetPath("palette/fdother_000.json"))
	if err != nil {
		return nil
	}
	var document separatedPaletteDocument
	if err := json.Unmarshal(raw, &document); err != nil || document.SchemaVersion != 1 ||
		document.AssetID != "palette/fdother_000" || len(document.Components) != 256*3 {
		return nil
	}
	dac := make([]byte, len(document.Components))
	for index, component := range document.Components {
		if component < 0 || component > 63 {
			return nil
		}
		dac[index] = byte(component)
	}
	palette, err := fdother.ParseVGAPalette(dac)
	if err != nil {
		return nil
	}
	return palette
}

func loadNativeActionCells(palette color.Palette) []*ebiten.Image {
	if len(palette) != 256 {
		return nil
	}
	images := make([]*ebiten.Image, nativeActionOverlayCellCount)
	for index := range images {
		path := separatedAssetPath(filepath.ToSlash(filepath.Join("ui", "action_cells", fmt.Sprintf("cell_%03d.png", index))))
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		decoded, _, decodeErr := image.Decode(file)
		closeErr := file.Close()
		if decodeErr != nil || closeErr != nil || decoded.Bounds().Dx() <= 0 || decoded.Bounds().Dy() <= 0 {
			return nil
		}
		images[index] = ebiten.NewImageFromImage(decoded)
	}
	return images
}
