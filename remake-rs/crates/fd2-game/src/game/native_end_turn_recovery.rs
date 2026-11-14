//! Go 來源：`remake/cmd/fd2/native_end_turn_recovery.go`（179 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Evidence/spec: docs/data/ida/fd2_end_turn_recovery_20261004.json (READY).
/// 0x1A30B publishes all mode2 candidates together, then all mode0 restores.
/// The two zero-tick waits retain Draw boundaries without claiming DOS duration.
///
/// Go：`native_end_turn_recovery.go:18`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeEndTurnRecoveryJob {
    pub(crate) plan: Vec<battle::NativeEndTurnRecovery>,
    pub(crate) entries: Vec<fdicon::NativeUnitLayerEntry>,
    /// initial, mask, raw restore
    pub(crate) frames: [Vec<u8>; 3],
    pub(crate) restore_work: Vec<u8>,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) ready_at: GameInstant,
    pub(crate) before_gate: u8,
    pub(crate) cue_count: i64,
    pub(crate) committed: bool,
    pub(crate) after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl Game {
    /// Go：`native_end_turn_recovery.go:32`，`func (g *Game) beginNativeEndTurnRecovery(after func()) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_end_turn_recovery(
        &mut self,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        todo!("native_end_turn_recovery.go:beginNativeEndTurnRecovery")
    }

    /// Go：`native_end_turn_recovery.go:113`，`func (g *Game) stepNativeEndTurnRecovery(now time.Time)`
    pub(crate) fn step_native_end_turn_recovery(&mut self, now: GameInstant) {
        todo!("native_end_turn_recovery.go:stepNativeEndTurnRecovery")
    }

    /// Go：`native_end_turn_recovery.go:162`，`func (g *Game) drawNativeEndTurnRecovery(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_end_turn_recovery(&mut self, screen: &mut Picture) -> bool {
        todo!("native_end_turn_recovery.go:drawNativeEndTurnRecovery")
    }
}
