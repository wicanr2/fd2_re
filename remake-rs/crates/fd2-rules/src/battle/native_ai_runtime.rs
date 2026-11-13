//! Go 來源：`remake/internal/battle/native_ai_runtime.go`（291 行）。

use super::*;

impl State {
    /// nextNativeAIPhysicalPlan 是原始 0x14237 候選橋的第一個正式消費端。
    /// 範圍刻意縮窄，只接受 runtime mode 2，不猜測 caller 擁有的 0x14ef0 前置選擇。
    /// 因此，移動／物品／地形來源不完整時回傳錯誤，不退回正規化目標選擇。
    ///
    /// Go：`native_ai_runtime.go:11`，`func (s *State) nextNativeAIPhysicalPlan(u *Unit) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn next_native_ai_physical_plan(&mut self, u: Option<UnitIdx>) -> Result<(Option<AIPlan>, bool)> {
        todo!("battle/native_ai_runtime.go:State.nextNativeAIPhysicalPlan")
    }
}

/// Go：`native_ai_runtime.go:132`，`func nativeAIPhysicalTargetCode(selector int) int`
pub(crate) fn native_ai_physical_target_code(selector: i64) -> i64 {
    todo!("battle/native_ai_runtime.go:nativeAIPhysicalTargetCode")
}

/// Go：`native_ai_runtime.go:139`，`func nativeAICostRowForRecord(record []byte, rows [][]byte) ([]byte, error)`
pub(crate) fn native_ai_cost_row_for_record(record: &[u8], rows: &[Vec<u8>]) -> Result<Vec<u8>> {
    todo!("battle/native_ai_runtime.go:nativeAICostRowForRecord")
}

/// nativeAIIsTerrainSpecialRecord 重現 `0x1F183`：`+7 == 0x1C` 時**回 0**
/// （`0x1F1A7 cmp edx,0x1C` / `0x1F1AA je 0x1F1C9` → `xor eax,eax`），只有
/// `+0x20 == 0x13` 或 `+0x1F` 是 4／5 才回 1。AI 兩個呼叫端（`0x1417B`、
/// `0x14BDB`）拿它的回傳值把成本列換成 19；`+7 == 0x1C` 在那兩處是另一條規則
/// （`+8 == 0x1C` → 列 1），玩家側 `0x188F9` 則是列 16。
///
/// 早期把 `+7 == 0x1C` 也算成 true，等於讓那些單位在 AI 回合拿到「每種地形都
/// 只要 1」的列，走進自己走不出來的山地。
///
/// Go：`native_ai_runtime.go:164`，`func nativeAIIsTerrainSpecialRecord(record []byte) bool`
pub(crate) fn native_aiis_terrain_special_record(record: &[u8]) -> bool {
    todo!("battle/native_ai_runtime.go:nativeAIIsTerrainSpecialRecord")
}

/// 0x15B77的指令10～12以目標runtime索引呼叫1F183。
/// 評分器先驗證candidate bounds，再消費這份同狀態raw record快照。
///
/// Go：`native_ai_runtime.go:171`，`func nativeAI1F183TargetGate(records []byte) func(int) bool`
pub(crate) fn native_ai1_f183_target_gate(records: &[u8]) -> Box<dyn FnMut(i64) -> bool + '_> {
    todo!("battle/native_ai_runtime.go:nativeAI1F183TargetGate")
}

impl State {
    /// Go：`native_ai_runtime.go:177`，`func (s *State) nativeAIPhysicalScoreInput( raw NativeAIPhysicalAttackRawCandidate, itemRows []byte, ) (NativePhysicalAttackScoreInput, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ai_physical_score_input(
        &self,
        raw: &NativeAIPhysicalAttackRawCandidate,
        item_rows: &[u8],
    ) -> Result<NativePhysicalAttackScoreInput> {
        todo!("battle/native_ai_runtime.go:State.nativeAIPhysicalScoreInput")
    }

    /// Go：`native_ai_runtime.go:214`，`func (s *State) nativeAITerrainAdjustedWords(record []byte, destination Cell, word48, word4a int) (int, int, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ai_terrain_adjusted_words(
        &self,
        record: &[u8],
        destination: Cell,
        word48: i64,
        word4a: i64,
    ) -> Result<(i64, i64)> {
        todo!("battle/native_ai_runtime.go:State.nativeAITerrainAdjustedWords")
    }
}

/// Go：`native_ai_runtime.go:232`，`func nativeAIPhysicalHelper1DEBE(targetRecord []byte, destination Cell, itemRows []byte) (int, error)`
pub(crate) fn native_ai_physical_helper1_debe(
    target_record: &[u8],
    destination: Cell,
    item_rows: &[u8],
) -> Result<i64> {
    todo!("battle/native_ai_runtime.go:nativeAIPhysicalHelper1DEBE")
}

/// Go：`native_ai_runtime.go:253`，`func nativeAIDirectionPath(start Cell, directions []byte) ([]Cell, error)`
pub(crate) fn native_ai_direction_path(start: Cell, directions: &[u8]) -> Result<Vec<Cell>> {
    todo!("battle/native_ai_runtime.go:nativeAIDirectionPath")
}

impl State {
    /// nativeAIMode2IdleRecoveryPlan 是 0x14237 回傳零之後的收尾：0x13C06/0x13C0F 消費 0x13FD4。
    /// 接受恢復與合法的閘門拒絕都要保留，兩者皆不可掉入正規化規劃器（planner）。
    ///
    /// Go：`native_ai_runtime.go:277`，`func (s *State) nativeAIMode2IdleRecoveryPlan(u *Unit, records []byte, actor int) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_ai_mode2_idle_recovery_plan(
        &self,
        u: Option<UnitIdx>,
        records: &[u8],
        actor: i64,
    ) -> Result<(AIPlan, bool)> {
        todo!("battle/native_ai_runtime.go:State.nativeAIMode2IdleRecoveryPlan")
    }
}
