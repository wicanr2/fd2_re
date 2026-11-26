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

/// Go：`native_ch23_post.go:93`，`func nativeCh23LoopSpec(loop campaign.NativeCh23Loop) fdother.NativeCh23LoopSpec`
pub(crate) fn native_ch23_loop_spec(lp: &campaign::NativeCh23Loop) -> fdother::NativeCh23LoopSpec {
    fdother::NativeCh23LoopSpec {
        phase: lp.phase.clone(),
        repeat: lp.repeat,
        stage_values: lp.stage_values.clone(),
        palette: lp.palette.is_some(),
    }
}

/// Go `snapshotNativeCh23LoopState` 回傳的復原閉包所保存的欄位。
#[derive(Clone)]
pub(crate) struct NativeCh23LoopSnapshot {
    st: Option<battle::State>,
    adapter: Option<NativeCh23AdapterState>,
    work: Vec<u8>,
    vga: Vec<u8>,
    dac: Vec<u8>,
    clock: NativeBIOSClock,
    palette_phase: i64,
    palette_tick: i64,
}

impl Game {
    /// Go：`native_ch23_post.go:101`，`func (g *Game) snapshotNativeCh23LoopState() func()`
    pub(crate) fn snapshot_native_ch23_loop_state(&self) -> NativeCh23LoopSnapshot {
        NativeCh23LoopSnapshot {
            st: self.st.clone(),
            adapter: self.native_ch23_state.clone(),
            work: self.native_map_work.clone(),
            vga: self.native_map_vga.clone(),
            dac: self.native_map_dac.clone(),
            clock: self.native_map_clock,
            palette_phase: self.native_fdother_palette_phase,
            palette_tick: self.native_fdother_palette_tick,
        }
    }

    fn restore_native_ch23_loop_state(&mut self, s: NativeCh23LoopSnapshot) {
        if self.st.is_some() {
            self.st = s.st;
        }
        self.native_ch23_state = s.adapter;
        (self.native_map_work, self.native_map_vga, self.native_map_dac) = (s.work, s.vga, s.dac);
        self.native_map_clock = s.clock;
        (self.native_fdother_palette_phase, self.native_fdother_palette_tick) = (s.palette_phase, s.palette_tick);
    }

    /// 0x10652 兩段 loop（initial／palette）：每次 draw 依 BIOS tick 轉動 #42 staging 列、palette 段另做
    /// DAC 減量，再走 11CAC 合成；失敗時整段復原。原版索引色分支，HD 另期。
    ///
    /// Go：`native_ch23_post.go:125`，`func (g *Game) startNativeCh23Loop(loop campaign.NativeCh23Loop, then func()) error`
    pub(crate) fn start_native_ch23_loop(
        &mut self,
        lp: &campaign::NativeCh23Loop,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<()> {
        if self.native_ch23_loop.is_some() || self.native_ch20_sky_key.is_some() || self.indexed_transition.is_some() {
            return Err(GameError::new("native ch23 loop is already active"));
        }
        let spec = native_ch23_loop_spec(lp);
        if !spec.is_recovered_contract() {
            return Err(GameError::new("native ch23 loop differs from recovered contract"));
        }
        if self.st.as_ref().is_none_or(|st| !st.has_native_map_view_state)
            || self.m.is_none()
            || !native_map_assets_available(self.native_map_assets.as_ref())
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || self.native_map_dac.len() != 256 * 3
        {
            return Err(GameError::new("native ch23 indexed map state is unavailable"));
        }
        let (current_tick, ok) = self.native_map_clock.current();
        if !ok {
            return Err(GameError::new("native ch23 BIOS tick provenance is unavailable"));
        }
        let rollback = self.snapshot_native_ch23_loop_state();
        if spec.phase == "initial" {
            if self.native_ch23_state.as_ref().is_some_and(|s| s.latch != 1 || s.initial_complete) {
                return Err(GameError::new("native ch23 initial loop state already exists"));
            }
            if self.native_ch23_state.is_none() {
                let frame = fdother::load_separated_native_ch23_stage(separated_asset_path("surfaces"))
                    .map_err(|err| game_err!("FDOTHER #42: {err}"))?;
                let mut staging =
                    vec![0u8; (fdother::NATIVE_CH23_STAGE_STRIDE * fdother::NATIVE_CH23_STAGE_HEIGHT) as usize];
                fdother::blit_native_ch23_stage(&frame, &mut staging)
                    .map_err(|err| game_err!("FDOTHER #42 staging: {err}"))?;
                self.native_ch23_state =
                    Some(NativeCh23AdapterState { staging, tick_snapshot: current_tick, ..Default::default() });
            }
        } else if self.native_ch23_state.as_ref().is_none_or(|s| !s.initial_complete) {
            return Err(GameError::new("native ch23 palette loop lacks completed initial state"));
        }
        let rollback: Box<dyn FnOnce(&mut Game) + Send> =
            Box::new(move |g: &mut Game| g.restore_native_ch23_loop_state(rollback));
        self.native_ch23_loop =
            Some(NativeCh23LoopJob { spec, then: Some(then), rollback: Some(rollback), ..Default::default() });
        let now = self.native_ch23_now();
        if let Err(err) = self.prepare_native_ch23_draw(now) {
            self.fail_native_ch23_loop(err.message());
            return Err(err);
        }
        Ok(())
    }

    /// 準備下一次 draw：候選狀態全部成功才發布（Go 以 `candidateGame := *g` 試做；Rust 先存快照、就地試做，失敗還原）。
    ///
    /// Go：`native_ch23_post.go:175`，`func (g *Game) prepareNativeCh23Draw(now time.Time) error`
    pub(crate) fn prepare_native_ch23_draw(&mut self, now: GameInstant) -> Result<()> {
        let valid = self.native_ch23_state.is_some()
            && self.native_ch23_loop.as_ref().is_some_and(|j| {
                j.stage >= 0 && (j.stage as usize) < j.spec.stage_values.len() && j.draw >= 0 && j.draw < j.spec.repeat
            });
        if !valid {
            return Err(GameError::new("native ch23 loop cursor is invalid"));
        }
        let (stage, draw, palette_on, raw_esi) = {
            let j = self.native_ch23_loop.as_ref().unwrap();
            (j.spec.stage_values[j.stage as usize], j.draw, j.spec.palette, j.raw_esi)
        };
        if draw == 0 {
            self.native_ch23_state.as_mut().unwrap().latch = stage;
        }
        let before = self.snapshot_native_ch23_loop_state();
        let result = (|| -> Result<ColorPalette> {
            let raw_tick = self.native_map_clock.sample(now);
            let adapter = self.native_ch23_state.as_mut().unwrap();
            if raw_tick != adapter.tick_snapshot {
                let latch = adapter.latch;
                adapter.rotate_rows(latch)?;
                adapter.tick_snapshot = raw_tick;
            }
            if palette_on {
                let baseline = self.native_map_assets.as_ref().unwrap().palette_dac.clone();
                fdother::apply_vga_palette_subtraction(&mut self.native_map_dac, &baseline, 0, 255, raw_esi)?;
            }
            let staging = self.native_ch23_state.as_ref().unwrap().staging.clone();
            indexedmap::seed_native_ch23_staging(&mut self.native_map_work, &staging)?;
            self.modern_hd_map_composing = false;
            self.compose_native_map_frame_at_with_palette_cycle(now, false, true)
                .map_err(|err| game_err!("indexed draw: {}", err.message()))?;
            Ok(color_palette_from_nrgba(&fdother::vga_palette_from_dac(&self.native_map_dac)?))
        })();
        match result {
            Ok(palette) => {
                let j = self.native_ch23_loop.as_mut().unwrap();
                (j.palette, j.wait_frames, j.drawn) = (palette, 3, false);
                Ok(())
            }
            Err(err) => {
                self.restore_native_ch23_loop_state(before);
                Err(err)
            }
        }
    }

    /// Go：`native_ch23_post.go:244`，`func (g *Game) failNativeCh23Loop(err error)`
    pub(crate) fn fail_native_ch23_loop(&mut self, err: &str) {
        if let Some(rollback) = self.native_ch23_loop.take().and_then(|mut j| j.rollback.take()) {
            rollback(self);
        }
        self.load_err = format!("native ch23 post: {err}");
    }

    /// Go：`native_ch23_post.go:252`，`func (g *Game) finishNativeCh23Loop()`
    pub(crate) fn finish_native_ch23_loop(&mut self) {
        let Some(mut j) = self.native_ch23_loop.take() else {
            return;
        };
        if j.spec.phase == "initial"
            && let Some(state) = self.native_ch23_state.as_mut()
        {
            state.initial_complete = true;
        }
        j.rollback = None;
        if let Some(then) = j.then.take() {
            then(self);
        }
    }

    /// Go：`native_ch23_post.go:268`，`func (g *Game) stepNativeCh23LoopAt(now time.Time)`
    pub(crate) fn step_native_ch23_loop_at(&mut self, now: GameInstant) {
        let Some(j) = self.native_ch23_loop.as_mut().filter(|j| j.drawn) else {
            return;
        };
        if j.wait_frames > 0 {
            j.wait_frames -= 1;
            if j.wait_frames > 0 {
                return;
            }
        }
        if j.spec.palette {
            j.raw_esi += 1;
        }
        j.draw += 1;
        if j.draw >= j.spec.repeat {
            (j.draw, j.stage) = (0, j.stage + 1);
        }
        if j.stage as usize >= j.spec.stage_values.len() {
            self.finish_native_ch23_loop();
            return;
        }
        if let Err(err) = self.prepare_native_ch23_draw(now) {
            self.fail_native_ch23_loop(err.message());
        }
    }

    /// Go：`native_ch23_post.go:296`，`func (g *Game) nativeCh23Now() time.Time`
    pub(crate) fn native_ch23_now(&mut self) -> GameInstant {
        if let Some(mut frozen) = self.native_map_frozen_now.take() {
            let now = frozen(self);
            if self.native_map_frozen_now.is_none() {
                self.native_map_frozen_now = Some(frozen);
            }
            return now;
        }
        self.game_now()
    }

    /// Go：`native_ch23_post.go:303`，`func (g *Game) stepNativeCh23Loop()`
    pub(crate) fn step_native_ch23_loop(&mut self) {
        let now = self.native_ch23_now();
        self.step_native_ch23_loop_at(now);
    }

    /// 原版索引色分支：以本次 draw 的調色盤畫目前 VGA 並承認呈現。
    ///
    /// Go：`native_ch23_post.go:307`，`func (g *Game) drawNativeCh23Loop(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_ch23_loop(&mut self, screen: Option<&mut Picture>) -> bool {
        let vga_ok = self.native_map_vga.len() as i64 == indexedmap::NATIVE_MAP_VGA_SIZE;
        let Some(j) = self.native_ch23_loop.as_mut().filter(|j| j.palette.len() == 256 && vga_ok) else {
            return false;
        };
        if let Some(screen) = screen {
            let img = indexed_frame(320, 200, &self.native_map_vga, j.palette.clone());
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            screen.draw_image(&Picture::from_indexed(&img), &options);
        }
        j.drawn = true;
        true
    }
}
