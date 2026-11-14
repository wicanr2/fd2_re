//! Go 來源：`remake/cmd/fd2/native_command33_presentation.go`（487 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command33_presentation.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand33PresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCompoundCommand33Plan>,
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) publish_at: i64,
    pub(crate) published: bool,
    pub(crate) drawn: bool,
    pub(crate) holding: bool,
    pub(crate) hold: i64,
    pub(crate) rng_before: u16,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) post_transaction_work: Vec<u8>,
    pub(crate) post_vga: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, battle::NativeCompoundCommand33Result) + Send>>,
}

impl Game {
    /// Go：`native_command33_presentation.go:396`，`func (g *Game) cancelNativeCommand33Presentation()`
    pub(crate) fn cancel_native_command33_presentation(&mut self) {
        todo!("native_command33_presentation.go:cancelNativeCommand33Presentation")
    }

    /// Go：`native_command33_presentation.go:413`，`func (g *Game) stepNativeCommand33Presentation()`
    pub(crate) fn step_native_command33_presentation(&mut self) {
        todo!("native_command33_presentation.go:stepNativeCommand33Presentation")
    }
}
