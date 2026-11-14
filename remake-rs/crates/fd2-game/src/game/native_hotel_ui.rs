//! Go 來源：`remake/cmd/fd2/native_hotel_ui.go`（284 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const NATIVE_HOTEL_PORTRAIT_ID: i64 = 0x81;

pub(crate) const NATIVE_HOTEL_PROMPT_TEXT: i64 = 0x249;

pub(crate) const NATIVE_HOTEL_SAVED_TEXT: i64 = 0x294;

pub(crate) const NATIVE_HOTEL_RESOURCE_ID: i64 = 13;

pub(crate) const NATIVE_HOTEL_SERVICE_SAVE: i64 = 1;

pub(crate) const NATIVE_HOTEL_SERVICE_LOAD: i64 = 2;

pub(crate) const NATIVE_HOTEL_SERVICE_EXIT: i64 = 3;

/// Go：`native_hotel_ui.go:30`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeHotelUIAssets {
    pub(crate) assets: Option<campaign::NativeShopAssets>,
    pub(crate) portraits: Vec<dato::Frame>,
    pub(crate) slots_box: fdother::LMI1Entry,
}

/// Go：`native_hotel_ui.go:36`，`func loadNativeHotelUIAssets(shared *nativeClassUIAssets) (*nativeHotelUIAssets, error)`
pub(crate) fn load_native_hotel_ui_assets(shared: Option<&NativeClassUIAssets>) -> Result<Option<NativeHotelUIAssets>> {
    todo!("native_hotel_ui.go:loadNativeHotelUIAssets")
}

impl Game {
    /// setupNativeHotel 進酒店節點時切到原生介面；素材不齊就維持舊版面（回 false）。
    ///
    /// Go：`native_hotel_ui.go:69`，`func (g *Game) setupNativeHotel() bool`
    pub(crate) fn setup_native_hotel(&mut self) -> bool {
        todo!("native_hotel_ui.go:setupNativeHotel")
    }
}

/// nativeHotelInput 是酒店原生介面的輸入；與 0x2D7BD 相同：左右換圖示、enter 選、
/// esc 離開（0x2FEA0）。存檔列表上下選槽、enter 寫檔、esc 回選單；「記錄儲存完畢」
/// 任一鍵回選單。
///
/// Go：`native_hotel_ui.go:169`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeHotelInput {
    pub(crate) delta: i64,
    pub(crate) up: bool,
    pub(crate) down: bool,
    pub(crate) enter: bool,
    pub(crate) esc: bool,
}

impl Game {
    /// Go：`native_hotel_ui.go:187`，`func (g *Game) handleNativeHotelInput(in nativeHotelInput) bool`
    pub(crate) fn handle_native_hotel_input(&mut self, r#in: NativeHotelInput) -> bool {
        todo!("native_hotel_ui.go:handleNativeHotelInput")
    }
}
