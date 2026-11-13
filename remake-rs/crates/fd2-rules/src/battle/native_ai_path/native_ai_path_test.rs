//! 移植自 `remake/internal/battle/native_ai_path_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

fn cost_row_with(first: &[u8]) -> Vec<u8> {
    let mut row = vec![0u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize];
    row[..first.len()].copy_from_slice(first);
    row
}

/// Go：`TestSelectNativeMovementDestinationUsesDistanceThenAxisBalance`
#[test]
fn select_native_movement_destination_uses_distance_then_axis_balance() {
    let candidates = [
        Cell { x: 3, y: 0 }, // distance 3, imbalance 3
        Cell { x: 2, y: 1 }, // distance 3, imbalance 1
        Cell { x: 1, y: 2 }, // exact metric tie: later must not replace
    ];
    let (got, ok) = select_native_movement_destination(&candidates, Cell::default());
    assert!(ok && got == candidates[1], "destination={got:?} ok={ok}");
}

/// Go：`TestSelectNativeMovementDestinationPreservesNativeFFSentinel`
#[test]
fn select_native_movement_destination_preserves_native_ff_sentinel() {
    let intended = Cell::default();
    let (got, ok) = select_native_movement_destination(&[Cell { x: 255, y: 0 }], intended);
    assert!(!ok && got == intended, "destination={got:?} ok={ok}");
}

/// Go：`TestSelectNativeNearestOppositeCoordinateKeepsFirstTie`
#[test]
fn select_native_nearest_opposite_coordinate_keeps_first_tie() {
    let mut records = vec![0u8; 4 * RS];
    for (i, (x, y, side)) in [(5, 5, 0), (4, 3, 1), (7, 4, 2), (5, 4, 0)].into_iter().enumerate() {
        (records[i * RS], records[i * RS + 1], records[i * RS + 6]) = (x, y, side);
    }
    let got = select_native_nearest_opposite_coordinate(&records, 4, 0, 0);
    assert!(matches!(got, Ok((c, true)) if c == Cell { x: 4, y: 3 }), "coordinate={got:?}");
}

/// Go：`TestNativePathDirectionsMatchesNativeOrderAndTerrainCosts`
#[test]
fn native_path_directions_matches_native_order_and_terrain_costs() {
    let cost_row = cost_row_with(&[1, 2]);
    let got = native_path_directions(
        3,
        3,
        Cell { x: 0, y: 1 },
        Cell { x: 2, y: 1 },
        4,
        0,
        &[0; 9],
        &[0, 0, 0, 0, 1, 0, 0, 0, 0],
        &cost_row,
    );
    assert!(matches!(&got, Ok((_, true))), "path={got:?}");
    // Direct right/right costs 3 and is explored before the detours.
    let (path, _) = got.unwrap();
    assert!(path == [NATIVE_PATH_RIGHT, NATIVE_PATH_RIGHT], "path={path:?}");
}

/// Go：`TestNativePathDirectionsHonoursBlockedAndTerminalFlags`
#[test]
fn native_path_directions_honours_blocked_and_terminal_flags() {
    let cost_row = cost_row_with(&[1]);
    let mut flags = vec![0u8; 5];
    flags[2] = NATIVE_COMMAND_GRID_BLOCKED;
    // Go 另查 `path != nil`；Rust 的 Vec 不分 nil，改查空。
    let got = native_path_directions(5, 1, Cell::default(), Cell { x: 4, y: 0 }, 5, 0, &flags, &[0; 5], &cost_row);
    assert!(matches!(&got, Ok((p, false)) if p.is_empty()), "blocked path={got:?}");

    flags[2] = NATIVE_COMMAND_GRID_ZERO_BUDGET;
    let got = native_path_directions(5, 1, Cell::default(), Cell { x: 4, y: 0 }, 5, 0, &flags, &[0; 5], &cost_row);
    assert!(matches!(&got, Ok((p, false)) if p.is_empty()), "terminal path={got:?}");
}

/// Go：`TestNativePathDirectionsReturnsEmptyPathAtDestination`
#[test]
fn native_path_directions_returns_empty_path_at_destination() {
    let cost_row = cost_row_with(&[1]);
    let got = native_path_directions(1, 1, Cell::default(), Cell::default(), 0, 0, &[0], &[0], &cost_row);
    assert!(matches!(&got, Ok((p, true)) if p.is_empty()), "path={got:?}");
}

/// Go：`TestNativePathBlockedCoordinateKeepsLastDFSBlockedCell`
#[test]
fn native_path_blocked_coordinate_keeps_last_dfs_blocked_cell() {
    let cost_row = cost_row_with(&[1]);
    let mut flags = vec![0u8; 5];
    flags[1] = NATIVE_COMMAND_GRID_BLOCKED;
    flags[4] = NATIVE_COMMAND_GRID_BLOCKED;
    let got = native_path_blocked_coordinate(5, 1, Cell::default(), 5, &flags, &[0; 5], &cost_row);
    assert!(matches!(got, Ok((c, true)) if c == Cell { x: 4, y: 0 }), "blocked coordinate={got:?}");
}
