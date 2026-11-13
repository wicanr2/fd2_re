//! 移植自 `remake/internal/battle/native_ai_spell_score_test.go`。

use super::*;
use crate::battle::native_ai_runtime::d_test_support::*;

/// Go：`TestScoreNativeAISpellAttackPreservesRawBranches`
#[test]
fn score_native_ai_spell_attack_preserves_raw_branches() {
    let mut records = vec![0u8; 2 * RS];
    put_record16(&mut records, 0, 0x40, 10);
    put_record16(&mut records, 1, 0x40, 20);
    let got = score_native_ai_spell_attack(&records, &[0, 1], 0, 15, None);
    assert!(matches!(got, Ok(48)), "score={got:?}"); // 24*1.5 + 8*1.5, toward-zero
    assert!(
        score_native_ai_spell_attack(&records, &[0], 10, 15, None).is_err(),
        "ID10 accepted without its raw inactive gate"
    );
    let mut gate = |index: i64| index == 1;
    let got = score_native_ai_spell_attack(&records, &[0, 1], 10, 15, Some(&mut gate));
    assert!(matches!(got, Ok(36)), "gated score={got:?}");
}

/// Go：`TestNativeAITargetGateFeedsAttackIDsTenThroughTwelve`
#[test]
fn native_ai_target_gate_feeds_attack_ids_ten_through_twelve() {
    let mut records = vec![0u8; 5 * RS];
    for i in 0..5 {
        records[i * RS + 8] = 1;
        put_record16(&mut records, i, 0x40, 10);
    }
    records[0x20] = 19;
    records[RS + 0x1f] = 4;
    records[2 * RS + 0x1f] = 5;
    // +7=0x1C優先回0，不能因race/class成立就略過這一筆。
    records[3 * RS + 7] = 0x1c;
    records[3 * RS + 0x20] = 19;
    records[3 * RS + 0x1f] = 4;
    let mut gate = native_ai1_f183_target_gate(&records);
    for command in [10, 11, 12] {
        let got = score_native_ai_spell_attack(&records, &[0, 1, 2, 3, 4], command, 15, Some(&mut *gate));
        assert!(matches!(got, Ok(48)), "command{command} score={got:?}");
    }
    assert!(
        score_native_ai_spell_attack(&records, &[5], 10, 15, Some(&mut *gate)).is_err(),
        "out-of-range target reached the raw record callback"
    );
}

/// Go：`TestScoreNativeAISpellRecoveryUsesHalfAndRawBit`
#[test]
fn score_native_ai_spell_recovery_uses_half_and_raw_bit() {
    let mut records = vec![0u8; RS];
    put_record16(&mut records, 0, 0x40, 4);
    put_record16(&mut records, 0, 0x42, 10);
    records[0x34] = 1;
    let got = score_native_ai_spell_recovery(&records, &[0], 13);
    assert!(matches!(got, Ok(6)), "score={got:?}");
    put_record16(&mut records, 0, 0x40, 2);
    let got = score_native_ai_spell_recovery(&records, &[0], 13);
    assert!(matches!(got, Ok(16)), "below-third score={got:?}");
    put_record16(&mut records, 0, 0x40, 5);
    let got = score_native_ai_spell_recovery(&records, &[0], 13);
    assert!(matches!(got, Ok(0)), "half-health score={got:?}");
}

/// Go：`TestScoreNativeAISpellFlagUsesVerifiedOffsets`
#[test]
fn score_native_ai_spell_flag_uses_verified_offsets() {
    let mut records = vec![0u8; 2 * RS];
    records[0x25] = 1;
    records[RS + 0x26] = 1;
    let got = score_native_ai_spell_flag(&records, &[0, 1], 20);
    assert!(matches!(got, Ok(6)), "ID20 score={got:?}");
    let got = score_native_ai_spell_flag(&records, &[0, 1], 21);
    assert!(matches!(got, Ok(6)), "ID21 score={got:?}");
    let got = score_native_ai_spell_zero_flag(&records, &[0, 1], 26);
    assert!(matches!(got, Ok(4)), "ID26 zero score={got:?}");
    let got = score_native_ai_spell_zero_flag(&records, &[0, 1], 27);
    assert!(matches!(got, Ok(4)), "ID27 zero score={got:?}");
}

/// Go：`TestScoreNativeAISpellZeroFlagUsesTransientOffsets`
#[test]
fn score_native_ai_spell_zero_flag_uses_transient_offsets() {
    let records = vec![0u8; RS];
    for id in [17, 18, 19] {
        let got = score_native_ai_spell_zero_flag(&records, &[0], id);
        assert!(matches!(got, Ok(3)), "ID{id} score={got:?}");
    }
}

/// Go：`TestScoreNativeAISpell22UsesRawGateAndBitRange`
#[test]
fn score_native_ai_spell22_uses_raw_gate_and_bit_range() {
    let mut records = vec![0u8; 2 * RS];
    records[0x1a] = 1;
    records[RS + 0x1f - 1] = 1;
    records[RS + 0x27] = 1;
    let got = score_native_ai_spell22(&records, &[0, 1]);
    assert!(matches!(got, Ok(6)), "score={got:?}");
}

/// Go：`TestScoreNativeAISpellRejectsMalformedCandidates`
#[test]
fn score_native_ai_spell_rejects_malformed_candidates() {
    assert!(score_native_ai_spell_recovery(&[0u8; RS - 1], &[0], 13).is_err(), "malformed record buffer accepted");
    assert!(score_native_ai_spell_attack(&[0u8; RS], &[1], 0, 1, None).is_err(), "out-of-range candidate accepted");
}
