//! Go 來源：`remake/cmd/fd2/native_turn_staging.go`（505 行）。
//!
//! 骨架：Go 檔的 13 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_turn_staging.go:46`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeTurnStagingPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_TURN_STAGING_PAN: NativeTurnStagingPhase = NativeTurnStagingPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_TURN_STAGING_DELAY: NativeTurnStagingPhase = NativeTurnStagingPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_TURN_STAGING_FLASH: NativeTurnStagingPhase = NativeTurnStagingPhase(2);

/// nativeTurnStagingJob 擁有已閉合 handler 的阻塞式 0x35822 呼叫。raw DAC 與
/// indexed frame 分開保存；第一個鏡頭 tick 前先私下建立所有 roster 快照，避免
/// 後續 call 失敗時留下半套已物化 group。
///
/// Go：`native_turn_staging.go:57`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeTurnStagingJob {
    pub(crate) event: battle::NativeTurnEvent,
    pub(crate) states: Vec<battle::State>,
    pub(crate) call: i64,
    pub(crate) phase: NativeTurnStagingPhase,
    pub(crate) ticks: i64,
    pub(crate) vga: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) drawn: bool,
    pub(crate) indexed: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// Go：`native_turn_staging.go:115`，`func cloneNativeTurnStagingState(source *battle.State) (*battle.State, error)`
pub(crate) fn clone_native_turn_staging_state(source: Option<&mut battle::State>) -> Result<Option<battle::State>> {
    todo!("native_turn_staging.go:cloneNativeTurnStagingState")
}

impl Game {
    /// startNativeRawCamp0TurnEvents is the sub_1A813(0) owner. Scenarios opt in by
    /// supplying native_turn_events; once opted in, every matching live row must
    /// have exactly one typed consumer or the enemy AI remains stopped.
    ///
    /// Go：`native_turn_staging.go:253`，`func (g *Game) startNativeRawCamp0TurnEvents() (bool, error)`
    pub(crate) fn start_native_raw_camp0_turn_events(&mut self) -> Result<bool> {
        todo!("native_turn_staging.go:startNativeRawCamp0TurnEvents")
    }

    /// READY: fd2_ch29_flow_20261017.json event79_phase_contract。
    ///
    /// Go：`native_turn_staging.go:286`，`func (g *Game) applyNativeRawCamp0PairEvent() error`
    pub(crate) fn apply_native_raw_camp0_pair_event(&mut self) -> Result<()> {
        todo!("native_turn_staging.go:applyNativeRawCamp0PairEvent")
    }

    /// Go：`native_turn_staging.go:404`，`func (g *Game) stepNativeTurnStaging()`
    pub(crate) fn step_native_turn_staging(&mut self) {
        todo!("native_turn_staging.go:stepNativeTurnStaging")
    }
}
