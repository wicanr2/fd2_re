package battle

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const (
	nativeBattlePanelWidth  = 149
	nativeBattlePanelHeight = 42
)

// NativeBattlePanelValues 是 0x18C6D 直接交給 bar／digit consumers 的可見值。
// 它不包含 raw +6／+8 selector；caller 若缺原始 selector，不得藉此宣稱完整
// RenderNativeBattlePanel parity。
type NativeBattlePanelValues struct {
	Level     int
	HP, MaxHP int
	MP, MaxMP int
}

// RenderNativeBattlePanelValuesAt 重用 0x18C6D 的框、bar與digit核心，但不畫姓名。
// 這讓已有可見值與明確panel原點的E1 caller避免另寫一套RGBA近似；完整raw caller
// 仍應使用 RenderNativeBattlePanel。
func RenderNativeBattlePanelValuesAt(
	assets NativeItemPanelDataAssets,
	dst []byte,
	x, y int,
	values NativeBattlePanelValues,
) error {
	if len(dst) != nativeItemPanelBytes || x < 0 || y < 0 ||
		x+nativeBattlePanelWidth > 320 || y+nativeBattlePanelHeight > 200 ||
		assets.BattlePanel.Width != nativeBattlePanelWidth ||
		assets.BattlePanel.Height != nativeBattlePanelHeight ||
		assets.RawCells == nil || assets.Frames == nil {
		return errors.New("battle: native battle panel value inputs are invalid")
	}
	staged := append([]byte(nil), dst...)
	if err := renderNativeBattlePanelValuesAt(assets, staged, x, y, values); err != nil {
		return err
	}
	copy(dst, staged)
	return nil
}

func renderNativeBattlePanelValuesAt(
	assets NativeItemPanelDataAssets,
	dst []byte,
	x, y int,
	values NativeBattlePanelValues,
) error {
	if err := assets.BattlePanel.BlitOpaqueAt(dst, 320, x, y, false); err != nil {
		return fmt.Errorf("battle: native battle panel base: %w", err)
	}
	for _, call := range []struct {
		point        NativeItemPanelPoint
		current, max int
		base         int
	}{
		{NativeItemPanelPoint{x + 21, y + 22}, values.HP, values.MaxHP, 23},
		{NativeItemPanelPoint{x + 21, y + 31}, values.MP, values.MaxMP, 26},
	} {
		if err := blitNativeItemPanelBar(assets.RawCells, dst, NativeItemPanelBarCall{
			Destination: call.point, FDOTHERBaseEntry: call.base,
		}, call.current, call.max); err != nil {
			return err
		}
	}
	if err := blitNativeItemPanelNumber(assets.Frames, dst,
		NativeItemPanelPoint{x + 132, y + 4}, values.Level, 31, 2); err != nil {
		return err
	}
	for _, call := range []struct {
		point        NativeItemPanelPoint
		current, max int
	}{
		{NativeItemPanelPoint{x + 126, y + 21}, values.HP, values.MaxHP},
		{NativeItemPanelPoint{x + 126, y + 30}, values.MP, values.MaxMP},
	} {
		base := 42
		if call.current == call.max {
			base = 31
		}
		if err := blitNativeItemPanelNumber(assets.Frames, dst, call.point, call.current, base, 3); err != nil {
			return err
		}
	}
	return nil
}

// NativeBattlePanelRecordForUnit materializes only the raw fields consumed by
// 0x2A289→0x18C6D. Raw +6 and +8 provenance remain mandatory; normalized Camp,
// Name, Portrait, and BattleFig are not substitutes for either selector.
func NativeBattlePanelRecordForUnit(unit *Unit) ([]byte, error) {
	if unit == nil || !unit.HasNativeRecordByte6 {
		return nil, errors.New("native battle panel: unit lacks raw record byte +6")
	}
	rawByte8, ok := nativeRecordByte8ForUnit(unit)
	if !ok {
		return nil, errors.New("native battle panel: unit lacks raw record byte +8")
	}
	if unit.Lv < 0 || unit.Lv > 0xff {
		return nil, errors.New("native battle panel: level is outside native byte range")
	}
	for _, value := range []int{unit.HP, unit.MaxHP, unit.MP, unit.MaxMP} {
		if value < math.MinInt16 || value > math.MaxInt16 {
			return nil, errors.New("native battle panel: HP/MP field is outside native word range")
		}
	}
	record := make([]byte, nativeRecordSize)
	record[6] = unit.NativeRecordByte6
	record[8] = rawByte8
	record[0x21] = byte(unit.Lv)
	putNativeItemPanelWord(record, 0x40, unit.HP)
	putNativeItemPanelWord(record, 0x42, unit.MaxHP)
	putNativeItemPanelWord(record, 0x44, unit.MP)
	putNativeItemPanelWord(record, 0x46, unit.MaxMP)
	return record, nil
}

// RenderNativeBattlePanel reproduces the complete narrow 0x2A289→0x18C6D
// indexed compositor. It stages the full 320x200 destination so missing raw
// provenance, assets, text, or geometry never leaves a partially drawn panel.
func RenderNativeBattlePanel(
	assets NativeItemPanelDataAssets,
	record, dst []byte,
	unitIndex, rawChapter int,
) error {
	if len(record) < nativeRecordSize || len(dst) != nativeItemPanelBytes {
		return errors.New("battle: native battle panel inputs are invalid")
	}
	if unitIndex < 0 || rawChapter < 0 || rawChapter > 0xff ||
		assets.BattlePanel.Width != nativeBattlePanelWidth ||
		assets.BattlePanel.Height != nativeBattlePanelHeight ||
		assets.RawCells == nil || assets.Frames == nil || assets.Strings == nil || assets.Font == nil {
		return errors.New("battle: native battle panel assets or selectors are invalid")
	}
	x, y := 171, 4
	if record[6] == 0 || rawChapter == 24 && unitIndex == 17 {
		x, y = 0, 154
	}
	staged := append([]byte(nil), dst...)
	hp := int(int16(binary.LittleEndian.Uint16(record[0x40:])))
	maxHP := int(int16(binary.LittleEndian.Uint16(record[0x42:])))
	mp := int(int16(binary.LittleEndian.Uint16(record[0x44:])))
	maxMP := int(int16(binary.LittleEndian.Uint16(record[0x46:])))
	if err := renderNativeBattlePanelValuesAt(assets, staged, x, y, NativeBattlePanelValues{
		Level: int(record[0x21]), HP: hp, MaxHP: maxHP, MP: mp, MaxMP: maxMP,
	}); err != nil {
		return err
	}
	if err := blitNativeItemPanelText(assets.Strings, assets.Font, staged,
		NativeItemPanelPoint{x + 5, y + 4}, int(record[8])+1, 205); err != nil {
		return err
	}
	copy(dst, staged)
	return nil
}
