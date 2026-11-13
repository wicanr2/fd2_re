//! 移植自 `remake/internal/battle/native_word_delta_remove_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestApplyNativeWordDeltaAndRemoveUsesRawWordWrapAndRemoval`
#[test]
fn apply_native_word_delta_and_remove_uses_raw_word_wrap_and_removal() {
    let mut records = vec![0u8; 2 * RS];
    put16(&mut records, 0x20, 0xfffe);
    for slot in 0..8 {
        let off = RS + 0x0a + slot * 2;
        (records[off], records[off + 1]) = (slot as u8, 0xc0 + slot as u8);
    }
    let got = apply_native_word_delta_and_remove(&mut records, 0, 0x20, 3, 1, 1).unwrap();
    assert!(got == 1 && get16(&records, 0x20) == 1, "word={:#x} result={got:#x}", get16(&records, 0x20));
    assert!(
        records[RS + 0x0c] == 2 && records[RS + 0x18] == 0x80,
        "removed inventory not compacted: {:#x} {:#x}",
        records[RS + 0x0c],
        records[RS + 0x18]
    );
}

/// Go：`TestApplyNativeWordDeltaAndRemoveRejectsBeforeMutation`
#[test]
fn apply_native_word_delta_and_remove_rejects_before_mutation() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x20, 9);
    assert!(
        apply_native_word_delta_and_remove(&mut records, 0, NATIVE_RECORD_SIZE, 1, 0, 0).is_err(),
        "expected invalid word offset"
    );
    assert_eq!(get16(&records, 0x20), 9, "invalid input mutated word");
    assert!(
        apply_native_word_delta_and_remove(&mut records, 0, 0x20, 1, 1, 0).is_err(),
        "expected invalid removal unit"
    );
}
