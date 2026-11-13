//! 移植自 `remake/internal/battle/native_ai_physical_candidates_test.go`。

use super::*;
use crate::battle::native_ai_runtime::d_test_support::*;

fn ones_cost_row() -> Vec<u8> {
    vec![1u8; NATIVE_MOVEMENT_COST_ROW_SIZE as usize]
}

/// Go：`TestBuildNativeAIPhysicalAttackCandidatesKeepsRawGeometryAndOrder`
#[test]
fn build_native_ai_physical_attack_candidates_keeps_raw_geometry_and_order() {
    let mut records = vec![0u8; 3 * RS];
    // actor: raw +6==0 at (0,0)
    (records[0], records[1], records[6]) = (0, 0, 0);
    // two opposite raw +6==1 targets
    let first = RS;
    (records[first], records[first + 1], records[first + 6]) = (2, 0, 1);
    let second = 2 * RS;
    (records[second], records[second + 1], records[second + 6]) = (1, 1, 1);
    let before = records.clone();

    let terrain = vec![0u8; 6];
    let cost = ones_cost_row();
    let mut seen: Vec<NativeAIPhysicalAttackRawCandidate> = Vec::new();
    let mut resolve = |raw: NativeAIPhysicalAttackRawCandidate| {
        assert!(
            raw.actor_record.len() == RS && raw.target_record.len() == RS,
            "raw record lengths={}/{}",
            raw.actor_record.len(),
            raw.target_record.len()
        );
        assert!(
            raw.actor_record[6] == 0 && raw.target_record[6] == 1,
            "raw groups actor={} target={}",
            raw.actor_record[6],
            raw.target_record[6]
        );
        // The resolver is caller-owned terrain/helper provenance. Keep the
        // test values raw-shaped; no Camp/X/Y projection is used.
        let target_word4_a = if raw.target_index == 2 { 1 } else { 10 };
        seen.push(raw);
        Ok(NativePhysicalAttackScoreInput {
            actor_word48: 14,
            target_word4_a,
            target_word40: 3,
            raw_target_byte8: 1,
            ..Default::default()
        })
    };
    let candidates = build_native_ai_physical_attack_candidates(
        3,
        2,
        &records,
        3,
        0,
        0,
        2,
        NativeAIPhysicalAttackGeometry { target_mode: 1, target_inner_mark: 0, target_code: 1 },
        &[0; 6],
        &[0; 6],
        &terrain,
        &cost,
        Some(&mut resolve),
    )
    .unwrap();
    assert!(records == before, "candidate builder mutated raw records");
    assert!(
        candidates.len() == 3 && seen.len() == candidates.len(),
        "candidate count={} seen={}, want 3",
        candidates.len(),
        seen.len()
    );
    assert!(
        candidates[0].destination_x == 1 && candidates[0].destination_y == 0 && candidates[0].target_index == 1,
        "first candidate={:?}, want row-major destination (1,0)/target 1",
        candidates[0]
    );
    let (selection, ok) = select_native_physical_attack_candidate(&candidates).unwrap();
    assert!(ok, "selection ok={ok}");
    assert!(
        selection.candidate.target_index == 2
            && selection.ranking == NativePhysicalAttackScore { priority: 18, score: 26 },
        "selection={selection:?}, want target 2 priority18 score26"
    );

    // The resolver receives detached snapshots, not aliases into the input.
    seen[0].actor_record[0] = 0xff;
    assert!(records[0] != 0xff, "resolver received aliased actor record");
}

/// Go：`TestBuildNativeAIPhysicalAttackCandidatesFailsClosedWithoutScoreProvenance`
#[test]
fn build_native_ai_physical_attack_candidates_fails_closed_without_score_provenance() {
    let records = vec![0u8; RS];
    let terrain = vec![0u8; 1];
    assert!(
        build_native_ai_physical_attack_candidates(
            1,
            1,
            &records,
            1,
            0,
            0,
            0,
            NativeAIPhysicalAttackGeometry { target_mode: 1, target_code: 1, ..Default::default() },
            &terrain,
            &terrain,
            &terrain,
            &ones_cost_row(),
            None,
        )
        .is_err(),
        "missing raw score resolver was accepted"
    );
}

/// Go：`TestBuildNativeAIPhysicalAttackCandidatesRejectsResolverError`
#[test]
fn build_native_ai_physical_attack_candidates_rejects_resolver_error() {
    let mut records = vec![0u8; 2 * RS];
    records[6] = 0;
    records[RS + 6] = 1;
    (records[RS], records[RS + 1]) = (1, 0);
    let terrain = vec![0u8; 2];
    let mut resolve =
        |_: NativeAIPhysicalAttackRawCandidate| -> Result<_> { Err(battle_err!("raw score provenance unavailable")) };
    let got = build_native_ai_physical_attack_candidates(
        2,
        1,
        &records,
        2,
        0,
        0,
        1,
        NativeAIPhysicalAttackGeometry { target_mode: 1, target_code: 1, ..Default::default() },
        &terrain,
        &terrain,
        &terrain,
        &ones_cost_row(),
        Some(&mut resolve),
    );
    assert!(got.is_err(), "resolver error was swallowed");
}
