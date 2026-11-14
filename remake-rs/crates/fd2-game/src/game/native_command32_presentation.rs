//! Go 來源：`remake/cmd/fd2/native_command32_presentation.go`（522 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command32_presentation.go:16`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeCompoundPresentedFrame {
    pub(crate) hd_map: Option<ModernHDCompoundMapSource>,
    pub(crate) hd_sources: Option<ModernHDBattleSources>,
    pub(crate) hd_palette: ColorPalette,
    pub(crate) hd_resource: i64,
    pub(crate) hd_mix: f32,
    pub(crate) hd_target: [f32; 3],
    pub(crate) hd_subtract: f32,
    pub(crate) hd_phase: String,
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) delay: i64,
    pub(crate) sound: Vec<u8>,
    pub(crate) sound_played: bool,
}

/// Go：`native_command32_presentation.go:31`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand32PresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCompoundCommand32Plan>,
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
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, battle::NativeCompoundCommand32Result) + Send>>,
}

impl Game {
    /// Go：`native_command32_presentation.go:429`，`func (g *Game) cancelNativeCommand32Presentation()`
    pub(crate) fn cancel_native_command32_presentation(&mut self) {
        todo!("native_command32_presentation.go:cancelNativeCommand32Presentation")
    }

    /// Go：`native_command32_presentation.go:446`，`func (g *Game) stepNativeCommand32Presentation()`
    pub(crate) fn step_native_command32_presentation(&mut self) {
        todo!("native_command32_presentation.go:stepNativeCommand32Presentation")
    }
}
