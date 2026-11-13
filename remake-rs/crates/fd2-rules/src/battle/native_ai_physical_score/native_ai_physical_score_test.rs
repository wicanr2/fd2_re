//! 移植自 `remake/internal/battle/native_ai_physical_score_test.go`。

use super::*;

fn score(priority: i64, score: i64) -> NativePhysicalAttackScore {
    NativePhysicalAttackScore { priority, score }
}

/// Go：`TestScoreNativePhysicalAttackCandidateRetainsLowScoreFallthrough`
#[test]
fn score_native_physical_attack_candidate_retains_low_score_fallthrough() {
    let got = score_native_physical_attack_candidate(NativePhysicalAttackScoreInput {
        actor_word48: 12,
        target_word4_a: 10,
        target_word40: 1,
        raw_target_byte8: 1,
        ..Default::default()
    });
    assert!(matches!(got, Ok((s, true)) if s == score(18, 4)), "score={got:?}，低分仍應通過生命比較");
}

/// Go：`TestSelectNativePhysicalAttackCandidateUsesZeroInitialRanking`
#[test]
fn select_native_physical_attack_candidate_uses_zero_initial_ranking() {
    for s in [-2, 0, 1, 2] {
        let candidate = NativePhysicalAttackCandidate {
            target_index: 7,
            inputs: NativePhysicalAttackScoreInput {
                actor_word48: 10 + s,
                target_word4_a: 10,
                target_word40: 99,
                raw_target_byte8: 1,
                ..Default::default()
            },
            ..Default::default()
        };
        let got = select_native_physical_attack_candidate(&[candidate, candidate]);
        let Ok((got, ok)) = got else { panic!("分數{s}：錯誤={got:?}") };
        assert!(ok == (s > 0), "分數{s}：候選={got:?} 選中={ok}");
        if ok {
            assert!(
                got.candidate.target_index == 7 && got.ranking.priority == 0 && got.ranking.score == s,
                "priority0 候選比較錯誤：{got:?}"
            );
        }
    }
}

/// Go：`TestScoreNativePhysicalAttackCandidateLowScoreKeepsCounterAdjustment`
#[test]
fn score_native_physical_attack_candidate_low_score_keeps_counter_adjustment() {
    let got = score_native_physical_attack_candidate(NativePhysicalAttackScoreInput {
        actor_word48: 9,
        target_word4_a: 10,
        target_word40: 99,
        actor_word4_a: 20,
        target_word48: 10,
        raw_helper1_debe_result: 1,
        raw_target_byte8: 0,
    });
    assert!(matches!(got, Ok((s, true)) if s == score(0, 13)), "低分反擊／身份加權={got:?}");
}

/// Go：`TestScoreNativePhysicalAttackCandidateUsesStrictHPComparison`
#[test]
fn score_native_physical_attack_candidate_uses_strict_hp_comparison() {
    let got = score_native_physical_attack_candidate(NativePhysicalAttackScoreInput {
        actor_word48: 13,
        target_word4_a: 10,
        target_word40: 3,
        raw_target_byte8: 1,
        ..Default::default()
    });
    assert!(matches!(got, Ok((s, true)) if s == score(8, 3)), "score={got:?}");

    let got = score_native_physical_attack_candidate(NativePhysicalAttackScoreInput {
        actor_word48: 14,
        target_word4_a: 10,
        target_word40: 3,
        raw_target_byte8: 1,
        ..Default::default()
    });
    assert!(matches!(got, Ok((s, true)) if s == score(18, 8)), "kill score={got:?}");
}

/// Go：`TestScoreNativePhysicalAttackCandidatePreservesRawAdjustments`
#[test]
fn score_native_physical_attack_candidate_preserves_raw_adjustments() {
    let got = score_native_physical_attack_candidate(NativePhysicalAttackScoreInput {
        actor_word48: 14,
        actor_word4_a: 5,
        target_word48: 8,
        target_word4_a: 10,
        target_word40: 3,
        raw_target_byte8: 0,
        raw_helper1_debe_result: 1,
    });
    assert!(matches!(got, Ok((s, true)) if s == score(18, 7)), "score={got:?}");
}

/// Go：`TestScoreNativePhysicalAttackCandidateRejectsNonNativeWords`
#[test]
fn score_native_physical_attack_candidate_rejects_non_native_words() {
    assert!(
        score_native_physical_attack_candidate(NativePhysicalAttackScoreInput {
            actor_word48: -1,
            ..Default::default()
        })
        .is_err(),
        "negative native word accepted"
    );
    assert!(
        score_native_physical_attack_candidate(NativePhysicalAttackScoreInput {
            actor_word48: 0x10000,
            ..Default::default()
        })
        .is_err(),
        "oversized native word accepted"
    );
}

/// Go：`TestSelectNativePhysicalAttackCandidateUsesPriorityThenScoreAndStableTie`
#[test]
fn select_native_physical_attack_candidate_uses_priority_then_score_and_stable_tie() {
    let candidate = |x, y, target, a48, t4a, t40| NativePhysicalAttackCandidate {
        destination_x: x,
        destination_y: y,
        target_index: target,
        inputs: NativePhysicalAttackScoreInput {
            actor_word48: a48,
            target_word4_a: t4a,
            target_word40: t40,
            raw_target_byte8: 1,
            ..Default::default()
        },
    };
    let candidates = [candidate(2, 3, 4, 110, 10, 100), candidate(5, 6, 7, 14, 10, 3), candidate(8, 9, 10, 14, 10, 3)];
    let (got, ok) = select_native_physical_attack_candidate(&candidates).unwrap();
    assert!(ok, "selection ok={ok}");
    assert!(
        got.candidate.target_index == 7 && got.ranking == score(18, 8),
        "selection={got:?}, want first priority-18 candidate"
    );
}

/// Go：`TestSelectNativePhysicalAttackCandidateFailsClosed`
#[test]
fn select_native_physical_attack_candidate_fails_closed() {
    let got = select_native_physical_attack_candidate(&[NativePhysicalAttackCandidate {
        destination_x: 0x100,
        inputs: NativePhysicalAttackScoreInput { actor_word48: 10, target_word4_a: 1, ..Default::default() },
        ..Default::default()
    }]);
    assert!(got.is_err(), "selection={got:?}, want rejected");
}
