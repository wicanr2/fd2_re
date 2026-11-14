//! Go 來源：`remake/cmd/fd2/native_field_event75.go`（92 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// beginNativeFieldEvent75 只由成功動作 owner 消費 map28 selector1。
    /// FDTXT_029 index1 必須先完成，之後才可一起提交 live rows 與 0x20-byte
    /// event-state table。
    ///
    /// Go：`native_field_event75.go:48`，`func (g *Game) beginNativeFieldEvent75(actor *battle.Unit, after func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_field_event75(
        &mut self,
        actor: Option<UnitIdx>,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        // R4 B 組：要播對白時呼叫 `event75DialogueActions`（`r8`，第 29 章），停在 `todo!`。
        let (Some(st), Some(actor)) = (self.st.as_ref(), actor) else {
            return false;
        };
        let (x, y) = (st.units[actor].x, st.units[actor].y);
        let (event_id, bound) = battle::native_field_event_id_at(Some(st), x, y, 1);
        if !bound || event_id != 75 {
            return false;
        }
        let plan = match battle::plan_native_field_event75(Some(st), Some(actor), x, y) {
            Ok(plan) => plan,
            Err(err) => {
                self.load_err = err.to_string();
                self.msg = String::from("事件 75 的原版證據不完整，已停止事件寫入");
                if let Some(after) = after {
                    after(self);
                }
                return true;
            }
        };
        if plan.noop {
            if let Some(after) = after {
                after(self);
            }
            return true;
        }
        todo!("R8: native_field_event75.go:event75DialogueActions（第一章沒有執行到）")
    }
}
