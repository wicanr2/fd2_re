//! Go 來源：`remake/internal/battle/native_map_unit_window.go`（61 行）。

use super::*;

// native_map_unit_window.go — 0x18B84 選單位／挑目的地時的單位資訊視窗（0x18C6D）。
//
// 0x18B84 每一幀在前景之後、HUD 之前呼叫 `0x18C6D(work+0x8088+off, 0x1C8, unit)`：
// `[0x53AB9]`（可見游標 x）< 7 時 off = 0x984（viewport 相對 (156,5)），否則
// off = 0x8ED（(5,5)）。視窗內容與全螢幕戰鬥的 0x2A289→0x18C6D 同一套：框（FDOTHER#5
// cell 22）、HP／MP 長條、LV／HP／MP 數字、姓名（FDTXT #0 第 `+8`+1 句，前景 0xCD）。

pub(crate) const NATIVE_MAP_UNIT_WINDOW_RIGHT_X: i64 = 156;

pub(crate) const NATIVE_MAP_UNIT_WINDOW_LEFT_X: i64 = 5;

pub(crate) const NATIVE_MAP_UNIT_WINDOW_Y: i64 = 5;

/// NativeMapUnitWindowOrigin 回傳 0x18B84 決定的 viewport 相對原點。
///
/// Go：`native_map_unit_window.go:22`，`func NativeMapUnitWindowOrigin(visibleCursorX int) (int, int)`
pub fn native_map_unit_window_origin(visible_cursor_x: i64) -> (i64, i64) {
    todo!("battle/native_map_unit_window.go:NativeMapUnitWindowOrigin")
}

/// RenderNativeMapUnitWindow 把 0x18C6D 的視窗畫進 viewport 原點為 dst[0]、列距 stride
/// 的工作緩衝。record 是該單位的 0x50 位元組（NativeItemPanelRecordForUnit）。
///
/// Go：`native_map_unit_window.go:31`，`func RenderNativeMapUnitWindow(assets NativeItemPanelDataAssets, record, dst []byte, stride, visibleCursorX int) error`
pub fn render_native_map_unit_window(
    assets: &NativeItemPanelDataAssets,
    record: &[u8],
    dst: &mut [u8],
    stride: i64,
    visible_cursor_x: i64,
) -> Result<()> {
    todo!("battle/native_map_unit_window.go:RenderNativeMapUnitWindow")
}
