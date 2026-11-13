//! 移植自 Go `terrain_phase.go`：`0x11eee` 的地形相位選擇器。

use super::{FdotherError, Result};

/// 擁有 `0x11eee` 的穩定地形全域 `[0x53c1f]`／`[0x539f4]`。計時器值是有號延伸的 BIOS 低位字。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeTerrainPhaseState {
    pub phase: i64,
    pub last_timer_tick: i64,
}

/// 重現 `0x11eee` 的選擇器更新。override 是原始 `[0x51a93]`：-1 啟用 BIOS 計時的 20 相位
/// 循環；0..19 直接選一個相位，不改 `last_timer_tick`。其他值在改動呼叫端狀態之前失敗即關閉。
///
/// Go 在錯誤時回傳未改動的輸入狀態；`NativeTerrainPhaseState` 是 `Copy`，呼叫端手上的值
/// 就是那個狀態，所以這裡只回傳錯誤。
pub fn advance_native_terrain_phase(
    state: NativeTerrainPhaseState,
    raw_timer_tick: i64,
    override_phase: i64,
) -> Result<NativeTerrainPhaseState> {
    let mut state = state;
    if state.phase < 0
        || state.phase >= 20
        || !(-0x8000..=0x7fff).contains(&state.last_timer_tick)
        || !(-0x8000..=0x7fff).contains(&raw_timer_tick)
    {
        return Err(FdotherError::fixed("fdother: native terrain phase state is outside raw bounds"));
    }
    if override_phase != -1 {
        if !(0..20).contains(&override_phase) {
            return Err(FdotherError::fixed("fdother: native terrain phase override is outside table"));
        }
        state.phase = override_phase;
        return Ok(state);
    }
    let delta = raw_timer_tick - state.last_timer_tick;
    if delta > 2 || raw_timer_tick < state.last_timer_tick {
        state.phase += 1;
        if state.phase == 20 {
            state.phase = 0;
        }
        state.last_timer_tick = raw_timer_tick;
    }
    Ok(state)
}
