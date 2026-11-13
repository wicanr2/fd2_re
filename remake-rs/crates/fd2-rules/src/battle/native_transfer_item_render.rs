//! Go 來源：`remake/internal/battle/native_transfer_item_render.go`（225 行）。

use super::*;

pub const NATIVE_TRANSFER_VISIBLE: i64 = 6;

/// Go：`native_transfer_item_render.go:13`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeFacilityPriceMode(pub u8);

/// Go 原式：`iota`
pub const NATIVE_FACILITY_FULL_PRICE: NativeFacilityPriceMode = NativeFacilityPriceMode(0);

pub const NATIVE_FACILITY_THREE_QUARTER_PRICE: NativeFacilityPriceMode = NativeFacilityPriceMode(1);

/// Go：`native_transfer_item_render.go:20`，`func NativeFacilityItemListPrice( effectRows []byte, itemID int, mode NativeFacilityPriceMode, ) (int, error)`
pub fn native_facility_item_list_price(effect_rows: &[u8], item_id: i64, mode: NativeFacilityPriceMode) -> Result<i64> {
    let offset = item_id.wrapping_mul(NATIVE_ITEM_EFFECT_ROW_SIZE);
    if item_id < 0
        || offset < 0
        || offset + NATIVE_ITEM_EFFECT_ROW_SIZE > effect_rows.len() as i64
        || (mode != NATIVE_FACILITY_FULL_PRICE && mode != NATIVE_FACILITY_THREE_QUARTER_PRICE)
    {
        return Err(battle_err!("battle: native facility item price state is invalid"));
    }
    let at = offset as usize + 19;
    let mut price = u16::from_le_bytes([effect_rows[at], effect_rows[at + 1]]) as i64;
    if mode == NATIVE_FACILITY_THREE_QUARTER_PRICE {
        price = (3 * price) >> 2;
    }
    Ok(price)
}

/// RenderNativeFacilityNumber exposes the already recovered 0x187d6 digit
/// primitive to facility scene owners. color is the native frame-bank base
/// (31 equal, 42 increased, 119 decreased in the equipment comparison caller).
///
/// Go：`native_transfer_item_render.go:42`，`func RenderNativeFacilityNumber( assets NativeItemPanelDataAssets, dst []byte, x, y, value, colorBase, width int, ) error`
pub fn render_native_facility_number(
    assets: &NativeItemPanelDataAssets,
    dst: &mut [u8],
    x: i64,
    y: i64,
    value: i64,
    color_base: i64,
    width: i64,
) -> Result<()> {
    if dst.len() as i64 != NATIVE_ITEM_PANEL_BYTES || width <= 0 {
        return Err(battle_err!("battle: native facility number state is invalid"));
    }
    blit_native_item_panel_number(
        assets.frames_or_empty(),
        dst,
        NativeItemPanelPoint { x, y },
        value,
        color_base,
        width,
    )
}

/// RenderNativeTransferItemRows reproduces 0x2dc55 mode 1, used by 0x2f8ea.
/// itemIDs is the caller's compact list of raw item bytes and start is the
/// stateful even viewport origin owned by 0x2df6b.
///
/// Go：`native_transfer_item_render.go:60`，`func RenderNativeTransferItemRows( assets NativeItemPanelDataAssets, facilityCell fdother.RawCell, itemIDs []int, start, selected int, effectRows []byte, dst []byte, ) error`
pub fn render_native_transfer_item_rows(
    assets: &NativeItemPanelDataAssets,
    facility_cell: &fdother::RawCell,
    item_ids: &[i64],
    start: i64,
    selected: i64,
    effect_rows: &[u8],
    dst: &mut [u8],
) -> Result<()> {
    render_native_facility_item_rows(
        assets,
        facility_cell,
        item_ids,
        start,
        selected,
        effect_rows,
        NATIVE_FACILITY_THREE_QUARTER_PRICE,
        dst,
    )
}

/// RenderNativeFacilityItemRows reproduces 0x2dc55's shared shop/facility
/// list. Mode zero displays row+0x13 unchanged for purchase; nonzero mode
/// displays trunc(3*price/4) for sale/transfer callers.
///
/// Go：`native_transfer_item_render.go:77`，`func RenderNativeFacilityItemRows( assets NativeItemPanelDataAssets, facilityCell fdother.RawCell, itemIDs []int, start, selected int, effectRows []byte, priceMode NativeFacilityPriceMode, dst []byte, ) error`
pub fn render_native_facility_item_rows(
    assets: &NativeItemPanelDataAssets,
    facility_cell: &fdother::RawCell,
    item_ids: &[i64],
    start: i64,
    selected: i64,
    effect_rows: &[u8],
    price_mode: NativeFacilityPriceMode,
    dst: &mut [u8],
) -> Result<()> {
    render_native_facility_item_rows_inner(
        assets,
        facility_cell,
        item_ids,
        start,
        selected,
        effect_rows,
        price_mode,
        dst,
        true,
    )
}

/// RenderNativeFacilityItemRowsWithoutNames 保留原版圖示、數值、價格與選取色，
/// 但將名稱安全矩形留白，供已驗證的多語 renderer 寫入。
///
/// Go：`native_transfer_item_render.go:93`，`func RenderNativeFacilityItemRowsWithoutNames( assets NativeItemPanelDataAssets, facilityCell fdother.RawCell, itemIDs []int, start, selected int, effectRows []byte, priceMode NativeFacilityPriceMode, dst []byte, ) error`
pub fn render_native_facility_item_rows_without_names(
    assets: &NativeItemPanelDataAssets,
    facility_cell: &fdother::RawCell,
    item_ids: &[i64],
    start: i64,
    selected: i64,
    effect_rows: &[u8],
    price_mode: NativeFacilityPriceMode,
    dst: &mut [u8],
) -> Result<()> {
    render_native_facility_item_rows_inner(
        assets,
        facility_cell,
        item_ids,
        start,
        selected,
        effect_rows,
        price_mode,
        dst,
        false,
    )
}

/// Go：`native_transfer_item_render.go:107`，`func renderNativeFacilityItemRows( assets NativeItemPanelDataAssets, facilityCell fdother.RawCell, itemIDs []int, start, selected int, effectRows []byte, priceMode NativeFacilityPriceMode, dst []byte, renderNames bool, ) error`
pub(crate) fn render_native_facility_item_rows_inner(
    assets: &NativeItemPanelDataAssets,
    facility_cell: &fdother::RawCell,
    item_ids: &[i64],
    start: i64,
    selected: i64,
    effect_rows: &[u8],
    price_mode: NativeFacilityPriceMode,
    dst: &mut [u8],
    render_names: bool,
) -> Result<()> {
    if dst.len() as i64 != NATIVE_ITEM_PANEL_BYTES
        || item_ids.is_empty()
        || start < 0
        || start % 2 != 0
        || selected < start
        || selected >= item_ids.len() as i64
        || selected >= start + NATIVE_TRANSFER_VISIBLE
        || (price_mode != NATIVE_FACILITY_FULL_PRICE && price_mode != NATIVE_FACILITY_THREE_QUARTER_PRICE)
    {
        return Err(battle_err!("battle: native transfer item rows/state are invalid"));
    }
    let visible = (item_ids.len() as i64 - start).min(NATIVE_TRANSFER_VISIBLE);
    let (raw_cells, frames) = (assets.raw_cells_or_empty(), assets.frames_or_empty());
    let row_size = NATIVE_ITEM_EFFECT_ROW_SIZE;
    let mut staged = dst.to_vec();
    for i in 0..visible {
        let item_id = item_ids[(start + i) as usize];
        let row_offset = item_id.wrapping_mul(row_size);
        if item_id < 0 || row_offset < 0 || row_offset + row_size > effect_rows.len() as i64 {
            return Err(battle_err!("battle: native transfer item row {item_id} is unavailable"));
        }
        let row = &effect_rows[row_offset as usize..(row_offset + row_size) as usize];
        let (column, line) = (i % 2, i / 2);
        let x = 10 + 148 * column;
        let y = 119 + 26 * line;
        let category = if row[0] < 0x15 {
            59
        } else if row[0] < 0x20 {
            60
        } else {
            61
        };
        blit_native_item_panel_raw_cell(raw_cells, category, &mut staged, y * 320 + x)?;
        let foreground = if start + i == selected { 201 } else { 205 };
        if render_names {
            blit_native_item_panel_text_opt(
                assets.strings.as_ref(),
                assets.font.as_ref(),
                &mut staged,
                NativeItemPanelPoint { x: x + 28, y: y + 3 },
                item_id + 181,
                foreground,
            )?;
        }
        let (stat_icon, stat_value, has_value) = native_transfer_item_stat(row);
        if stat_icon == 41 {
            blit_native_item_panel_digit_frame(frames, &mut staged, NativeItemPanelPoint { x: x + 95, y: y + 4 }, 41)?;
        } else {
            blit_native_item_panel_raw_cell(raw_cells, stat_icon, &mut staged, (y + 2) * 320 + x + 95)?;
        }
        if has_value {
            blit_native_item_panel_number(
                frames,
                &mut staged,
                NativeItemPanelPoint { x: x + 118, y: y + 2 },
                stat_value,
                42,
                3,
            )?;
        }
        facility_cell.blit_opaque_at_offset(&mut staged, 320, (y + 12) * 320 + x + 95)?;
        let price = native_facility_item_list_price(effect_rows, item_id, price_mode)?;
        blit_native_item_panel_number(
            frames,
            &mut staged,
            NativeItemPanelPoint { x: x + 104, y: y + 12 },
            price,
            119,
            5,
        )?;
    }
    dst.copy_from_slice(&staged);
    Ok(())
}

/// Go：`native_transfer_item_render.go:212`，`func nativeTransferItemStat(row []byte) (icon, value int, hasValue bool)`
/// Go 具名回傳值依序為：icon、value、hasValue。
pub(crate) fn native_transfer_item_stat(row: &[u8]) -> (i64, i64, bool) {
    let word = |off: usize| u16::from_le_bytes([row[off], row[off + 1]]) as i16 as i64;
    match (row[0], row[13]) {
        (t, _) if t < 0x15 => (64, word(1), true),
        (t, _) if t < 0x20 => (65, word(5), true),
        (0x20, 5) => (66, word(14), true),
        (0x20, 11) => (67, word(14), true),
        _ => (41, 0, false),
    }
}

#[cfg(test)]
mod native_transfer_item_render_test;
