//! 移植自 `remake/internal/battle/native_persistent_lookup_test.go`。

use super::*;

const PS: usize = NATIVE_PERSISTENT_RECORD_SIZE as usize;

/// Go：`TestFindNativePersistentIdentityUsesCountStrideAndByteOffset`
#[test]
fn find_native_persistent_identity_uses_count_stride_and_byte_offset() {
    let mut records = vec![0u8; 4 * PS];
    records[PS + 0x08] = 12;
    records[3 * PS + 0x08] = 12;
    let before = records.clone();
    assert_eq!(find_native_persistent_identity(&records, 3, 12), Ok((1, true)));
    assert_eq!(records, before, "persistent lookup mutated records");
    assert_eq!(find_native_persistent_identity(&records, 3, 13).map(|r| r.1), Ok(false), "missing");
}

/// Go：`TestFindNativePersistentIdentityRejectsCountAndShortBuffer`
#[test]
fn find_native_persistent_identity_rejects_count_and_short_buffer() {
    assert!(
        find_native_persistent_identity(&vec![0u8; 32 * PS], 33, 1).is_err(),
        "count above native persistent capacity unexpectedly accepted"
    );
    assert!(
        find_native_persistent_identity(&[0u8; PS - 1], 1, 1).is_err(),
        "short persistent record unexpectedly accepted"
    );
}
