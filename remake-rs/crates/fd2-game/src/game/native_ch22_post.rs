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
///
/// R4 B 組：回傳值借用 `in` 的素材（生命週期 `'a`），骨架的 `'static` 做不到。
pub(crate) fn native_transition_input<'a>(
    r#in: &indexedmap::NativeFrameInput<'a>,
) -> indexedmap::NativeTransitionFrameInput<'a> {
    let f = &r#in.frame;
    indexedmap::NativeTransitionFrameInput {
        parallax: f.parallax,
        parallax_scroll_x: f.parallax_scroll_x,
        parallax_scroll_y: f.parallax_scroll_y,
        terrain_bank: f.terrain_bank,
        unit_bank: f.unit_bank,
        foreground_bank: f.foreground_bank,
        selector_cache: f.selector_cache,
        cells: f.cells.clone(),
        controls: f.controls,
        terrain_lut: f.lut,
        map_width: f.map_width,
        camera_x: f.camera_x,
        camera_y: f.camera_y,
        flip: f.flip,
        terrain_cycle: f.terrain_cycle,
        idle_cycle: f.idle_cycle,
        moving_cycle: f.moving_cycle,
        pixel_shift: f.pixel_shift,
        units: f.units.clone(),
        foreground_units: f.foreground_units.clone(),
    }
}
