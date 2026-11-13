//! 移植自 `split_slide_test.go`。

use crate::fdother::*;

#[test]
fn native_battle_entry_split_slide_steps_match_verified_abi() {
    let steps = native_battle_entry_split_slide_steps();
    let want = [
        [100, -15, 82, 265, 81],
        [75, 10, 82, 240, 81],
        [50, 35, 82, 215, 81],
        [25, 60, 82, 190, 81],
        [0, 85, 82, 165, 81],
    ];
    assert_eq!(steps.len(), 5);
    for (i, step) in steps.iter().enumerate() {
        assert_eq!([step.offset, step.left_x, step.left_y, step.right_x, step.right_y], want[i], "step {i}");
    }
}

#[test]
fn run_native_battle_entry_split_slide_preserves_present_restore_order() {
    let cell = LMI1Entry { width: 2, height: 1, pixels: vec![7, 0] };
    let mut dst = vec![0u8; 456 * 200];
    let events = std::cell::RefCell::new(Vec::new());
    run_native_battle_entry_split_slide::<FdotherError>(
        &cell,
        &mut dst,
        || {
            events.borrow_mut().push("present");
            Ok(())
        },
        || {
            events.borrow_mut().push("restore");
            Ok(())
        },
    )
    .unwrap();
    let events = events.into_inner();
    assert_eq!(events.len(), 10, "events={events:?}, want five present/restore pairs");
    for pair in events.chunks(2) {
        assert_eq!(pair, ["present", "restore"], "events={events:?}");
    }
    // 最後一次在未位移的左錨點；透明的 0 必須保留它右邊那個位元組。
    assert!(dst[82 * 456 + 85] == 7 && dst[82 * 456 + 86] == 0, "last cell pixels");
}
