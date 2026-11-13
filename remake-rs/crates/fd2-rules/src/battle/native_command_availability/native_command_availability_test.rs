//! 移植自 `remake/internal/battle/native_command_availability_test.go`。

use super::*;

fn book(cost: impl Fn(i64) -> i64) -> Vec<NativeCommandRecord> {
    (0..36).map(|id| NativeCommandRecord { id, mp_cost: cost(id), ..Default::default() }).collect()
}

/// Go：`TestNativeCommandAvailabilityUsesRawBitAndMPCost`
#[test]
fn native_command_availability_uses_raw_bit_and_mp_cost() {
    let book = book(|id| id);
    let mut u = Unit { mp: 3, native_command_mask: [0x0f, 0, 0, 0, 0], ..Default::default() };
    for id in 0..4 {
        assert!(native_command_available(Some(&u), &book, id), "command {id} unexpectedly unavailable");
    }
    assert!(!native_command_available(Some(&u), &book, 4), "command over current MP accepted");
    u.native_command_mask[0] &= !(1 << 1);
    assert!(!native_command_available(Some(&u), &book, 1), "missing raw command bit accepted");
}

/// Go：`TestNativeAvailableCommandIDsOmitsUnknownPhysicalBits`
#[test]
fn native_available_command_ids_omits_unknown_physical_bits() {
    let book = book(|_| 0);
    let mut u = Unit { mp: 0, native_command_mask: [0, 0, 0, 0, 0x90], ..Default::default() };
    let got = native_available_command_ids(Some(&u), &book);
    assert!(got.is_empty(), "unknown IDs 36/39 should not be returned: {got:?}");
    u.native_command_mask[4] = 1;
    assert_eq!(native_available_command_ids(Some(&u), &book), [32], "closed ID32 missing from availability");
}

/// Go：`TestNativeCommandAvailabilityFailsClosedOnMalformedBook`
#[test]
fn native_command_availability_fails_closed_on_malformed_book() {
    let u = Unit { mp: 99, native_command_mask: [1, 0, 0, 0, 0], ..Default::default() };
    assert!(
        !native_command_available(Some(&u), &[], 0) && native_available_command_ids(Some(&u), &[]).is_empty(),
        "malformed command book did not fail closed"
    );
}

/// Go：`TestNativeAvailableAICommandIDsAppliesRaw27Gate`
#[test]
fn native_available_ai_command_ids_applies_raw27_gate() {
    let book = book(|_| 0);
    let mut u = Unit { mp: 1, native_command_mask: [1, 0, 0, 0, 0], ..Default::default() };
    assert_eq!(native_available_ai_command_ids(Some(&u), &book), [0], "ungated AI commands");
    u.native_transient[5] = 1;
    let got = native_available_ai_command_ids(Some(&u), &book);
    assert!(got.is_empty(), "raw +0x27 gate ignored: {got:?}");
}

/// Go：`TestNativeAvailableAIScoredCommandIDsKeepsLowAndHighRawIndices`
#[test]
fn native_available_ai_scored_command_ids_keeps_low_and_high_raw_indices() {
    let book = book(|_| 0);
    let u = Unit { mp: 4, native_command_mask: [1 << 2, 0, 0, 1 << 4, 0], ..Default::default() };
    assert_eq!(native_available_ai_scored_command_ids(Some(&u), &book), [2, 28], "raw AI scored commands");
}

/// Go：`TestNativeAvailableAIScoredCommandIDsFailsClosedOnRaw27Gate`
#[test]
fn native_available_ai_scored_command_ids_fails_closed_on_raw27_gate() {
    let book = book(|_| 0);
    let u = Unit { native_transient: [0, 0, 0, 0, 0, 1], native_command_mask: [0xff; 5], ..Default::default() };
    let got = native_available_ai_scored_command_ids(Some(&u), &book);
    assert!(got.is_empty(), "raw +0x27 gate must reject AI scored commands: {got:?}");
}
