//! 移植自 `remake/internal/battle/native_command_compound_test.go`。

use super::*;

fn step(callee: u32, command_id: i64, marker_offset: i64, amount: i64) -> NativeCompoundStep {
    NativeCompoundStep { callee, command_id, marker_offset, amount }
}

/// Go：`TestNativeCompoundCommandPlanPreservesRawOrder`
#[test]
fn native_compound_command_plan_preserves_raw_order() {
    let checks: Vec<(i64, Vec<NativeCompoundStep>)> = vec![
        (32, vec![step(0x2111a, 32, 0, 0)]),
        (33, vec![step(0, 33, 0x25, 0), step(0, 33, 0x26, 0), step(0, 33, 0x27, 0), step(0x211a4, 33, 0, 0x320)]),
        (34, vec![step(0x22721, 17, 0x22, 0), step(0x22866, 18, 0x23, 0), step(0x22997, 19, 0x24, 0)]),
        (35, vec![step(0x22d1b, 26, 0x25, 0), step(0x22d1b, 22, 0x27, 0), step(0x22d1b, 27, 0x26, 0)]),
    ];
    for (id, want) in &checks {
        let (got, ok) = native_compound_command_plan(*id);
        assert!(ok && got.len() == want.len(), "id {id} plan = {got:?}, {ok}; want {want:?}");
        for i in 0..want.len() {
            assert!(got[i] == want[i], "id {id} step {i} = {:?}; want {:?}", got[i], want[i]);
        }
    }
    let (got, ok) = native_compound_command_plan(31);
    // Go 檢查 `got != nil`；Rust 的 Vec 不分 nil，改查空。
    assert!(!ok && got.is_empty(), "unsupported compound command accepted: {got:?}, {ok}");
}
