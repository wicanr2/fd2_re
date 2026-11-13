//! 移植自 `preparation_input_test.go`。

use crate::fdother::*;

// Go: TestApplyNativePreparationInputMatchesScancodeBranches
#[test]
fn apply_native_preparation_input_matches_scancode_branches() {
    for scan in [0xe0, 0x52, 0x1c, 0x39] {
        let mut state = 1;
        let got = apply_native_preparation_input(scan, Some(&mut state));
        assert!(got == NativePreparationInputResult::CONFIRM && state == 0, "confirm {scan:#x} => {got:?} {state}");
    }
    for scan in [0x01, 0x53] {
        let mut state = 1;
        let got = apply_native_preparation_input(scan, Some(&mut state));
        assert!(got == NativePreparationInputResult::CANCEL && state == 0, "cancel {scan:#x} => {got:?} {state}");
    }
    for (scan, start, want) in [(0x4b, 1, 0), (0x4d, 0, 1), (0x7f, 1, 1)] {
        let mut state = start;
        let got = apply_native_preparation_input(scan, Some(&mut state));
        assert!(
            got == NativePreparationInputResult::CONTINUE && state == want,
            "continue {scan:#x} => {got:?} {state}"
        );
    }
}
