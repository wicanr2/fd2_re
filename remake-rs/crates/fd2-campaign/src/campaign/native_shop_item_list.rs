//! Go 來源：`remake/internal/campaign/native_shop_item_list.go`（71 行）。

use super::*;

/// ComposeNativeShopItemListFrame reproduces 0x2e0bd's completed child-panel
/// target: shop resource entry16 at (5,112), followed by 0x2dc55's two-column
/// item rows and caller-selected price mode. Opening/closing band timing stays
/// outside this stable-frame compositor.
///
/// Go：`native_shop_item_list.go:13`，`func ComposeNativeShopItemListFrame( stable []byte, shop *NativeShopAssets, itemAssets battle.NativeItemPanelDataAssets, itemIDs []int, start, selected int, effectRows []byte, priceMode battle.NativeFacilityPriceMode, ) ([]byte, error)`
pub fn compose_native_shop_item_list_frame(
    stable: &[u8],
    shop: Option<&NativeShopAssets>,
    item_assets: &battle::NativeItemPanelDataAssets,
    item_ids: &[i64],
    start: i64,
    selected: i64,
    effect_rows: &[u8],
    price_mode: battle::NativeFacilityPriceMode,
) -> Result<Vec<u8>> {
    compose_native_shop_item_list_frame_inner(
        stable,
        shop,
        item_assets,
        item_ids,
        start,
        selected,
        effect_rows,
        price_mode,
        true,
    )
}

/// ComposeNativeShopItemListFrameWithoutNames 建立同一原版商品列，但保留名稱矩形，
/// 讓正式語系層用 item ID 對應的文字覆蓋。
///
/// Go：`native_shop_item_list.go:29`，`func ComposeNativeShopItemListFrameWithoutNames( stable []byte, shop *NativeShopAssets, itemAssets battle.NativeItemPanelDataAssets, itemIDs []int, start, selected int, effectRows []byte, priceMode battle.NativeFacilityPriceMode, ) ([]byte, error)`
pub fn compose_native_shop_item_list_frame_without_names(
    stable: &[u8],
    shop: Option<&NativeShopAssets>,
    item_assets: &battle::NativeItemPanelDataAssets,
    item_ids: &[i64],
    start: i64,
    selected: i64,
    effect_rows: &[u8],
    price_mode: battle::NativeFacilityPriceMode,
) -> Result<Vec<u8>> {
    compose_native_shop_item_list_frame_inner(
        stable,
        shop,
        item_assets,
        item_ids,
        start,
        selected,
        effect_rows,
        price_mode,
        false,
    )
}

/// Go：`native_shop_item_list.go:43`，`func composeNativeShopItemListFrame( stable []byte, shop *NativeShopAssets, itemAssets battle.NativeItemPanelDataAssets, itemIDs []int, start, selected int, effectRows []byte, priceMode battle.NativeFacilityPriceMode, renderNames bool, ) ([]byte, error)`
pub(crate) fn compose_native_shop_item_list_frame_inner(
    stable: &[u8],
    shop: Option<&NativeShopAssets>,
    item_assets: &battle::NativeItemPanelDataAssets,
    item_ids: &[i64],
    start: i64,
    selected: i64,
    effect_rows: &[u8],
    price_mode: battle::NativeFacilityPriceMode,
    render_names: bool,
) -> Result<Vec<u8>> {
    let shop = match shop {
        Some(s) if stable.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT => s,
        _ => return Err(campaign_err!("campaign: native shop item-list state is invalid")),
    };
    let mut frame = stable.to_vec();
    shop.panel.blit_opaque_at(&mut frame, NATIVE_SHOP_WIDTH, 5, 112, false)?;
    let render = if render_names {
        battle::render_native_facility_item_rows
    } else {
        battle::render_native_facility_item_rows_without_names
    };
    render(item_assets, &shop.price_cell, item_ids, start, selected, effect_rows, price_mode, &mut frame)?;
    Ok(frame)
}
