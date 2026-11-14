//! Go `remake/internal/campaign/menu_state_test.go` 的逐條移植。

use super::*;

/// Go：`TestMenuStateTraceIsBoundedAndDataDriven`
#[test]
fn menu_state_trace_is_bounded_and_data_driven() {
    let mut s = MenuState { selection: 9, count: 3 };
    let (got, confirm) = s.step(MENU_TICK);
    assert!(got == 2 && !confirm, "initial clamp=({got},{confirm}), want (2,false)");
    s.step(MENU_DOWN);
    assert_eq!(s.selection, 2, "down at last option wrapped to {}", s.selection);
    s.step(MENU_UP);
    assert_eq!(s.selection, 1, "up selection={}, want 1", s.selection);
    let (got, confirm) = s.step(MENU_CONFIRM);
    assert!(got == 1 && confirm, "confirm=({got},{confirm}), want (1,true)");
    s.count = 0;
    let (got, confirm) = s.step(MENU_CONFIRM);
    assert!(got == 0 && !confirm, "empty menu confirm=({got},{confirm})");
}
