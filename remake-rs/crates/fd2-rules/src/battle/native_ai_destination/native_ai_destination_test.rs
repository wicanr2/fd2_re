//! 移植自 `remake/internal/battle/native_ai_destination_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestNativeAIPhysicalDestinationsUsesNativeCostsAndRowMajorOrder`
#[test]
fn native_ai_physical_destinations_uses_native_costs_and_row_major_order() {
    let mut records = vec![0u8; RS];
    (records[0], records[1], records[6]) = (1, 1, 0);
    let mut cost_row = vec![0u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize];
    (cost_row[1], cost_row[2]) = (1, 2);
    let codes = [1, 1, 1, 2, 1, 1, 1, 1, 1];
    let got = native_ai_physical_destinations(3, 3, &records, 1, 0, 0, 2, &[0; 9], &codes, &cost_row).unwrap();
    let want: Vec<Cell> = (0..3).flat_map(|y| (0..3).map(move |x| Cell { x, y })).collect();
    assert_eq!(got, want, "destinations");
}

/// Go：`TestNativeAIPhysicalDestinationsAppliesOccupancyBeforeAndAfterSearch`
#[test]
fn native_ai_physical_destinations_applies_occupancy_before_and_after_search() {
    let mut records = vec![0u8; 3 * RS];
    (records[0], records[1], records[6]) = (0, 1, 0);
    (records[RS], records[RS + 1], records[RS + 6]) = (1, 1, 0);
    (records[2 * RS], records[2 * RS + 1], records[2 * RS + 6]) = (3, 1, 1);
    let mut cost_row = vec![0u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize];
    cost_row[0] = 1;
    let got = native_ai_physical_destinations(5, 3, &records, 3, 0, 0, 5, &[0; 15], &[0; 15], &cost_row).unwrap();
    for forbidden in [Cell { x: 1, y: 1 }, Cell { x: 3, y: 1 }, Cell { x: 4, y: 1 }] {
        assert!(!got.contains(&forbidden), "occupied/blocked destination {forbidden:?} leaked into {got:?}");
    }
    // The opposite group's adjacent cell is enterable but terminal.
    assert!(got.contains(&Cell { x: 2, y: 1 }), "terminal adjacent cell missing from {got:?}");
}

/// Go：`TestNativeAIPhysicalDestinationsRejectsIncompleteRawInputs`
#[test]
fn native_ai_physical_destinations_rejects_incomplete_raw_inputs() {
    let records = vec![0u8; RS];
    let cost_row = vec![0u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize];
    let codes = [NATIVE_MOVEMENT_COST_ROW_SIZE as u8];
    assert!(
        native_ai_physical_destinations(1, 1, &records, 1, 0, 0, 1, &[0], &codes, &cost_row).is_err(),
        "out-of-range native terrain code accepted"
    );
}
