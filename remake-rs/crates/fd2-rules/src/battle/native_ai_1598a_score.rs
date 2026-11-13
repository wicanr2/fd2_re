//! Go 來源：`remake/internal/battle/native_ai_1598a_score.go`（100 行）。

use super::*;

/// NativeAI1598AScoreResult preserves the numeric maximum written to
/// [0x53c23]. PositiveWinner is present only when a candidate scored above
/// zero. The all-zero command-word local remains unproven and is not invented.
///
/// Go：`native_ai_1598a_score.go:11`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAI1598AScoreResult {
    pub max_score: i64,
    pub positive_winner: NativeAISpellCandidate,
    pub has_positive_winner: bool,
}

/// nativeAICommandCostRow 是 0x4E555(0) 的成本列：0x55446 的 20 個 1
/// （remake/assets/data/native_movement_cost_rows.json selector 0）。
///
/// Go：`native_ai_1598a_score.go:19`
pub(crate) static NATIVE_AI_COMMAND_COST_ROW: [u8; 20] = [1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1];

/// ScoreNativeAI1598A connects the proven 0x1598a slices without executing a
/// command: availability, destination/target groups, 0x15b77 family scoring,
/// and the strict positive-score winner comparison.
///
/// Go：`native_ai_1598a_score.go:26`，`func ScoreNativeAI1598A( w, h int, records []byte, count, actor, selector int, unit *Unit, book []NativeCommandRecord, baseFlags, terrainMoveCodes, costRow []byte, skip func(int) bool, ) (NativeAI1598AScoreResult, error)`
pub fn score_native_ai1598_a(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    selector: i64,
    unit: Option<&Unit>,
    book: &[NativeCommandRecord],
    base_flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
    skip: Option<&mut dyn FnMut(i64) -> bool>,
) -> Result<NativeAI1598AScoreResult> {
    todo!("battle/native_ai_1598a_score.go:ScoreNativeAI1598A")
}
