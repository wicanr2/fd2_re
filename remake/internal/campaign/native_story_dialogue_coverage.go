package campaign

import (
	"errors"
	"fmt"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// NativeStoryDialogueFrameBounds derives presentation bounds from the same
// opaque writes as the native frame. The restored background and cursor tail
// carry no dialogue border.
func NativeStoryDialogueFrameBounds(cells []fdother.RawCell, control string, phase NativeDialogueFramePhase, index int) (image.Rectangle, error) {
	if phase == NativeDialogueClosing && index >= len(nativeStoryClosingGridSizes) {
		// Validate the source/control through the existing blank-frame contract.
		_, err := NativeStoryDialogueCoverage(cells, control, phase, len(nativeStoryClosingGridSizes))
		return image.Rectangle{}, err
	}
	mask, err := NativeStoryDialogueCoverage(cells, control, phase, index)
	if err != nil {
		return image.Rectangle{}, err
	}
	var bounds image.Rectangle
	for i, painted := range mask {
		if painted {
			bounds = bounds.Union(image.Rect(i%320, i/320, i%320+1, i/320+1))
		}
	}
	return bounds, nil
}

type NativeDialogueFramePhase uint8

const (
	NativeDialogueOpening NativeDialogueFramePhase = iota
	NativeDialogueSteady
	NativeDialogueClosing
)

// NativeStoryDialogueCoverage follows the actual opaque raw-cell writes.
// Closing covers the five snapshot restores, without the optional cursor tail.
func NativeStoryDialogueCoverage(cells []fdother.RawCell, control string, phase NativeDialogueFramePhase, index int) ([]bool, error) {
	if len(cells) < 18 {
		return nil, errors.New("campaign: dialogue coverage cells unavailable")
	}
	y := nativeStoryLowerFrameY
	switch control {
	case "FFEF", "FFED":
		y = nativeStoryUpperFrameY
	case "FFEE", "FFEC":
	default:
		return nil, errors.New("campaign: dialogue coverage control unavailable")
	}
	columns, rows := 19, 5
	switch phase {
	case NativeDialogueOpening:
		if index < 0 || index >= len(nativeStoryOpeningGridSizes) {
			return nil, errors.New("campaign: dialogue opening coverage index unavailable")
		}
		columns, rows = nativeStoryOpeningGridSizes[index][0], nativeStoryOpeningGridSizes[index][1]
	case NativeDialogueClosing:
		if index < 0 || index > len(nativeStoryClosingGridSizes) {
			return nil, errors.New("campaign: dialogue closing coverage index unavailable")
		}
		if index == len(nativeStoryClosingGridSizes) {
			return make([]bool, 320*200), nil
		}
		columns, rows = nativeStoryClosingGridSizes[index][0], nativeStoryClosingGridSizes[index][1]
	case NativeDialogueSteady:
		if index != 0 {
			return nil, errors.New("campaign: dialogue steady coverage index unavailable")
		}
	default:
		return nil, errors.New("campaign: dialogue coverage phase unavailable")
	}
	placements, err := fdother.PlanNativeDialogueFrameGrid(320, 5, y, columns, rows)
	if err != nil {
		return nil, err
	}
	mask := make([]bool, 320*200)
	for _, placement := range placements {
		cell := cells[placement.ResourceIndex]
		x, y := placement.DestinationByte%320, placement.DestinationByte/320
		if cell.Width <= 0 || cell.Height <= 0 || len(cell.Pixels) != cell.Width*cell.Height ||
			x < 0 || y < 0 || x+cell.Width > 320 || y+cell.Height > 200 {
			return nil, fmt.Errorf("campaign: dialogue coverage cell %d exceeds source", placement.ResourceIndex)
		}
		for row := 0; row < cell.Height; row++ {
			for column := 0; column < cell.Width; column++ {
				mask[(y+row)*320+x+column] = true
			}
		}
	}
	return mask, nil
}
