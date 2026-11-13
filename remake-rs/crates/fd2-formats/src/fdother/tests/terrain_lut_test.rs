//! 移植自 `terrain_lut_test.go`。

use crate::fdother::*;

#[test]
fn native_terrain_lut_index_matches_51a97() {
    let want = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1];
    for (phase, expected) in want.into_iter().enumerate() {
        assert_eq!(native_terrain_lut_index(phase as i64).unwrap(), expected, "phase={phase}");
    }
    assert!(native_terrain_lut_index(-1).is_err(), "negative phase accepted");
    assert!(native_terrain_lut_index(20).is_err(), "out-of-range phase accepted");
}
