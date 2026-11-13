//! 移植自 `remake/internal/battle/native_unit_byte_clear_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestClearNativeUnitByteMatchesRawMutationAndEffectiveValue`
#[test]
fn clear_native_unit_byte_matches_raw_mutation_and_effective_value() {
    let mut records = vec![0u8; 2 * RS];
    records[0x14] = 1;
    records[0x20] = 9;
    records[0x21] = 2;
    records[RS + 0x14] = 0;
    let (got, total) = clear_native_unit_byte(&mut records, &[0, 1], 0x14).unwrap();
    assert!(
        records[0x14] == 0
            && got.len() == 2
            && got[0].cleared
            && got[0].raw_value == 4 * 32
            && !got[1].cleared
            && total == 4 * 32,
        "results={got:?} total={total} records={:#x}",
        records[0x14]
    );
}

/// Go：`TestClearNativeUnitByteDoesNotAddThirtyAtBoundary`
#[test]
fn clear_native_unit_byte_does_not_add_thirty_at_boundary() {
    let mut records = vec![0u8; RS];
    records[0x16] = 1;
    records[0x20] = 25;
    records[0x21] = 3;
    let (got, total) = clear_native_unit_byte(&mut records, &[0], 0x16).unwrap();
    assert!(got[0].cleared && total == 12, "results={got:?} total={total}");
}

/// Go：`TestClearNativeUnitByteRejectsBadFieldOrIndex`
#[test]
fn clear_native_unit_byte_rejects_bad_field_or_index() {
    assert!(
        clear_native_unit_byte(&mut [0u8; RS], &[0], NATIVE_RECORD_SIZE).is_err(),
        "expected bad field offset to fail closed"
    );
    assert!(clear_native_unit_byte(&mut [0u8; RS], &[1], 0x14).is_err(), "expected bad unit index to fail closed");
}
