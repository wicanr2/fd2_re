//! 移植自 `remake/internal/battle/native_record_byte5_bit0_test.go`。

use super::*;

const RS: usize = NATIVE_RECORD_SIZE as usize;

/// Go：`TestNativeRecordByte5Bit0MasksOnlyLowBit`
#[test]
fn native_record_byte5_bit0_masks_only_low_bit() {
    let mut records = vec![0u8; 2 * RS];
    records[5] = 0xff;
    records[RS + 5] = 0xfe;
    assert_eq!(native_record_byte5_bit0(&records, 0), Ok(1), "record0");
    assert_eq!(native_record_byte5_bit0(&records, 1), Ok(0), "record1");
}

/// Go：`TestNativeRecordByte5Bit0RejectsBounds`
#[test]
fn native_record_byte5_bit0_rejects_bounds() {
    assert!(native_record_byte5_bit0(&[0u8; RS], 1).is_err(), "out-of-range record unexpectedly accepted");
}
