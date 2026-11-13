//! 移植自 `separated_load_slots_test.go`。

use super::{TempDir, original_entry, pack};
use crate::fdother::*;

#[test]
fn separated_load_slots_frame_matches_original_entry16() {
    let Some(raw) = original_entry("FDOTHER.DAT", 13) else { return };
    let Some(ui_root) = pack("ui", "fdother_013_load_slots/resource.json") else { return };
    let want = parse_lmi1_opaque_entry(&raw, 16).unwrap();
    let got = load_separated_load_slots_frame(&ui_root).unwrap();
    assert_eq!(got, want, "separated load-slots frame differs from FDOTHER #13 entry16");
}

#[test]
fn separated_load_slots_frame_fails_closed_without_pack() {
    let dir = TempDir::new("load-slots");
    assert!(load_separated_load_slots_frame(dir.path()).is_err(), "incomplete separated load-slots pack was accepted");
}
