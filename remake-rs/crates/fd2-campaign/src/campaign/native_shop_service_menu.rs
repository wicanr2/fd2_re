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
    let assets = match assets {
        Some(a) if stable.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT && (0..=3).contains(&step) => a,
        _ => return Err(campaign_err!("campaign: native shop service opening state is invalid")),
    };
    native_shop_service_frame(stable, assets, |option| (0, NATIVE_SHOP_SERVICE_DISPLACEMENTS[option] / (4 - step)))
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
    let assets = match assets {
        Some(a) if stable.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT && (0..4).contains(&step) => a,
        _ => return Err(campaign_err!("campaign: native shop service closing state is invalid")),
    };
    native_shop_service_frame(stable, assets, |option| (0, NATIVE_SHOP_SERVICE_DISPLACEMENTS[option] / (step + 1)))
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
    let assets = match assets {
        Some(a)
            if stable.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT
                && (0..4).contains(&selected)
                && (0..4).contains(&phase) =>
        {
            a
        }
        _ => return Err(campaign_err!("campaign: native shop service steady state is invalid")),
    };
    native_shop_service_frame(stable, assets, |option| {
        let variant = if option as i64 == selected { (phase / 2) as usize } else { 0 };
        (variant, NATIVE_SHOP_SERVICE_DISPLACEMENTS[option])
    })
}

/// 0x2d669／0x2d9fe 共用的貼圖：從穩定畫面複本開始，四個選項各貼一格。`pick(option)` 回傳
/// （正常 0／選取 1 變體, 相對共同錨點的 x 位移）。Go 三個函式各寫一份同樣的迴圈
/// （`native_shop_service_menu.go:24`、`:51`、`:77`）。
fn native_shop_service_frame(
    stable: &[u8],
    assets: &NativeShopAssets,
    pick: impl Fn(usize) -> (usize, i64),
) -> Result<Vec<u8>> {
    let mut frame = stable.to_vec();
    let base_x = NATIVE_SHOP_SERVICE_BASE_OFFSET % NATIVE_SHOP_WIDTH;
    let base_y = NATIVE_SHOP_SERVICE_BASE_OFFSET / NATIVE_SHOP_WIDTH;
    for option in 0..4 {
        let (variant, dx) = pick(option);
        assets.service_cells[option][variant].blit_at(&mut frame, NATIVE_SHOP_WIDTH, base_x + dx, base_y)?;
    }
    Ok(frame)
}
