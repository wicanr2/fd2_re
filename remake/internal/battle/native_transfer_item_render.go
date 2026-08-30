package battle

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const NativeTransferVisible = 6

type NativeFacilityPriceMode byte

const (
	NativeFacilityFullPrice NativeFacilityPriceMode = iota
	NativeFacilityThreeQuarterPrice
)

func NativeFacilityItemListPrice(
	effectRows []byte,
	itemID int,
	mode NativeFacilityPriceMode,
) (int, error) {
	offset := itemID * NativeItemEffectRowSize
	if itemID < 0 || offset < 0 ||
		offset+NativeItemEffectRowSize > len(effectRows) ||
		(mode != NativeFacilityFullPrice &&
			mode != NativeFacilityThreeQuarterPrice) {
		return 0, errors.New("battle: native facility item price state is invalid")
	}
	price := int(binary.LittleEndian.Uint16(effectRows[offset+19:]))
	if mode == NativeFacilityThreeQuarterPrice {
		price = (3 * price) >> 2
	}
	return price, nil
}

// RenderNativeFacilityNumber exposes the already recovered 0x187d6 digit
// primitive to facility scene owners. color is the native frame-bank base
// (31 equal, 42 increased, 119 decreased in the equipment comparison caller).
func RenderNativeFacilityNumber(
	assets NativeItemPanelDataAssets,
	dst []byte,
	x, y, value, colorBase, width int,
) error {
	if len(dst) != nativeItemPanelBytes || width <= 0 {
		return errors.New("battle: native facility number state is invalid")
	}
	return blitNativeItemPanelNumber(
		assets.Frames, dst,
		NativeItemPanelPoint{X: x, Y: y},
		value, colorBase, width,
	)
}

// RenderNativeTransferItemRows reproduces 0x2dc55 mode 1, used by 0x2f8ea.
// itemIDs is the caller's compact list of raw item bytes and start is the
// stateful even viewport origin owned by 0x2df6b.
func RenderNativeTransferItemRows(
	assets NativeItemPanelDataAssets,
	facilityCell fdother.RawCell,
	itemIDs []int,
	start, selected int,
	effectRows []byte,
	dst []byte,
) error {
	return RenderNativeFacilityItemRows(
		assets, facilityCell, itemIDs, start, selected, effectRows,
		NativeFacilityThreeQuarterPrice, dst,
	)
}

// RenderNativeFacilityItemRows reproduces 0x2dc55's shared shop/facility
// list. Mode zero displays row+0x13 unchanged for purchase; nonzero mode
// displays trunc(3*price/4) for sale/transfer callers.
func RenderNativeFacilityItemRows(
	assets NativeItemPanelDataAssets,
	facilityCell fdother.RawCell,
	itemIDs []int,
	start, selected int,
	effectRows []byte,
	priceMode NativeFacilityPriceMode,
	dst []byte,
) error {
	return renderNativeFacilityItemRows(
		assets, facilityCell, itemIDs, start, selected, effectRows, priceMode, dst, true,
	)
}

// RenderNativeFacilityItemRowsWithoutNames 保留原版圖示、數值、價格與選取色，
// 但將名稱安全矩形留白，供已驗證的多語 renderer 寫入。
func RenderNativeFacilityItemRowsWithoutNames(
	assets NativeItemPanelDataAssets,
	facilityCell fdother.RawCell,
	itemIDs []int,
	start, selected int,
	effectRows []byte,
	priceMode NativeFacilityPriceMode,
	dst []byte,
) error {
	return renderNativeFacilityItemRows(
		assets, facilityCell, itemIDs, start, selected, effectRows, priceMode, dst, false,
	)
}

func renderNativeFacilityItemRows(
	assets NativeItemPanelDataAssets,
	facilityCell fdother.RawCell,
	itemIDs []int,
	start, selected int,
	effectRows []byte,
	priceMode NativeFacilityPriceMode,
	dst []byte,
	renderNames bool,
) error {
	if len(dst) != nativeItemPanelBytes || len(itemIDs) == 0 ||
		start < 0 || start%2 != 0 || selected < start ||
		selected >= len(itemIDs) || selected >= start+NativeTransferVisible ||
		(priceMode != NativeFacilityFullPrice &&
			priceMode != NativeFacilityThreeQuarterPrice) {
		return errors.New("battle: native transfer item rows/state are invalid")
	}
	visible := len(itemIDs) - start
	if visible > NativeTransferVisible {
		visible = NativeTransferVisible
	}
	staged := append([]byte(nil), dst...)
	for i := 0; i < visible; i++ {
		itemID := itemIDs[start+i]
		rowOffset := itemID * NativeItemEffectRowSize
		if itemID < 0 || rowOffset < 0 || rowOffset+NativeItemEffectRowSize > len(effectRows) {
			return fmt.Errorf("battle: native transfer item row %d is unavailable", itemID)
		}
		row := effectRows[rowOffset : rowOffset+NativeItemEffectRowSize]
		column, line := i%2, i/2
		x := 10 + 148*column
		y := 119 + 26*line
		category := 59
		switch {
		case row[0] < 0x15:
			category = 59
		case row[0] < 0x20:
			category = 60
		default:
			category = 61
		}
		if err := blitNativeItemPanelRawCell(
			assets.RawCells, category, staged, y*320+x,
		); err != nil {
			return err
		}
		foreground := byte(205)
		if start+i == selected {
			foreground = 201
		}
		if renderNames {
			if err := blitNativeItemPanelText(
				assets.Strings, assets.Font, staged,
				NativeItemPanelPoint{X: x + 28, Y: y + 3},
				itemID+181, foreground,
			); err != nil {
				return err
			}
		}
		statIcon, statValue, hasValue := nativeTransferItemStat(row)
		if statIcon == 41 {
			if err := blitNativeItemPanelDigitFrame(
				assets.Frames, staged, NativeItemPanelPoint{X: x + 95, Y: y + 4}, 41,
			); err != nil {
				return err
			}
		} else {
			if err := blitNativeItemPanelRawCell(
				assets.RawCells, statIcon, staged, (y+2)*320+x+95,
			); err != nil {
				return err
			}
		}
		if hasValue {
			if err := blitNativeItemPanelNumber(
				assets.Frames, staged,
				NativeItemPanelPoint{X: x + 118, Y: y + 2},
				statValue, 42, 3,
			); err != nil {
				return err
			}
		}
		if err := facilityCell.BlitOpaqueAtOffset(
			staged, 320, (y+12)*320+x+95,
		); err != nil {
			return err
		}
		price, err := NativeFacilityItemListPrice(
			effectRows, itemID, priceMode,
		)
		if err != nil {
			return err
		}
		if err := blitNativeItemPanelNumber(
			assets.Frames, staged,
			NativeItemPanelPoint{X: x + 104, Y: y + 12},
			price, 119, 5,
		); err != nil {
			return err
		}
	}
	copy(dst, staged)
	return nil
}

func nativeTransferItemStat(row []byte) (icon, value int, hasValue bool) {
	switch {
	case row[0] < 0x15:
		return 64, int(int16(binary.LittleEndian.Uint16(row[1:3]))), true
	case row[0] < 0x20:
		return 65, int(int16(binary.LittleEndian.Uint16(row[5:7]))), true
	case row[0] == 0x20 && row[13] == 5:
		return 66, int(int16(binary.LittleEndian.Uint16(row[14:16]))), true
	case row[0] == 0x20 && row[13] == 11:
		return 67, int(int16(binary.LittleEndian.Uint16(row[14:16]))), true
	default:
		return 41, 0, false
	}
}
