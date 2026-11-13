//! 移植自 `terrain_phase_test.go`。

use crate::fdother::*;

#[test]
fn advance_native_terrain_phase_matches_11eee() {
    let s = |phase, last_timer_tick| NativeTerrainPhaseState { phase, last_timer_tick };
    let mut state = s(0, 0);
    state = advance_native_terrain_phase(state, 2, -1).unwrap();
    assert_eq!(state, s(0, 0), "two-tick gate");
    state = advance_native_terrain_phase(state, 3, -1).unwrap();
    assert_eq!(state, s(1, 3), "three-tick advance");
    state = advance_native_terrain_phase(state, 4, -1).unwrap();
    assert_eq!(state, s(1, 3), "gated advance");
    state = advance_native_terrain_phase(s(19, 0x7fff), -0x8000, -1).unwrap();
    assert_eq!(state, s(0, -0x8000), "signed wrap advance");
}

#[test]
fn advance_native_terrain_phase_override_does_not_touch_timer() {
    let state = NativeTerrainPhaseState { phase: 7, last_timer_tick: 123 };
    assert_eq!(
        advance_native_terrain_phase(state, 456, 12).unwrap(),
        NativeTerrainPhaseState { phase: 12, last_timer_tick: 123 }
    );
    // Go 另外確認錯誤時回傳的狀態等於輸入；Rust 錯誤時呼叫端的 `state`（Copy）本來就沒變。
    for override_phase in [-2, 20] {
        assert!(advance_native_terrain_phase(state, 456, override_phase).is_err(), "invalid override {override_phase}");
    }
}
