//! Go 來源：`remake/cmd/fd2/native_battle_staging_present.go`（87 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// Go：`native_battle_staging_present.go:75`，`func (g *Game) beginNativeFieldEvent80(actor *battle.Unit) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn begin_native_field_event80(&mut self, actor: Option<UnitIdx>) -> bool {
        let (Some(st), Some(actor)) = (self.st.as_mut(), actor) else {
            return false;
        };
        let (x, y) = (st.units[actor].x, st.units[actor].y);
        let (id, ok) = battle::native_field_event_id_at(Some(st), x, y, 1);
        if !ok || id != 80 {
            return false;
        }
        let (_, applied) = battle::apply_native_field_mode_event(Some(st), Some(actor), x, y, 1);
        if !applied {
            self.load_err = String::from("event80: sourced mode rule unavailable");
        }
        true
    }
}
