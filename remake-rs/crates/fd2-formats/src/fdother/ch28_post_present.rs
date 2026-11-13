//! 移植自 Go `ch28_post_present.go`：第 28 章戰後 `0x1DB65` 的 LMI 覆蓋呈現。

use super::{FdotherError, LMI1Entry, Result};

pub const NATIVE_CH28_POST_LMI_RESOURCE: i64 = 5;
pub const NATIVE_CH28_POST_LMI_FIRST: i64 = 0x44;
pub const NATIVE_CH28_POST_LMI_LAST: i64 = 0x4f;
pub const NATIVE_CH28_POST_SFX_RESOURCE: i64 = 31;
pub const NATIVE_CH28_POST_SFX_INDEX: i64 = 3;
pub const NATIVE_CH28_POST_SFX_ARG: i64 = 1;

pub const NATIVE_CH28_POST_POSE_FRAMES: i64 = 13;
pub const NATIVE_CH28_POST_WORK_SIZE: i64 = 0x25680;
pub const NATIVE_CH28_POST_STRIDE: i64 = 0x1c8;
pub const NATIVE_CH28_POST_VIEW_BASE: i64 = 0x8088;

/// 保存一個 `0x1DB65` 呈現邊界。`phase` 仍是 raw 控制流標籤；LMI 格還沒有已證實的玩家可見名稱。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCh28PostOverlayFrame {
    pub phase: String,
    pub entry: i64,
}

/// 回傳 `0x1DDB2..0x1DE9C` 的兩段六幀迴圈。第二段在每幀之前還原暫存的快照；那個緩衝交易歸
/// 呈現擁有者。
pub fn native_ch28_post_overlay_frames() -> Vec<NativeCh28PostOverlayFrame> {
    let mut frames = Vec::with_capacity(12);
    for entry in NATIVE_CH28_POST_LMI_FIRST..NATIVE_CH28_POST_LMI_FIRST + 6 {
        frames.push(NativeCh28PostOverlayFrame { phase: "overlay_a".to_string(), entry });
    }
    for entry in NATIVE_CH28_POST_LMI_FIRST + 6..=NATIVE_CH28_POST_LMI_LAST {
        frames.push(NativeCh28PostOverlayFrame { phase: "overlay_b".to_string(), entry });
    }
    frames
}

/// 重現 `0x1DBE7..0x1DC23`，回傳原版 0x25680 位元組、stride 456 工作面內的位元組偏移。座標與
/// 鏡頭值是 raw 單位／全域整數，不是正規化像素。乘加與 Go 一樣以 64 位元繞回。
pub fn native_ch28_post_overlay_origin(x: i64, y: i64, camera_x: i64, camera_y: i64) -> Result<i64> {
    let offset = NATIVE_CH28_POST_VIEW_BASE
        .wrapping_add(24i64.wrapping_mul(x.wrapping_sub(1).wrapping_sub(camera_x)))
        .wrapping_add((24 * NATIVE_CH28_POST_STRIDE).wrapping_mul(y.wrapping_sub(1).wrapping_sub(camera_y)))
        .wrapping_sub(6 * NATIVE_CH28_POST_STRIDE);
    if !(0..NATIVE_CH28_POST_WORK_SIZE).contains(&offset) {
        return Err(FdotherError::message(format!(
            "fdother: ch28 post overlay origin {} is outside work surface",
            go_hex(offset)
        )));
    }
    Ok(offset)
}

/// Go `%#x`：負值寫成 `-0x...`。
pub(super) fn go_hex(v: i64) -> String {
    if v < 0 { format!("-{:#x}", v.unsigned_abs()) } else { format!("{v:#x}") }
}

/// 在預先算好的原生工作緩衝位址套用 `0x4E85B` 的透明格寫入。呈現與時序仍是呼叫端的必要責任。
pub fn blit_native_ch28_post_overlay(entry: &LMI1Entry, work: &mut [u8], origin: i64) -> Result<()> {
    if work.len() as i64 != NATIVE_CH28_POST_WORK_SIZE || origin < 0 || origin >= work.len() as i64 {
        return Err(FdotherError::fixed("fdother: invalid ch28 post work surface or origin"));
    }
    entry.blit_at(
        work,
        NATIVE_CH28_POST_STRIDE,
        origin % NATIVE_CH28_POST_STRIDE,
        origin / NATIVE_CH28_POST_STRIDE,
        false,
    )
}

/// 拒收不完整或被呼叫端改名的資源庫。這個呈現端只需要 0x44..0x4F。
pub fn validate_native_ch28_post_lmi(entries: &[LMI1Entry]) -> Result<()> {
    if entries.len() as i64 <= NATIVE_CH28_POST_LMI_LAST {
        return Err(FdotherError::message(format!(
            "fdother: FDOTHER#5 lacks ch28 post entries {NATIVE_CH28_POST_LMI_FIRST:#x}..{NATIVE_CH28_POST_LMI_LAST:#x}"
        )));
    }
    for index in NATIVE_CH28_POST_LMI_FIRST..=NATIVE_CH28_POST_LMI_LAST {
        let entry = &entries[index as usize];
        if entry.width <= 0 || entry.height <= 0 || entry.pixels.len() as i64 != entry.width.wrapping_mul(entry.height)
        {
            return Err(FdotherError::message(format!("fdother: invalid FDOTHER#5 entry {index:#x}")));
        }
    }
    Ok(())
}
