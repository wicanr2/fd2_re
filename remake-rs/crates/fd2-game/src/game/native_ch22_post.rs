//! Go 來源：`remake/cmd/fd2/native_ch22_post.go`（248 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// native2189AJob is the exact ten-pass indexed presentation owned by 0x2189A.
/// The address-shaped name is deliberate: evidence proves its buffer schedule,
/// not a universal gameplay or narrative meaning.
///
/// Go：`native_ch22_post.go:18`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct Native2189AJob {
    pub(crate) work_frames: Vec<Vec<u8>>,
    pub(crate) vga_frames: Vec<Vec<u8>>,
    pub(crate) palette: ColorPalette,
    pub(crate) index: i64,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) rollback: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd_input: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) hd_terrain: Vec<u8>,
    pub(crate) hd_radial: Vec<fdother::RadialLUTRemap>,
    pub(crate) hd_luts: Vec<Vec<u8>>,
    pub(crate) hd_frames: Vec<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) map_stage: Option<NativeCh23AdapterState>,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_steady: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_signature: [u8; 32],
}

/// Go：`native_ch22_post.go:39`，`func nativeTransitionInput(in indexedmap.NativeFrameInput) indexedmap.NativeTransitionFrameInput`
pub(crate) fn native_transition_input(
    r#in: &indexedmap::NativeFrameInput<'_>,
) -> indexedmap::NativeTransitionFrameInput<'static> {
    todo!("native_ch22_post.go:nativeTransitionInput")
}
