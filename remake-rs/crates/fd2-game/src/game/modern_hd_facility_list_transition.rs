//! Go 來源：`remake/cmd/fd2/modern_hd_facility_list_transition.go`（476 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_facility_list_transition.go:16`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeFacilityListPresentation {
    pub(crate) variant: i64,
    pub(crate) mode: String,
    pub(crate) owner: String,
    pub(crate) closing: bool,
}

/// Go：`modern_hd_facility_list_transition.go:35`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct ModernFacilityListRoster {
    pub(crate) job: Option<NativeClassUIJob>,
    pub(crate) frame: i64,
    pub(crate) clip: Rectangle,
}

/// Go：`modern_hd_facility_list_transition.go:44`，`func captureModernFacilityListText(g *Game, compose func() ([]byte, bool)) ([]byte, []modernHDTextRun, bool)`
pub(crate) fn capture_modern_facility_list_text(
    g: &mut Game,
    compose: &mut dyn FnMut() -> (Vec<u8>, bool),
) -> (Vec<u8>, Vec<ModernHDTextRun>, bool) {
    todo!("modern_hd_facility_list_transition.go:captureModernFacilityListText")
}

impl Game {
    /// Go：`modern_hd_facility_list_transition.go:64`，`func (g *Game) composeModernShopListMarker(fill byte, compose func() ([]byte, bool)) ([]byte, bool)`
    pub(crate) fn compose_modern_shop_list_marker(
        &mut self,
        fill: u8,
        compose: &mut dyn FnMut() -> (Vec<u8>, bool),
    ) -> (Vec<u8>, bool) {
        todo!("modern_hd_facility_list_transition.go:composeModernShopListMarker")
    }

    /// Go：`modern_hd_facility_list_transition.go:88`，`func (g *Game) freezeModernFacilityListWriter(owner string, frame []byte, compose func() ([]byte, bool)) ([]byte, bool)`
    /// Go 的 func 參數 `compose` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn freeze_modern_facility_list_writer(
        &mut self,
        owner: &str,
        frame: &[u8],
        compose: Option<Box<dyn FnMut(&mut Game) -> (Vec<u8>, bool) + Send>>,
    ) -> (Vec<u8>, bool) {
        todo!("modern_hd_facility_list_transition.go:freezeModernFacilityListWriter")
    }
}
