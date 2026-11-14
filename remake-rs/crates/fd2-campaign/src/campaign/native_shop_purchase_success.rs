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
    match hub_variant {
        1 => Ok(NativeShopPurchaseSuccessPlan {
            hub_variant: 1,
            resource_id: 12,
            frame_count: 5,
            effect_offset: 45 * NATIVE_SHOP_WIDTH + 169,
            per_frame_delay_bios_ticks: 2,
            restore_portrait_mode0: true,
            optional_equip_before: true,
            debit_after_presentation: true,
            return_to_product_loop: true,
            ..Default::default()
        }),
        3 => Ok(NativeShopPurchaseSuccessPlan {
            hub_variant: 3,
            resource_id: 29,
            frame_count: 1,
            effect_offset: 39 * NATIVE_SHOP_WIDTH + 148,
            pre_delay_bios_ticks: 1,
            post_delay_bios_ticks: 8,
            restore_portrait_mode0: true,
            optional_equip_before: true,
            debit_after_presentation: true,
            return_to_product_loop: true,
            ..Default::default()
        }),
        5 => Ok(NativeShopPurchaseSuccessPlan {
            hub_variant: 5,
            resource_id: 63,
            frame_count: 7,
            effect_offset: 28 * NATIVE_SHOP_WIDTH + 131,
            per_frame_delay_bios_ticks: 2,
            optional_equip_before: true,
            debit_after_presentation: true,
            return_to_product_loop: true,
            ..Default::default()
        }),
        _ => Err(campaign_err!("campaign: unsupported native shop success variant")),
    }
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
    let plan = plan_native_shop_purchase_success(hub_variant)?;
    let shop = match shop {
        Some(s)
            if background.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT
                && s.resource_id == plan.resource_id
                && s.success_frames.len() as i64 == plan.frame_count =>
        {
            s
        }
        _ => return Err(campaign_err!("campaign: native shop success assets do not match the hub variant")),
    };
    let mut staged = background.to_vec();
    let mut animation = Vec::with_capacity(shop.success_frames.len());
    for effect in &shop.success_frames {
        effect.blit_at(&mut staged, NATIVE_SHOP_WIDTH, plan.effect_offset, -1)?;
        animation.push(staged.clone());
    }
    let mut last = staged;
    if plan.restore_portrait_mode0 {
        portrait.blit_at_offset(&mut last, NATIVE_SHOP_WIDTH, native_facility_portrait_offset_inner(portrait_id))?;
    }
    Ok((animation, last))
}
