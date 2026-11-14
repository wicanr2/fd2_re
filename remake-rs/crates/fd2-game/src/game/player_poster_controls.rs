//! Go 來源：`remake/cmd/fd2/player_poster_controls.go`（190 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

impl Game {
    /// Go：`player_poster_controls.go:23`，`func (g *Game) playerPosterControlsAllowed() bool`
    pub(crate) fn player_poster_controls_allowed(&self) -> bool {
        !self.startup_blocked
            && self.load_err.is_empty()
            && self.native_defeat.is_none()
            && self.native_ending.is_none()
            && !self.help_visible
            && !self.transient_ui
            && !native_modifier_held()
            && self.native_system_end_turn_ui.as_ref().is_none_or(|ui| ui.reward_message.is_none())
    }
}
