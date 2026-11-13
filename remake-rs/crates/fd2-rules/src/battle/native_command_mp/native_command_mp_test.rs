//! 移植自 `remake/internal/battle/native_command_mp_test.go`。

use super::*;

/// Go：`TestSpendNativeCommandMPMatchesConfirmedNativeDeduction`
#[test]
fn spend_native_command_mp_matches_confirmed_native_deduction() {
    let mut u = Unit { mp: 9, max_mp: 12, ..Default::default() };
    assert!(spend_native_command_mp(Some(&mut u), 6) && u.mp == 3, "successful native deduction: mp={}", u.mp);
}

/// Go：`TestSpendNativeCommandMPFailsClosedBeforeNativeConfirm`
#[test]
fn spend_native_command_mp_fails_closed_before_native_confirm() {
    let mut u = Unit { mp: 5, max_mp: 5, ..Default::default() };
    for cost in [6, -1, 0x100] {
        assert!(
            !spend_native_command_mp(Some(&mut u), cost) && u.mp == 5,
            "cost={cost} changed unit before valid confirm"
        );
    }
    assert!(!spend_native_command_mp(None, 0), "nil unit must fail closed");
}
