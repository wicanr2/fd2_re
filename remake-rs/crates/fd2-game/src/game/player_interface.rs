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
        match self.interface_mode.as_str() {
            PLAYER_INTERFACE_ORIGINAL => false,
            PLAYER_INTERFACE_HD => true,
            _ => self.modern_story_portraits.as_ref().is_some_and(|p| p.has_hd),
        }
    }

    /// Go：`player_interface.go:36`，`func (g *Game) interfaceThemeForArt(art *modernStoryPortraitSet) *modernStoryPortraitSet`
    /// Go 回傳指標（`art` 本身或 `g` 的欄位）；Rust 回傳借用，不複製整份主題（簽名變更見 `PORTING.md`）。
    pub(crate) fn interface_theme_for_art<'a>(
        &'a self,
        art: Option<&'a ModernStoryPortraitSet>,
    ) -> Option<&'a ModernStoryPortraitSet> {
        if self.interface_mode == PLAYER_INTERFACE_ORIGINAL {
            return None;
        }
        if art.is_some() {
            return art;
        }
        if self.interface_mode == PLAYER_INTERFACE_HD {
            if self.modern_theme_cache.is_some() {
                return self.modern_theme_cache.as_ref();
            }
            return self.modern_story_portraits.as_ref();
        }
        None
    }

    /// Go：`player_interface.go:52`，`func (g *Game) interfaceAssetTheme() *modernStoryPortraitSet`
    /// Rust 回傳借用（簽名變更見 `PORTING.md`）。
    pub(crate) fn interface_asset_theme(&self) -> Option<&ModernStoryPortraitSet> {
        self.interface_theme_for_art(self.modern_story_portraits.as_ref())
    }
}
