package ending

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// RenderDialogueFrameGrid executes only the recovered 0x168b6→0x4e9bb raw
// cell copies into the caller's portrait restore buffer. It does not render
// text, DATO, input, or any semantic UI layer.
func RenderDialogueFrameGrid(m Montage, cells []fdother.RawCell, dst []byte) error {
	if len(dst) != Bytes {
		return fmt.Errorf("ending: dialogue frame destination must be %d bytes", Bytes)
	}
	placements, err := m.PlanDialogueFrameGrid()
	if err != nil {
		return err
	}
	for _, placement := range placements {
		if placement.ResourceIndex < 0 || placement.ResourceIndex >= len(cells) {
			return fmt.Errorf("ending: dialogue frame resource index %d is unavailable", placement.ResourceIndex)
		}
		if err := cells[placement.ResourceIndex].BlitOpaqueAtOffset(dst, m.PartyCycle.DialogueFrameLayout.Stride, placement.DestinationByte); err != nil {
			return fmt.Errorf("ending: dialogue frame resource %d at %#x: %w", placement.ResourceIndex, placement.DestinationByte, err)
		}
	}
	return nil
}

// RenderDialogueFrameGridResource loads the exact FDOTHER#5 cells referenced
// by the recovered plan, then executes the raw compositor. Missing player
// assets fail closed; no fallback UI is synthesized.
func RenderDialogueFrameGridResource(m Montage, datPath string, dst []byte) error {
	placements, err := m.PlanDialogueFrameGrid()
	if err != nil {
		return err
	}
	maxIndex := 0
	for _, placement := range placements {
		if placement.ResourceIndex > maxIndex {
			maxIndex = placement.ResourceIndex
		}
	}
	cells := make([]fdother.RawCell, maxIndex+1)
	loaded := make([]bool, maxIndex+1)
	for _, placement := range placements {
		index := placement.ResourceIndex
		if loaded[index] {
			continue
		}
		cell, err := fdother.DecodeLMI1RawEntry(datPath, m.PartyCycle.DialogueFrameLayout.Resource, index)
		if err != nil {
			return fmt.Errorf("ending: FDOTHER#5 raw cell %d: %w", index, err)
		}
		cells[index] = cell
		loaded[index] = true
	}
	return RenderDialogueFrameGrid(m, cells, dst)
}

// RenderDialogueFrameGridSeparated loads the exact FDOTHER#5 raw cells from
// the separated item-panel contract. It never falls back to FDOTHER.DAT.
func RenderDialogueFrameGridSeparated(m Montage, itemPanelRoot string, dst []byte) error {
	placements, err := m.PlanDialogueFrameGrid()
	if err != nil {
		return err
	}
	entries, err := fdother.LoadSeparatedItemPanelEntries(itemPanelRoot)
	if err != nil {
		return err
	}
	maxIndex := 0
	for _, placement := range placements {
		if placement.ResourceIndex > maxIndex {
			maxIndex = placement.ResourceIndex
		}
	}
	cells := make([]fdother.RawCell, maxIndex+1)
	for _, placement := range placements {
		cell, ok := entries.Raw[placement.ResourceIndex]
		if !ok {
			return fmt.Errorf("ending: separated FDOTHER#5 raw cell %d is unavailable", placement.ResourceIndex)
		}
		cells[placement.ResourceIndex] = cell
	}
	return RenderDialogueFrameGrid(m, cells, dst)
}
