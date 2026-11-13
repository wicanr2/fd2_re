//! 移植自 `binary_tick_test.go`。

use crate::fdicon::*;

#[test]
fn advance_native_binary_tick_matches_native_latch() {
    let state = NativeBinaryTickState::default();
    let state = advance_native_binary_tick(state, 0).unwrap();
    assert_eq!(state, NativeBinaryTickState::default(), "same tick");
    let state = advance_native_binary_tick(state, 1).unwrap();
    assert_eq!(state, NativeBinaryTickState { value: 1, last_timer_tick: 1 }, "first tick");
    let state = advance_native_binary_tick(state, 1).unwrap();
    assert_eq!(state, NativeBinaryTickState { value: 1, last_timer_tick: 1 }, "repeated tick");
    let state = advance_native_binary_tick(state, -0x8000).unwrap();
    assert_eq!(state, NativeBinaryTickState { value: 0, last_timer_tick: -0x8000 }, "signed wrap tick");
}
