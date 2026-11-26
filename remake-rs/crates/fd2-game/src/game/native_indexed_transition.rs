//! Go 來源：`remake/cmd/fd2/native_indexed_transition.go`（361 行）。
//!
//! 骨架：Go 檔的 11 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! R4 B 組：3 個函式移植完；`stepNativeIndexedTransition` 呼叫的 `nativeDelayTicks`、
//! `applyNativeIndexedTransitionPalette` 與 `ensureNativePaletteFrame` 用到的 `buildNativeIndexedTransitionInput`
//! 原標 `r8`，隨呼叫端補移植。`indexedmap::NativeTransitionFrameInput` 借用呼叫端的素材與選擇器快取，
//! 不能存進 `Game` 或從函式回傳臨時載體的借用，所以本檔新增擁有資源的 [`NativeTransitionFrameSource`]，
//! 每次合成前以 [`NativeTransitionFrameSource::input`] 組出借用的輸入（120「fd2-game 保存資源本身」）。

use super::*;

/// Go：`native_indexed_transition.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeIndexedTransitionPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_TRANSITION_PASS: NativeIndexedTransitionPhase = NativeIndexedTransitionPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_TRANSITION_TAIL: NativeIndexedTransitionPhase = NativeIndexedTransitionPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_TRANSITION_PALETTE: NativeIndexedTransitionPhase = NativeIndexedTransitionPhase(2);

/// nativeIndexedTransitionJob owns one complete 0x24618 presentation. Indexed
/// pixels and the six-bit DAC remain separate, just as they do in mode 13h.
/// Draw acknowledges every native present so multiple 5/4ms operations cannot
/// collapse into an unobservable final frame on the host's 60Hz update clock.
///
/// Go：`native_indexed_transition.go:29`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
///
/// R4 B 組：`input` 由 `NativeTransitionFrameInput<'static>` 改成擁有資源的 [`NativeTransitionFrameSource`]。
#[derive(Default)]
pub(crate) struct NativeIndexedTransitionJob {
    pub(crate) input: NativeTransitionFrameSource,
    pub(crate) schedule: fdother::NativeIndexedTransitionSchedule,
    pub(crate) spec: campaign::HandlerIndexedTransition,
    pub(crate) work: Vec<u8>,
    pub(crate) vga: Vec<u8>,
    pub(crate) frame_vga: Vec<Vec<u8>>,
    pub(crate) frame_work: Vec<Vec<u8>>,
    pub(crate) dac: Vec<u8>,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) phase: NativeIndexedTransitionPhase,
    pub(crate) frame: i64,
    pub(crate) palette_step: i64,
    pub(crate) tail_ticks: i64,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd_source: Option<ModernHDIndexedTransitionSource>,
    pub(crate) hd_signature: [u8; 32],
    pub(crate) hd_frames: Vec<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) hd_cached: Vec<ModernHDMapFrame>,
    pub(crate) hd_cached_theme: Option<ModernStoryPortraitSet>,
}

/// R4 B 組新增：擁有資源的 0x24618 輸入（Go 直接存 `indexedmap.NativeTransitionFrameInput`）。
///
/// Go 的輸入含兩類指標：素材庫 `nativeMapAssets` 的 bank／parallax／controls／LUT，以及戰場（或
/// `buildNativeIndexedTransitionInputForActors` 的臨時載體）的選擇器快取。後者在這裡存複本；前者每次以
/// [`NativeTransitionFrameSource::input`] 從呼叫端交來的素材庫借用（Go 在同一個工作期間不換素材庫）。
#[derive(Debug, Clone, Default, PartialEq)]
pub(crate) struct NativeTransitionFrameSource {
    pub(crate) parallax_scroll_x: i64,
    pub(crate) parallax_scroll_y: i64,
    pub(crate) selector_cache: Option<fdicon::NativeSelectorCache>,
    pub(crate) cells: Vec<fdicon::NativeTerrainCell>,
    /// Go `TerrainLUT: a.LUTs[lutIndex]`：素材庫 LUT 表的索引。
    pub(crate) lut_index: usize,
    pub(crate) map_width: i64,
    pub(crate) camera_x: i64,
    pub(crate) camera_y: i64,
    pub(crate) flip: i64,
    pub(crate) terrain_cycle: i64,
    pub(crate) idle_cycle: i64,
    pub(crate) moving_cycle: i64,
    pub(crate) pixel_shift: i64,
    pub(crate) units: Vec<fdicon::NativeUnitLayerEntry>,
    pub(crate) foreground_units: Vec<fdicon::NativeForegroundLayerEntry>,
}

impl NativeTransitionFrameSource {
    /// 組出借用 `a` 與本身的 `indexedmap::NativeTransitionFrameInput`；欄位對應 Go
    /// `buildNativeIndexedTransitionInputForState` 的回傳值。`TerrainBank` 與 `ForegroundBank` 都是 `a.Terrain`。
    pub(crate) fn input<'a>(&'a self, a: &'a NativeMapAssets) -> indexedmap::NativeTransitionFrameInput<'a> {
        indexedmap::NativeTransitionFrameInput {
            parallax: a.parallax.as_ref(),
            parallax_scroll_x: self.parallax_scroll_x,
            parallax_scroll_y: self.parallax_scroll_y,
            terrain_bank: a.terrain.as_deref(),
            unit_bank: a.units.as_ref(),
            foreground_bank: a.terrain.as_deref(),
            selector_cache: self.selector_cache.as_ref(),
            cells: self.cells.clone(),
            controls: &a.controls,
            terrain_lut: &a.luts[self.lut_index],
            map_width: self.map_width,
            camera_x: self.camera_x,
            camera_y: self.camera_y,
            flip: self.flip,
            terrain_cycle: self.terrain_cycle,
            idle_cycle: self.idle_cycle,
            moving_cycle: self.moving_cycle,
            pixel_shift: self.pixel_shift,
            units: self.units.clone(),
            foreground_units: self.foreground_units.clone(),
        }
    }
}

/// R4 B 組隨呼叫端補移植（原標 `r8`）：Go `native_indexed_transition.go:55`，`func nativeDelayTicks(ms int) int`。
pub(crate) fn native_delay_ticks(ms: i64) -> i64 {
    if ms <= 0 {
        return 0;
    }
    (ms * 60 + 999) / 1000
}

/// Go：`native_indexed_transition.go:62`，`func validateNativeIndexedTransitionSpec(s campaign.HandlerIndexedTransition) error`
pub(crate) fn validate_native_indexed_transition_spec(s: &campaign::HandlerIndexedTransition) -> Result<()> {
    if s.frames != 9
        || s.frame_delay_ms != 5
        || s.tail_delay_ms != 500
        || s.start_y != 0
        || s.end_y != fdother::NATIVE_TRANSITION_STAGE_HEIGHT
        || s.clip_width != fdother::NATIVE_TRANSITION_STAGE_WIDTH
        || s.clip_height != fdother::NATIVE_TRANSITION_STAGE_HEIGHT
        || s.palette_range_start != 0
        || s.palette_range_end != 255
        || s.palette_delta_start != 0
        || s.palette_delta_end != 62
        || s.palette_delta_step != 2
        || s.palette_delay_ms != 4
    {
        return Err(GameError::new("native 0x24618 payload differs from recovered geometry/timing"));
    }
    Ok(())
}

impl NativeIndexedTransitionJob {
    /// Go：`native_indexed_transition.go:237`，`func (j *nativeIndexedTransitionJob) composePass(index int, luts [][]byte) error`
    pub(crate) fn compose_pass(&mut self, index: usize, assets: &NativeMapAssets) -> Result<()> {
        let Some(frame) = self.schedule.frames.get(index).copied() else {
            return Err(GameError::new("native 0x24618 pass outside schedule"));
        };
        let lut = fdother::native_indexed_transition_lut(&assets.luts, frame.lut_index)?;
        let pass = fdother::build_native_indexed_transition_pass(
            self.spec.tile_x,
            self.spec.tile_y,
            frame.radius,
            self.spec.start_y,
            self.spec.end_y,
        )?;
        indexedmap::compose_native_transition_frame(
            &mut self.work,
            &mut self.vga,
            &self.input.input(assets),
            pass,
            &lut,
        )?;
        self.frame_vga.push(self.vga.clone());
        self.frame_work.push(self.work.clone());
        Ok(())
    }
}

impl Game {
    /// 第一、二個參數是動態游標運算式，與靜態幾何分開；呼叫點屬於 `Beat.Source`，未證實的偏移失敗即關閉。
    ///
    /// Go：`native_indexed_transition.go:82`，`func (g *Game) resolveNativeIndexedTransitionSpec(spec campaign.HandlerIndexedTransition, source string) (campaign.HandlerIndexedTransition, error)`
    pub(crate) fn resolve_native_indexed_transition_spec(
        &self,
        spec: &campaign::HandlerIndexedTransition,
        source: &str,
    ) -> Result<campaign::HandlerIndexedTransition> {
        let mut spec = spec.clone();
        if spec.cursor_source.is_empty() {
            return Ok(spec);
        }
        if spec.cursor_source != "native_relative_cursor" {
            return Err(GameError::new("native 0x24618 cursor source is not proven"));
        }
        let (dx, dy) = (spec.cursor_x_offset, spec.cursor_y_offset);
        let valid_offset = matches!(
            (source, dx, dy),
            ("0x245ce", 0, 3)
                | ("0x336e5", 0, 5)
                | ("0x33ce2", 6, 5)
                // 第27章READY caller資料：fd2_ch27_flow_20261016.json。
                | ("0x33bb9", 0, 3)
                | ("0x33c09", 0, 0)
                | ("0x33c66", 2, 0)
                | ("0x252ee", 0, -1)
        );
        if !valid_offset {
            return Err(game_err!("native 0x24618 cursor offset is not proven for source {source}"));
        }
        let view = match self.st.as_ref() {
            Some(st) if st.has_native_map_view_state => st.native_map_view_state,
            // LOADCH scenes keep the six raw map-view globals outside battle.State.
            _ if self.has_story_native_map_view => self.story_native_map_view,
            _ => return Err(GameError::new("native 0x24618 relative cursor provenance unavailable")),
        };
        let (x, y) = (view.visible_cursor_x + dx, view.visible_cursor_y + dy);
        if !(0..fdother::NATIVE_TRANSITION_STAGE_WIDTH).contains(&x)
            || !(0..fdother::NATIVE_TRANSITION_STAGE_HEIGHT).contains(&y)
        {
            return Err(GameError::new("native 0x24618 relative cursor is outside indexed stage"));
        }
        (spec.tile_x, spec.tile_y) = (x, y);
        Ok(spec)
    }

    /// 0x24618 圓形轉場：九段 LUT 合成全部預先算好再啟動；HD 綁定與其他 Rust 演出相同另期。
    ///
    /// Go：`native_indexed_transition.go:184`，`func (g *Game) startNativeIndexedTransition(spec campaign.HandlerIndexedTransition, source string, then func()) error`
    pub(crate) fn start_native_indexed_transition(
        &mut self,
        spec: &campaign::HandlerIndexedTransition,
        source: &str,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<()> {
        if self.indexed_transition.is_some() {
            return Err(GameError::new("native 0x24618 transition already active"));
        }
        let spec = self.resolve_native_indexed_transition_spec(spec, source)?;
        validate_native_indexed_transition_spec(&spec)?;
        let schedule = fdother::build_native_indexed_transition_schedule(spec.radial_radius, spec.radial_radius_step)?;
        let input = self.build_native_indexed_transition_input()?;
        let assets = self.native_map_assets.as_ref().ok_or_else(|| GameError::new("native map assets unavailable"))?;
        let mut job = NativeIndexedTransitionJob {
            input,
            schedule,
            spec,
            work: vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize],
            vga: vec![0; indexedmap::NATIVE_MAP_VGA_SIZE as usize],
            dac: assets.palette_dac.clone(),
            baseline: assets.palette_dac.clone(),
            phase: NATIVE_TRANSITION_PASS,
            then: Some(then),
            ..Default::default()
        };
        for i in 0..job.schedule.frames.len() {
            job.compose_pass(i, assets).map_err(|err| game_err!("preflight pass {i}: {}", err.message()))?;
        }
        let palette = fdother::vga_palette_from_dac(&job.dac)?;
        (job.frame, job.palette, job.drawn) = (0, color_palette_from_nrgba(&palette), false);
        let first = job.frame_vga[0].clone();
        job.vga.copy_from_slice(&first);
        self.indexed_transition = Some(job);
        Ok(())
    }

    /// 原版索引色分支：以目前 DAC 調色盤畫 320×200 VGA 並承認呈現。
    ///
    /// Go：`native_indexed_transition.go:331`，`func (g *Game) drawNativeIndexedTransition(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_indexed_transition(&mut self, screen: &mut Picture) -> bool {
        let Some(j) = self.indexed_transition.as_mut() else {
            return false;
        };
        if j.vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE || j.palette.len() != 256 {
            return false;
        }
        let img = indexed_frame(320, 200, &j.vga, j.palette.clone());
        let mut options = DrawImageOptions::default();
        options.geo_m.scale(2.0, 2.0);
        screen.draw_image(&Picture::from_indexed(&img), &options);
        j.drawn = true;
        true
    }
}

impl Game {
    /// R4 B 組隨呼叫端補移植（原標 `r8`）：Go `native_indexed_transition.go:122`，
    /// `func (g *Game) buildNativeIndexedTransitionInput() (indexedmap.NativeTransitionFrameInput, error)`。
    pub(crate) fn build_native_indexed_transition_input(&self) -> Result<NativeTransitionFrameSource> {
        self.build_native_indexed_transition_input_for_actors(&self.story_actors)
    }

    /// Go：`native_indexed_transition.go:126`，`func (g *Game) buildNativeIndexedTransitionInputForActors(actorsSource []battle.Unit) (indexedmap.NativeTransitionFrameInput, error)`
    ///
    /// R4 B 組：回傳 [`NativeTransitionFrameSource`]（Go 的回傳值借用這裡新建的臨時載體的選擇器快取）；
    /// 接收者改 `&self`（只讀）。
    pub(crate) fn build_native_indexed_transition_input_for_actors(
        &self,
        actors_source: &[battle::Unit],
    ) -> Result<NativeTransitionFrameSource> {
        let Some(m) = self.m.as_ref() else {
            return Err(GameError::new("native map unavailable"));
        };
        let mut state = battle::State::default();
        (state.w, state.h) = (m.w, m.h);
        if let Err(err) = state.append_native_map_selector_batch(actors_source.to_vec()) {
            return Err(game_err!("selector roster: {err}"));
        }
        self.build_native_indexed_transition_input_for_state(Some(&state))
    }

    /// buildNativeIndexedTransitionInputForState preserves the battle-session
    /// selector cache and animation/timing globals.  Reconstructing those from a
    /// flat unit slice would reset a turn-four transition to the chapter-opening
    /// phase even when every unit record was otherwise correct.
    ///
    /// Go：`native_indexed_transition.go:147`，`func (g *Game) buildNativeIndexedTransitionInputForState(state *battle.State) (indexedmap.NativeTransitionFrameInput, error)`
    ///
    /// R4 B 組：回傳 [`NativeTransitionFrameSource`]；接收者改 `&self`、`state` 改 `Option<&State>`（都只讀；
    /// 呼叫端常傳 `self.st.as_ref()`，骨架的 `&mut self` 加 `Option<&mut State>` 無法同時借用）。
    pub(crate) fn build_native_indexed_transition_input_for_state(
        &self,
        state: Option<&battle::State>,
    ) -> Result<NativeTransitionFrameSource> {
        let (a, field) = (self.native_map_assets.as_ref(), self.m.as_ref());
        let unavailable = || GameError::new("native map assets/field unavailable");
        if !native_map_assets_available(a) {
            return Err(unavailable());
        }
        let (Some(a), Some(field), Some(state)) = (a, field, state) else {
            return Err(unavailable());
        };
        if field.tile_w <= 0
            || field.tile_h <= 0
            || field.w <= 0
            || field.h <= 0
            || field.tiles.len() as i64 != field.w * field.h
            || field.native_terrain_control != a.controls
            || state.w != field.w
            || state.h != field.h
        {
            return Err(unavailable());
        }
        if self.cam_x as i64 % field.tile_w != 0
            || self.cam_y as i64 % field.tile_h != 0
            || self.cam_x != (self.cam_x as i64) as f64
            || self.cam_y != (self.cam_y as i64) as f64
        {
            return Err(GameError::new("native camera is not tile-aligned"));
        }
        let cells = indexedmap::build_native_terrain_cells(&field.tiles, &field.native_tile_blit_modes)
            .map_err(|err| game_err!("terrain cells: {err}"))?;
        let roster = state.native_map_frame_roster().map_err(|err| game_err!("native roster: {err}"))?;
        let lut_index = fdother::native_terrain_lut_index(roster.terrain_phase);
        let lut_index = match lut_index {
            Ok(i) if i >= 0 && (i as usize) < a.luts.len() && a.luts[i as usize].len() == 256 => i as usize,
            _ => return Err(GameError::new("terrain phase LUT unavailable")),
        };
        Ok(NativeTransitionFrameSource {
            parallax_scroll_x: 0,
            parallax_scroll_y: 0,
            selector_cache: state.native_map_selector_cache.clone(),
            cells,
            lut_index,
            map_width: field.w,
            camera_x: self.cam_x as i64 / field.tile_w,
            camera_y: self.cam_y as i64 / field.tile_h,
            flip: roster.terrain_flip,
            terrain_cycle: roster.cycles.idle,
            idle_cycle: roster.cycles.idle,
            moving_cycle: roster.cycles.moving,
            pixel_shift: roster.unit_pixel_shift,
            units: roster.units,
            foreground_units: roster.foreground,
        })
    }

    /// Go：`native_indexed_transition.go:260`，`func (g *Game) stepNativeIndexedTransition()`
    ///
    /// R4 B 組：job 留在 `self.indexed_transition` 原地推進（`applyNativeIndexedTransitionPalette` 讀
    /// `g.indexedTransition`）；`drawn` 由呈現回報設定。
    pub(crate) fn step_native_indexed_transition(&mut self) {
        let Some(j) = self.indexed_transition.as_mut() else {
            return;
        };
        match j.phase {
            NATIVE_TRANSITION_PASS => {
                if !j.drawn {
                    return;
                }
                if j.frame + 1 < j.schedule.frames.len() as i64 {
                    j.frame += 1;
                    let frame = j.frame as usize;
                    j.vga.copy_from_slice(&j.frame_vga[frame]);
                    j.drawn = false;
                    return;
                }
                j.phase = NATIVE_TRANSITION_TAIL;
                j.tail_ticks = native_delay_ticks(j.schedule.tail_delay_ms);
            }
            NATIVE_TRANSITION_TAIL => {
                if j.tail_ticks > 0 {
                    j.tail_ticks -= 1;
                    return;
                }
                j.phase = NATIVE_TRANSITION_PALETTE;
                j.palette_step = 0;
                j.drawn = false;
                if let Err(err) = self.apply_native_indexed_transition_palette() {
                    self.load_err = format!("beat indexed_transition: {err}");
                    self.indexed_transition = None;
                }
            }
            NATIVE_TRANSITION_PALETTE => {
                if !j.drawn {
                    return;
                }
                j.palette_step += 1;
                if j.palette_step >= j.schedule.palette_deltas.len() as i64 {
                    let mut j = self.indexed_transition.take().unwrap();
                    let then = j.then.take();
                    self.native_map_work.clear();
                    self.native_map_work.extend_from_slice(&j.work);
                    self.native_map_vga.clear();
                    self.native_map_vga.extend_from_slice(&j.vga);
                    if let Some(then) = then {
                        then(self);
                    }
                    return;
                }
                if let Err(err) = self.apply_native_indexed_transition_palette() {
                    self.load_err = format!("beat indexed_transition: {err}");
                    self.indexed_transition = None;
                }
            }
            _ => {}
        }
    }

    /// R4 B 組隨呼叫端補移植（原標 `r8`）：Go `native_indexed_transition.go:312`，
    /// `func (g *Game) applyNativeIndexedTransitionPalette() error`。
    pub(crate) fn apply_native_indexed_transition_palette(&mut self) -> Result<()> {
        let Some(j) = self
            .indexed_transition
            .as_mut()
            .filter(|j| j.palette_step >= 0 && j.palette_step < j.schedule.palette_deltas.len() as i64)
        else {
            return Err(GameError::new("native 0x24618 palette step unavailable"));
        };
        fdother::apply_vga_palette_delta(
            &mut j.dac,
            &j.baseline,
            j.spec.palette_range_start,
            j.spec.palette_range_end,
            j.schedule.palette_deltas[j.palette_step as usize],
        )?;
        let palette = fdother::vga_palette_from_dac(&j.dac)?;
        (j.palette, j.drawn) = (color_palette_from_nrgba(&palette), false);
        Ok(())
    }
}
