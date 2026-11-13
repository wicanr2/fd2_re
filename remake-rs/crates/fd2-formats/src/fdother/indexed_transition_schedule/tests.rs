//! 移植自 `indexed_transition_schedule_test.go`。

use crate::fdother::*;

// Go: TestBuildNativeIndexedTransitionSchedulePreservesRawLoop
#[test]
fn build_native_indexed_transition_schedule_preserves_raw_loop() {
    let s = build_native_indexed_transition_schedule(10, 8).unwrap();
    assert!(s.frames.len() == 9 && s.frames[0].lut_index == 9 && s.frames[8].lut_index == 1, "frames={:?}", s.frames);
    assert!(s.frames[0].radius == 10 && s.frames[8].radius == 74 && s.tail_delay_ms == 500, "radius/tail");
    assert!(
        s.palette_deltas.len() == 32
            && s.palette_deltas[0] == 0
            && s.palette_deltas[31] == 62
            && s.palette_delay_ms == 4,
        "palette={:?}",
        s.palette_deltas
    );
}

// Go: TestBuildNativeIndexedTransitionScheduleRejectsInvalidRadius
#[test]
fn build_native_indexed_transition_schedule_rejects_invalid_radius() {
    assert!(build_native_indexed_transition_schedule(-1, 0).is_err(), "negative radius unexpectedly accepted");
    assert!(
        build_native_indexed_transition_schedule(1, 0x8000).is_err(),
        "oversized radius step unexpectedly accepted"
    );
}
