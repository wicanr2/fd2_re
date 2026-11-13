//! 移植自 `preparation_dispatch_test.go`。

use crate::fdother::*;

// Go: TestFindNativePreparationDispatchUsesOverlappingThreeByteStride
#[test]
fn find_native_preparation_dispatch_uses_overlapping_three_byte_stride() {
    let mut table = vec![0u8; 51];
    // slot 2 從 6 開始；它的 +3/+5 閘門與 +4 callback 索引和相鄰 raw 位元組重疊，與原版位址運算相同。
    table[6 + 3..6 + 6].copy_from_slice(&[0x2a, 0x17, 0x09]);
    // 第二個符合的 slot 證明掃描不會在第一個命中後停止。
    table[9 + 3..9 + 6].copy_from_slice(&[0x2a, 0x22, 0x09]);
    let matches = find_native_preparation_dispatch(&table, 0x2a, 0x09).unwrap();
    assert_eq!(
        matches,
        [
            NativePreparationDispatchMatch { slot: 2, entry_start: 6, function_index: 0x17 },
            NativePreparationDispatchMatch { slot: 3, entry_start: 9, function_index: 0x22 },
        ]
    );
}

// Go: TestFindNativePreparationDispatchRejectsShortTable
#[test]
fn find_native_preparation_dispatch_rejects_short_table() {
    assert!(find_native_preparation_dispatch(&[0u8; 50], 0, 0).is_err(), "short overlapping dispatch table");
}
