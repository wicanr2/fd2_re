//! Go 來源：`remake/cmd/fd2/native_continue_handoff.go`（389 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// consumeNativeContinueOpeningConfirm 僅辨識剛由原版 CONTINUE 發布的
    /// 第一個玩家確認，而且游標必須還在 save header 原樣記錄的位置。它只記錄
    /// 該 E2 首次輸入；共用空游標面板本身另由 0x117E7 直接指令證實。
    ///
    /// Go：`native_continue_handoff.go:379`，`func (g *Game) consumeNativeContinueOpeningConfirm() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn consume_native_continue_opening_confirm(&mut self) -> bool {
        todo!("native_continue_handoff.go:consumeNativeContinueOpeningConfirm")
    }
}
