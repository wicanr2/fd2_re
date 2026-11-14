//! Go 來源：`remake/cmd/fd2/native_command_modifier_presentation.go`（138 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command_modifier_presentation.go:13`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommandModifierPresentationJob {
    pub(crate) command_id: i64,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palettes: Vec<ColorPalette>,
    pub(crate) phase: i64,
    pub(crate) drawn: bool,
    pub(crate) transaction: Option<Box<dyn FnMut(&mut Game) -> Result<()> + Send>>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd_frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) unit_present: Option<NativeUnitPresentJob>,
}

impl Game {
    /// Go：`native_command_modifier_presentation.go:95`，`func (g *Game) stepNativeCommandModifierPresentation()`
    pub(crate) fn step_native_command_modifier_presentation(&mut self) {
        todo!("native_command_modifier_presentation.go:stepNativeCommandModifierPresentation")
    }
}
