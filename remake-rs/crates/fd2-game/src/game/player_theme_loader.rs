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
    ///
    /// Go 的背景 goroutine 把結果寫進 `load.set`／`load.err` 再 `close(done)`。Rust 的
    /// `ModernThemeLoad` 只有 `Receiver<()>`，背景載入端（`requestModernThemeLoad`，R6）要在送出
    /// 完成通知前把結果交回；忠實模式下 `modern_theme_loading` 永遠是 `None`，這裡只會走第一個 return。
    pub(crate) fn poll_modern_theme_load(&mut self) {
        let Some(load) = self.modern_theme_loading.as_ref() else {
            return;
        };
        // Go `select { case <-load.done: default: return }`：已關閉的 channel 立刻可讀；
        // Rust 的傳送端關閉（Disconnected）或送出通知（Ok）都算完成。
        let finished = match &load.done {
            None => false,
            Some(done) => !matches!(done.try_recv(), Err(std::sync::mpsc::TryRecvError::Empty)),
        };
        if !finished {
            return;
        }
        let load = self.modern_theme_loading.take().unwrap();
        if let Some(err) = load.err {
            self.msg = format!("現代主題不可用：{}", err.message());
            return;
        }
        self.modern_theme_cache = load.set;
        match load.pending.as_str() {
            MODERN_THEME_PENDING_ART => {
                if self.modern_story_portraits.is_none() {
                    todo!("R6: player_theme.go:cycleTheme（HD 美術切換）")
                }
            }
            MODERN_THEME_PENDING_INTERFACE if !self.modern_hd_interface_enabled() => {
                todo!("R6: player_interface.go:cycleInterfaceMode（HD 介面切換）")
            }
            _ => {}
        }
    }
}
