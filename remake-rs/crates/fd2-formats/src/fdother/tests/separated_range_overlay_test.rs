//! 移植自 `separated_range_overlay_test.go`。

use super::{TempDir, original, pack};
use crate::fdother::*;

#[test]
fn separated_range_overlay_bank_matches_original() {
    let Some(archive) = original("FDOTHER.DAT") else { return };
    let Some(root) = pack("sprites/fdother_001_range_overlay", "bank.json") else { return };
    let want = decode_native_range_overlay_bank(&archive).unwrap();
    let got = load_separated_range_overlay_bank(&root).unwrap();
    assert_eq!(got.sprites.len(), want.sprites.len(), "sprite count");
    for (index, (g, w)) in got.sprites.iter().zip(&want.sprites).enumerate() {
        assert_eq!(g, w, "range-overlay sprite {index} differs");
    }
}

#[test]
fn separated_range_overlay_bank_fails_closed_without_pack() {
    let dir = TempDir::new("range-overlay");
    assert!(load_separated_range_overlay_bank(dir.path()).is_err(), "incomplete range-overlay pack was accepted");
}
