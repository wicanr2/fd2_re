package battle

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

type NativeItemPanelDataAssets struct {
	BattlePanel fdother.LMI1Entry
	RawCells    map[int]fdother.RawCell
	Frames      map[int]fdother.Frame
	Strings     *fdtxt.Strings
	Font        *fdtxt.Font
}

// LoadNativeBattlePanelValueAssets只解碼0x18C6D框、bar與digit consumer需要的
// FDOTHER#5資料；姓名由另一個具FDTXT／FDOTHER#4 provenance的caller處理時，
// 不應因此強迫載入不會消費的文字資產。
func LoadNativeBattlePanelValueAssets(fdotherPath string) (NativeItemPanelDataAssets, error) {
	raw, err := fdother.ReadResource(fdotherPath, 5)
	if err != nil {
		return NativeItemPanelDataAssets{}, fmt.Errorf("battle: native battle panel FDOTHER#5: %w", err)
	}
	assets := NativeItemPanelDataAssets{
		RawCells: make(map[int]fdother.RawCell),
		Frames:   make(map[int]fdother.Frame),
	}
	assets.BattlePanel, err = fdother.ParseLMI1OpaqueEntry(raw, 22)
	if err != nil {
		return NativeItemPanelDataAssets{}, fmt.Errorf("battle: native battle panel cell 22: %w", err)
	}
	for index := 23; index <= 30; index++ {
		assets.RawCells[index], err = fdother.ParseLMI1RawEntry(raw, index)
		if err != nil {
			return NativeItemPanelDataAssets{}, fmt.Errorf("battle: native battle panel raw cell %d: %w", index, err)
		}
	}
	for index := 31; index <= 52; index++ {
		assets.Frames[index], err = fdother.ParseLMI1FrameEntry(raw, index)
		if err != nil {
			return NativeItemPanelDataAssets{}, fmt.Errorf("battle: native battle panel frame %d: %w", index, err)
		}
	}
	assets.Frames[93], err = fdother.ParseLMI1FrameEntry(raw, 93)
	if err != nil {
		return NativeItemPanelDataAssets{}, fmt.Errorf("battle: native battle panel frame 93: %w", err)
	}
	return assets, nil
}

// RenderNativeItemPanelResources composes 0x17eef and 0x17fc0 as one
// transaction. The DATO resource selector is read from the native unit record
// at +7; no normalized portrait or class field is substituted.
func RenderNativeItemPanelResources(
	fdotherPath, fdtxtPath, datoPath string,
	record, dst []byte,
) error {
	if len(record) < nativeRecordSize {
		return errors.New("battle: native item panel record is shorter than 80 bytes")
	}
	if len(dst) != nativeItemPanelBytes {
		return fmt.Errorf("battle: native item panel destination=%d, want %d", len(dst), nativeItemPanelBytes)
	}
	staged := append([]byte(nil), dst...)
	if err := RenderNativeItemPanelBaseResources(
		fdotherPath, datoPath, int(record[7]), staged,
	); err != nil {
		return err
	}
	assets, err := LoadNativeItemPanelDataAssets(fdotherPath, fdtxtPath)
	if err != nil {
		return err
	}
	if err := RenderNativeItemPanelData(assets, record, staged); err != nil {
		return err
	}
	copy(dst, staged)
	return nil
}

// RenderNativeItemPanelData executes the recovered 0x17fc0 overlay against
// an existing 0x17eef base. It preserves the three distinct native paths:
// opaque raw cells (0x1685c), transparent four-mode frames (0x16886), and
// FDOTHER#4 1bpp glyphs (0x4ea2a).
func RenderNativeItemPanelData(assets NativeItemPanelDataAssets, record, dst []byte) error {
	if len(record) < nativeRecordSize {
		return errors.New("battle: native item panel record is shorter than 80 bytes")
	}
	if len(dst) != nativeItemPanelBytes {
		return fmt.Errorf("battle: native item panel destination=%d, want %d", len(dst), nativeItemPanelBytes)
	}
	if assets.RawCells == nil || assets.Frames == nil || assets.Strings == nil || assets.Font == nil {
		return errors.New("battle: native item panel data assets are incomplete")
	}

	staged := append([]byte(nil), dst...)
	plan := NativeItemPanelDataPlanFor()
	for _, call := range plan.Bars {
		current := int(int16(binary.LittleEndian.Uint16(record[call.CurrentOffset:])))
		maximum := int(int16(binary.LittleEndian.Uint16(record[call.MaximumOffset:])))
		if err := blitNativeItemPanelBar(assets.RawCells, staged, call, current, maximum); err != nil {
			return err
		}
	}
	for _, call := range plan.ComparedNumbers {
		value := int(int16(binary.LittleEndian.Uint16(record[call.ValueOffset:])))
		compare := int(int16(binary.LittleEndian.Uint16(record[call.CompareOffset:])))
		base := 42
		if value == compare {
			base = 31
		}
		if err := blitNativeItemPanelNumber(assets.Frames, staged, call.Destination, value, base, call.Width); err != nil {
			return err
		}
	}
	for _, call := range plan.RawNumbers {
		value := int(record[call.ValueOffset])
		if call.ValueBytes == 2 {
			value = int(int16(binary.LittleEndian.Uint16(record[call.ValueOffset:])))
		}
		base := call.Color
		if call.AlternateFlagOffset != 0 && record[call.AlternateFlagOffset] != 0 {
			base = call.AlternateColor
		}
		if err := blitNativeItemPanelNumber(assets.Frames, staged, call.Destination, value, base, call.Width); err != nil {
			return err
		}
	}
	for _, call := range plan.Text {
		index := int(record[call.RecordOffset]) + call.FDTXTBase
		if err := blitNativeItemPanelText(assets.Strings, assets.Font, staged, call.Destination, index, 205); err != nil {
			return err
		}
	}
	if err := blitNativeItemPanelIcon(assets.RawCells, staged, plan.BaseIcon, record); err != nil {
		return err
	}
	for _, call := range plan.FlagIcons {
		if err := blitNativeItemPanelIcon(assets.RawCells, staged, call, record); err != nil {
			return err
		}
	}
	copy(dst, staged)
	return nil
}

// RenderNativeItemPanelRows executes 0x184c0's compact two-column item list
// over the completed 0x17eef/0x17fc0 panel. selectedRawSlot follows the
// native raw inventory cell; layout compaction remains display-only.
func RenderNativeItemPanelRows(
	assets NativeItemPanelDataAssets,
	record []byte,
	selectedRawSlot int,
	effectRows []byte,
	dst []byte,
) error {
	if len(record) < nativeRecordSize || len(dst) != nativeItemPanelBytes {
		return errors.New("battle: native item panel row inputs are invalid")
	}
	cells, err := NativeItemSelectorCells(record, 0, selectedRawSlot, effectRows)
	if err != nil {
		return err
	}
	staged := append([]byte(nil), dst...)
	for _, cell := range cells {
		category := NativeItemPanelPoint{X: cell.LabelX - 29, Y: cell.LabelY - 2}
		if err := blitNativeItemPanelRawCell(
			assets.RawCells, cell.CategoryIcon, staged,
			category.Y*320+category.X,
		); err != nil {
			return err
		}
		foreground := byte(205)
		if cell.Selected {
			foreground = 201
		}
		if err := blitNativeItemPanelText(
			assets.Strings, assets.Font, staged,
			NativeItemPanelPoint{X: cell.LabelX, Y: cell.LabelY},
			int(cell.ItemID)+181, foreground,
		); err != nil {
			return err
		}
		statPoint := NativeItemPanelPoint{X: cell.LabelX + 68, Y: cell.LabelY + 4}
		if cell.StatIcon == 41 {
			if err := blitNativeItemPanelDigitFrame(assets.Frames, staged, statPoint, 41); err != nil {
				return err
			}
		} else {
			if err := blitNativeItemPanelRawCell(
				assets.RawCells, cell.StatIcon, staged,
				statPoint.Y*320+statPoint.X,
			); err != nil {
				return err
			}
		}
		if cell.HasStatValue {
			if err := blitNativeItemPanelNumber(
				assets.Frames, staged,
				NativeItemPanelPoint{X: cell.LabelX + 93, Y: cell.LabelY + 4},
				cell.StatValue, 42, 3,
			); err != nil {
				return err
			}
		}
	}
	copy(dst, staged)
	return nil
}

func blitNativeItemPanelBar(
	cells map[int]fdother.RawCell,
	dst []byte,
	call NativeItemPanelBarCall,
	current, maximum int,
) error {
	if maximum == 0 {
		return nil
	}
	origin := call.Destination.Y*320 + call.Destination.X
	if current == 0 {
		for x := 1; x <= 101; x++ {
			if err := blitNativeItemPanelRawCell(cells, 29, dst, origin+x); err != nil {
				return err
			}
		}
		return blitNativeItemPanelRawCell(cells, 30, dst, origin+102)
	}
	length := 101*current/maximum + 1
	if err := blitNativeItemPanelRawCell(cells, call.FDOTHERBaseEntry, dst, origin); err != nil {
		return err
	}
	x := 1
	for ; x < length; x++ {
		if err := blitNativeItemPanelRawCell(cells, call.FDOTHERBaseEntry+1, dst, origin+x); err != nil {
			return err
		}
	}
	return blitNativeItemPanelRawCell(cells, call.FDOTHERBaseEntry+2, dst, origin+x)
}

func blitNativeItemPanelRawCell(cells map[int]fdother.RawCell, index int, dst []byte, offset int) error {
	cell, ok := cells[index]
	if !ok {
		return fmt.Errorf("battle: native item panel raw cell %d is unavailable", index)
	}
	if err := cell.BlitOpaqueAtOffset(dst, 320, offset); err != nil {
		return fmt.Errorf("battle: native item panel raw cell %d at %#x: %w", index, offset, err)
	}
	return nil
}

func blitNativeItemPanelNumber(
	frames map[int]fdother.Frame,
	dst []byte,
	destination NativeItemPanelPoint,
	value, base, width int,
) error {
	if value < 0 {
		value = 0
	}
	if width == 3 && value > 999 {
		return blitNativeItemPanelDigitFrame(frames, dst, destination, base+10)
	}
	if width == 2 && value > 99 {
		return blitNativeItemPanelDigitFrame(frames, dst, destination, 93)
	}
	// sub_187D6 is also called with width 8 by sub_1B41D for the
	// battlefield-information currency field.
	if width != 2 && width != 3 && width != 5 && width != 8 {
		return fmt.Errorf("battle: native item panel number width %d is unsupported", width)
	}
	text := fmt.Sprintf("%0*d", width, value)
	for i := 0; i < width; i++ {
		point := destination
		point.X += 6 * i
		if err := blitNativeItemPanelDigitFrame(frames, dst, point, base+int(text[i]-'0')); err != nil {
			return err
		}
	}
	return nil
}

func blitNativeItemPanelDigitFrame(
	frames map[int]fdother.Frame,
	dst []byte,
	destination NativeItemPanelPoint,
	index int,
) error {
	frame, ok := frames[index]
	if !ok {
		return fmt.Errorf("battle: native item panel frame %d is unavailable", index)
	}
	offset := destination.Y*320 + destination.X
	if err := frame.BlitAt(dst, 320, offset, -1); err != nil {
		return fmt.Errorf("battle: native item panel frame %d at %#x: %w", index, offset, err)
	}
	return nil
}

func blitNativeItemPanelText(
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	dst []byte,
	destination NativeItemPanelPoint,
	index int,
	foreground byte,
) error {
	words, err := strings.Words(index)
	if err != nil {
		return fmt.Errorf("battle: native item panel text %d: %w", index, err)
	}
	style := fdtxt.NativeGlyphStyle{Foreground: foreground, Shadow: 76}
	for i, word := range words {
		if word >= fdtxt.ControlMin {
			return fmt.Errorf("battle: native item panel text %d contains control %#x", index, word)
		}
		offset := destination.Y*320 + destination.X + 16*i
		if err := font.BlitNativeGlyph(dst, 320, offset, int(word), style); err != nil {
			return fmt.Errorf("battle: native item panel text %d glyph %d: %w", index, i, err)
		}
	}
	return nil
}

func blitNativeItemPanelIcon(
	cells map[int]fdother.RawCell,
	dst []byte,
	call NativeItemPanelIconCall,
	record []byte,
) error {
	if call.DrawWhenFlagNonzero && record[call.FlagOffset] == 0 {
		return nil
	}
	index := call.FDOTHEREntry
	if call.UseAlternateWhenFlagZero && record[call.FlagOffset] == 0 {
		index = call.AlternateFDOTHEREntry
	}
	return blitNativeItemPanelRawCell(
		cells, index, dst,
		call.Destination.Y*320+call.Destination.X,
	)
}

// LoadNativeItemPanelDataAssets decodes only the FDOTHER/FDTXT resources and
// mixed-codec LMI1 entries proven by 0x17fc0.
func LoadNativeItemPanelDataAssets(fdotherPath, fdtxtPath string) (NativeItemPanelDataAssets, error) {
	raw, err := fdother.ReadResource(fdotherPath, 5)
	if err != nil {
		return NativeItemPanelDataAssets{}, fmt.Errorf("battle: native item panel FDOTHER#5: %w", err)
	}
	assets := NativeItemPanelDataAssets{
		RawCells: make(map[int]fdother.RawCell),
		Frames:   make(map[int]fdother.Frame),
	}
	assets.BattlePanel, err = fdother.ParseLMI1OpaqueEntry(raw, 22)
	if err != nil {
		return NativeItemPanelDataAssets{}, fmt.Errorf("battle: native battle panel cell 22: %w", err)
	}
	for _, index := range []int{
		23, 24, 25, 26, 27, 28, 29, 30,
		53, 54, 55, 56, 57,
		59, 60, 61, 62, 63, 64, 65, 66, 67,
		92,
	} {
		assets.RawCells[index], err = fdother.ParseLMI1RawEntry(raw, index)
		if err != nil {
			return NativeItemPanelDataAssets{}, fmt.Errorf("battle: native item panel raw cell %d: %w", index, err)
		}
	}
	frameIndexes := make([]int, 0, 34)
	for index := 31; index <= 52; index++ {
		frameIndexes = append(frameIndexes, index)
	}
	frameIndexes = append(frameIndexes, 93)
	for index := 119; index <= 129; index++ {
		frameIndexes = append(frameIndexes, index)
	}
	for _, index := range frameIndexes {
		assets.Frames[index], err = fdother.ParseLMI1FrameEntry(raw, index)
		if err != nil {
			return NativeItemPanelDataAssets{}, fmt.Errorf("battle: native item panel frame %d: %w", index, err)
		}
	}
	textResource, err := fdother.ReadResource(fdtxtPath, 0)
	if err != nil {
		return NativeItemPanelDataAssets{}, fmt.Errorf("battle: native item panel FDTXT#0: %w", err)
	}
	assets.Strings, err = fdtxt.Parse(textResource)
	if err != nil {
		return NativeItemPanelDataAssets{}, err
	}
	fontResource, err := fdother.ReadResource(fdotherPath, 4)
	if err != nil {
		return NativeItemPanelDataAssets{}, fmt.Errorf("battle: native item panel FDOTHER#4: %w", err)
	}
	assets.Font, err = fdtxt.ParseFont(fontResource)
	if err != nil {
		return NativeItemPanelDataAssets{}, err
	}
	return assets, nil
}
