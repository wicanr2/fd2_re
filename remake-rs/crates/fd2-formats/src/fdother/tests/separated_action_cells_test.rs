//! 移植自 `separated_action_cells_test.go`。

use super::{TempDir, original_entry, pack};
use crate::fdother::*;

#[test]
fn separated_action_cells_match_original_bank() {
    let Some(raw) = original_entry("FDOTHER.DAT", 2) else { return };
    let Some(ui_root) = pack("ui", "action_cells/resource.json") else { return };
    let want = parse_raw_cell_bank(&raw).unwrap();
    let got = load_separated_action_cells(&ui_root).unwrap();
    assert_eq!(got.len(), want.len(), "cell count");
    for (index, (g, w)) in got.iter().zip(&want).enumerate() {
        assert_eq!(g, w, "action cell {index} differs");
    }
}

#[test]
fn separated_action_cells_fail_closed_without_pack() {
    let dir = TempDir::new("action-cells");
    assert!(load_separated_action_cells(dir.path()).is_err(), "incomplete separated action-cell pack was accepted");
}
