//! Go 來源：`remake/internal/battle/native_ai_physical_score.go`（114 行）。

use super::*;

/// NativePhysicalAttackScoreInput preserves the word/byte inputs consumed by
/// FD2.EXE 0x14237 after caller-owned terrain adjustment. RawTargetByte8 and
/// RawHelper1DEBEResult deliberately retain address-level names until their
/// writer/consumer contracts are independently closed.
///
/// Go：`native_ai_physical_score.go:9`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhysicalAttackScoreInput {
    pub actor_word48: i64,
    pub actor_word4_a: i64,
    pub target_word48: i64,
    pub target_word4_a: i64,
    pub target_word40: i64,
    pub raw_target_byte8: u8,
    pub raw_helper1_debe_result: i64,
}

/// NativePhysicalAttackScore is the priority/score pair compared by 0x14237.
///
/// Go：`native_ai_physical_score.go:20`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhysicalAttackScore {
    pub priority: i64,
    pub score: i64,
}

/// NativePhysicalAttackCandidate is one destination/target pair in 0x14237's
/// nested enumeration order.
///
/// Go：`native_ai_physical_score.go:27`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhysicalAttackCandidate {
    pub destination_x: i64,
    pub destination_y: i64,
    pub target_index: i64,
    pub inputs: NativePhysicalAttackScoreInput,
}

/// NativePhysicalAttackSelection retains the winning raw pair and its ranking.
///
/// Go：`native_ai_physical_score.go:35`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhysicalAttackSelection {
    pub candidate: NativePhysicalAttackCandidate,
    pub ranking: NativePhysicalAttackScore,
}

/// ScoreNativePhysicalAttackCandidate preserves 0x14586..0x144D5:
/// candidates with actor word48-target word4A <= 2 start at priority zero; a strict
/// score>target word40 raises priority from 8 to 18 and doubles the score;
/// helper result 1 adds actor word4A-target word48; raw target byte8 zero
/// applies the native signed truncation-toward-zero 3/2 multiplier.
///
/// Go：`native_ai_physical_score.go:45`，`func ScoreNativePhysicalAttackCandidate(in NativePhysicalAttackScoreInput) (NativePhysicalAttackScore, bool, error)`
pub fn score_native_physical_attack_candidate(
    r#in: NativePhysicalAttackScoreInput,
) -> Result<(NativePhysicalAttackScore, bool)> {
    // Go 迭代 map，多個欄位同時越界時報哪一個隨 map 順序而定；Rust 依鍵名升冪（battle.rs 模組文件）。
    let fields = BTreeMap::from([
        ("actor_word48", r#in.actor_word48),
        ("actor_word4a", r#in.actor_word4_a),
        ("target_word48", r#in.target_word48),
        ("target_word4a", r#in.target_word4_a),
        ("target_word40", r#in.target_word40),
    ]);
    for (name, value) in fields {
        if !(0..=0xffff).contains(&value) {
            return Err(battle_err!("{name} outside native u16 range: {value}"));
        }
    }

    let mut score = r#in.actor_word48 - r#in.target_word4_a;
    let mut priority = 0;
    if score > 2 {
        priority = 8;
    }
    if score > r#in.target_word40 {
        score *= 2;
        priority = 18;
    }
    if r#in.raw_helper1_debe_result == 1 {
        score += r#in.actor_word4_a - r#in.target_word48;
    }
    if r#in.raw_target_byte8 == 0 {
        score = native_mul_three_halves_toward_zero(score);
    }
    Ok((NativePhysicalAttackScore { priority, score }, true))
}

/// SelectNativePhysicalAttackCandidate preserves 0x144C5..0x144F4: priority is
/// compared before score, and an exact tie keeps the earlier enumeration.
///
/// Go：`native_ai_physical_score.go:78`，`func SelectNativePhysicalAttackCandidate(candidates []NativePhysicalAttackCandidate) (NativePhysicalAttackSelection, bool, error)`
pub fn select_native_physical_attack_candidate(
    candidates: &[NativePhysicalAttackCandidate],
) -> Result<(NativePhysicalAttackSelection, bool)> {
    let mut best = NativePhysicalAttackSelection::default();
    let mut found = false;
    for (i, candidate) in candidates.iter().enumerate() {
        // 同上：多欄同時越界時 Go 報哪一個隨 map 順序而定，Rust 依鍵名升冪。
        let fields = BTreeMap::from([
            ("destination_x", candidate.destination_x),
            ("destination_y", candidate.destination_y),
            ("target_index", candidate.target_index),
        ]);
        for (name, value) in fields {
            if !(0..=0xff).contains(&value) {
                return Err(battle_err!("candidate {i} {name} outside native byte range: {value}"));
            }
        }
        let (ranking, accepted) = score_native_physical_attack_candidate(candidate.inputs)
            .map_err(|err| err.wrap(format!("candidate {i}: ")))?;
        if !accepted {
            continue;
        }
        // 0x14248／0x1427E 的初值為 score/priority=0/0；priority0
        // 的非正分不能靠「尚未找到候選」繞過原版比較。
        if ranking.priority > best.ranking.priority
            || (ranking.priority == best.ranking.priority && ranking.score > best.ranking.score)
        {
            best = NativePhysicalAttackSelection { candidate: *candidate, ranking };
            found = true;
        }
    }
    Ok((best, found))
}

/// Go：`native_ai_physical_score.go:109`，`func nativeMulThreeHalvesTowardZero(value int) int`
pub(crate) fn native_mul_three_halves_toward_zero(value: i64) -> i64 {
    if value < 0 {
        return (value.wrapping_neg().wrapping_mul(3) / 2).wrapping_neg();
    }
    value.wrapping_mul(3) / 2
}

#[cfg(test)]
mod native_ai_physical_score_test;
