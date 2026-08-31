package campaign

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

const NativeShopEquipmentRecipientVisible = 3

type NativeShopEquipmentRecipientRow struct {
	Sprite        fdicon.Sprite
	NameTextIndex int
	Current       [4]int
	Candidate     [4]int
}

// NativeShopEquipmentRecordForUnit materializes the exact 0x50-byte fields
// consumed by 0x2efb7. The general item-panel adapter intentionally omits
// base AP/DP at +0x37/+0x39, so equipment preview must require independently
// initialized equipment-base provenance rather than silently previewing zero.
func NativeShopEquipmentRecordForUnit(unit *battle.Unit) ([]byte, error) {
	if unit == nil || !unit.EquipmentBaseSet ||
		unit.BaseAP < -0x8000 || unit.BaseAP > 0x7fff ||
		unit.BaseDP < -0x8000 || unit.BaseDP > 0x7fff {
		return nil, errors.New(
			"campaign: native shop equipment record lacks base AP/DP provenance",
		)
	}
	record, err := battle.NativeItemPanelRecordForUnit(unit)
	if err != nil {
		return nil, err
	}
	binary.LittleEndian.PutUint16(record[0x37:], uint16(int16(unit.BaseAP)))
	binary.LittleEndian.PutUint16(record[0x39:], uint16(int16(unit.BaseDP)))
	return record, nil
}

// NativeShopEquipmentCandidateStats reproduces 0x2efb7. It previews replacing
// the equipped item in the selected item's <=0x14 or >0x14 category while
// retaining equipped contributions from the opposite category.
func NativeShopEquipmentCandidateStats(
	record []byte,
	itemID int,
	effectRows []byte,
) ([4]int, error) {
	var result [4]int
	const recordSize = 0x50
	if len(record) < recordSize || itemID < 0 {
		return result, errors.New(
			"campaign: native shop equipment preview state is invalid",
		)
	}
	item, err := nativeShopEffectRow(effectRows, itemID)
	if err != nil {
		return result, err
	}
	result = [4]int{
		nativeShopSignedWord(record, 0x37) + nativeShopSignedWord(item, 1),
		nativeShopSignedWord(record, 0x39) + nativeShopSignedWord(item, 5),
		nativeShopSignedWord(record, 0x3e) + nativeShopSignedWord(item, 3),
		nativeShopSignedWord(record, 0x3e) + nativeShopSignedWord(item, 7),
	}
	for slot := 0; slot < 8; slot++ {
		cell := 0x0a + 2*slot
		if record[cell]&0x40 == 0 {
			continue
		}
		equipped, err := nativeShopEffectRow(effectRows, int(record[cell+1]))
		if err != nil {
			return [4]int{}, err
		}
		if (item[0] <= 0x14) == (equipped[0] <= 0x14) {
			continue
		}
		result[0] += nativeShopSignedWord(equipped, 1)
		result[1] += nativeShopSignedWord(equipped, 5)
		result[2] += nativeShopSignedWord(equipped, 3)
		result[3] += nativeShopSignedWord(equipped, 7)
	}
	return result, nil
}

func NativeShopEquipmentCurrentStats(record []byte) ([4]int, error) {
	if len(record) < 0x50 {
		return [4]int{}, errors.New(
			"campaign: native shop equipment record is shorter than 80 bytes",
		)
	}
	return [4]int{
		int(int16(binary.LittleEndian.Uint16(record[0x48:]))),
		int(int16(binary.LittleEndian.Uint16(record[0x4a:]))),
		int(int16(binary.LittleEndian.Uint16(record[0x4c:]))),
		int(int16(binary.LittleEndian.Uint16(record[0x4e:]))),
	}, nil
}

// ComposeNativeShopEquipmentRecipientFrame reproduces the stable
// 0x2e8cf→0x2ebe0 target: three rows, selected FDTXT name color, and AP/DP/
// HIT/EV current→candidate comparisons. Opening/closing remain caller-owned.
func ComposeNativeShopEquipmentRecipientFrame(
	stable []byte,
	shop *NativeShopAssets,
	itemAssets battle.NativeItemPanelDataAssets,
	rows []NativeShopEquipmentRecipientRow,
	selected int,
) ([]byte, error) {
	return composeNativeShopEquipmentRecipientFrame(stable, shop, itemAssets, rows, selected, true)
}

func ComposeNativeShopEquipmentRecipientFrameWithoutNames(
	stable []byte,
	shop *NativeShopAssets,
	itemAssets battle.NativeItemPanelDataAssets,
	rows []NativeShopEquipmentRecipientRow,
	selected int,
) ([]byte, error) {
	return composeNativeShopEquipmentRecipientFrame(stable, shop, itemAssets, rows, selected, false)
}

func composeNativeShopEquipmentRecipientFrame(
	stable []byte,
	shop *NativeShopAssets,
	itemAssets battle.NativeItemPanelDataAssets,
	rows []NativeShopEquipmentRecipientRow,
	selected int,
	renderNames bool,
) ([]byte, error) {
	if len(stable) != NativeShopWidth*NativeShopHeight || shop == nil ||
		len(rows) == 0 || len(rows) > NativeShopEquipmentRecipientVisible ||
		selected < 0 || selected >= len(rows) ||
		(renderNames && (itemAssets.Strings == nil || itemAssets.Font == nil)) {
		return nil, errors.New(
			"campaign: native shop equipment recipient state is invalid",
		)
	}
	frame := append([]byte(nil), stable...)
	if err := shop.Panel.BlitOpaqueAt(frame, NativeShopWidth, 5, 112, false); err != nil {
		return nil, err
	}
	for i, row := range rows {
		y := 117 + 26*i
		if err := row.Sprite.BlitAt(frame, NativeShopWidth, 14, y); err != nil {
			return nil, fmt.Errorf("campaign: equipment recipient row %d sprite: %w", i, err)
		}
		foreground := byte(205)
		if i == selected {
			foreground = 201
		}
		if renderNames {
			if err := blitNativeClassListText(
				frame, itemAssets.Strings, itemAssets.Font,
				40, y+4, row.NameTextIndex, foreground,
			); err != nil {
				return nil, fmt.Errorf("campaign: equipment recipient row %d name: %w", i, err)
			}
		}
		positions := [4]struct {
			cell                                     int
			x, dy                                    int
			currentOffset, arrowOffset, resultOffset int
		}{
			{0, 122, 3, 15, 35, 43},
			{1, 122, 12, 15, 35, 43},
			{2, 196, 3, 18, 38, 46},
			{3, 196, 12, 18, 38, 46},
		}
		for stat, position := range positions {
			if err := shop.CompareCells[position.cell].BlitOpaqueAt(
				frame, NativeShopWidth, position.x, y+position.dy, false,
			); err != nil {
				return nil, err
			}
			colorBase := nativeShopComparisonColor(
				row.Current[stat], row.Candidate[stat],
			)
			if err := battle.RenderNativeFacilityNumber(
				itemAssets, frame, position.x+position.currentOffset, y+position.dy,
				row.Current[stat], colorBase, 3,
			); err != nil {
				return nil, err
			}
			arrowX := position.x + position.arrowOffset
			arrowY := y + position.dy + 1
			if err := shop.CompareCells[4].BlitOpaqueAt(
				frame, NativeShopWidth, arrowX, arrowY, false,
			); err != nil {
				return nil, err
			}
			if err := battle.RenderNativeFacilityNumber(
				itemAssets, frame, position.x+position.resultOffset, y+position.dy,
				row.Candidate[stat], colorBase, 3,
			); err != nil {
				return nil, err
			}
		}
	}
	return frame, nil
}

func nativeShopComparisonColor(current, candidate int) int {
	switch {
	case current == candidate:
		return 31
	case current < candidate:
		return 42
	default:
		return 119
	}
}

func nativeShopEffectRow(effectRows []byte, itemID int) ([]byte, error) {
	start := itemID * battle.NativeItemEffectRowSize
	if itemID < 0 || start < 0 ||
		start+battle.NativeItemEffectRowSize > len(effectRows) {
		return nil, fmt.Errorf(
			"campaign: native shop item row %d is unavailable", itemID,
		)
	}
	return effectRows[start : start+battle.NativeItemEffectRowSize], nil
}

func nativeShopSignedWord(raw []byte, offset int) int {
	return int(int16(binary.LittleEndian.Uint16(raw[offset:])))
}
