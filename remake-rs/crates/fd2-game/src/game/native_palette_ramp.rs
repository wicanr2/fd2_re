//! Go 來源：`remake/cmd/fd2/native_palette_ramp.go`（281 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 8 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativePaletteRampJob owns the exact whole-DAC 0x11d40 writes used by
/// 0x1f882 and 0x1f525. Each step must be presented before it advances; this
/// intentionally stretches sub-frame DOS waits on a 60Hz host rather than
/// collapsing the complete effect into one unobservable update.
///
/// Go：`native_palette_ramp.go:20`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativePaletteRampJob {
    pub(crate) hd_battle_sources: Option<ModernHDBattleSources>,
    pub(crate) deltas: Vec<i64>,
    pub(crate) delay_ms: i64,
    pub(crate) step: i64,
    pub(crate) vga: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// Go：`native_palette_ramp.go:33`，`func buildInclusivePaletteDeltas(start, end int) ([]int, error)`
pub(crate) fn build_inclusive_palette_deltas(start: i64, end: i64) -> Result<Vec<i64>> {
    if !(0..=255).contains(&start) || !(0..=255).contains(&end) {
        return Err(GameError::new("palette delta endpoint outside byte range"));
    }
    let step = if start > end { -1 } else { 1 };
    let mut deltas = Vec::with_capacity((abs_int(start - end) + 1) as usize);
    let mut value = start;
    loop {
        deltas.push(value);
        if value == end {
            break;
        }
        value += step;
    }
    Ok(deltas)
}

impl Game {
    /// Go：`native_palette_ramp.go:51`，`func (g *Game) startNativePaletteRamp(start, end, delayMs int, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_palette_ramp(
        &mut self,
        start: i64,
        end: i64,
        delay_ms: i64,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        if self.native_palette_ramp.is_some() {
            return Err(GameError::new("native palette ramp already active"));
        }
        if delay_ms != 2 || !native_map_assets_available(self.native_map_assets.as_ref()) {
            return Err(GameError::new("native indexed framebuffer or DAC baseline unavailable"));
        }
        self.ensure_native_palette_frame()?;
        let vga = self.native_map_vga.clone();
        self.start_native_palette_ramp_on_frame(&vga, start, end, delay_ms, then)
    }

    /// startNativePaletteRampOnFrame accepts the caller's current indexed VGA.
    /// Normal physical returns fade their last body present, before the map VGA
    /// is cleared. Contract: physical_owner_return_dac_spec in the canonical evidence.
    ///
    /// Go：`native_palette_ramp.go:67`，`func (g *Game) startNativePaletteRampOnFrame(vga []byte, start, end, delayMs int, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_palette_ramp_on_frame(
        &mut self,
        vga: &[u8],
        start: i64,
        end: i64,
        delay_ms: i64,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        if self.native_palette_ramp.is_some() || vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE {
            return Err(GameError::new("native palette ramp owner or indexed frame unavailable"));
        }
        if delay_ms != 2 || !native_map_assets_available(self.native_map_assets.as_ref()) {
            return Err(GameError::new("native indexed framebuffer or DAC baseline unavailable"));
        }
        let deltas = build_inclusive_palette_deltas(start, end)?;
        let baseline = self.native_map_assets.as_ref().unwrap().palette_dac.clone();
        let mut dac = baseline.clone();
        if self.native_map_dac.len() == baseline.len() {
            dac.copy_from_slice(&self.native_map_dac);
        }
        let mut job =
            NativePaletteRampJob { deltas, delay_ms, vga: vga.to_vec(), dac, baseline, then, ..Default::default() };
        job.apply_current()?;
        self.native_palette_ramp = Some(job);
        Ok(())
    }

    /// ensureNativePaletteFrame uses the already published steady battle frame
    /// when available. A handler cutscene has no battle State/HUD; its acting
    /// redraw path is the separately recovered terrain → unit/foreground →
    /// 0x11eb0 compositor, represented by the three strict indexedmap operations
    /// below. No normalized PNG pixels are converted into fabricated palette
    /// indices.
    ///
    /// Go：`native_palette_ramp.go:101`，`func (g *Game) ensureNativePaletteFrame() error`
    pub(crate) fn ensure_native_palette_frame(&mut self) -> Result<()> {
        if self.native_map_vga.len() as i64 == indexedmap::NATIVE_MAP_VGA_SIZE {
            return Ok(());
        }
        if self.story_actors.is_empty() {
            return Err(GameError::new("native indexed framebuffer unavailable"));
        }
        let source =
            self.build_native_indexed_transition_input().map_err(|err| game_err!("native handler frame: {err}"))?;
        let a = self.native_map_assets.as_ref().expect("buildNativeIndexedTransitionInput 已檢查素材庫");
        let r#in = source.input(a);
        let mut work = vec![0u8; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
        let mut vga = vec![0u8; indexedmap::NATIVE_MAP_VGA_SIZE as usize];
        indexedmap::compose_native_unit_present_terrain_snapshot(&mut work, &r#in)
            .map_err(|err| game_err!("native handler terrain: {err}"))?;
        indexedmap::redraw_native_unit_present_objects(&mut work, &r#in)
            .map_err(|err| game_err!("native handler objects: {err}"))?;
        indexedmap::copy_native_unit_present_viewport(&mut vga, &work)
            .map_err(|err| game_err!("native handler viewport: {err}"))?;
        self.native_map_work = work;
        self.native_map_vga = vga;
        Ok(())
    }
}

impl NativePaletteRampJob {
    /// Go：`native_palette_ramp.go:128`，`func (j *nativePaletteRampJob) applyCurrent() error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn apply_current(&mut self) -> Result<()> {
        if self.step < 0 || self.step >= self.deltas.len() as i64 || self.delay_ms != 2 {
            return Err(GameError::new("native palette ramp step unavailable"));
        }
        fdother::apply_vga_palette_subtraction(&mut self.dac, &self.baseline, 0, 255, self.deltas[self.step as usize])?;
        let palette = fdother::vga_palette_from_dac(&self.dac)?;
        (self.palette, self.drawn) = (color_palette_from_nrgba(&palette), false);
        Ok(())
    }
}

impl Game {
    /// Go：`native_palette_ramp.go:143`，`func (g *Game) stepNativePaletteRamp()`
    ///
    /// R4 B 組：`drawn` 由呈現回報設定（Go 在 `drawNativePaletteRamp` 設），沒有回報就不前進。
    pub(crate) fn step_native_palette_ramp(&mut self) {
        let Some(j) = self.native_palette_ramp.as_mut().filter(|j| j.drawn) else {
            return;
        };
        j.step += 1;
        if j.step >= j.deltas.len() as i64 {
            let mut j = self.native_palette_ramp.take().unwrap();
            let then = j.then.take();
            self.native_map_dac.clear();
            self.native_map_dac.extend_from_slice(&j.dac);
            if let Some(then) = then {
                then(self);
            }
            return;
        }
        if let Err(err) = j.apply_current() {
            self.load_err = format!("native palette ramp: {err}");
            self.native_palette_ramp = None;
        }
    }

    /// Go：`native_palette_ramp.go:164`，`func (g *Game) drawNativePaletteRamp(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 `r5p`：照 Go 逐行移植，`j.drawn = true` 與 Go 一樣在這裡設（擷取畫格時走 `Game::draw`；
    /// 無畫面重播走 `acknowledge_present`，兩者不在同一幀呼叫）。HD 實體戰鬥回場的 `publishModernHDBattle`
    /// 是 R6；忠實模式下 `modern_hd_battle_wanted` 為 false，不會走到。
    pub(crate) fn draw_native_palette_ramp(&mut self, screen: &mut Picture) -> bool {
        let Some(j) = self.native_palette_ramp.as_ref() else {
            return false;
        };
        if j.vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE || j.palette.len() != 256 {
            return false;
        }
        // Go 的 `&&` 短路：`hdBattleSources` 為 nil 時不呼叫 `modernHDBattleWanted`。
        if j.hd_battle_sources.is_some()
            && self.atk.is_some()
            && self.modern_hd_battle_wanted(self.atk.as_ref().and_then(|a| a.native_scene.as_ref()))
        {
            todo!("r6:native_palette_ramp.go:169 publishModernHDBattle（HD 物理演出返回 DAC）")
        }
        let j = self.native_palette_ramp.as_mut().unwrap();
        let img = IndexedFrame { width: 320, height: 200, pix: j.vga.clone(), palette: j.palette.clone() };
        let mut op = DrawImageOptions::default();
        op.geo_m.scale(2.0, 2.0);
        screen.draw_image(&Picture::from_indexed(&img), &op);
        j.drawn = true;
        true
    }
}

/// Go：`native_palette_ramp.go:190`，`func nativeDACIsBlack(dac []byte) bool`
pub(crate) fn native_dacis_black(dac: &[u8]) -> bool {
    if dac.len() != 256 * 3 {
        return false;
    }
    dac.iter().all(|&c| c == 0)
}

impl Game {
    /// 戰場上寫 `g.st`；戰前處理器（例如 0x3332B 在 LOADCH 之後）寫同一份 runtime slot 的劇情演員陣列，
    /// 戰場接手時由 AdoptHandlerBattleState 原樣帶進去。先核對全部 sparse 寫入與視圖，再一次寫入，
    /// 不留半套。
    ///
    /// Go：`native_palette_ramp.go:202`，`func (g *Game) applyHandlerDirectRecordPatch(patch *campaign.HandlerDirectRecordPatch) error`
    pub(crate) fn apply_handler_direct_record_patch(
        &mut self,
        patch: &campaign::HandlerDirectRecordPatch,
    ) -> Result<()> {
        if self.handler_unit_count() == 0 || patch.units.is_empty() || (patch.view.is_some() && self.st.is_none()) {
            return Err(GameError::new("runtime battle records unavailable"));
        }
        let mut seen = BTreeSet::new();
        for entry in &patch.units {
            if entry.slot < 0 || entry.slot >= self.handler_unit_count() || seen.contains(&entry.slot) {
                return Err(GameError::new(format!("slot{} unavailable or duplicated", entry.slot)));
            }
            if entry.x.is_none() != entry.y.is_none()
                || (entry.x.is_none() && entry.pose.is_none() && entry.raw_bytes.is_empty())
            {
                return Err(GameError::new(format!("slot{} has an incomplete or empty sparse write", entry.slot)));
            }
            if let (Some(x), Some(y)) = (entry.x, entry.y)
                && (!(0..=0xff).contains(&x) || !(0..=0xff).contains(&y))
            {
                return Err(GameError::new(format!("slot{} coordinate outside byte range", entry.slot)));
            }
            if entry.pose.is_some_and(|pose| !(0..=3).contains(&pose)) {
                return Err(GameError::new(format!("slot{} pose outside native range", entry.slot)));
            }
            seen.insert(entry.slot);
            let Some(r) = self.handler_unit_at(entry.slot) else {
                return Err(GameError::new(format!("slot{} lacks native map-record provenance", entry.slot)));
            };
            let unit = self.handler_unit(r);
            if (entry.x.is_some() || entry.pose.is_some()) && !unit.has_native_map_presentation {
                return Err(GameError::new(format!("slot{} lacks native map-record provenance", entry.slot)));
            }
            let mut seen_offsets = BTreeSet::new();
            for raw in &entry.raw_bytes {
                if !(0..=0xff).contains(&raw.value) || seen_offsets.contains(&raw.offset) {
                    return Err(GameError::new(format!(
                        "slot{} raw offset {:#x} has invalid or duplicate byte value",
                        entry.slot, raw.offset
                    )));
                }
                seen_offsets.insert(raw.offset);
                if raw.offset == 5 && !unit.has_native_record_byte5 {
                    return Err(GameError::new(format!("slot{} lacks raw byte+5 provenance", entry.slot)));
                }
                if raw.offset != 5 && !(0x22..=0x27).contains(&raw.offset) {
                    return Err(GameError::new(format!("slot{} raw offset {:#x} unsupported", entry.slot, raw.offset)));
                }
            }
        }
        if let Some(view) = patch.view.as_ref() {
            let mut candidate = self.st.as_ref().expect("view 已確認 st").clone();
            if let Err(err) = candidate.materialize_native_map_view_state(battle_native_map_view(view)) {
                return Err(GameError::new(format!("native view: {err}")));
            }
        }
        for entry in &patch.units {
            let r = self.handler_unit_at(entry.slot).expect("前面已核對 slot");
            let unit = self.handler_unit_mut(r);
            if let (Some(x), Some(y)) = (entry.x, entry.y)
                && !unit.set_native_map_coordinates_raw(x, y)
            {
                return Err(GameError::new(format!("slot{} coordinate write failed", entry.slot)));
            }
            if let Some(pose) = entry.pose
                && !unit.set_map_pose(pose)
            {
                return Err(GameError::new(format!("slot{} pose write failed", entry.slot)));
            }
            for raw in &entry.raw_bytes {
                match raw.offset {
                    5 => unit.native_record_byte5 = raw.value as u8,
                    offset => unit.native_transient[(offset - 0x22) as usize] = raw.value as u8,
                }
            }
        }
        if let Some(view) = patch.view.as_ref() {
            let st = self.st.as_mut().expect("view 已確認 st");
            if let Err(err) = st.materialize_native_map_view_state(battle_native_map_view(view)) {
                return Err(GameError::new(format!("native view commit: {err}")));
            }
            self.sync_native_map_view();
        }
        Ok(())
    }
}

/// Go：`native_palette_ramp.go:275`，`func battleNativeMapView(view campaign.NativeMapViewConfig) battle.NativeMapViewState`
pub(crate) fn battle_native_map_view(view: &campaign::NativeMapViewConfig) -> battle::NativeMapViewState {
    battle::NativeMapViewState {
        camera_x: view.camera_x,
        camera_y: view.camera_y,
        cursor_x: view.cursor_x,
        cursor_y: view.cursor_y,
        visible_cursor_x: view.visible_cursor_x,
        visible_cursor_y: view.visible_cursor_y,
    }
}

#[cfg(test)]
mod native_palette_ramp_test;
