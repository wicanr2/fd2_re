//! Go 來源：`remake/cmd/fd2/modern_hd_service_menu_transition.go`（252 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_service_menu_transition.go:14`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
pub(crate) struct NativeServiceMenuPresentation {
    pub(crate) owner: String,
    pub(crate) locale: String,
    pub(crate) variant: i64,
    pub(crate) resource: i64,
    pub(crate) closing: bool,
    pub(crate) font: Option<Font>,
    pub(crate) palette: [ColorRgba; 256],
    pub(crate) source: ModernChurchConfirmationSource,
    pub(crate) cells: [fdother::RawCell; 4],
    pub(crate) raw_frames: Vec<Vec<u8>>,
    pub(crate) chrome: Option<ModernHDDialogueChrome>,
    pub(crate) shop_job: Option<Box<NativeClassUIJob>>,
    pub(crate) church_job: Option<Box<NativeChurchUIJob>>,
    pub(crate) last_draw: Option<ModernChurchConfirmationDraw>,
}

impl Default for NativeServiceMenuPresentation {
    fn default() -> Self {
        Self {
            owner: Default::default(),
            locale: Default::default(),
            variant: Default::default(),
            resource: Default::default(),
            closing: Default::default(),
            font: Default::default(),
            palette: std::array::from_fn(|_| Default::default()),
            source: Default::default(),
            cells: Default::default(),
            raw_frames: Default::default(),
            chrome: Default::default(),
            shop_job: Default::default(),
            church_job: Default::default(),
            last_draw: Default::default(),
        }
    }
}

impl Game {
    /// Go：`modern_hd_service_menu_transition.go:31`，`func (g *Game) bindModernFacilityServiceMenu(shopJob *nativeClassUIJob, churchJob *nativeChurchUIJob, closing bool) *nativeServiceMenuPresentation`
    pub(crate) fn bind_modern_facility_service_menu(
        &mut self,
        shop_job: Option<&mut NativeClassUIJob>,
        church_job: Option<&mut NativeChurchUIJob>,
        closing: bool,
    ) -> Option<NativeServiceMenuPresentation> {
        todo!("modern_hd_service_menu_transition.go:bindModernFacilityServiceMenu")
    }
}

impl NativeServiceMenuPresentation {
    /// All four entries are normal cells. Preserve native signed division and order.
    ///
    /// Go：`modern_hd_service_menu_transition.go:101`，`func (d *nativeServiceMenuPresentation) compose(index int, hd bool) (p modernChurchConfirmationFrame, ok bool)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn compose(&mut self, index: i64, hd: bool) -> (ModernChurchConfirmationFrame, bool) {
        todo!("modern_hd_service_menu_transition.go:compose")
    }
}
