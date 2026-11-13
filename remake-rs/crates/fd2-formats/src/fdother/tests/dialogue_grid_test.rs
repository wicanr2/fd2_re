//! 移植自 `dialogue_grid_test.go`。

use crate::fdother::*;

#[test]
fn plan_native_dialogue_frame_grid_matches_168b6_item_invocation() {
    let got = plan_native_dialogue_frame_grid(320, 5, 7, 5, 5).unwrap();
    assert_eq!(got.len(), 49, "placements");
    for (index, resource_index, destination_byte) in [
        (0, 1, 2245),
        (1, 2, 2328),
        (2, 3, 28805),
        (3, 4, 28888),
        (11, 17, 23768),
        (12, 9, 2264),
        (18, 10, 8325),
        (24, 13, 3208),
        (48, 13, 23752),
    ] {
        assert_eq!(got[index], NativeDialogueGridPlacement { resource_index, destination_byte }, "placement[{index}]");
    }
}

#[test]
fn plan_native_dialogue_frame_grid_rejects_invalid_geometry() {
    // Go 另外確認回傳的切片是 nil；Rust 錯誤時沒有切片。
    assert!(plan_native_dialogue_frame_grid(320, 5, 7, 1, 5).is_err());
    assert!(plan_native_dialogue_frame_grid(16, 5, 7, 5, 5).is_err());
}
