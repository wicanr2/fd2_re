//! Go 來源：`remake/cmd/fd2/native_command34_presentation.go`（536 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command34_presentation.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand34PresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCompoundCommand34Plan>,
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) publish_at: [i64; 3],
    pub(crate) stage_end: [i64; 3],
    pub(crate) published: i64,
    pub(crate) drawn: bool,
    pub(crate) holding: bool,
    pub(crate) hold: i64,
    pub(crate) hold_stage: i64,
    pub(crate) rng_before: u16,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) stage_work: [Vec<u8>; 3],
    pub(crate) stage_vga: [Vec<u8>; 3],
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, battle::NativeCompoundCommand34Result) + Send>>,
}

impl Game {
    /// Go：`native_command34_presentation.go:431`，`func (g *Game) cancelNativeCommand34Presentation()`
    pub(crate) fn cancel_native_command34_presentation(&mut self) {
        todo!("native_command34_presentation.go:cancelNativeCommand34Presentation")
    }

    /// Go：`native_command34_presentation.go:448`，`func (g *Game) stepNativeCommand34Presentation()`
    pub(crate) fn step_native_command34_presentation(&mut self) {
        todo!("native_command34_presentation.go:stepNativeCommand34Presentation")
    }
}
