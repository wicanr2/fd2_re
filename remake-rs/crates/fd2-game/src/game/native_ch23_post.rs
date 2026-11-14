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
        if !self.native_map_assets.as_ref().is_some_and(|a| a.map_index == 23) || self.native_ch23_loop.is_some() {
            return Ok(None);
        }
        // Go `cloneNativeCh23AdapterState`（`r8`）：nil 回 nil，否則連 staging 一起複製；Rust 的 clone 相同。
        let state = match self.native_ch23_state.clone() {
            None => {
                let frame = fdother::load_separated_native_ch23_stage(separated_asset_path("surfaces"))
                    .map_err(|err| game_err!("FDOTHER #42: {err}"))?;
                let mut state = NativeCh23AdapterState {
                    staging: vec![
                        0u8;
                        (fdother::NATIVE_CH23_STAGE_STRIDE * fdother::NATIVE_CH23_STAGE_HEIGHT) as usize
                    ],
                    latch: 1,
                    tick_snapshot: raw_tick,
                    ..Default::default()
                };
                fdother::blit_native_ch23_stage(&frame, &mut state.staging)?;
                // 0x10842 的 loader 尾端先執行一次 sub_24D22(0)。
                let latch = state.latch;
                state.rotate_rows(latch)?;
                state
            }
            Some(mut state) => {
                if state.tick_snapshot != raw_tick {
                    let latch = state.latch;
                    state.rotate_rows(latch)?;
                    state.tick_snapshot = raw_tick;
                }
                state
            }
        };
        Ok(Some(state))
    }
}

impl NativeCh23AdapterState {
    /// R4 B 組隨呼叫端補移植（原標 `r8`）：Go `native_ch23_post.go:44`，`func (s *nativeCh23AdapterState) rotateRows(latch int) error`。
    pub(crate) fn rotate_rows(&mut self, latch: i64) -> Result<()> {
        fdother::rotate_native_ch23_rows(&mut self.staging, latch)?;
        self.row_shift = (self.row_shift + latch) % 192;
        Ok(())
    }
}
