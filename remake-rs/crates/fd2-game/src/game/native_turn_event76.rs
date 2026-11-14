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
        // R4 B 組：最後一輪（`plan.Final`）的預檢與演出呼叫 `preflightNativeEvent76Final`、
        // `event76DialogueActions`、`runNativeEvent76Presentation`（都是 `r8`，第 29 章），停在 `todo!`。
        let Some(sc) = self.sc.as_ref().filter(|sc| !sc.native_turn_events.is_empty()) else {
            return Ok(false);
        };
        let events = sc.native_turn_events_at(self.st.as_ref(), 2)?;
        if events.is_empty() {
            return Ok(false);
        }
        if events.len() != 1 || events[0].event_id != 76 {
            return Err(game_err!("raw camp2 has {} handlers without a unique event76 adapter", events.len()));
        }
        let event = &events[0];
        let plan = battle::plan_native_turn_event76(self.st.as_ref(), event)?;
        if !plan.r#final {
            battle::commit_native_turn_event76_repeat(self.st.as_mut(), &plan)?;
            if let Some(then) = then {
                then(self);
            }
            return Ok(true);
        }
        todo!(
            "R8: native_turn_event76.go:preflightNativeEvent76Final／runNativeEvent76Presentation（第一章沒有執行到）"
        )
    }
}
