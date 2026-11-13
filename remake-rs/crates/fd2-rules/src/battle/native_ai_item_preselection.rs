//! Go 來源：`remake/internal/battle/native_ai_item_preselection.go`（206 行）。

use super::*;

/// NativeAI1567EScoreResult preserves the positive best tuple written by
/// 0x1567e to [0x53c33/37/3b/3f]. InventorySlot is the raw slot index saved in
/// [0x53c3f], not the item command byte.
///
/// Go：`native_ai_item_preselection.go:8`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAI1567EScoreResult {
    pub max_score: i64,
    pub x: i64,
    pub y: i64,
    pub inventory_slot: i64,
    pub item_id: i64,
    /// TargetIndices is the detached raw target list for the winning
    /// destination.  It is retained so the execution owner can re-run the
    /// second target/effect stage instead of inventing a target from normalized
    /// party data.
    pub target_indices: Vec<u8>,
    pub has_positive_winner: bool,
}

/// ScoreNativeAI1567E connects the proven item-command preselection slices:
/// 0x1b8a6 count-sized slot scan, caller-specific 0x14818/0x149f8 geometry,
/// 0x15880 scoring, and strict greater-than winner replacement. It does not
/// execute the selected item or infer an effect name.
///
/// Go：`native_ai_item_preselection.go:26`，`func ScoreNativeAI1567E( w, h int, records []byte, count, actor, selector int, itemRows []byte, book []NativeCommandRecord, baseFlags []byte, ) (NativeAI1567EScoreResult, error)`
pub fn score_native_ai1567_e(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    selector: i64,
    item_rows: &[u8],
    book: &[NativeCommandRecord],
    base_flags: &[u8],
) -> Result<NativeAI1567EScoreResult> {
    todo!("battle/native_ai_item_preselection.go:ScoreNativeAI1567E")
}

/// Go：`native_ai_item_preselection.go:129`，`func nativeAI1567EAreaTargets( w, h int, records []byte, count int, origin Cell, mode, targetCode int, baseFlags []byte, ) ([]byte, error)`
pub(crate) fn native_ai1567_e_area_targets(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    origin: Cell,
    mode: i64,
    target_code: i64,
    base_flags: &[u8],
) -> Result<Vec<u8>> {
    todo!("battle/native_ai_item_preselection.go:nativeAI1567EAreaTargets")
}

/// Go：`native_ai_item_preselection.go:161`，`func nativeAI1567ECardinalTargets( w, h int, records []byte, count int, start, destination Cell, steps int, ) ([]byte, error)`
pub(crate) fn native_ai1567_e_cardinal_targets(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    start: Cell,
    destination: Cell,
    steps: i64,
) -> Result<Vec<u8>> {
    todo!("battle/native_ai_item_preselection.go:nativeAI1567ECardinalTargets")
}
