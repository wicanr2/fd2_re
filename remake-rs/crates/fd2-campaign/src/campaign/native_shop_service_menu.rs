//! Go 來源：`remake/internal/campaign/native_shop_service_menu.go`（92 行）。

use super::*;

pub(crate) const NATIVE_SHOP_SERVICE_BASE_OFFSET: i64 = 0xd430;

/// Go：`native_shop_service_menu.go:7`
pub(crate) static NATIVE_SHOP_SERVICE_DISPLACEMENTS: [i64; 4] = [-39, -13, 13, 39];

/// ComposeNativeShopServiceOpeningFrame reproduces one of 0x2d669's four
/// option-spread frames. Step zero is nearest the common anchor; step three
/// reaches the native steady destinations. All four entries use their odd
/// (normal) resource variants exactly as the opening loop does.
///
/// Go：`native_shop_service_menu.go:13`，`func ComposeNativeShopServiceOpeningFrame( stable []byte, assets *NativeShopAssets, step int, ) ([]byte, error)`
pub fn compose_native_shop_service_opening_frame(
    stable: &[u8],
    assets: Option<&NativeShopAssets>,
    step: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_shop_service_menu.go:ComposeNativeShopServiceOpeningFrame")
}

/// ComposeNativeShopServiceClosingFrame reproduces 0x2d669(a1!=0): each
/// option contracts from its full displacement using divisor step+1. After
/// step three the native caller restores the cleared stable snapshot.
///
/// Go：`native_shop_service_menu.go:40`，`func ComposeNativeShopServiceClosingFrame( stable []byte, assets *NativeShopAssets, step int, ) ([]byte, error)`
pub fn compose_native_shop_service_closing_frame(
    stable: &[u8],
    assets: Option<&NativeShopAssets>,
    step: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_shop_service_menu.go:ComposeNativeShopServiceClosingFrame")
}

/// ComposeNativeShopServiceSteadyFrame reproduces 0x2d9fe's selected-option
/// redraw on the fully spread menu. Native phase/2 selects the odd normal cell
/// for phases 0..1 and the adjacent even selected cell for phases 2..3.
///
/// Go：`native_shop_service_menu.go:67`，`func ComposeNativeShopServiceSteadyFrame( stable []byte, assets *NativeShopAssets, selected, phase int, ) ([]byte, error)`
pub fn compose_native_shop_service_steady_frame(
    stable: &[u8],
    assets: Option<&NativeShopAssets>,
    selected: i64,
    phase: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_shop_service_menu.go:ComposeNativeShopServiceSteadyFrame")
}
