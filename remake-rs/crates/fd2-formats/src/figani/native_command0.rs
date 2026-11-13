//! 指令 0（`sub_26152`）的演出排程與亂數走訪。移植自 Go `native_command0.go`。

use super::{Animation, FiganiError, Resolver, RngWalkError, fail, frame_shape_ok, msg};
use crate::native_rng::native_rng_step;

pub const NATIVE_COMMAND0_PRESENTATION_FRAMES: i64 = 28;
pub const NATIVE_COMMAND0_EFFECT_FRAME_COUNT: i64 = 16;
pub const NATIVE_COMMAND0_SOUND_RESOURCE: i64 = 82;
pub const NATIVE_COMMAND0_SOUND_INDEX: i64 = 1;
pub const NATIVE_COMMAND0_DAMAGE_STAGES: i64 = 7;

const NATIVE_COMMAND0_LAYER_FLAGS: [u8; 7] = [0, 0, 1, 0, 1, 0, 0];
const NATIVE_COMMAND0_X_OFFSETS: [i64; 7] = [40, 70, 120, 80, 50, 100, 70];
const NATIVE_COMMAND0_Y_OFFSETS: [i64; 7] = [0, -10, -20, 0, -15, -5, 0];
const NATIVE_COMMAND0_RAW_BYTE4: [u8; 16] = [0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0];

/// 保存 `sub_26152` 原始 mode 3/4/5 契約。`effect_resource` 仍由呼叫端依 actor raw byte +6 選定；
/// 這些欄位都不替幀指派遊戲或顏色名稱。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand0PresentationSchedule {
    pub effect_resource: i64,
    pub sound_resource: i64,
    pub sound_index: i64,
    pub frames: i64,
    pub damage_stages: i64,
    pub side_x_shift: i64,
    pub layer_flags: [u8; 7],
    pub x_offsets: [i64; 7],
    pub y_offsets: [i64; 7],
}

/// 在任何演出端發布 MP 或 HP 之前，驗證實際的 FDOTHER #18/#20 幀庫。
pub fn build_native_command0_presentation_schedule(
    raw_side: u8,
    effect: Option<&Animation>,
) -> Result<NativeCommand0PresentationSchedule, FiganiError> {
    let (resource, shift) = if raw_side == 0 { (20, 148) } else { (18, 0) };
    let Some(effect) = effect.filter(|e| {
        e.header_byte1 == 0
            && e.header_byte2 == 16
            && e.header_byte4 == 0
            && e.frames.len() as i64 == NATIVE_COMMAND0_EFFECT_FRAME_COUNT
    }) else {
        return Err(msg(format!("figani: command0 FDOTHER #{resource} signature mismatch")));
    };
    for (index, frame) in effect.frames.iter().enumerate() {
        if !frame_shape_ok(frame)
            || frame.raw_byte4 != NATIVE_COMMAND0_RAW_BYTE4[index]
            || frame.delay != 0
            || frame.raw_byte5 != 0
            || frame.raw_byte7 != 0
        {
            return Err(msg(format!("figani: command0 FDOTHER #{resource} frame {index} malformed")));
        }
    }
    Ok(NativeCommand0PresentationSchedule {
        effect_resource: resource,
        sound_resource: NATIVE_COMMAND0_SOUND_RESOURCE,
        sound_index: NATIVE_COMMAND0_SOUND_INDEX,
        frames: NATIVE_COMMAND0_PRESENTATION_FRAMES,
        damage_stages: NATIVE_COMMAND0_DAMAGE_STAGES,
        side_x_shift: shift,
        layer_flags: NATIVE_COMMAND0_LAYER_FLAGS,
        x_offsets: NATIVE_COMMAND0_X_OFFSETS,
        y_offsets: NATIVE_COMMAND0_Y_OFFSETS,
    })
}

/// 某個交錯疊層在某一步選到的原始效果幀。`None` 代表 counter 不在 0..15，`sub_26152` 會略過
/// `0x2935B`。Go 版在 ok=false 時仍回傳計算值（超出範圍的幀或 0），這裡以 `(frame, ok)` 保留。
pub fn native_command0_effect_frame(step: i64, layer: i64) -> (i64, bool) {
    if !(0..NATIVE_COMMAND0_PRESENTATION_FRAMES).contains(&step) || !(0..7).contains(&layer) {
        return (0, false);
    }
    let frame = step - 2 * layer;
    (frame, (0..NATIVE_COMMAND0_EFFECT_FRAME_COUNT).contains(&frame))
}

/// 七個 counter==3 的標記。原始子取樣播放與分段數值發布都由呼叫端負責。
pub fn native_command0_impact(step: i64, layer: i64) -> bool {
    let (frame, ok) = native_command0_effect_frame(step, layer);
    ok && frame == 3
}

/// 重現 `sub_26152` mode 5 的回傳值：七個 counter 由 mode 3 寫成 0,-2,…,-12，每一步各加 1，
/// 任一 counter 加完等於 9 就回傳 1。`0x2A6BD` 只在回傳 1 的那一步做 HP 分段與
/// `0x2AF40 call 0x4E893` 抖動擲骰。
pub fn native_command0_marker_step(step: i64) -> bool {
    if !(0..NATIVE_COMMAND0_PRESENTATION_FRAMES).contains(&step) {
        return false;
    }
    (0..7).any(|layer| step + 1 - 2 * layer == 9)
}

/// 走指令 0 單一目標的全域亂數序列：mode 0..3 不擲骰，`0x2B114` 先對目標呼叫 `0x1C75E`
/// （命中、傷害各一步），之後 28 步裡 mode 5 回傳 1 的七步各吃一次 `0x2AF40` 抖動亂數（r4 收據
/// seq 114：`0x1C7F2`、`0x1C86E`、七次 `0x2AF45`）；未命中只有 `0x1C7F2` 一步（r4 收據 seq 1013）。
pub fn walk_native_command0_rng(
    mut rng: u16,
    schedule: NativeCommand0PresentationSchedule,
    target_count: i64,
    resolve: Resolver<'_>,
) -> Result<u16, RngWalkError> {
    let Some(resolve) = resolve.filter(|_| target_count > 0 && schedule.frames == NATIVE_COMMAND0_PRESENTATION_FRAMES)
    else {
        return Err(fail(rng, FiganiError::Static("figani: command0 RNG walk needs targets and the 28-step schedule")));
    };
    for target in 0..target_count {
        let (next, hit) = resolve(target, rng).map_err(|e| fail(rng, FiganiError::Resolver(e)))?;
        rng = next;
        if !hit {
            // 0x2ADFE：未命中走 0x2AF61，不看 mode 5 回傳，沒有抖動擲骰（r4 seq 1013）。
            continue;
        }
        for step in 0..schedule.frames {
            if native_command0_marker_step(step) {
                rng = native_rng_step(rng);
            }
        }
    }
    Ok(rng)
}
