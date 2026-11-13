//! Go 來源：`remake/internal/battle/native_ai_scored_group_score.go`（67 行）。

use super::*;

/// NativeAIScoredCommandGroupScore is the score returned by 0x15b77 for one
/// already-resolved 0x1598a destination/target group.
///
/// Go：`native_ai_scored_group_score.go:7`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAIScoredCommandGroupScore {
    pub destination: Cell,
    pub target_indices: Vec<u8>,
    pub score: i64,
}

/// ScoreNativeAIScoredCommandGroups dispatches the complete command-index
/// families visible in 0x15b77. IDs 23..25 and 28..35 reach the native zero
/// return path. The caller still owns comparison against [0x53c23] and the
/// command-word tie state; this function deliberately does not select a winner.
///
/// Go：`native_ai_scored_group_score.go:17`，`func ScoreNativeAIScoredCommandGroups( records []byte, commandID, commandValue int, groups []NativeAIScoredCommandCandidateGroup, skip func(int) bool, ) ([]NativeAIScoredCommandGroupScore, error)`
pub fn score_native_ai_scored_command_groups(
    records: &[u8],
    command_id: i64,
    command_value: i64,
    groups: &[NativeAIScoredCommandCandidateGroup],
    skip: Option<&mut dyn FnMut(i64) -> bool>,
) -> Result<Vec<NativeAIScoredCommandGroupScore>> {
    let mut skip = skip;
    if !(0..NATIVE_COMMAND_RECORD_COUNT).contains(&command_id) || command_value < 0 {
        return Err(battle_err!("native AI group score: invalid command id/value"));
    }
    let mut result = Vec::with_capacity(groups.len());
    for group in groups {
        let score = score_native_ai_scored_command_targets(
            records,
            &group.target_indices,
            command_id,
            command_value,
            native_ai_reborrow_skip(&mut skip),
        )?;
        result.push(NativeAIScoredCommandGroupScore {
            destination: group.destination,
            target_indices: group.target_indices.clone(),
            score,
        });
    }
    Ok(result)
}

/// Go：`native_ai_scored_group_score.go:43`，`func scoreNativeAIScoredCommandTargets( records []byte, targets []byte, commandID, commandValue int, skip func(int) bool, ) (int, error)`
pub(crate) fn score_native_ai_scored_command_targets(
    records: &[u8],
    targets: &[u8],
    command_id: i64,
    command_value: i64,
    skip: Option<&mut dyn FnMut(i64) -> bool>,
) -> Result<i64> {
    match command_id {
        ..=12 => score_native_ai_spell_attack(records, targets, command_id, command_value, skip),
        13..=16 => score_native_ai_spell_recovery(records, targets, command_id),
        17..=19 => score_native_ai_spell_zero_flag(records, targets, command_id),
        20..=21 => score_native_ai_spell_flag(records, targets, command_id),
        22 => score_native_ai_spell22(records, targets),
        23..=25 => Ok(0),
        26..=27 => score_native_ai_spell_zero_flag(records, targets, command_id),
        _ => Ok(0),
    }
}

/// 在迴圈裡重複借出呼叫端的 `skip` 閘門（Go 的 `func(int) bool` 可以直接重複傳遞）。
/// Rust 新增的小工具，沒有對應的 Go 函式。
pub(crate) fn native_ai_reborrow_skip<'a>(
    skip: &'a mut Option<&mut dyn FnMut(i64) -> bool>,
) -> Option<&'a mut dyn FnMut(i64) -> bool> {
    match skip {
        Some(f) => Some(&mut **f),
        None => None,
    }
}

#[cfg(test)]
mod native_ai_scored_group_score_test;
