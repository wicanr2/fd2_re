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
        if self.native_death_reward_ui.is_some() || self.pending_native_death_rewards.is_empty() {
            return false;
        }
        let pending = self.pending_native_death_rewards[0];
        let st = self.st.as_ref().expect("Go：pending.killer 指向 g.st.Units");
        let killer = pending.killer.map(|k| &st.units[k]);
        let valid = killer.is_some_and(|k| {
            k.inventory.len() == 8 && battle::validate_native_inventory_projection(Some(k)).is_ok() && k.has_battle_fig
        });
        if !valid
            || pending.item < 0
            || pending.item > 0xff
            || self.native_preparation_ui.is_none()
            || self.native_class_ui.is_none()
        {
            self.load_err = String::from("native death reward: full inventory or dialogue provenance unavailable");
            return true;
        }
        let killer = killer.unwrap();
        if killer.native_inventory_flags.iter().any(|&flag| flag & 0x80 != 0) {
            self.load_err = String::from("native death reward: full raw inventory contains an empty slot");
            return true;
        }
        match load_native_separated_portrait(killer.battle_fig) {
            Ok(portraits) if portraits.len() > 3 => {}
            _ => {
                self.load_err = String::from("native death reward: killer portrait unavailable");
                return true;
            }
        }
        if let Err(err) = self.compose_native_map_frame() {
            self.load_err = format!("native death reward: {}", err.message());
            return true;
        }
        let _ = then;
        todo!(
            "R8: native_death_reward_ui.go:58 prepareNativeItemPanel 之後的滿欄丟棄提示（0x1AA56；第一章重播只走到開頭的早退）"
        )
    }
}
