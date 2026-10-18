package campaign

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

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
