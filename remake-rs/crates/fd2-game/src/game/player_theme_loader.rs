//! Go 來源：`remake/cmd/fd2/player_theme_loader.go`（82 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const MODERN_THEME_PENDING_ART: &str = "art";

pub(crate) const MODERN_THEME_PENDING_INTERFACE: &str = "interface";

/// Go：`player_theme_loader.go:22`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Debug, Default)]
pub(crate) struct ModernThemeLoad {
    pub(crate) done: Option<std::sync::mpsc::Receiver<()>>,
    pub(crate) set: Option<ModernStoryPortraitSet>,
    pub(crate) err: Option<GameError>,
    /// 載入完成後要補做的切換
    pub(crate) pending: String,
}

impl Game {
    /// pollModernThemeLoad 在每次 Update 開頭收回已完成的背景載入，成功就寫進快取並補做切換。
    ///
    /// Go：`player_theme_loader.go:56`，`func (g *Game) pollModernThemeLoad()`
    pub(crate) fn poll_modern_theme_load(&mut self) {
        todo!("player_theme_loader.go:pollModernThemeLoad")
    }
}
