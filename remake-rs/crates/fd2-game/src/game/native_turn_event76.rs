//! Go 來源：`remake/cmd/fd2/native_turn_event76.go`（188 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// startNativeRawCamp2TurnEvents 是 round increment 後、玩家重新取得輸入前的
    /// sub_1A813(2) owner。缺少唯一 typed consumer 時維持失敗即關閉。
    ///
    /// Go：`native_turn_event76.go:112`，`func (g *Game) startNativeRawCamp2TurnEvents(then func()) (bool, error)`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_raw_camp2_turn_events(
        &mut self,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<bool> {
        todo!("native_turn_event76.go:startNativeRawCamp2TurnEvents")
    }
}
