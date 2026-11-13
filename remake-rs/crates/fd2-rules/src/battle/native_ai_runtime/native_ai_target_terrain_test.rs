//! 移植自 `remake/internal/battle/native_ai_target_terrain_test.go`。
//!
//! 地形修正查 E 組的 `native_terrain_apdp_pct`（`terrain.go`）。

use super::*;
use crate::battle::native_ai_runtime::d_test_support::*;

/// Go：`TestNativeAIPhysicalTargetTerrainReadsTargetCell`
#[test]
fn native_ai_physical_target_terrain_reads_target_cell() {
    let (mut actor, mut target) = (vec![0u8; RS], vec![0u8; RS]);
    (actor[0x20], target[0x1f]) = (2, 4);
    (target[0], target[1], target[8]) = (1, 1, 17);
    put_record16(&mut actor, 0, 0x48, 200);
    put_record16(&mut actor, 0, 0x4a, 78);
    put_record16(&mut target, 0, 0x48, 216);
    put_record16(&mut target, 0, 0x4a, 154);
    put_record16(&mut target, 0, 0x40, 157);
    let state = State { w: 3, h: 2, native_terrain_move_codes: vec![0, 0, 0, 0, 1, 2], ..Default::default() };
    let item_rows = vec![0u8; NATIVE_ITEM_EFFECT_ROW_SIZE as usize];
    for destination in [Cell { x: 0, y: 1 }, Cell { x: 2, y: 1 }] {
        let raw = NativeAIPhysicalAttackRawCandidate {
            actor_record: actor.clone(),
            target_record: target.clone(),
            destination,
            ..Default::default()
        };
        let got = state.native_ai_physical_score_input(&raw, &item_rows).unwrap();
        assert!(
            got.target_word48 == 216 && got.target_word4_a == 154 && got.actor_word48 == 200 && got.actor_word4_a == 78,
            "目的地{destination:?} 混入目標格地形：{got:?}"
        );
    }
    // 相同 raw 閘門仍只對非零 sub_1F183 的 actor 套目的地地形。
    actor[0x1f] = 4;
    let raw = NativeAIPhysicalAttackRawCandidate {
        actor_record: actor,
        target_record: target,
        destination: Cell { x: 2, y: 1 },
        ..Default::default()
    };
    let got = state.native_ai_physical_score_input(&raw, &item_rows);
    assert!(
        matches!(&got, Ok(g) if g.actor_word48 == 190 && g.actor_word4_a == 85 && g.target_word48 == 216 && g.target_word4_a == 154),
        "actor／target 地形來源未分開：{got:?}"
    );
}
