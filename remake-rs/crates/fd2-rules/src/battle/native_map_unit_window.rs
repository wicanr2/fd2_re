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
    if visible_cursor_x < 7 {
        return (NATIVE_MAP_UNIT_WINDOW_RIGHT_X, NATIVE_MAP_UNIT_WINDOW_Y);
    }
    (NATIVE_MAP_UNIT_WINDOW_LEFT_X, NATIVE_MAP_UNIT_WINDOW_Y)
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
    if (record.len() as i64) < NATIVE_RECORD_SIZE
        || stride < NATIVE_BATTLE_PANEL_WIDTH
        || assets.strings.is_none()
        || assets.font.is_none()
    {
        return Err(battle_err!("battle: native map unit window inputs are invalid"));
    }
    let (x, y) = native_map_unit_window_origin(visible_cursor_x);
    if (y + NATIVE_BATTLE_PANEL_HEIGHT) * stride > dst.len() as i64 || x + NATIVE_BATTLE_PANEL_WIDTH > stride {
        return Err(battle_err!("battle: native map unit window is outside the work buffer"));
    }
    // 既有的 0x18C6D 核心以 320×200 為畫布；先畫在原點，再把 149×42 的不透明框
    // 整塊搬到工作緩衝的目標位置。
    let mut scratch = vec![0u8; NATIVE_ITEM_PANEL_BYTES as usize];
    let word = |off: usize| u16::from_le_bytes([record[off], record[off + 1]]) as i16 as i64;
    let values = NativeBattlePanelValues {
        level: record[0x21] as i64,
        hp: word(0x40),
        max_hp: word(0x42),
        mp: word(0x44),
        max_mp: word(0x46),
    };
    render_native_battle_panel_values_at_inner(assets, &mut scratch, 0, 0, values)?;
    blit_native_item_panel_text_opt(
        assets.strings.as_ref(),
        assets.font.as_ref(),
        &mut scratch,
        NativeItemPanelPoint { x: 5, y: 4 },
        record[8] as i64 + 1,
        205,
    )
    .map_err(|e| e.wrap("battle: native map unit window name: "))?;
    let (x, y, stride, width) = (x as usize, y as usize, stride as usize, NATIVE_BATTLE_PANEL_WIDTH as usize);
    for row in 0..NATIVE_BATTLE_PANEL_HEIGHT as usize {
        let at = (y + row) * stride + x;
        dst[at..at + width].copy_from_slice(&scratch[row * 320..row * 320 + width]);
    }
    Ok(())
}
