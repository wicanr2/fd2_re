package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

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
	images, _ := loadNativeActionCellsWithRaw(palette)
	return images
}

// loadNativeActionCellsWithRaw 同時回傳 indexed 原格（給 indexed 整幀的指令環用）。
func loadNativeActionCellsWithRaw(palette color.Palette) ([]*ebiten.Image, []fdother.RawCell) {
	if len(palette) != 256 {
		return nil, nil
	}
	cells, err := fdother.LoadSeparatedActionCells(separatedAssetPath("ui"))
	if err != nil || len(cells) != nativeActionOverlayCellCount {
		return nil, nil
	}
	images := make([]*ebiten.Image, len(cells))
	for index, cell := range cells {
		decoded, err := cell.Paletted(palette)
		if err != nil {
			return nil, nil
		}
		images[index] = ebiten.NewImageFromImage(decoded)
	}
	return images, cells
}
