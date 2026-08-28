package main

import (
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
	cells, err := fdother.LoadSeparatedActionCells(separatedAssetPath("ui"))
	if err != nil || len(cells) != nativeActionOverlayCellCount {
		return nil
	}
	images := make([]*ebiten.Image, len(cells))
	for index, cell := range cells {
		decoded, err := cell.Paletted(palette)
		if err != nil {
			return nil
		}
		images[index] = ebiten.NewImageFromImage(decoded)
	}
	return images
}
