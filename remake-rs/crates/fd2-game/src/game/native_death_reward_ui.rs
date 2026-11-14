//! Go 來源：`remake/cmd/fd2/native_death_reward_ui.go`（258 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_death_reward_ui.go:10`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct PendingNativeDeathReward {
    pub(crate) killer: Option<UnitIdx>,
    pub(crate) item: i64,
}

/// Go：`native_death_reward_ui.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Clone, Default)]
pub(crate) struct NativeDeathRewardUIState {
    pub(crate) killer: Option<UnitIdx>,
    pub(crate) previous_selection: Option<UnitIdx>,
    pub(crate) portrait: dato::Frame,
    pub(crate) item: i64,
    pub(crate) selected: i64,
    pub(crate) source: Vec<u8>,
    pub(crate) dialogue: Vec<u8>,
    pub(crate) hd: Option<ModernHDFullReward>,
}

impl Game {
    /// runPendingNativeDeathRewards starts the first 0x1AA56 modal only at action
    /// completion, after the attack presentation. It preflights both the indexed
    /// question and the later item panel before exposing any interactive state.
    ///
    /// Go：`native_death_reward_ui.go:32`，`func (g *Game) runPendingNativeDeathRewards(then func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn run_pending_native_death_rewards(&mut self, then: Option<Box<dyn FnOnce(&mut Game) + Send>>) -> bool {
        todo!("native_death_reward_ui.go:runPendingNativeDeathRewards")
    }
}
