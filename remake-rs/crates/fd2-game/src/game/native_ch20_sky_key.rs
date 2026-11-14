//! Go 來源：`remake/cmd/fd2/native_ch20_sky_key.go`（390 行）。
//!
//! 骨架：Go 檔的 15 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_ch20_sky_key.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCh20SkyKeyPhase(pub(crate) u8);

/// nativeCh20SkyKeyJob owns the complete blocking presentation called at
/// 0x242c9. Indexed framebuffer and DAC snapshots remain separate throughout:
/// the white flash and baseline restore must recolour the final ANI indices,
/// not replace them with a generic RGBA overlay.
///
/// Go：`native_ch20_sky_key.go:32`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCh20SkyKeyJob {
    pub(crate) hd_world: Option<ModernSkyKeyWorld>,
    pub(crate) hd_frame: Option<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) spec: campaign::NativeCh20SkyKeySequence,
    pub(crate) fd_frames: Vec<fdother::Frame>,
    pub(crate) ani_pixels: Vec<Vec<u8>>,
    pub(crate) ani_dac: Vec<Vec<u8>>,
    pub(crate) base: Vec<u8>,
    pub(crate) vga: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) phase: NativeCh20SkyKeyPhase,
    pub(crate) frame: i64,
    pub(crate) ticks: i64,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) rollback: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}
