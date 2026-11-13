//! 移植自 `native_unit_mode_test.go`。

use crate::fdother::*;

// Go: TestPlanNativeUnitModeMasksModeAndPreservesRawArguments
#[test]
fn plan_native_unit_mode_masks_mode_and_preserves_raw_arguments() {
    let mut raw = vec![0u8; 0x3e];
    raw[0x34] = 0xf9;
    raw[0x35] = 0x80;
    raw[0x36] = 0x17;
    raw[0x3d] = 0x04;
    let got = plan_native_unit_mode(&raw, 7, 1).unwrap();
    assert_eq!(
        got,
        Some(NativeUnitModePlan {
            unit_index: 7,
            caller_arg: 1,
            mode: 9,
            raw_arg35: 0x80,
            raw_arg36: 0x17,
            raw_counter: 4
        })
    );
}

// Go: TestPlanNativeUnitModeRejectsGatedRecordWithoutMutation
#[test]
fn plan_native_unit_mode_rejects_gated_record_without_mutation() {
    let mut raw = vec![0u8; 0x3e];
    raw[5] = 0x04;
    raw[0x34] = 3;
    assert_eq!(plan_native_unit_mode(&raw, 0, 1), Ok(None));
}

// Go: TestPlanNativeUnitModeRejectsShortRecord
#[test]
fn plan_native_unit_mode_rejects_short_record() {
    assert!(plan_native_unit_mode(&[0u8; 0x3d], 0, 0).is_err(), "expected short record to fail closed");
}
