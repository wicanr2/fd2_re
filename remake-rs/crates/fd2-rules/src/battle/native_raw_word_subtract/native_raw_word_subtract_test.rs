//! 移植自 `remake/internal/battle/native_raw_word_subtract_test.go`。

use super::*;
use crate::test_support::{get16, put16};

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestApplyNativeRawWordSubtractWrapsLow16`
#[test]
fn apply_native_raw_word_subtract_wraps_low16() {
    let mut records = vec![0u8; RS];
    put16(&mut records, 0x44, 2);
    let got = apply_native_raw_word_subtract(&mut records, 0, 0x44, 5).unwrap();
    assert!(got == 0xfffd && get16(&records, 0x44) == 0xfffd, "got={got:#x} raw={:#x}", get16(&records, 0x44));
}

/// Go：`TestApplyNativeRawWordSubtractPreflights`（Go 的 `t.Run` 子測試改成迴圈）
#[test]
fn apply_native_raw_word_subtract_preflights() {
    for (name, unit, off, amount) in
        [("unit", -1, 0x44, 1), ("offset", 0, NATIVE_RECORD_SIZE, 1), ("amount", 0, 0x44, 0x100)]
    {
        assert!(
            apply_native_raw_word_subtract(&mut [0u8; RS], unit, off, amount).is_err(),
            "{name}: invalid raw subtract unexpectedly accepted"
        );
    }
}
