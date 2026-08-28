package battle

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const nativeItemPanelBytes = 320 * 200

// RenderNativeItemPanelBase executes the complete recovered 0x17eef ordering:
// 0x168b6 raw grid, DATO frame zero, then FDOTHER#5 entries 20 and 21 through
// opaque 0x4e8af writes. 0x17fc0's dynamic data overlays are deliberately a
// separate pass.
func RenderNativeItemPanelBase(
	cells []fdother.RawCell,
	portrait dato.Frame,
	upper, bottom fdother.LMI1Entry,
	dst []byte,
) error {
	if len(dst) != nativeItemPanelBytes {
		return fmt.Errorf("battle: native item panel destination=%d, want %d", len(dst), nativeItemPanelBytes)
	}
	layout := NativeItemPanelBaseLayoutFor()
	placements, err := fdother.PlanNativeDialogueFrameGrid(
		layout.Stride, layout.GridOrigin.X, layout.GridOrigin.Y,
		layout.GridColumns, layout.GridRows,
	)
	if err != nil {
		return err
	}
	if len(cells) <= 17 {
		return errors.New("battle: native item panel frame cells 1..17 are unavailable")
	}

	staged := append([]byte(nil), dst...)
	for _, placement := range placements {
		cell := cells[placement.ResourceIndex]
		if err := cell.BlitOpaqueAtOffset(staged, layout.Stride, placement.DestinationByte); err != nil {
			return fmt.Errorf(
				"battle: native item panel frame cell %d at %#x: %w",
				placement.ResourceIndex, placement.DestinationByte, err,
			)
		}
	}
	if err := portrait.BlitAt(
		staged, layout.Stride,
		layout.PortraitDestination.X, layout.PortraitDestination.Y,
	); err != nil {
		return fmt.Errorf("battle: native item panel portrait: %w", err)
	}
	if err := upper.BlitOpaqueAt(
		staged, layout.Stride,
		layout.UpperDestination.X, layout.UpperDestination.Y, false,
	); err != nil {
		return fmt.Errorf("battle: native item panel upper entry: %w", err)
	}
	if err := bottom.BlitOpaqueAt(
		staged, layout.Stride,
		layout.BottomDestination.X, layout.BottomDestination.Y, false,
	); err != nil {
		return fmt.Errorf("battle: native item panel bottom entry: %w", err)
	}
	copy(dst, staged)
	return nil
}

// RenderNativeItemPanelBaseResources loads the separated FDOTHER #5 UI bank
// and portrait pack proven by 0x17eef. The portrait resource is the caller's
// unit-record byte +7; frame zero is the exact pointer selected by native's
// first DATO offset. It never falls back to the original archives.
func RenderNativeItemPanelBaseResources(
	assetPackRoot, portraitRoot string,
	portraitResource int,
	dst []byte,
) error {
	layout := NativeItemPanelBaseLayoutFor()
	entries, err := fdother.LoadSeparatedItemPanelEntries(filepath.Join(assetPackRoot, "ui"))
	if err != nil {
		return fmt.Errorf("battle: separated native item panel base: %w", err)
	}
	cells := make([]fdother.RawCell, 18)
	for index := 1; index <= 17; index++ {
		var ok bool
		cells[index], ok = entries.Raw[index]
		if !ok {
			return fmt.Errorf("battle: separated native item panel raw cell %d is unavailable", index)
		}
	}
	upper, upperOK := entries.Opaque[layout.UpperDirectoryEntry]
	bottom, bottomOK := entries.Opaque[layout.BottomDirectoryEntry]
	if !upperOK || !bottomOK {
		return errors.New("battle: separated native item panel large entries are unavailable")
	}
	portraits, err := dato.LoadSeparatedResource(portraitRoot, portraitResource)
	if err != nil {
		return fmt.Errorf("battle: native item panel separated portrait %d: %w", portraitResource, err)
	}
	if len(portraits) == 0 {
		return errors.New("battle: native item panel DATO frame zero is unavailable")
	}
	return RenderNativeItemPanelBase(
		cells, portraits[0],
		upper, bottom,
		dst,
	)
}

// RenderNativeItemPanelBaseResourcesArchive is retained only for fixed-source
// oracle comparisons. Production callers use RenderNativeItemPanelBaseResources.
func RenderNativeItemPanelBaseResourcesArchive(
	fdotherPath, portraitRoot string,
	portraitResource int,
	dst []byte,
) error {
	layout := NativeItemPanelBaseLayoutFor()
	raw, err := fdother.ReadResource(fdotherPath, layout.FDOTHERResource)
	if err != nil {
		return fmt.Errorf("battle: native item panel FDOTHER resource: %w", err)
	}
	cells := make([]fdother.RawCell, 18)
	for index := 1; index <= 17; index++ {
		cells[index], err = fdother.ParseLMI1RawEntry(raw, index)
		if err != nil {
			return fmt.Errorf("battle: native item panel raw cell %d: %w", index, err)
		}
	}
	entries, err := fdother.ParseLMI1(raw)
	if err != nil {
		return fmt.Errorf("battle: native item panel LMI1 entries: %w", err)
	}
	if layout.UpperDirectoryEntry >= len(entries) || layout.BottomDirectoryEntry >= len(entries) {
		return errors.New("battle: native item panel large entries are unavailable")
	}
	portraits, err := dato.LoadSeparatedResource(portraitRoot, portraitResource)
	if err != nil {
		return fmt.Errorf("battle: native item panel separated portrait %d: %w", portraitResource, err)
	}
	if len(portraits) == 0 {
		return errors.New("battle: native item panel DATO frame zero is unavailable")
	}
	return RenderNativeItemPanelBase(cells, portraits[0], entries[layout.UpperDirectoryEntry], entries[layout.BottomDirectoryEntry], dst)
}
