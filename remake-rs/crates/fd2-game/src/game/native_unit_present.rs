//! Go 來源：`remake/cmd/fd2/native_unit_present.go`（359 行）。
//!
//! 原版索引色分支已移植；HD 的 `freezeModernHDUnitPresentSource`／`prepareModernHDUnitPresent` 另期（R6）。

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

/// Go：`native_unit_present.go:48`，`func nativeUnitPresentAssetsAvailable(a *nativeMapAssets) bool`
pub(crate) fn native_unit_present_assets_available(a: Option<&NativeMapAssets>) -> bool {
    let Some(a) = a.filter(|a| native_map_assets_available(Some(a)) && a.fdother6.len() > 0x7c) else {
        return false;
    };
    a.fdother6[0x72..=0x7c]
        .iter()
        .all(|entry| entry.width > 0 && entry.height > 0 && entry.pixels.len() as i64 == entry.width * entry.height)
}

/// Go：`native_unit_present.go:61`，`func resolveNativeUnitPresentCall(spec campaign.NativeUnitPresent, slotCount int) (fdother.NativeUnitPresentCall, error)`
pub(crate) fn resolve_native_unit_present_call(
    spec: &campaign::NativeUnitPresent,
    slot_count: usize,
) -> Result<fdother::NativeUnitPresentCall> {
    if slot_count == 0 || (spec.last_runtime_slot && spec.slot != 0) {
        return Err(game_err!("native 0x22253 slot selector is invalid"));
    }
    let slot = if spec.last_runtime_slot { slot_count as i64 - 1 } else { spec.slot };
    Ok(fdother::plan_native_unit_present_call(slot, spec.new_x, spec.new_y, spec.visual_x, spec.visual_y)?)
}

/// Go：`native_unit_present.go:72`，`func cloneNativeUnitPresentState(source *battle.State) (*battle.State, error)`。
/// Rust 的單位是值，整份 `clone` 即可。
pub(crate) fn clone_native_unit_present_state(source: Option<&battle::State>) -> Result<battle::State> {
    match source {
        Some(source) if source.native_map_selector_cache.is_some() && source.native_map_selector_error.is_none() => {
            Ok(source.clone())
        }
        _ => Err(game_err!("native 0x22253 selector state is unavailable")),
    }
}

impl Game {
    /// Go：`native_unit_present.go:84`，`func (g *Game) startNativeUnitPresent(spec campaign.NativeUnitPresent, then func()) error`
    pub(crate) fn start_native_unit_present(
        &mut self,
        spec: &campaign::NativeUnitPresent,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        if self.native_unit_present.is_some()
            || self.native2189_a.is_some()
            || self.transition_reveal.is_some()
            || self.indexed_transition.is_some()
            || self.native_palette_ramp.is_some()
            || self.native_palette_pulse.is_some()
            || self.native_ch20_sky_key.is_some()
            || self.native_ch23_loop.is_some()
        {
            return Err(game_err!("native 0x22253 presentation is already active"));
        }
        let (work, vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        let mut job = self.build_native_unit_present_job(spec, self.st.as_ref(), &work, &vga, then)?;
        let units = self.st.as_ref().map_or(0, |st| st.units.len());
        let call = resolve_native_unit_present_call(spec, units)?;
        let slot = call.unit_slot as usize;
        let before_unit = self.st.as_ref().unwrap().units[slot].clone();
        let (new_x, new_y) = (i64::from(call.new_x), i64::from(call.new_y));
        job.mutate = Some(Box::new(move |g: &mut Game| {
            g.st.as_mut()
                .and_then(|st| st.units.get_mut(slot))
                .is_some_and(|unit| unit.set_native_map_coordinates_raw(new_x, new_y))
        }));
        job.rollback = Some(Box::new(move |g: &mut Game| {
            (g.native_map_work, g.native_map_vga) = (work, vga);
            if let Some(unit) = g.st.as_mut().and_then(|st| st.units.get_mut(slot)) {
                *unit = before_unit;
            }
        }));
        self.native_unit_present = Some(job);
        self.publish_native_unit_present_frame();
        Ok(())
    }

    /// 不發布畫面、不改正式狀態地預算一次 0x22253：11 張 intro、6 張收縮、逐列 bridge、其餘釋放格。
    /// 原版索引色分支，HD 另期。
    ///
    /// Go：`native_unit_present.go:121`，`func (g *Game) buildNativeUnitPresentJob(...)`
    pub(crate) fn build_native_unit_present_job(
        &self,
        spec: &campaign::NativeUnitPresent,
        source: Option<&battle::State>,
        initial_work: &[u8],
        initial_vga: &[u8],
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<NativeUnitPresentJob> {
        let unavailable = || game_err!("native 0x22253 indexed map state is unavailable");
        let source = source.ok_or_else(unavailable)?;
        let assets = self.native_map_assets.as_ref().ok_or_else(unavailable)?;
        if self.m.is_none()
            || !native_unit_present_assets_available(Some(assets))
            || !source.has_native_map_view_state
            || initial_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || initial_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
        {
            return Err(unavailable());
        }
        let call = resolve_native_unit_present_call(spec, source.units.len())?;
        let slot = call.unit_slot as usize;
        if slot >= source.units.len() || !source.units[slot].has_native_map_presentation {
            return Err(game_err!("native 0x22253 slot {slot} lacks raw map presentation"));
        }
        fdother::validate_native_unit_present_schedule(&fdother::native_unit_present_schedule())?;
        let bridge_lut = fdother::native_unit_present_bridge_lut(&assets.luts)?;
        let before = self
            .build_native_indexed_transition_input_for_state(Some(source))
            .map_err(|err| game_err!("native 0x22253 pre-mutation input: {err}"))?;
        let mut candidate = clone_native_unit_present_state(Some(source))?;
        let (visual_x, visual_y) = (i64::from(call.visual_x), i64::from(call.visual_y));
        if !candidate.units[slot].set_native_map_coordinates_raw(i64::from(call.new_x), i64::from(call.new_y)) {
            return Err(game_err!("native 0x22253 candidate coordinate write failed"));
        }
        let after = self
            .build_native_indexed_transition_input_for_state(Some(&candidate))
            .map_err(|err| game_err!("native 0x22253 post-mutation input: {err}"))?;
        let (before_input, after_input) = (before.input(assets), after.input(assets));

        let mut terrain = vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
        if assets.map_index == 23 {
            let stage =
                self.native_ch23_state.as_ref().ok_or_else(|| game_err!("native 0x22253 same-compose stage absent"))?;
            indexedmap::seed_native_ch23_staging(&mut terrain, &stage.staging)?;
        }
        indexedmap::compose_native_unit_present_terrain_snapshot(&mut terrain, &before_input)?;
        let mut job = NativeUnitPresentJob {
            palette: assets.palette.clone(),
            unit_slot: call.unit_slot,
            new_x: i64::from(call.new_x),
            new_y: i64::from(call.new_y),
            then,
            ..Default::default()
        };
        if self.native_map_dac.len() == 256 * 3
            && let Ok(current) = fdother::vga_palette_from_dac(&self.native_map_dac)
        {
            job.palette = color_palette_from_nrgba(&current);
        }
        fn append_frame(job: &mut NativeUnitPresentJob, work: &[u8], vga: &[u8], wait: i64) {
            job.work_frames.push(work.to_vec());
            job.vga_frames.push(vga.to_vec());
            job.waits.push(wait.max(1));
        }
        let mut work = vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
        let mut vga = initial_vga.to_vec();
        for index in 0x72..=0x7c {
            indexedmap::compose_native_unit_present_intro_frame(
                &mut work,
                &mut vga,
                &terrain,
                &before_input,
                &assets.fdother6[index],
                visual_x,
                visual_y,
            )
            .map_err(|err| game_err!("native 0x22253 intro {index:#x}: {err}"))?;
            append_frame(&mut job, &work, &vga, 3); // 原版一個 BIOS tick，以 60 Hz 近似
        }
        let mut lut_snapshot = vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
        indexedmap::compose_native_unit_present_lut_snapshot(
            &mut lut_snapshot,
            &terrain,
            &before_input,
            &assets.fdother6[0x7c],
            visual_x,
            visual_y,
        )?;
        let view = source.native_map_view_state;
        // 可見游標在這裡當視窗內格座標用（24*visible 的 LUT 幾何）；狀態本身可以合法出界，界線在消費端問。
        if !view.visible_cursor_in_viewport() {
            return Err(game_err!(
                "native 0x22253 可見游標 ({},{}) 不在 13×8 視窗內，無法定位 LUT 幾何",
                view.visible_cursor_x,
                view.visible_cursor_y
            ));
        }
        let lut_frames = fdother::native_unit_present_lut_frames(view.visible_cursor_x, view.visible_cursor_y)?;
        if lut_frames.len() < 6 {
            return Err(game_err!("native 0x22253 LUT schedule too short"));
        }
        for (index, frame) in lut_frames[..6].iter().enumerate() {
            let lut = &assets.luts[frame.lut_index as usize];
            indexedmap::compose_native_unit_present_lut_frame(
                &mut work,
                &mut vga,
                &lut_snapshot,
                lut,
                &before_input,
                frame,
            )
            .map_err(|err| game_err!("native 0x22253 contract {index}: {err}"))?;
            append_frame(&mut job, &work, &vga, native_delay_ticks(frame.delay_ms) + frame.delay_ticks * 3);
        }
        job.mutation_at = job.vga_frames.len() as i64;
        let mut bridge_rows: Vec<Vec<u8>> = Vec::new();
        // Go 的列延遲回呼逐列抄 work／vga。Rust 的 RowDelay 只交 vga；bridge 在第一列輸出前 work 已完成重映射
        // （`compose_native_unit_present_strip_bridge`），所以每列沿用最終 work，與 Go 相同。
        let mut capture = |_row: i64, vga: &[u8]| -> indexedmap::Result<()> {
            bridge_rows.push(vga.to_vec());
            Ok(())
        };
        indexedmap::compose_native_unit_present_strip_bridge(
            &mut work,
            &mut vga,
            &lut_snapshot,
            &bridge_lut,
            &after_input,
            visual_x,
            visual_y,
            view.visible_cursor_x,
            view.visible_cursor_y,
            Some(&mut capture),
        )?;
        for row in bridge_rows {
            append_frame(&mut job, &work, &row, native_delay_ticks(10));
        }
        for (index, frame) in lut_frames[6..].iter().enumerate() {
            let lut = &assets.luts[frame.lut_index as usize];
            indexedmap::compose_native_unit_present_lut_frame(
                &mut work,
                &mut vga,
                &lut_snapshot,
                lut,
                &after_input,
                frame,
            )
            .map_err(|err| game_err!("native 0x22253 release {index}: {err}"))?;
            append_frame(&mut job, &work, &vga, frame.delay_ticks * 3);
        }
        if job.work_frames.len() != job.vga_frames.len()
            || job.waits.len() != job.vga_frames.len()
            || job.mutation_at != 17
            || job.vga_frames.len() < 45
        {
            return Err(game_err!("native 0x22253 preflight produced an invalid schedule"));
        }
        Ok(job)
    }

    /// Go：`native_unit_present.go:277`，`func (g *Game) publishNativeUnitPresentFrame()`
    pub(crate) fn publish_native_unit_present_frame(&mut self) {
        let Some(j) = self.native_unit_present.as_mut() else { return };
        let Some(index) = usize::try_from(j.frame).ok().filter(|&index| index < j.vga_frames.len()) else {
            return;
        };
        self.native_map_work.clone_from(&j.work_frames[index]);
        self.native_map_vga.clone_from(&j.vga_frames[index]);
        j.drawn = false;
        j.wait = (j.waits[index] - 1).max(0);
    }

    /// Go：`native_unit_present.go:291`，`func (g *Game) stepNativeUnitPresent()`
    pub(crate) fn step_native_unit_present(&mut self) {
        let Some(j) = self.native_unit_present.as_mut().filter(|j| j.drawn) else { return };
        if j.wait > 0 {
            j.wait -= 1;
            return;
        }
        let next = j.frame + 1;
        if next >= j.vga_frames.len() as i64 {
            let mut j = self.native_unit_present.take().unwrap();
            j.rollback = None;
            if let Some(then) = j.then.take() {
                then(self);
            }
            return;
        }
        if next == j.mutation_at && !j.mutated {
            let (slot, x, y) = (j.unit_slot, j.new_x, j.new_y);
            let mutate = j.mutate.take();
            let mutated = match mutate {
                Some(mut mutate) => {
                    let result = mutate(self);
                    if let Some(j) = self.native_unit_present.as_mut() {
                        j.mutate = Some(mutate);
                    }
                    result
                }
                None => self
                    .st
                    .as_mut()
                    .and_then(|st| usize::try_from(slot).ok().and_then(|slot| st.units.get_mut(slot)))
                    .is_some_and(|unit| unit.set_native_map_coordinates_raw(x, y)),
            };
            if !mutated {
                self.fail_native_unit_present(&game_err!("coordinate mutation boundary unavailable slot={slot}"));
                return;
            }
            self.native_unit_present.as_mut().unwrap().mutated = true;
        }
        self.native_unit_present.as_mut().unwrap().frame = next;
        self.publish_native_unit_present_frame();
    }

    /// Go：`native_unit_present.go:332`，`func (g *Game) failNativeUnitPresent(err error)`
    pub(crate) fn fail_native_unit_present(&mut self, err: &GameError) {
        if let Some(rollback) = self.native_unit_present.as_mut().and_then(|j| j.rollback.take()) {
            rollback(self);
        }
        self.native_unit_present = None;
        self.load_err = format!("native 0x22253: {err}");
    }

    /// 原版索引色分支；`screen` 為 `None` 時只標記已呈現（無畫面重播）。
    ///
    /// Go：`native_unit_present.go:340`，`func (g *Game) drawNativeUnitPresent(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_unit_present(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(j) = self.native_unit_present.as_mut() else { return false };
        if j.palette.len() != 256 || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE {
            return false;
        }
        if let Some(screen) = screen {
            let image = Picture::from_indexed(&IndexedFrame {
                width: 320,
                height: 200,
                pix: self.native_map_vga.clone(),
                palette: j.palette.clone(),
            });
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            screen.draw_image(&image, &options);
        }
        j.drawn = true;
        true
    }
}
