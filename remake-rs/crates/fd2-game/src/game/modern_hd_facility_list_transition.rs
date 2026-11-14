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
///
/// 簽名改寫：`compose` 收 `&mut dyn FnMut(&mut Game)`（Go 是讀 `g` 的方法值）。
///
/// Go 暫時換上空的 `modernHDText`，以 `beginModernHDFacilityText` 把設施字型換成會記錄 HD 文字的 presenter，
/// 再重組一次畫面。那個 presenter 要以 TTF 查字形並在繪字時寫回擷取結果，屬於 HD 文字層（R6）；這裡照 Go
/// 移植 `capture == nil` 的退出（`nativeClassUI` 為 nil；Go 在函式開頭已解參考 `g.font`）。
pub(crate) fn capture_modern_facility_list_text(
    g: &mut Game,
    compose: &mut dyn FnMut(&mut Game) -> (Vec<u8>, bool),
) -> (Vec<u8>, Vec<ModernHDTextRun>, bool) {
    assert!(g.font.is_some(), "captureModernFacilityListText: g.font is nil");
    if g.native_class_ui.is_none() {
        return (Vec::new(), Vec::new(), false);
    }
    todo!("R6: modern_hd_facility_list_transition.go:captureModernFacilityListText（HD 文字擷取 presenter）")
}

impl Game {
    /// Go：`modern_hd_facility_list_transition.go:64`，`func (g *Game) composeModernShopListMarker(fill byte, compose func() ([]byte, bool)) ([]byte, bool)`
    ///
    /// 簽名改寫：`compose` 收 `&mut dyn FnMut(&mut Game)`。Go 暫時把 `g.nativeShopUI` 換成背景與店員頭像填滿
    /// `fill` 的複本再重組，`defer` 換回；Rust 照樣換上複本、呼叫後放回原值。
    pub(crate) fn compose_modern_shop_list_marker(
        &mut self,
        fill: u8,
        compose: &mut dyn FnMut(&mut Game) -> (Vec<u8>, bool),
    ) -> (Vec<u8>, bool) {
        // Go 直接解參考 `g.nativeShopUI` 與 `original.shops[g.nativeShopVariant]`。
        let original = self.native_shop_ui.take().expect("composeModernShopListMarker: nativeShopUI is nil");
        let mut copy_ui = original.clone();
        let assets = copy_ui
            .shops
            .get_mut(&self.native_shop_variant)
            .expect("composeModernShopListMarker: shop variant is missing");
        assets.background = vec![fill; 64000];
        copy_ui
            .portraits
            .insert(self.native_shop_variant, dato::Frame { width: 80, height: 80, pixels: vec![fill; 6400] });
        self.native_shop_ui = Some(copy_ui);
        let result = compose(self);
        self.native_shop_ui = Some(original);
        result
    }

    /// Go：`modern_hd_facility_list_transition.go:88`，`func (g *Game) freezeModernFacilityListWriter(owner string, frame []byte, compose func() ([]byte, bool)) ([]byte, bool)`
    ///
    /// 簽名改寫：`compose` 收 `&mut dyn FnMut(&mut Game)`（Go 只在本次呼叫內使用，不存起來）。
    pub(crate) fn freeze_modern_facility_list_writer(
        &mut self,
        owner: &str,
        frame: &[u8],
        compose: &mut dyn FnMut(&mut Game) -> (Vec<u8>, bool),
    ) -> (Vec<u8>, bool) {
        let mut marker = |g: &mut Game, fill: u8| -> (Vec<u8>, bool) {
            match owner {
                "shop" => g.compose_modern_shop_list_marker(fill, compose),
                "church" => {
                    g.compose_modern_facility_marker(&ModernStableFacilitySource { resource: 14 }, fill, compose)
                }
                _ => (Vec::new(), false),
            }
        };
        let (a, ok_a) = marker(self, 0);
        let (b, ok_b) = marker(self, 255);
        if !ok_a || !ok_b || frame.len() != 64000 || a.len() != 64000 || b.len() != 64000 {
            return (Vec::new(), false);
        }
        let mut mask = vec![0u8; 64000];
        for i in 0..mask.len() {
            if a[i] == b[i] {
                if a[i] != frame[i] {
                    return (Vec::new(), false);
                }
                mask[i] = 255;
            }
        }
        (mask, true)
    }
}
