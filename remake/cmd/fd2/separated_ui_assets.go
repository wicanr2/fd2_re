package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func separatedAssetPath(relative string) string {
	if root := os.Getenv("FD2_ASSET_PACK"); root != "" {
		return filepath.Join(root, filepath.FromSlash(relative))
	}
	return assetPath(filepath.ToSlash(filepath.Join("assets", relative)))
}

func loadNativeBattlePalette() ([]byte, color.Palette, error) {
	return fdother.LoadSeparatedFDOTHERPalette(separatedAssetPath("palette"), 0)
}

func loadNativeUIPalette() color.Palette {
	_, palette, err := loadNativeBattlePalette()
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
