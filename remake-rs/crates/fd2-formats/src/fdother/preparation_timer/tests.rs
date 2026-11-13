//! 移植自 `preparation_timer_test.go`。

use crate::fdother::*;

// Go: TestTickNativePreparationTimersOnlyEmitsOneToZeroTransitions
#[test]
fn tick_native_preparation_timers_only_emits_one_to_zero_transitions() {
    let mut records = vec![0u8; 2 * 0x50];
    records[6] = 4;
    records[5] = 0;
    records[0x22] = 1;
    records[0x23] = 2;
    records[0x24] = 0;
    // 停用與 selector 不符的記錄不動。
    records[0x50 + 6] = 4;
    records[0x50 + 5] = 1;
    records[0x50 + 0x22] = 1;
    let expired = tick_native_preparation_timers(&mut records, 2, 4).unwrap();
    assert_eq!(
        expired,
        [NativePreparationTimerExpiry {
            record_index: 0,
            counter_index: 0,
            counter_offset: 0x22,
            visual_source_offset: 0x1e1
        }]
    );
    assert!(
        records[0x22] == 0 && records[0x23] == 1 && records[0x50 + 0x22] == 1,
        "records mutated unexpectedly: {},{},{}",
        records[0x22],
        records[0x23],
        records[0x50 + 0x22]
    );
}

// Go: TestTickNativePreparationTimersRejectsTruncatedInput
#[test]
fn tick_native_preparation_timers_rejects_truncated_input() {
    assert!(tick_native_preparation_timers(&mut [0u8; 0x4f], 1, 0).is_err(), "truncated records must fail closed");
}
