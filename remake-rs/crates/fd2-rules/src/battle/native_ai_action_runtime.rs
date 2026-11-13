//! Go 來源：`remake/internal/battle/native_ai_action_runtime.go`（809 行）。

use super::*;

/// DebugAI 是對拍診斷用的可選記錄器；nil 時不輸出。由測試端設定。
///
/// Go：`native_ai_action_runtime.go:9`
///
/// Rust 骨架：Go 簽名是 `func(format string, args ...any)`。Rust 改收已格式化的
/// `fmt::Arguments`，呼叫端以 `format_args!` 取代 Go 的 format 字串與參數；`None` 對應 Go 的 nil。
pub static DEBUG_AI: RwLock<Option<Callback<dyn Fn(fmt::Arguments<'_>) + Send + Sync>>> = RwLock::new(None);

impl State {
    /// nextNativeAI14EF0Plan is the first runtime consumer of the three raw
    /// producers called by 0x14ef0.  It is intentionally limited to records whose
    /// mode reaches that helper and to a complete command/item/movement export.
    /// The selected route remains an address-level label; no command or item name
    /// is inferred here.
    ///
    /// Go：`native_ai_action_runtime.go:16`，`func (s *State) nextNativeAI14EF0Plan(u *Unit) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn next_native_ai14_ef0_plan(&mut self, u: Option<UnitIdx>) -> Result<(Option<AIPlan>, bool)> {
        todo!("battle/native_ai_action_runtime.go:State.nextNativeAI14EF0Plan")
    }
}

/// Go：`native_ai_action_runtime.go:231`，`func nativeAIModeUses14EF0(mode int) bool`
pub(crate) fn native_ai_mode_uses14_ef0(mode: i64) -> bool {
    todo!("battle/native_ai_action_runtime.go:nativeAIModeUses14EF0")
}

impl State {
    /// Go：`native_ai_action_runtime.go:240`，`func (s *State) nativeAI14EF0PhysicalSelection( actor, selector int, records, actorRecord, baseFlags, costRow []byte, ) (NativePhysicalAttackSelection, bool, error)`
    pub(crate) fn native_ai14_ef0_physical_selection(
        &self,
        actor: i64,
        selector: i64,
        records: &[u8],
        actor_record: &[u8],
        base_flags: &[u8],
        cost_row: &[u8],
    ) -> Result<(NativePhysicalAttackSelection, bool)> {
        todo!("battle/native_ai_action_runtime.go:State.nativeAI14EF0PhysicalSelection")
    }

    /// nativeAIPlanForStationarySelection owns 0x15311's selected command result.
    /// Unlike 0x1548E, 0x15311 never calls 0x14B78: its winner coordinates feed
    /// target/effect presentation while the actor remains at the current cell.
    ///
    /// Go：`native_ai_action_runtime.go:287`，`func (s *State) nativeAIPlanForStationarySelection( u *Unit, actorRecord []byte, destination Cell, targetIndex int, ) (*AIPlan, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ai_plan_for_stationary_selection(
        &self,
        u: Option<UnitIdx>,
        actor_record: &[u8],
        destination: Cell,
        target_index: i64,
    ) -> Result<AIPlan> {
        todo!("battle/native_ai_action_runtime.go:State.nativeAIPlanForStationarySelection")
    }

    /// nativeAIPlanForDestination replays 0x14b78's movement transaction for a
    /// selected raw destination.  The caller still owns the effect/target stage;
    /// this helper only returns the detached path and verified target pointer.
    ///
    /// Go：`native_ai_action_runtime.go:309`，`func (s *State) nativeAIPlanForDestination( u *Unit, actorRecord []byte, selector int, destination Cell, targetIndex int, costRow []byte, ) (*AIPlan, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_ai_plan_for_destination(
        &mut self,
        u: Option<UnitIdx>,
        actor_record: &[u8],
        selector: i64,
        destination: Cell,
        target_index: i64,
        cost_row: &mut [u8],
    ) -> Result<AIPlan> {
        todo!("battle/native_ai_action_runtime.go:State.nativeAIPlanForDestination")
    }

    /// nextNativeAIModeFallbackPlan consumes only the mode branches whose inputs
    /// are explicit in the 0x13a9f disassembly.  Mode 5 now owns the raw
    /// 0x15df3/event-row state tail; its raw 0x25b45 AIL sample cue remains a
    /// separate gate. Mode 11's two-action transaction and the 0x13fd4 raw HP
    /// write still have no complete runtime owner here.
    ///
    /// Go：`native_ai_action_runtime.go:350`，`func (s *State) nextNativeAIModeFallbackPlan(u *Unit) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn next_native_ai_mode_fallback_plan(&mut self, u: Option<UnitIdx>) -> Result<(Option<AIPlan>, bool)> {
        todo!("battle/native_ai_action_runtime.go:State.nextNativeAIModeFallbackPlan")
    }

    /// nativeAIIdleRecoveryOnlyPlan 是移動後備完全沒有落點時的 0x13C06→0x13FD4：沒有聚焦、
    /// 沒有走行，只有回復（或閘門拒絕）與共用收尾，和 mode 2 無物理候選同一種計畫形狀。
    ///
    /// Go：`native_ai_action_runtime.go:571`，`func (s *State) nativeAIIdleRecoveryOnlyPlan(u *Unit, records []byte, actor, mode int) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_ai_idle_recovery_only_plan(
        &self,
        u: Option<UnitIdx>,
        records: &[u8],
        actor: i64,
        mode: i64,
    ) -> Result<(AIPlan, bool)> {
        todo!("battle/native_ai_action_runtime.go:State.nativeAIIdleRecoveryOnlyPlan")
    }

    /// Go：`native_ai_action_runtime.go:587`，`func (s *State) attachNativeFallbackIdleRecovery(plan *AIPlan, records []byte, actor int) error`
    pub(crate) fn attach_native_fallback_idle_recovery(
        &self,
        plan: &mut AIPlan,
        records: &[u8],
        actor: i64,
    ) -> Result<()> {
        todo!("battle/native_ai_action_runtime.go:State.attachNativeFallbackIdleRecovery")
    }

    /// nativeAIMovementApproachPoint 是 0x14B78 的第 1～3 段（0x14C16..0x14D43）：
    /// intended 直接走得到就原樣回傳；否則沿不看單位的長路徑（預算 0x1C、mode 1）
    /// 找 MV 預算場內走得到的最後一格。長路徑也找不到時同樣原樣回傳。
    ///
    /// Go：`native_ai_action_runtime.go:602`，`func (s *State) nativeAIMovementApproachPoint( origin, intended Cell, mv, actor, selector int, records, baseFlags, runtimeFlags, costRow []byte, ) (Cell, error)`
    pub(crate) fn native_ai_movement_approach_point(
        &mut self,
        origin: Cell,
        intended: Cell,
        mv: i64,
        actor: i64,
        selector: i64,
        records: &[u8],
        base_flags: &mut [u8],
        runtime_flags: &mut [u8],
        cost_row: &mut [u8],
    ) -> Result<Cell> {
        todo!("battle/native_ai_action_runtime.go:State.nativeAIMovementApproachPoint")
    }
}

/// nativeAIRawRecord8Index mirrors 0x12c60's first-match scan.  The caller
/// supplies the complete scoring-record export, so a missing +0x08 provenance
/// is rejected by NativeAIScoringRecords before this helper is reached.
///
/// Go：`native_ai_action_runtime.go:657`，`func nativeAIRawRecord8Index(records []byte, count, rawKey int) (int, bool)`
pub(crate) fn native_ai_raw_record8_index(records: &[u8], count: i64, raw_key: i64) -> (i64, bool) {
    todo!("battle/native_ai_action_runtime.go:nativeAIRawRecord8Index")
}

impl State {
    /// Go：`native_ai_action_runtime.go:671`，`func (s *State) nativeAIModeRuntimeContext(u *Unit) (int, int, []byte, []byte, []byte, []byte, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ai_mode_runtime_context(
        &self,
        u: Option<UnitIdx>,
    ) -> Result<(i64, i64, Vec<u8>, Vec<u8>, Vec<u8>, Vec<u8>)> {
        todo!("battle/native_ai_action_runtime.go:State.nativeAIModeRuntimeContext")
    }
}

/// Go：`native_ai_action_runtime.go:709`，`func nativeAIModeBlockedSearchFlags(w, h int, records []byte, count, selector int, baseFlags []byte) ([]byte, error)`
pub(crate) fn native_ai_mode_blocked_search_flags(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    selector: i64,
    base_flags: &mut [u8],
) -> Result<Vec<u8>> {
    todo!("battle/native_ai_action_runtime.go:nativeAIModeBlockedSearchFlags")
}

/// Go：`native_ai_action_runtime.go:734`，`func nativeAIActiveUnitAtRawCell(records []byte, count, selector int, cell Cell) (int, bool)`
pub(crate) fn native_ai_active_unit_at_raw_cell(records: &[u8], count: i64, selector: i64, cell: Cell) -> (i64, bool) {
    todo!("battle/native_ai_action_runtime.go:nativeAIActiveUnitAtRawCell")
}

impl State {
    /// nativeAIPlanTowardRawDestination 重現 0x14B78 的四段：
    ///
    ///  1. 0x145CD 標記後以 MV 預算、mode 0 直接找到 intended 的路徑；找到就跳到第 4 段。
    ///  2. 找不到就 0x4DBFC 清掉單位標記（只剩 FDFIELD 基底旗標），以預算 0x1C、mode 1
    ///     找一條穿過單位的長路徑；找不到也跳到第 4 段。
    ///  3. 重新標記、0x4E040 建 MV 預算場（不做 0x146D1 同組清除），沿長路徑逐格走，
    ///     取「預算場走得到的最後一格」當新的 intended。
    ///  4. 0x14B16 候選（含 0x146D1）依 距離→XY 軸差→列優先 選實際落點，再以 MV、mode 0
    ///     取最終路徑。
    ///
    /// 第四章原版 eip-trace（work/parity-slot-ch04/ai-trace-r1）證實每個敵人都是
    /// 0x4E1A6(MV,mode0)→0x4E1A6(0x1C,mode1)→0x4E040×2→0x14B16→0x4E1A6(MV,mode0) 這個序列。
    ///
    /// Go：`native_ai_action_runtime.go:759`，`func (s *State) nativeAIPlanTowardRawDestination( u *Unit, actor, selector int, records, actorRecord, baseFlags, costRow []byte, intended Cell, ) (*AIPlan, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ai_plan_toward_raw_destination(
        &mut self,
        u: Option<UnitIdx>,
        actor: i64,
        selector: i64,
        records: &[u8],
        actor_record: &[u8],
        base_flags: &mut [u8],
        cost_row: &mut [u8],
        intended: Cell,
    ) -> Result<AIPlan> {
        todo!("battle/native_ai_action_runtime.go:State.nativeAIPlanTowardRawDestination")
    }
}
