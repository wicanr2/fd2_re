//! 移植自 `separated_lutbank_test.go`。

use super::{TempDir, original_entry, pack};
use crate::fdother::*;

#[test]
fn separated_lut_bank_matches_fixed_resource() {
    let Some(raw) = original_entry("FDOTHER.DAT", 3) else { return };
    let Some(root) = pack("palette", "fdother_003_luts.json") else { return };
    let want = parse_lut_bank(&raw).unwrap();
    assert_eq!(load_separated_lut_bank(&root).unwrap(), want);
}

#[test]
fn separated_lut_bank_fails_closed() {
    let dir = TempDir::new("lut");
    assert!(load_separated_lut_bank(dir.path()).is_err(), "missing LUT bank was accepted");
}
