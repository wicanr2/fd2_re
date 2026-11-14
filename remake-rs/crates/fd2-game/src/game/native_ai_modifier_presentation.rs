//! Go 來源：`remake/cmd/fd2/native_ai_modifier_presentation.go`（604 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_ai_modifier_presentation.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeAICommandModifierPresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeAICommandModifierPlan>,
    pub(crate) plan2022: Option<battle::NativeAICommand2022Plan>,
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) publish_at: i64,
    pub(crate) end_at: i64,
    pub(crate) drawn: bool,
    pub(crate) holding: bool,
    pub(crate) published: bool,
    pub(crate) skip_hold: bool,
    pub(crate) hold: i64,
    pub(crate) rng_before: u16,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) stage_work: Vec<u8>,
    pub(crate) stage_vga: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, battle::NativeCommandModifierResult) + Send>>,
    pub(crate) then2022: Option<Box<dyn FnOnce(&mut Game, &battle::NativeAICommand2022Plan) + Send>>,
}

impl Game {
    /// Go：`native_ai_modifier_presentation.go:483`，`func (g *Game) cancelNativeAICommandModifierPresentation()`
    pub(crate) fn cancel_native_ai_command_modifier_presentation(&mut self) {
        todo!("native_ai_modifier_presentation.go:cancelNativeAICommandModifierPresentation")
    }

    /// Go：`native_ai_modifier_presentation.go:504`，`func (g *Game) stepNativeAICommandModifierPresentation()`
    pub(crate) fn step_native_ai_command_modifier_presentation(&mut self) {
        todo!("native_ai_modifier_presentation.go:stepNativeAICommandModifierPresentation")
    }
}
