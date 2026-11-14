//! Go 來源：`remake/cmd/fd2/native_battle_staging_present.go`（87 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// Go：`native_battle_staging_present.go:75`，`func (g *Game) beginNativeFieldEvent80(actor *battle.Unit) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn begin_native_field_event80(&mut self, actor: Option<UnitIdx>) -> bool {
        todo!("native_battle_staging_present.go:beginNativeFieldEvent80")
    }
}
