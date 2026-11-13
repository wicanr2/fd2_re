//! 移植自 `remake/internal/battle/native_ai_scored_group_score_test.go`。

use super::*;
use crate::battle::native_ai_runtime::d_test_support::*;

fn group(x: i64, y: i64, targets: &[u8]) -> NativeAIScoredCommandCandidateGroup {
    NativeAIScoredCommandCandidateGroup { destination: Cell { x, y }, target_indices: targets.to_vec() }
}

/// Go：`TestScoreNativeAIScoredCommandGroupsDispatchesAttackAndKeepsOrder`
#[test]
fn score_native_ai_scored_command_groups_dispatches_attack_and_keeps_order() {
    let mut records = vec![0u8; 3 * RS];
    put_record16(&mut records, 1, 0x40, 20);
    put_record16(&mut records, 2, 0x40, 80);
    records[RS + 8] = 1;
    records[2 * RS + 8] = 1;
    let groups = [group(2, 1, &[2]), group(1, 1, &[1, 2])];
    let got = score_native_ai_scored_command_groups(&records, 2, 50, &groups, None).unwrap();
    let want = vec![
        NativeAIScoredCommandGroupScore { destination: Cell { x: 2, y: 1 }, target_indices: vec![2], score: 8 },
        NativeAIScoredCommandGroupScore { destination: Cell { x: 1, y: 1 }, target_indices: vec![1, 2], score: 32 },
    ];
    assert!(got == want, "scores={got:?} want={want:?}");
}

/// Go：`TestScoreNativeAIScoredCommandGroupsDispatchesRecoveryAndZeroFamilies`
#[test]
fn score_native_ai_scored_command_groups_dispatches_recovery_and_zero_families() {
    let mut records = vec![0u8; RS];
    put_record16(&mut records, 0, 0x40, 10);
    put_record16(&mut records, 0, 0x42, 60);
    let groups = [group(0, 0, &[0])];
    let recovery = score_native_ai_scored_command_groups(&records, 13, 0, &groups, None);
    assert!(matches!(&recovery, Ok(r) if r.len() == 1 && r[0].score == 8), "recovery={recovery:?}");
    let zero = score_native_ai_scored_command_groups(&records, 28, 0, &groups, None);
    assert!(matches!(&zero, Ok(z) if z.len() == 1 && z[0].score == 0), "zero family={zero:?}");
}

/// Go：`TestScoreNativeAIScoredCommandGroupsRequiresGateForIDs10To12`
#[test]
fn score_native_ai_scored_command_groups_requires_gate_for_ids10_to12() {
    let records = vec![0u8; RS];
    assert!(
        score_native_ai_scored_command_groups(&records, 10, 1, &[group(0, 0, &[0])], None).is_err(),
        "command 10 accepted without 0x1f183 gate"
    );
}

/// Go：`TestScoreNativeAIScoredCommandGroupsReturnsDetachedTargets`
#[test]
fn score_native_ai_scored_command_groups_returns_detached_targets() {
    let records = vec![0u8; RS];
    let g = group(0, 0, &[0]);
    let mut got = score_native_ai_scored_command_groups(&records, 28, 0, std::slice::from_ref(&g), None).unwrap();
    got[0].target_indices[0] = 9;
    assert!(g.target_indices[0] == 0, "score result aliases candidate group targets");
}

/// Go：`TestMap0AssetsScoreNativeAIScoredCommandGroup`
#[test]
fn map0_assets_score_native_ai_scored_command_group() {
    let (Some(mut st), Some(spells), Some(rows_path)) = (
        load_asset_state("maps/map0/map0_units.json"),
        asset_path("spells.json"),
        asset_path("data/native_movement_cost_rows.json"),
    ) else {
        return;
    };
    for unit in &mut st.units {
        unit.materialize_native_map_presentation().unwrap();
    }
    let records = native_ai_scoring_records(&st.units).unwrap();
    let book = load_native_command_records(&spells).unwrap();
    let rows = load_native_movement_cost_rows(&rows_path).unwrap();
    let actor = st
        .units
        .iter()
        .position(|u| u.native_record_byte8 == 103 && u.has_native_record_byte8 && !u.has_native_identity)
        .expect("map0 raw +8 value 103 actor missing");
    let groups = native_ai_scored_command_candidate_groups(
        st.w,
        st.h,
        &records,
        st.units.len() as i64,
        actor as i64,
        0,
        book[0],
        &native_composition_base_flags_for_test(&st),
        &st.native_terrain_move_codes,
        &rows[0],
    )
    .unwrap();
    let scores = score_native_ai_scored_command_groups(&records, book[0].id, book[0].damage, &groups, None).unwrap();
    let Some(score) = scores.iter().find(|s| s.destination == Cell { x: 23, y: 14 }) else {
        panic!("map0 scored ally destination missing: {scores:?}");
    };
    assert!(
        score.target_indices.len() == 4 && score.score == 96,
        "map0 ally group targets={:?} score={} want four/96",
        score.target_indices,
        score.score
    );
}
