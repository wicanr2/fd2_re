//! 移植自 `separated_lmi1_bank_test.go`。

use super::{TempDir, original_entry, pack};
use crate::fdother::*;

#[test]
fn separated_complete_lmi1_banks_match_fixed_resources() {
    for (resource, rel) in [
        (5, "ui/fdother_005_lmi1_opaque"),
        (6, "effects/fdother_006_lmi1_opaque"),
        (9, "animations/fdother_009_spawn_intro"),
    ] {
        let Some(raw) = original_entry("FDOTHER.DAT", resource as usize) else { return };
        let Some(root) = pack(rel, "bank.json") else { return };
        let want = parse_lmi1(&raw).unwrap();
        let got = load_separated_lmi1_bank(&root, resource).unwrap();
        assert_eq!(got.len(), want.len(), "FDOTHER #{resource} entries");
        for (index, (g, w)) in got.iter().zip(&want).enumerate() {
            assert_eq!(g, w, "FDOTHER #{resource} entry {index} differs");
        }
    }
}

#[test]
fn separated_complete_lmi1_banks_fail_closed() {
    let a = TempDir::new("lmi1-a");
    assert!(load_separated_lmi1_bank(a.path(), 5).is_err(), "missing FDOTHER #5 bank was accepted");
    let b = TempDir::new("lmi1-b");
    assert!(load_separated_lmi1_bank(b.path(), 4).is_err(), "unsupported FDOTHER bank was accepted");
}
