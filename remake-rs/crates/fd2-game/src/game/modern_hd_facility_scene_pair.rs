//! Go 來源：`remake/cmd/fd2/modern_hd_facility_scene_pair.go`（157 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_facility_scene_pair.go:44`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernStableFacilitySource {
    pub(crate) resource: i64,
}

impl Game {
    /// Marker passes replay the real producer. Only the source-proved main DATO
    /// patch belongs to the HD scene; later UI/party-portrait writes stay foreground.
    ///
    /// Go：`modern_hd_facility_scene_pair.go:103`，`func (g *Game) composeModernFacilityMarker(s *modernStableFacilitySource, fill byte, compose func() ([]byte, bool)) ([]byte, bool)`
    ///
    /// 簽名改寫：`compose` 收 `&mut dyn FnMut(&mut Game)`。Go 暫時換上背景與頭像填滿 `fill` 的素材複本，
    /// `defer` 換回；Rust 照樣換上複本、呼叫後放回原值。
    pub(crate) fn compose_modern_facility_marker(
        &mut self,
        s: &ModernStableFacilitySource,
        fill: u8,
        compose: &mut dyn FnMut(&mut Game) -> (Vec<u8>, bool),
    ) -> (Vec<u8>, bool) {
        let background = vec![fill; 64000];
        let portrait = dato::Frame { width: 80, height: 80, pixels: vec![fill; 6400] };
        if s.resource == 13 {
            // Go 直接解參考 `g.nativeHotelUI` 與 `original.assets`。
            let original = self.native_hotel_ui.take().expect("composeModernFacilityMarker: nativeHotelUI is nil");
            let mut copy_ui = original.clone();
            copy_ui.assets.as_mut().expect("composeModernFacilityMarker: hotel assets are nil").background = background;
            copy_ui.portraits[0] = portrait;
            self.native_hotel_ui = Some(copy_ui);
            let result = compose(self);
            self.native_hotel_ui = Some(original);
            result
        } else {
            // Go 直接解參考 `g.nativeClassUI`。
            let original = self.native_class_ui.take().expect("composeModernFacilityMarker: nativeClassUI is nil");
            let mut copy_ui = original.clone();
            (copy_ui.background, copy_ui.portrait) = (background, portrait);
            self.native_class_ui = Some(copy_ui);
            let result = compose(self);
            self.native_class_ui = Some(original);
            result
        }
    }
}
