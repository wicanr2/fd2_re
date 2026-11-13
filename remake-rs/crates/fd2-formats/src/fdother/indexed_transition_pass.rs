//! 移植自 Go `indexed_transition_pass.go`：`0x22046` 的 raw 參數對應。

use super::{
    CenteredRectLUTRemap, FdotherError, IndexedTransitionPass, NATIVE_TRANSITION_STAGE_HEIGHT,
    NATIVE_TRANSITION_STAGE_WIDTH, RadialLUTRemap, Result,
};

/// 轉寫 raw `0x22046` 參數對應。呼叫端的 a1/a2/a3/a4/a5 成為放射狀中心／半徑與列界線；特別是 a2
/// 在原生程序裡同時是放射狀 CenterY 與最後矩形的不含 EndY。這個奇特的別名刻意保留，不正規化成
/// 猜測的畫面矩形。
///
/// `radius*16/10` 與 Go 一樣以 64 位元繞回（radius 沒有上限檢查）。
pub fn build_native_indexed_transition_pass(
    center_x: i64,
    center_y: i64,
    radius: i64,
    start_y: i64,
    end_y: i64,
) -> Result<IndexedTransitionPass> {
    if !(0..NATIVE_TRANSITION_STAGE_WIDTH).contains(&center_x)
        || center_y < 0
        || radius <= 0
        || start_y < 0
        || end_y < start_y
        || end_y > NATIVE_TRANSITION_STAGE_HEIGHT
    {
        return Err(FdotherError::fixed("indexed transition: invalid raw 0x22046 args"));
    }
    let radial = |start_y| RadialLUTRemap {
        center_x,
        center_y,
        radius,
        scale: 16,
        start_y,
        end_y,
        clip_width: NATIVE_TRANSITION_STAGE_WIDTH,
    };
    Ok(IndexedTransitionPass {
        first_radial: radial(start_y),
        second_radial: radial(center_y),
        final_rect: CenteredRectLUTRemap {
            center_x,
            horizontal_radius: radius.wrapping_mul(16) / 10,
            start_y,
            end_y: center_y,
            clip_width: NATIVE_TRANSITION_STAGE_WIDTH,
        },
    })
}
