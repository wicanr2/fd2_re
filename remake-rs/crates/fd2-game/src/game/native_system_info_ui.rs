//! Go 來源：`remake/cmd/fd2/native_system_info_ui.go`（233 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_system_info_ui.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeSystemInfoUIState {
    pub(crate) opening: Vec<Vec<u8>>,
    pub(crate) steady: Vec<u8>,
    pub(crate) closing: Vec<Vec<u8>>,
    pub(crate) phase: String,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) dac: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) input: campaign::NativeSystemInfoInput,
    pub(crate) text_index: i64,
    pub(crate) hd: Option<ModernHDSystemInfo>,
    pub(crate) hd_presented: bool,
}

/// Go：`native_system_info_ui.go:30`，`func loadNativeSystemInfoAssets() (*campaign.NativeSystemInfoAssets, error)`
pub(crate) fn load_native_system_info_assets() -> Result<Option<campaign::NativeSystemInfoAssets>> {
    todo!("native_system_info_ui.go:loadNativeSystemInfoAssets")
}

impl Game {
    /// Go：`native_system_info_ui.go:140`，`func (g *Game) stepNativeSystemInfoUI()`
    pub(crate) fn step_native_system_info_ui(&mut self) {
        todo!("native_system_info_ui.go:stepNativeSystemInfoUI")
    }
}
