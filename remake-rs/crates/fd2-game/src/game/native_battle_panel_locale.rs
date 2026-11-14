//! Go 來源：`remake/cmd/fd2/native_battle_panel_locale.go`（135 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const LOCALIZED_BATTLE_PANEL_NAME_WIDTH: i64 = 120;

pub(crate) const LOCALIZED_BATTLE_PANEL_NAME_HEIGHT: i64 = 16;

pub(crate) const LOCALIZED_BATTLE_PANEL_NAME_SCALE: f64 = 0.65;

impl Game {
    /// Go：`native_battle_panel_locale.go:18`，`func (g *Game) renderLocalizedNativeBattlePanel( assets battle.NativeItemPanelDataAssets, record, dst []byte, unitIndex, rawChapter int, ) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn render_localized_native_battle_panel(
        &mut self,
        assets: &battle::NativeItemPanelDataAssets,
        record: &[u8],
        dst: &mut [u8],
        unit_index: i64,
        raw_chapter: i64,
    ) -> Result<()> {
        todo!("native_battle_panel_locale.go:renderLocalizedNativeBattlePanel")
    }
}
