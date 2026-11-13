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
    todo!("battle/native_ai_physical_score.go:ScoreNativePhysicalAttackCandidate")
}

/// SelectNativePhysicalAttackCandidate preserves 0x144C5..0x144F4: priority is
/// compared before score, and an exact tie keeps the earlier enumeration.
///
/// Go：`native_ai_physical_score.go:78`，`func SelectNativePhysicalAttackCandidate(candidates []NativePhysicalAttackCandidate) (NativePhysicalAttackSelection, bool, error)`
pub fn select_native_physical_attack_candidate(
    candidates: &[NativePhysicalAttackCandidate],
) -> Result<(NativePhysicalAttackSelection, bool)> {
    todo!("battle/native_ai_physical_score.go:SelectNativePhysicalAttackCandidate")
}

/// Go：`native_ai_physical_score.go:109`，`func nativeMulThreeHalvesTowardZero(value int) int`
pub(crate) fn native_mul_three_halves_toward_zero(value: i64) -> i64 {
    todo!("battle/native_ai_physical_score.go:nativeMulThreeHalvesTowardZero")
}
