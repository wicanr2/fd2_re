//! Go 來源：`remake/internal/battle/native_ai_physical_candidates.go`（149 行）。

use super::*;

/// NativeAIPhysicalAttackGeometry 保存 0x14237 呼叫 0x14818 時由 caller
/// 提供的原始幾何參數。欄位只描述 call-site 角色，不把 mode、inner marker
/// 或 target code 命名成武器射程、範圍或陣營。
///
/// Go：`native_ai_physical_candidates.go:8`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAIPhysicalAttackGeometry {
    pub target_mode: i64,
    pub target_inner_mark: i64,
    pub target_code: i64,
}

/// NativeAIPhysicalAttackRawCandidate 是一個落點／目標索引及其 0x50-byte
/// 原始記錄快照。resolver 只能從這些快照與 caller-owned 的位址資料產生
/// NativePhysicalAttackScoreInput；不可以用 Unit 的 Camp、X/Y 或 normalized
/// 戰鬥數值補齊缺少的 raw provenance。
///
/// Go：`native_ai_physical_candidates.go:18`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAIPhysicalAttackRawCandidate {
    pub destination: Cell,
    pub target_index: i64,
    pub actor_record: Vec<u8>,
    pub target_record: Vec<u8>,
}

/// NativeAIPhysicalAttackScoreResolver 將已閉合的 raw 候選轉成評分器的
/// word/byte 輸入。地形修正、0x1DEBE 結果及 target +8 等 caller-owned
/// provenance 必須由 resolver 明示提供；缺資料時應回傳錯誤。
///
/// Go：`native_ai_physical_candidates.go:28`
pub type NativeAIPhysicalAttackScoreResolver<'a> =
    dyn FnMut(NativeAIPhysicalAttackRawCandidate) -> Result<NativePhysicalAttackScoreInput> + 'a;

/// BuildNativeAIPhysicalAttackCandidates 串起 0x14237 已證實的無副作用前綴：
///
///     0x145cd→0x4e040→0x146d1→0x14b16
///     →0x14818 target geometry→raw score-input resolver
///
/// movementFlags 與 targetFlags 都是 caller-owned 的 live composition snapshot，
/// 不會從 immutable FDFIELD 或 normalized State 推導另一份旗標。落點維持
/// NativeAIPhysicalDestinations 的 Y-first/X-second 順序；每個落點的目標維持
/// runtime record 順序。這個函式只產生候選，不選勝者、不移動、不攻擊，也不
/// 接入 NextAIPlan；任何缺少 raw record、幾何或 score provenance 都失敗即關閉。
///
/// Go：`native_ai_physical_candidates.go:42`，`func BuildNativeAIPhysicalAttackCandidates( w, h int, records []byte, count, actor, selector, initialBudget int, geometry NativeAIPhysicalAttackGeometry, movementFlags, targetFlags, terrainMoveCodes, costRow []byte, resolve NativeAIPhysicalAttackScoreResolver, ) ([]NativePhysicalAttackCandidate, error)`
pub fn build_native_ai_physical_attack_candidates(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    selector: i64,
    initial_budget: i64,
    geometry: NativeAIPhysicalAttackGeometry,
    movement_flags: &mut [u8],
    target_flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
    resolve: Option<&mut NativeAIPhysicalAttackScoreResolver<'_>>,
) -> Result<Vec<NativePhysicalAttackCandidate>> {
    todo!("battle/native_ai_physical_candidates.go:BuildNativeAIPhysicalAttackCandidates")
}

/// nativeAIPhysicalTargetIndices 保存 0x14818 的 cell geometry 與 raw +5/+6
/// roster 過濾。它刻意不讀 Unit 投影；targetFlags 必須是該 caller 在這個
/// candidate lifetime 中提供的 composition snapshot。
///
/// Go：`native_ai_physical_candidates.go:111`，`func nativeAIPhysicalTargetIndices( w, h int, records []byte, count int, origin Cell, mode, innerMark, targetCode int, targetFlags []byte, ) ([]byte, error)`
pub(crate) fn native_ai_physical_target_indices(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    origin: Cell,
    mode: i64,
    inner_mark: i64,
    target_code: i64,
    target_flags: &[u8],
) -> Result<Vec<u8>> {
    todo!("battle/native_ai_physical_candidates.go:nativeAIPhysicalTargetIndices")
}
