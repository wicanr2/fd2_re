//! Go 來源：`remake/cmd/fd2/native_field_event61.go`（269 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_field_event61.go:12`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeFieldEvent61Job {
    pub(crate) plan: battle::NativeFieldEvent61Plan,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) last_tick: i64,
    pub(crate) has_tick: bool,
    pub(crate) clock: NativeBIOSClock,
    pub(crate) after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd: Option<ModernHDFieldEvent61>,
}

impl Game {
    /// beginNativeFieldEvent61 owns the selector1 event only after a successful
    /// unit action. A coordinate without event61 returns false; an event61 whose
    /// editable/original provenance is malformed blocks its mutation and reports
    /// the error rather than silently finishing a guessed handler.
    ///
    /// Go：`native_field_event61.go:51`，`func (g *Game) beginNativeFieldEvent61(actor *battle.Unit, after func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_field_event61(
        &mut self,
        actor: Option<UnitIdx>,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        // R4 B 組：事件 61 確實命中之後的兩條路徑都先呼叫 `event61DialogueActions`（`r8`，第一章沒有
        // 第 26 章地圖），停在 `todo!`；第一章只走「格子沒有事件 61」的早退。
        let (Some(st), Some(actor)) = (self.st.as_ref(), actor) else {
            return false;
        };
        if self.native_field_event61.is_some() {
            return false;
        }
        let (x, y) = (st.units[actor].x, st.units[actor].y);
        let (event_id, bound) = battle::native_field_event_id_at(Some(st), x, y, 1);
        if !bound || event_id != 61 {
            return false;
        }
        let plan = match battle::plan_native_field_event61(Some(st), Some(actor), x, y) {
            Ok(plan) => plan,
            Err(err) => {
                self.load_err = err.to_string();
                self.msg = String::from("事件 61 的原版證據不完整，已停止事件寫入");
                if let Some(after) = after {
                    after(self);
                }
                return true;
            }
        };
        let _ = plan;
        todo!(
            "R8: native_field_event61.go:event61DialogueActions／beginNativeFieldEvent61Presentation（第一章沒有執行到）"
        )
    }
}
