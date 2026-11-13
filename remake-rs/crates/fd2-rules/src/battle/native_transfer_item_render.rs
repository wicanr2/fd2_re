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
    todo!("battle/native_transfer_item_render.go:NativeFacilityItemListPrice")
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
    todo!("battle/native_transfer_item_render.go:RenderNativeFacilityNumber")
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
    todo!("battle/native_transfer_item_render.go:RenderNativeTransferItemRows")
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
    todo!("battle/native_transfer_item_render.go:RenderNativeFacilityItemRows")
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
    todo!("battle/native_transfer_item_render.go:RenderNativeFacilityItemRowsWithoutNames")
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
    todo!("battle/native_transfer_item_render.go:renderNativeFacilityItemRows")
}

/// Go：`native_transfer_item_render.go:212`，`func nativeTransferItemStat(row []byte) (icon, value int, hasValue bool)`
/// Go 具名回傳值依序為：icon、value、hasValue。
pub(crate) fn native_transfer_item_stat(row: &[u8]) -> (i64, i64, bool) {
    todo!("battle/native_transfer_item_render.go:nativeTransferItemStat")
}
