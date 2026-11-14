//! Go 來源：`remake/cmd/fd2/native_unit_present.go`（359 行）。
//!
//! 骨架：Go 檔的 9 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeUnitPresentJob owns one complete 0x22253 transaction. Frames are
/// precomputed before publication; mutationAt is the first bridge row, after
/// all 11 intro and six contract full presents have actually been drawn.
///
/// Go：`native_unit_present.go:19`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeUnitPresentJob {
    pub(crate) work_frames: Vec<Vec<u8>>,
    pub(crate) vga_frames: Vec<Vec<u8>>,
    pub(crate) waits: Vec<i64>,
    pub(crate) palette: ColorPalette,
    pub(crate) frame: i64,
    pub(crate) wait: i64,
    pub(crate) drawn: bool,
    pub(crate) mutation_at: i64,
    pub(crate) unit_slot: i64,
    pub(crate) new_x: i64,
    pub(crate) new_y: i64,
    pub(crate) mutate: Option<Box<dyn FnMut(&mut Game) -> bool + Send>>,
    pub(crate) mutated: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) rollback: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd_source: Option<ModernHDUnitPresentSource>,
    pub(crate) hd_signature: [u8; 32],
    pub(crate) hd_frames: Vec<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) hd_cached: Vec<ModernHDMapFrame>,
    pub(crate) hd_cached_theme: Option<ModernStoryPortraitSet>,
    pub(crate) hd_cached_bridge: Option<ModernHDMapFrame>,
    /// Asset preflight follows the next leg without executing its callback.
    pub(crate) hd_next: Option<Box<NativeUnitPresentJob>>,
}
