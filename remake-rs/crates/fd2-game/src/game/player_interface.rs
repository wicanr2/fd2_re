//! Go 來源：`remake/cmd/fd2/player_interface.go`（165 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const PLAYER_INTERFACE_ORIGINAL: &str = "original";

pub(crate) const PLAYER_INTERFACE_HD: &str = "hd";

impl Game {
    /// Empty mode retains compatibility with isolated legacy render fixtures. A
    /// normal game resolves it once at startup, then F2 and F11 stay independent.
    ///
    /// Go：`player_interface.go:18`，`func (g *Game) modernHDInterfaceEnabled() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn modern_hd_interface_enabled(&self) -> bool {
        todo!("player_interface.go:modernHDInterfaceEnabled")
    }

    /// Go：`player_interface.go:36`，`func (g *Game) interfaceThemeForArt(art *modernStoryPortraitSet) *modernStoryPortraitSet`
    /// Go 第 1 個回傳值是指向參數 `art` 的指標；骨架回傳擁有的值，移植時再決定借用方式。
    pub(crate) fn interface_theme_for_art(
        &self,
        art: Option<&ModernStoryPortraitSet>,
    ) -> Option<ModernStoryPortraitSet> {
        todo!("player_interface.go:interfaceThemeForArt")
    }

    /// Go：`player_interface.go:52`，`func (g *Game) interfaceAssetTheme() *modernStoryPortraitSet`
    pub(crate) fn interface_asset_theme(&mut self) -> Option<ModernStoryPortraitSet> {
        todo!("player_interface.go:interfaceAssetTheme")
    }
}
