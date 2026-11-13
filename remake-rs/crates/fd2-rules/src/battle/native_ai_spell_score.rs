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
    if !(0..=12).contains(&spell_id) || spell_value < 0 {
        return Err(battle_err!("native AI attack score: invalid spell id/value"));
    }
    let mut skip = skip;
    if spell_id >= 10 && skip.is_none() {
        return Err(battle_err!("native AI attack score: missing 0x1f183 gate"));
    }
    validate_native_ai_spell_candidates(records, candidates)?;
    let mut score = 0;
    for &raw_index in candidates {
        let index = raw_index as i64;
        if spell_id >= 10 && skip.as_mut().is_some_and(|f| f(index)) {
            continue;
        }
        let record = native_ai_spell_record(records, raw_index);
        let mut base = 8;
        if (u16::from_le_bytes([record[0x40], record[0x41]]) as i64) < spell_value {
            base = 24;
        }
        // 0x15bbf..0x15bce multiplies the raw +0x08==0 path by the
        // double at 0x50144 (1.5), with x87 toward-zero conversion.
        if record[0x08] == 0 {
            base = base * 3 / 2;
        }
        score += base;
    }
    Ok(score)
}

/// ScoreNativeAISpellRecovery preserves the raw spell IDs 13..16 branch:
/// score 8 below max/3, score 3 below max/2, otherwise zero; raw +0x34 bit0
/// doubles the selected score.
///
/// Go：`native_ai_spell_score.go:45`，`func ScoreNativeAISpellRecovery(records []byte, candidates []byte, spellID int) (int, error)`
pub fn score_native_ai_spell_recovery(records: &[u8], candidates: &[u8], spell_id: i64) -> Result<i64> {
    if !(13..=16).contains(&spell_id) {
        return Err(battle_err!("native AI recovery score: unsupported spell id {spell_id}"));
    }
    validate_native_ai_spell_candidates(records, candidates)?;
    let mut score = 0;
    for &raw_index in candidates {
        let record = native_ai_spell_record(records, raw_index);
        let current = u16::from_le_bytes([record[0x40], record[0x41]]) as i64;
        let max = u16::from_le_bytes([record[0x42], record[0x43]]) as i64;
        let mut value = 0;
        if max / 3 > current {
            value = 8;
        } else if max / 2 > current {
            value = 3;
        }
        if record[0x34] & 1 != 0 {
            value *= 2;
        }
        score += value;
    }
    Ok(score)
}

/// ScoreNativeAISpellFlag preserves the ID20/21 branches: each nonzero raw
/// +0x25/+0x26 flag contributes six points. The byte remains unnamed and this
/// function does not apply or clear it.
///
/// Go：`native_ai_spell_score.go:74`，`func ScoreNativeAISpellFlag(records []byte, candidates []byte, spellID int) (int, error)`
pub fn score_native_ai_spell_flag(records: &[u8], candidates: &[u8], spell_id: i64) -> Result<i64> {
    let offset = match spell_id {
        20 => 0x25,
        21 => 0x26,
        _ => return Err(battle_err!("native AI flag score: unsupported spell id {spell_id}")),
    };
    validate_native_ai_spell_candidates(records, candidates)?;
    let mut score = 0;
    for &raw_index in candidates {
        if native_ai_spell_record(records, raw_index)[offset] != 0 {
            score += 6;
        }
    }
    Ok(score)
}

/// ScoreNativeAISpellZeroFlag preserves the 0x15b77 helper branches that add
/// a score when a raw byte is zero: IDs17/18/19 use +0x22/+0x23/+0x24 and add
/// three; IDs26/27 use +0x25/+0x26 and add four.
///
/// Go：`native_ai_spell_score.go:100`，`func ScoreNativeAISpellZeroFlag(records []byte, candidates []byte, spellID int) (int, error)`
pub fn score_native_ai_spell_zero_flag(records: &[u8], candidates: &[u8], spell_id: i64) -> Result<i64> {
    let (offset, increment) = match spell_id {
        17 => (0x22, 3),
        18 => (0x23, 3),
        19 => (0x24, 3),
        26 => (0x25, 4),
        27 => (0x26, 4),
        _ => return Err(battle_err!("native AI zero-flag score: unsupported spell id {spell_id}")),
    };
    validate_native_ai_spell_candidates(records, candidates)?;
    let mut score = 0;
    for &raw_index in candidates {
        if native_ai_spell_record(records, raw_index)[offset] == 0 {
            score += increment;
        }
    }
    Ok(score)
}

/// ScoreNativeAISpell22 preserves the ID22 branch at 0x15d30: a zero raw
/// +0x27 gate followed by 0x1c269(unit, nil), which scans raw +0x1a..+0x1e
/// bits. A candidate with any set bit contributes six points. The bytes are
/// intentionally left unnamed and untouched.
///
/// Go：`native_ai_spell_score.go:133`，`func ScoreNativeAISpell22(records []byte, candidates []byte) (int, error)`
pub fn score_native_ai_spell22(records: &[u8], candidates: &[u8]) -> Result<i64> {
    validate_native_ai_spell_candidates(records, candidates)?;
    let mut score = 0;
    for &raw_index in candidates {
        let record = native_ai_spell_record(records, raw_index);
        if record[0x27] != 0 {
            continue;
        }
        if record[0x1a..0x1f].iter().any(|&b| b != 0) {
            score += 6;
        }
    }
    Ok(score)
}

/// Go：`native_ai_spell_score.go:157`，`func validateNativeAISpellCandidates(records []byte, candidates []byte) error`
pub(crate) fn validate_native_ai_spell_candidates(records: &[u8], candidates: &[u8]) -> Result<()> {
    if records.len() as i64 % NATIVE_RECORD_SIZE != 0 {
        return Err(battle_err!("native AI spell score: malformed record buffer"));
    }
    for &raw_index in candidates {
        if raw_index as i64 >= records.len() as i64 / NATIVE_RECORD_SIZE {
            return Err(battle_err!("native AI spell score: candidate index {raw_index} out of range"));
        }
    }
    Ok(())
}

/// Go `records[int(rawIndex)*nativeRecordSize : (int(rawIndex)+1)*nativeRecordSize]`；呼叫端已先做
/// `validate_native_ai_spell_candidates`。Rust 新增的小工具，沒有對應的 Go 函式。
fn native_ai_spell_record(records: &[u8], raw_index: u8) -> &[u8] {
    let rs = NATIVE_RECORD_SIZE as usize;
    &records[raw_index as usize * rs..(raw_index as usize + 1) * rs]
}

#[cfg(test)]
mod native_ai_spell_score_test;
