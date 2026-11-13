//! 移植自 `remake/internal/battle/native_command_grid_test.go`。

use super::*;

/// Go：`TestNativeCommandGridMatchesRecoveredFourRowLayout`
#[test]
fn native_command_grid_matches_recovered_four_row_layout() {
    let grid = native_command_grid(&[0, 1, 2, 3, 4], 4);
    let got = grid[0];
    assert!(got.x == 0x12 && got.y == 0x67 && got.column == 0 && got.row == 0 && !got.selected, "first cell={got:?}");
    let got = grid[4];
    assert!(got.x == 0x76 && got.y == 0x67 && got.column == 1 && got.row == 0 && got.selected, "fifth cell={got:?}");
}

/// Go：`TestNativeCommandGridMoveMatchesOriginalBounds`
#[test]
fn native_command_grid_move_matches_original_bounds() {
    assert_eq!(native_command_grid_move(0, 6, 0), 5, "up wrap");
    assert_eq!(native_command_grid_move(5, 6, 1), 0, "down wrap");
    assert_eq!(native_command_grid_move(1, 6, 2), 1, "left edge");
    assert_eq!(native_command_grid_move(1, 6, 3), 5, "right");
    assert_eq!(native_command_grid_move(5, 6, 3), 5, "right bound");
}

/// Go：`TestNativeCommandGridMissingInputsRemainInert`
///
/// Go 另外檢查空輸入回傳非 nil 的空切片；Rust `Vec` 沒有 nil，只比長度。
#[test]
fn native_command_grid_missing_inputs_remain_inert() {
    assert!(native_command_grid(&[], 0).is_empty(), "empty command ids");
    let ids = [0, 7];
    for selected in [-1, ids.len() as i64, 99] {
        let grid = native_command_grid(&ids, selected);
        assert_eq!(grid.len(), ids.len(), "selected={selected} changed grid length");
        for (i, cell) in grid.iter().enumerate() {
            assert!(!cell.selected, "selected={selected} marked cell {i} despite out-of-range selection");
        }
    }
}
