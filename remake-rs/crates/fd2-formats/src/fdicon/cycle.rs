//! 移植自 Go `cycle.go`。

/// `0x127e0` 消費、`0x1297d` 修改的三個戰場全域。
///
/// serde 欄位名是 Go 型別沒有標籤時 `encoding/json` 的輸出（Go 欄位名），供 `fd2-rules` 的戰場快照讀寫。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeMapSpriteCycleState {
    #[serde(rename = "Idle")]
    pub idle: i64,
    #[serde(rename = "Moving")]
    pub moving: i64,
    #[serde(rename = "LastTimerTick")]
    pub last_timer_tick: i64,
}

/// 重現 `sub_1297d` 的全部修改。`raw_timer_tick` 是 BIOS timer `[0x46c]` 符號延伸的低 word。
/// idle 只在 tick-last 為負或大於 4 時前進；moving 每次合成都前進。兩者都 3→0 繞回。
/// 減法與遞增沿用 Go `int` 的 64 位元繞回。
pub fn advance_native_map_sprite_cycles(
    state: NativeMapSpriteCycleState,
    raw_timer_tick: i64,
) -> NativeMapSpriteCycleState {
    let mut state = state;
    let delta = raw_timer_tick.wrapping_sub(state.last_timer_tick);
    if !(0..=4).contains(&delta) {
        state.idle = state.idle.wrapping_add(1) & 3;
        state.last_timer_tick = raw_timer_tick;
    }
    state.moving = state.moving.wrapping_add(1) & 3;
    state
}
