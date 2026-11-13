//! Go 來源：`remake/internal/battle/native_ai_scored_candidates.go`（160 行）。

use super::*;

/// NativeAIScoredCommandCandidateGroup is one 0x1598a destination followed by
/// the record indices emitted by its 0x14818 call. Groups retain 0x14b16's
/// row-major destination order and the runtime roster's target order.
///
/// Go：`native_ai_scored_candidates.go:8`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAIScoredCommandCandidateGroup {
    pub destination: Cell,
    pub target_indices: Vec<u8>,
}

/// NativeAIScoredCommandCandidateGroups preserves 0x1598a's geometry pipeline:
/// command record +3 -> 0x4e040 -> 0x14b16, then command record +4 -> 0x14818.
/// Unlike 0x14237's physical path, this caller does not invoke 0x145cd/0x146d1
/// to derive occupancy flags from the roster; baseFlags must therefore be the
/// exact caller-owned grid snapshot.
///
/// Go：`native_ai_scored_candidates.go:18`，`func NativeAIScoredCommandCandidateGroups( w, h int, records []byte, count, actor, selector int, command NativeCommandRecord, baseFlags, terrainMoveCodes, costRow []byte, ) ([]NativeAIScoredCommandCandidateGroup, error)`
pub fn native_ai_scored_command_candidate_groups(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    selector: i64,
    command: NativeCommandRecord,
    base_flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<Vec<NativeAIScoredCommandCandidateGroup>> {
    todo!("battle/native_ai_scored_candidates.go:NativeAIScoredCommandCandidateGroups")
}

/// Go：`native_ai_scored_candidates.go:55`，`func nativeAIScoredCommandDestinations( w, h int, records []byte, count, actor, initialBudget int, baseFlags, terrainMoveCodes, costRow []byte, ) ([]Cell, error)`
pub(crate) fn native_ai_scored_command_destinations(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    initial_budget: i64,
    base_flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<Vec<Cell>> {
    todo!("battle/native_ai_scored_candidates.go:nativeAIScoredCommandDestinations")
}

/// Go：`native_ai_scored_candidates.go:104`，`func nativeAIScoredCommandTargetCode(commandTargetCode, selector int) (int, error)`
pub(crate) fn native_ai_scored_command_target_code(command_target_code: i64, selector: i64) -> Result<i64> {
    todo!("battle/native_ai_scored_candidates.go:nativeAIScoredCommandTargetCode")
}

/// Go：`native_ai_scored_candidates.go:118`，`func nativeAIScoredCommandTargetIndices( w, h int, records []byte, count int, origin Cell, effectMode, targetCode int, baseFlags []byte, ) ([]byte, error)`
pub(crate) fn native_ai_scored_command_target_indices(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    origin: Cell,
    effect_mode: i64,
    target_code: i64,
    base_flags: &[u8],
) -> Result<Vec<u8>> {
    todo!("battle/native_ai_scored_candidates.go:nativeAIScoredCommandTargetIndices")
}

/// Go：`native_ai_scored_candidates.go:147`，`func nativeAIScoredRawTargetMatches(targetCode int, recordByte6 byte) bool`
pub(crate) fn native_ai_scored_raw_target_matches(target_code: i64, record_byte6: u8) -> bool {
    todo!("battle/native_ai_scored_candidates.go:nativeAIScoredRawTargetMatches")
}
