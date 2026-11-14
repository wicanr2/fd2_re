//! Go 來源：`remake/cmd/fd2/native_ai_idle_recovery.go`（350 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 6 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_WORK_BASE: i64 = 0x8088;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_VGA_OFFSET: i64 = 0x504;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_WIDTH: i64 = 312;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_HEIGHT: i64 = 192;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_STRIDE: i64 = 0x1c8;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_VGA_STRIDE: i64 = 320;

/// nativeAIIdleRecoveryJob owns the asynchronous presentation portion of
/// 0x13FD4. The job presents initial, mode2 mask and mode0 raw-restore frames,
/// as specified by docs/data/ida/fd2_ai_idle_recovery_20261004.json, one per
/// 0x17AA9(1) wait.  HP is committed only after the third present acknowledgement
/// and a fresh raw-record preflight.
///
/// Go：`native_ai_idle_recovery.go:29`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeAIIdleRecoveryJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: battle::NativeAIIdleRecoveryPresentation,
    pub(crate) decision: battle::NativeAIIdleRecoveryDecision,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) before_range: i64,
    pub(crate) before_has: bool,
    pub(crate) after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl Game {
    /// beginNativeAIIdleRecovery is the concrete indexed/audio owner for the
    /// 0x14121→0x13FD4 edge.  It deliberately uses the already verified native map
    /// compositor, FDICON 24×24 RLE path and FDOTHER#31 index-4 sample.  It does not
    /// assign a high-level name to either 0x1DA16 decode mode.
    ///
    /// Go：`native_ai_idle_recovery.go:45`，`func (g *Game) beginNativeAIIdleRecovery( actor *battle.Unit, decision battle.NativeAIIdleRecoveryDecision, after func(), ) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_ai_idle_recovery(
        &mut self,
        actor: Option<UnitIdx>,
        decision: battle::NativeAIIdleRecoveryDecision,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        todo!("native_ai_idle_recovery.go:beginNativeAIIdleRecovery")
    }

    /// buildNativeAIIdleRecoveryFrames consumes the proven 24×24 FDICON raw
    /// decoder and the corrected 312×192 viewport-copy ABI. The 0x1402B wait
    /// presents the initial source; 0x14094 presents the C8 write mask, and
    /// 0x140EA presents the mode0 raw restore before HP is committed.
    ///
    /// Go：`native_ai_idle_recovery.go:167`，`func (g *Game) buildNativeAIIdleRecoveryFrames( actor *battle.Unit, presentation battle.NativeAIIdleRecoveryPresentation, ) ([][]byte, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn build_native_ai_idle_recovery_frames(
        &mut self,
        actor: Option<UnitIdx>,
        presentation: &battle::NativeAIIdleRecoveryPresentation,
    ) -> Result<Vec<Vec<u8>>> {
        todo!("native_ai_idle_recovery.go:buildNativeAIIdleRecoveryFrames")
    }
}

/// Go：`native_ai_idle_recovery.go:240`，`func nativeAIIdleRecoveryViewport(work, baseVGA []byte) ([]byte, error)`
pub(crate) fn native_ai_idle_recovery_viewport(work: &[u8], base_vga: &[u8]) -> Result<Vec<u8>> {
    todo!("native_ai_idle_recovery.go:nativeAIIdleRecoveryViewport")
}

/// Go：`native_ai_idle_recovery.go:260`，`func osMuteOrShot(g *Game) bool`
pub(crate) fn os_mute_or_shot(g: Option<&Game>) -> bool {
    todo!("native_ai_idle_recovery.go:osMuteOrShot")
}

/// Go：`native_ai_idle_recovery.go:264`，`func unitIndex(st *battle.State, actor *battle.Unit) int`
pub(crate) fn unit_index(st: Option<&battle::State>, actor: Option<UnitIdx>) -> i64 {
    todo!("native_ai_idle_recovery.go:unitIndex")
}

impl Game {
    /// Go：`native_ai_idle_recovery.go:276`，`func (g *Game) stepNativeAIIdleRecovery()`
    pub(crate) fn step_native_ai_idle_recovery(&mut self) {
        todo!("native_ai_idle_recovery.go:stepNativeAIIdleRecovery")
    }
}
