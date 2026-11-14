//! Go 來源：`remake/internal/campaign/native_shop_purchase_success.go`（100 行）。

use super::*;

/// Go：`native_shop_purchase_success.go:9`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeShopPurchaseSuccessPlan {
    pub hub_variant: i64,
    pub resource_id: i64,
    pub frame_count: i64,
    pub effect_offset: i64,
    pub pre_delay_bios_ticks: i64,
    pub per_frame_delay_bios_ticks: i64,
    pub post_delay_bios_ticks: i64,
    pub restore_portrait_mode0: bool,
    pub optional_equip_before: bool,
    pub debit_after_presentation: bool,
    pub return_to_product_loop: bool,
}

/// PlanNativeShopPurchaseSuccess preserves the three shop cases in 0x2f4c6.
/// Church case 4 is intentionally owned by PlanNativeChurchReviveSuccess.
///
/// Go：`native_shop_purchase_success.go:25`，`func PlanNativeShopPurchaseSuccess( hubVariant int, ) (NativeShopPurchaseSuccessPlan, error)`
pub fn plan_native_shop_purchase_success(hub_variant: i64) -> Result<NativeShopPurchaseSuccessPlan> {
    todo!("campaign/native_shop_purchase_success.go:PlanNativeShopPurchaseSuccess")
}

/// ComposeNativeShopPurchaseSuccessFrames applies resource entries 23 onward
/// cumulatively through 0x16886. For variants 1/3, 0x16559(0) then restores
/// the current facility portrait; variant 5 has no portrait restore call.
///
/// Go：`native_shop_purchase_success.go:64`，`func ComposeNativeShopPurchaseSuccessFrames( background []byte, shop *NativeShopAssets, portrait dato.Frame, portraitID, hubVariant int, ) (animation [][]byte, final []byte, err error)`
/// Go 具名回傳值依序為：animation、final、err。
pub fn compose_native_shop_purchase_success_frames(
    background: &[u8],
    shop: Option<&NativeShopAssets>,
    portrait: &dato::Frame,
    portrait_id: i64,
    hub_variant: i64,
) -> Result<(Vec<Vec<u8>>, Vec<u8>)> {
    todo!("campaign/native_shop_purchase_success.go:ComposeNativeShopPurchaseSuccessFrames")
}
