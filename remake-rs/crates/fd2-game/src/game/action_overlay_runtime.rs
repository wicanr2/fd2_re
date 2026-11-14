//! Go 來源：`remake/cmd/fd2/action_overlay_runtime.go`（898 行）。
//!
//! 骨架：Go 檔的 28 個函式中產生 11 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`action_overlay_runtime.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeSystemEndTurnUIState {
    pub(crate) hd: Option<ModernHDSystemConfirmation>,
    pub(crate) source: Vec<u8>,
    pub(crate) dialogue: Vec<u8>,
    pub(crate) question: Vec<u8>,
    pub(crate) portraits: Vec<dato::Frame>,
    pub(crate) accepted: Vec<Vec<u8>>,
    pub(crate) canceled: Vec<Vec<u8>>,
    pub(crate) choice: i64,
    pub(crate) accepted_outcome: bool,
    pub(crate) exit_program: bool,
    pub(crate) group_march: bool,
    pub(crate) group_march_plan: battle::NativeSystemGroupMarchPlan,
    pub(crate) save_current: bool,
    pub(crate) save_path: String,
    pub(crate) save_stored: Vec<u8>,
    pub(crate) load_current: bool,
    pub(crate) load_candidate: Option<Box<Game>>,
    pub(crate) treasure: Option<NativeTreasurePrompt>,
    pub(crate) death_reward: Option<NativeDeathRewardUIState>,
    pub(crate) reward_message: Option<NativeDeathRewardMessageState>,
}

pub(crate) const ACTION_OVERLAY_OPENING: &str = "opening";

pub(crate) const ACTION_OVERLAY_OPEN: &str = "open";

pub(crate) const ACTION_OVERLAY_CLOSING: &str = "closing";

impl Game {
    /// beginActionOverlayOpen starts the four presents recovered at 0x1741c.
    /// There is no delay call between native presents, so the remake assigns one
    /// presented Ebiten frame to each step without claiming an original duration.
    ///
    /// Go：`action_overlay_runtime.go:44`，`func (g *Game) beginActionOverlayOpen(selection int)`
    pub(crate) fn begin_action_overlay_open(&mut self, selection: i64) {
        todo!("action_overlay_runtime.go:beginActionOverlayOpen")
    }

    /// beginActionOverlayClose starts the independent four-present sequence from
    /// 0x176b4. The selected action is deferred until all four close frames have
    /// been presented; it must not appear beneath an overlay that native code was
    /// still closing.
    ///
    /// Go：`action_overlay_runtime.go:59`，`func (g *Game) beginActionOverlayClose(after func())`
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_action_overlay_close(&mut self, after: Option<Box<dyn FnOnce(&mut Game) + Send>>) {
        todo!("action_overlay_runtime.go:beginActionOverlayClose")
    }

    /// Go：`action_overlay_runtime.go:73`，`func (g *Game) actionOverlayBlocksInput() bool`
    pub(crate) fn action_overlay_blocks_input(&self) -> bool {
        todo!("action_overlay_runtime.go:actionOverlayBlocksInput")
    }

    /// Go：`action_overlay_runtime.go:78`，`func (g *Game) actionOverlayRenderState() (frame int, closing bool)`
    pub(crate) fn action_overlay_render_state(&self) -> (i64, bool) {
        todo!("action_overlay_runtime.go:actionOverlayRenderState")
    }

    /// stepActionOverlayLifecycle runs once near the start of Update. A sequence is
    /// initialized later in an input Update, so frame zero is drawn before the next
    /// call advances it. The callback similarly runs only after close frame three
    /// was available to Draw for a complete update interval.
    ///
    /// Go：`action_overlay_runtime.go:93`，`func (g *Game) stepActionOverlayLifecycle()`
    pub(crate) fn step_action_overlay_lifecycle(&mut self) {
        todo!("action_overlay_runtime.go:stepActionOverlayLifecycle")
    }

    /// Go：`action_overlay_runtime.go:133`，`func (g *Game) markActionOverlayDrawn()`
    pub(crate) fn mark_action_overlay_drawn(&mut self) {
        todo!("action_overlay_runtime.go:markActionOverlayDrawn")
    }

    /// Go：`action_overlay_runtime.go:139`，`func (g *Game) resetActionOverlayLifecycle()`
    pub(crate) fn reset_action_overlay_lifecycle(&mut self) {
        todo!("action_overlay_runtime.go:resetActionOverlayLifecycle")
    }

    /// Go：`action_overlay_runtime.go:171`，`func (g *Game) currentNativeSystemOptions() fdother.NativeSystemOptions`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn current_native_system_options(&self) -> fdother::NativeSystemOptions {
        todo!("action_overlay_runtime.go:currentNativeSystemOptions")
    }

    /// Go：`action_overlay_runtime.go:701`，`func (g *Game) stepNativeSystemEndTurn()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn step_native_system_end_turn(&mut self) {
        todo!("action_overlay_runtime.go:stepNativeSystemEndTurn")
    }

    /// applyNativeOverlayDirectionInput 重現 sub_177FC 於 `0x17835..0x17897` 的四個
    /// 方向分支：掃描碼只有在目前 caller 的 availability 為零時才寫入 `[0x53c57]`；
    /// 不可用的方向按下去原版完全沒有反應，等待迴圈繼續。0x16F55、0x1728C、
    /// 0x18D8C、0x19DF7 與 0x1BBDC 五個 chooser 共用這個閘門，各自帶自己的表。
    ///
    /// Go：`action_overlay_runtime.go:879`，`func (g *Game) applyNativeOverlayDirectionInput()`
    pub(crate) fn apply_native_overlay_direction_input(&mut self) {
        todo!("action_overlay_runtime.go:applyNativeOverlayDirectionInput")
    }

    /// beginBattleActionOverlay 依 sub_173E7（0x18E53／0x18ED6 各呼叫一次）挑起始
    /// 方向：第一個 availability 為零的方向。硬寫 1 會讓起始選擇落在原版根本不會
    /// 停留的不可用格上，方向鍵閘門一旦成立就再也移不開。
    ///
    /// Go：`action_overlay_runtime.go:896`，`func (g *Game) beginBattleActionOverlay()`
    pub(crate) fn begin_battle_action_overlay(&mut self) {
        todo!("action_overlay_runtime.go:beginBattleActionOverlay")
    }
}
