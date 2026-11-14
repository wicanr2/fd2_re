//! Go 來源：`remake/cmd/fd2/native_defeat.go`（256 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeDefeatJob 擁有 READY §5.2.1 的有限呈現，不讀鍵盤、不寫存檔。
/// 原始定位與分級：fd2_pending_code1_return_title_20261001.json。
///
/// Go：`native_defeat.go:18`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeDefeatJob {
    pub(crate) plan: Vec<fdother::PendingCode1PresentationStep>,
    pub(crate) pos: i64,
    pub(crate) frames: Vec<fdother::Frame>,
    pub(crate) vga: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) ramp: Option<NativePaletteRampJob>,
    pub(crate) drawn: bool,
    pub(crate) wait_start: GameInstant,
    pub(crate) title: Option<TitleAssets>,
}

impl Game {
    /// Go：`native_defeat.go:30`，`func (g *Game) beginNativeDefeatIfConfigured()`
    pub(crate) fn begin_native_defeat_if_configured(&mut self) {
        todo!("native_defeat.go:beginNativeDefeatIfConfigured")
    }
}
