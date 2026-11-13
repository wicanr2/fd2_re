//! Go 來源：`remake/internal/battle/native_ai_spell_score.go`（167 行）。

use super::*;

/// ScoreNativeAISpellAttack preserves the raw scoring loop at 0x15b77 for
/// spell IDs 0..12.  The callback is the caller-owned 0x1f183 gate; it is
/// required for IDs 10..12 and intentionally has no gameplay name here.
///
/// Go：`native_ai_spell_score.go:11`，`func ScoreNativeAISpellAttack(records []byte, candidates []byte, spellID, spellValue int, skip func(int) bool) (int, error)`
pub fn score_native_ai_spell_attack(
    records: &[u8],
    candidates: &[u8],
    spell_id: i64,
    spell_value: i64,
    skip: Option<&mut dyn FnMut(i64) -> bool>,
) -> Result<i64> {
    todo!("battle/native_ai_spell_score.go:ScoreNativeAISpellAttack")
}

/// ScoreNativeAISpellRecovery preserves the raw spell IDs 13..16 branch:
/// score 8 below max/3, score 3 below max/2, otherwise zero; raw +0x34 bit0
/// doubles the selected score.
///
/// Go：`native_ai_spell_score.go:45`，`func ScoreNativeAISpellRecovery(records []byte, candidates []byte, spellID int) (int, error)`
pub fn score_native_ai_spell_recovery(records: &[u8], candidates: &[u8], spell_id: i64) -> Result<i64> {
    todo!("battle/native_ai_spell_score.go:ScoreNativeAISpellRecovery")
}

/// ScoreNativeAISpellFlag preserves the ID20/21 branches: each nonzero raw
/// +0x25/+0x26 flag contributes six points. The byte remains unnamed and this
/// function does not apply or clear it.
///
/// Go：`native_ai_spell_score.go:74`，`func ScoreNativeAISpellFlag(records []byte, candidates []byte, spellID int) (int, error)`
pub fn score_native_ai_spell_flag(records: &[u8], candidates: &[u8], spell_id: i64) -> Result<i64> {
    todo!("battle/native_ai_spell_score.go:ScoreNativeAISpellFlag")
}

/// ScoreNativeAISpellZeroFlag preserves the 0x15b77 helper branches that add
/// a score when a raw byte is zero: IDs17/18/19 use +0x22/+0x23/+0x24 and add
/// three; IDs26/27 use +0x25/+0x26 and add four.
///
/// Go：`native_ai_spell_score.go:100`，`func ScoreNativeAISpellZeroFlag(records []byte, candidates []byte, spellID int) (int, error)`
pub fn score_native_ai_spell_zero_flag(records: &[u8], candidates: &[u8], spell_id: i64) -> Result<i64> {
    todo!("battle/native_ai_spell_score.go:ScoreNativeAISpellZeroFlag")
}

/// ScoreNativeAISpell22 preserves the ID22 branch at 0x15d30: a zero raw
/// +0x27 gate followed by 0x1c269(unit, nil), which scans raw +0x1a..+0x1e
/// bits. A candidate with any set bit contributes six points. The bytes are
/// intentionally left unnamed and untouched.
///
/// Go：`native_ai_spell_score.go:133`，`func ScoreNativeAISpell22(records []byte, candidates []byte) (int, error)`
pub fn score_native_ai_spell22(records: &[u8], candidates: &[u8]) -> Result<i64> {
    todo!("battle/native_ai_spell_score.go:ScoreNativeAISpell22")
}

/// Go：`native_ai_spell_score.go:157`，`func validateNativeAISpellCandidates(records []byte, candidates []byte) error`
pub(crate) fn validate_native_ai_spell_candidates(records: &[u8], candidates: &[u8]) -> Result<()> {
    todo!("battle/native_ai_spell_score.go:validateNativeAISpellCandidates")
}
