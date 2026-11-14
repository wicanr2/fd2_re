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
        todo!("native_field_event75.go:beginNativeFieldEvent75")
    }
}
