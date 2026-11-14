//! Go 來源：`remake/cmd/fd2/modern_hd_church_confirmation_transition.go`（419 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_church_confirmation_transition.go:16`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernChurchConfirmationSource {
    pub(crate) raw: Vec<u8>,
    pub(crate) hd: Vec<u8>,
    pub(crate) mask: Vec<u8>,
    pub(crate) runs: Vec<ModernHDTextRun>,
}

/// Go：`modern_hd_church_confirmation_transition.go:21`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernChurchConfirmationDraw {
    pub(crate) frame: i64,
    pub(crate) choice_frame: i64,
    pub(crate) phase: String,
    pub(crate) interface_hd: bool,
    pub(crate) art_hd: bool,
}

/// Go：`modern_hd_church_confirmation_transition.go:27`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeChurchConfirmationPresentation {
    pub(crate) owner: String,
    pub(crate) variant: i64,
    pub(crate) resource: i64,
    pub(crate) pane_opening: bool,
    pub(crate) mode: String,
    pub(crate) kind: String,
    pub(crate) last_draw: Option<ModernChurchConfirmationDraw>,
}

/// Go：`modern_hd_church_confirmation_transition.go:43`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernChurchConfirmationFrame {
    pub(crate) pixels: Vec<u8>,
    pub(crate) mask: Vec<u8>,
    pub(crate) runs: Vec<ModernHDTextRun>,
    pub(crate) panels: Vec<ModernHDDialoguePanel>,
    pub(crate) name: String,
}

impl Game {
    /// Go：`modern_hd_church_confirmation_transition.go:54`，`func (g *Game) captureModernFacilityConfirmationSource(owner string, compose func() ([]byte, bool)) (s modernChurchConfirmationSource, valid bool)`
    /// Go 的 func 參數 `compose` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn capture_modern_facility_confirmation_source(
        &mut self,
        owner: &str,
        compose: Option<Box<dyn FnMut(&mut Game) -> (Vec<u8>, bool) + Send>>,
    ) -> (ModernChurchConfirmationSource, bool) {
        todo!("modern_hd_church_confirmation_transition.go:captureModernFacilityConfirmationSource")
    }
}
