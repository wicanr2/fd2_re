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
    todo!("battle/native_ai_scored_group_score.go:ScoreNativeAIScoredCommandGroups")
}

/// Go：`native_ai_scored_group_score.go:43`，`func scoreNativeAIScoredCommandTargets( records []byte, targets []byte, commandID, commandValue int, skip func(int) bool, ) (int, error)`
pub(crate) fn score_native_ai_scored_command_targets(
    records: &[u8],
    targets: &[u8],
    command_id: i64,
    command_value: i64,
    skip: Option<&mut dyn FnMut(i64) -> bool>,
) -> Result<i64> {
    todo!("battle/native_ai_scored_group_score.go:scoreNativeAIScoredCommandTargets")
}
