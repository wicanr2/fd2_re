//! 移植自 `remake/internal/battle/native_ai_choice_test.go`。

use super::*;

fn candidate(command_id: i64, command_value: i64, x: i64, y: i64, score: i64) -> NativeAISpellCandidate {
    NativeAISpellCandidate { command_id, command_value, x, y, score, ..Default::default() }
}

/// Go：`TestSelectNativeAISpellCandidateUsesScoreThenRawCommandValue`
#[test]
fn select_native_ai_spell_candidate_uses_score_then_raw_command_value() {
    let (got, ok) = select_native_ai_spell_candidate(&[
        candidate(17, 4, 1, 2, 6),
        candidate(18, 7, 3, 4, 6),
        candidate(19, 99, 5, 6, 5),
    ]);
    assert!(ok && got.command_id == 18 && got.x == 3 && got.y == 4, "choice={got:?} ok={ok}");
    let (got, ok) = select_native_ai_spell_candidate(&[candidate(17, 7, 0, 0, 6), candidate(18, 7, 0, 0, 6)]);
    assert!(ok && got.command_id == 17, "exact tie should keep first candidate: {got:?}");
}

/// Go：`TestSelectNativeAISpellCandidateEmptyIsFailClosed`
#[test]
fn select_native_ai_spell_candidate_empty_is_fail_closed() {
    let (_, ok) = select_native_ai_spell_candidate(&[]);
    assert!(!ok, "empty native AI candidate list selected a command");
}
