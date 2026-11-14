//! Go 來源：`remake/cmd/fd2/native_class_ui.go`（244 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_class_ui.go:16`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeClassUIAssets {
    pub(crate) background: Vec<u8>,
    pub(crate) entries: Vec<fdother::LMI1Entry>,
    pub(crate) panel: fdother::LMI1Entry,
    pub(crate) price_cell: fdother::RawCell,
    pub(crate) choices: Vec<fdother::RawCell>,
    pub(crate) dialogue: Vec<fdother::RawCell>,
    pub(crate) digits: Vec<fdother::Frame>,
    pub(crate) portrait: dato::Frame,
    pub(crate) portraits: Vec<dato::Frame>,
    pub(crate) units: Option<fdicon::Bank>,
    pub(crate) strings: Option<fdtxt::Strings>,
    pub(crate) font: Option<fdtxt::Font>,
    pub(crate) palette: ColorPalette,
    pub(crate) palette_dac: Vec<u8>,
    pub(crate) revive_fx: Vec<fdother::Frame>,
}

/// Go：`native_class_ui.go:34`，`func loadNativeClassUIAssets() (*nativeClassUIAssets, error)`
pub(crate) fn load_native_class_ui_assets() -> Result<Option<NativeClassUIAssets>> {
    todo!("native_class_ui.go:loadNativeClassUIAssets")
}
