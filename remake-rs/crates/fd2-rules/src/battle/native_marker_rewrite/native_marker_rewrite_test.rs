//! 移植自 `remake/internal/battle/native_marker_rewrite_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestRewriteNativeMarker1FMutatesOnlyMatchingTailSlots`
#[test]
fn rewrite_native_marker1_f_mutates_only_matching_tail_slots() {
    let mut records = vec![0u8; 19 * RS];
    records[16 * RS + 7] = 0x1f;
    records[17 * RS + 7] = 0x20;
    records[18 * RS + 7] = 0x1f;
    rewrite_native_marker1_f(&mut records, 19).unwrap();
    assert!(records[16 * RS] == 0x10 && records[16 * RS + 1] == 0x06, "slot16 prefix");
    assert!(records[17 * RS] == 0 && records[17 * RS + 1] == 0, "nonmatching marker mutated");
    assert!(records[18 * RS] == 0x10 && records[18 * RS + 1] == 0x06, "slot18 marker not rewritten");
}

/// Go：`TestRewriteNativeMarker1FLeavesPrefixAndRejectsCount`
#[test]
fn rewrite_native_marker1_f_leaves_prefix_and_rejects_count() {
    let mut records = vec![0u8; 0x10 * RS];
    rewrite_native_marker1_f(&mut records, 0x10).unwrap();
    assert!(rewrite_native_marker1_f(&mut records, 0x11).is_err(), "short record buffer unexpectedly accepted");
    assert!(rewrite_native_marker1_f(&mut records, -1).is_err(), "negative count unexpectedly accepted");
}
