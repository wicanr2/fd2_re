//! 移植自 `indexed_transition_lut_test.go`。

use crate::fdother::*;

// Go: TestNativeIndexedTransitionLUTResolvesOnlyOuterPassRange
#[test]
fn native_indexed_transition_lut_resolves_only_outer_pass_range() {
    let bank: Vec<Vec<u8>> = (0..10)
        .map(|i| {
            let mut entry = vec![0u8; 256];
            entry[0] = i as u8;
            entry
        })
        .collect();
    let mut got = native_indexed_transition_lut(&bank, 9).unwrap();
    assert_eq!(got[0], 9);
    got[0] = 0xff;
    assert_eq!(bank[9][0], 9, "selector returned mutable bank storage");
    for descriptor in [0, 10, -1] {
        assert!(native_indexed_transition_lut(&bank, descriptor).is_err(), "descriptor {descriptor} accepted");
    }
}

// Go: TestNativeIndexedTransitionLUTRejectsMalformedEntry
#[test]
fn native_indexed_transition_lut_rejects_malformed_entry() {
    let mut bank = vec![Vec::new(); 10];
    bank[1] = vec![0u8; 255];
    assert!(native_indexed_transition_lut(&bank, 1).is_err(), "short LUT accepted");
}
