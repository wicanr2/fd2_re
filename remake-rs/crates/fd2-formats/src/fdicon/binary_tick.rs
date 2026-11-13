//! 移植自 Go `binary_tick.go`。

use super::{Result, fixed};

/// 一個原版 0/1 動畫全域及其符號延伸的低 BIOS word 閂鎖。`0x11eee` 的地形用一份，
/// `0x127e0` 的單位像素位移另用一份獨立實例。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeBinaryTickState {
    pub value: i64,
    pub last_timer_tick: i64,
}

/// 重現共用的「BIOS word 變了就翻轉位元」序列。兩個原版呼叫點各自擁有實例與呼叫時機。
/// Go 版出錯時連同未改動的輸入狀態一起回傳；Rust 的狀態是 `Copy`，呼叫端手上仍有原值。
pub fn advance_native_binary_tick(state: NativeBinaryTickState, raw_timer_tick: i64) -> Result<NativeBinaryTickState> {
    let raw = -0x8000..=0x7fff;
    if (state.value != 0 && state.value != 1) || !raw.contains(&state.last_timer_tick) || !raw.contains(&raw_timer_tick)
    {
        return fixed("fdicon: native binary tick state is outside raw bounds");
    }
    let mut state = state;
    if raw_timer_tick != state.last_timer_tick {
        state.value ^= 1;
        state.last_timer_tick = raw_timer_tick;
    }
    Ok(state)
}
