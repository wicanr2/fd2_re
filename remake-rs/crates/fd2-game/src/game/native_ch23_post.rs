//! Go 來源：`remake/cmd/fd2/native_ch23_post.go`（327 行）。
//!
//! 骨架：Go 檔的 13 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeCh23AdapterState is the caller-owned raw state shared by the two
/// separated native_ch23_loop beats around FDTXT_024 index 3. Names remain
/// address-shaped because the original evidence only proves staging, latch and
/// timer contracts, not a higher-level transition meaning.
///
/// Go：`native_ch23_post.go:20`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCh23AdapterState {
    pub(crate) staging: Vec<u8>,
    pub(crate) row_shift: i64,
    pub(crate) latch: i64,
    pub(crate) tick_snapshot: i64,
    pub(crate) initial_complete: bool,
}

/// Go：`native_ch23_post.go:28`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCh23LoopJob {
    pub(crate) hd_source: Option<ModernHDCh23Source>,
    pub(crate) hd_frame: Option<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) spec: fdother::NativeCh23LoopSpec,
    pub(crate) stage: i64,
    pub(crate) draw: i64,
    pub(crate) raw_esi: i64,
    pub(crate) wait_frames: i64,
    pub(crate) drawn: bool,
    pub(crate) palette: ColorPalette,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) rollback: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl Game {
    /// #144：0x10652 的 #42 owner 也由正常 0x11EEE case23 消費。
    /// 契約見 fd2_ch24_stage_runtime_20261003.json；此處只準備候選，呼叫端提交。
    ///
    /// Go：`native_ch23_post.go:63`，`func (g *Game) prepareNativeCh23SteadyStage(rawTick int) (*nativeCh23AdapterState, error)`
    pub(crate) fn prepare_native_ch23_steady_stage(&self, raw_tick: i64) -> Result<Option<NativeCh23AdapterState>> {
        todo!("native_ch23_post.go:prepareNativeCh23SteadyStage")
    }
}
