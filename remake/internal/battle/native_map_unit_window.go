package battle

import (
	"errors"
	"fmt"
)

// native_map_unit_window.go — 0x18B84 選單位／挑目的地時的單位資訊視窗（0x18C6D）。
//
// 0x18B84 每一幀在前景之後、HUD 之前呼叫 `0x18C6D(work+0x8088+off, 0x1C8, unit)`：
// `[0x53AB9]`（可見游標 x）< 7 時 off = 0x984（viewport 相對 (156,5)），否則
// off = 0x8ED（(5,5)）。視窗內容與全螢幕戰鬥的 0x2A289→0x18C6D 同一套：框（FDOTHER#5
// cell 22）、HP／MP 長條、LV／HP／MP 數字、姓名（FDTXT #0 第 `+8`+1 句，前景 0xCD）。

const (
	nativeMapUnitWindowRightX = 156
	nativeMapUnitWindowLeftX  = 5
	nativeMapUnitWindowY      = 5
)

// NativeMapUnitWindowOrigin 回傳 0x18B84 決定的 viewport 相對原點。
func NativeMapUnitWindowOrigin(visibleCursorX int) (int, int) {
	if visibleCursorX < 7 {
		return nativeMapUnitWindowRightX, nativeMapUnitWindowY
	}
	return nativeMapUnitWindowLeftX, nativeMapUnitWindowY
}

// RenderNativeMapUnitWindow 把 0x18C6D 的視窗畫進 viewport 原點為 dst[0]、列距 stride
// 的工作緩衝。record 是該單位的 0x50 位元組（NativeItemPanelRecordForUnit）。
func RenderNativeMapUnitWindow(assets NativeItemPanelDataAssets, record, dst []byte, stride, visibleCursorX int) error {
	if len(record) < nativeRecordSize || stride < nativeBattlePanelWidth || assets.Strings == nil || assets.Font == nil {
		return errors.New("battle: native map unit window inputs are invalid")
	}
	x, y := NativeMapUnitWindowOrigin(visibleCursorX)
	if (y+nativeBattlePanelHeight)*stride > len(dst) || x+nativeBattlePanelWidth > stride {
		return errors.New("battle: native map unit window is outside the work buffer")
	}
	// 既有的 0x18C6D 核心以 320×200 為畫布；先畫在原點，再把 149×42 的不透明框
	// 整塊搬到工作緩衝的目標位置。
	scratch := make([]byte, nativeItemPanelBytes)
	values := NativeBattlePanelValues{
		Level: int(record[0x21]),
		HP:    int(int16(uint16(record[0x40]) | uint16(record[0x41])<<8)),
		MaxHP: int(int16(uint16(record[0x42]) | uint16(record[0x43])<<8)),
		MP:    int(int16(uint16(record[0x44]) | uint16(record[0x45])<<8)),
		MaxMP: int(int16(uint16(record[0x46]) | uint16(record[0x47])<<8)),
	}
	if err := renderNativeBattlePanelValuesAt(assets, scratch, 0, 0, values); err != nil {
		return err
	}
	if err := blitNativeItemPanelText(assets.Strings, assets.Font, scratch,
		NativeItemPanelPoint{5, 4}, int(record[8])+1, 205); err != nil {
		return fmt.Errorf("battle: native map unit window name: %w", err)
	}
	for row := 0; row < nativeBattlePanelHeight; row++ {
		copy(dst[(y+row)*stride+x:(y+row)*stride+x+nativeBattlePanelWidth],
			scratch[row*320:row*320+nativeBattlePanelWidth])
	}
	return nil
}
