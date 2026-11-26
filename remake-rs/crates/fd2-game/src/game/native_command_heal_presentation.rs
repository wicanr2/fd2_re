//! Go 來源：`remake/cmd/fd2/native_command_heal_presentation.go`（509 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command_heal_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommandHealPresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND_HEAL_FRAMES: NativeCommandHealPresentationPhase = NativeCommandHealPresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_MID_HOLD: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_FRONT_HOLD: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_EFFECT_FRAMES: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(3);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_MASK_FRAMES: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(4);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_DIGIT_FRAMES: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(5);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_DIGIT_HOLD: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(6);

/// Go：`native_command_heal_presentation.go:27`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommandHealPresentationJob {
    /// Original wrapper identity, also used by read-only input audit.
    pub(crate) command_id: i64,
    pub(crate) front_schedule: fdother::NativeCommandHealPresentationSchedule,
    pub(crate) tail_schedule: fdother::NativeCommandHealTailSchedule,
    pub(crate) front_frames: Vec<Vec<u8>>,
    pub(crate) effect_frames: Vec<Vec<u8>>,
    pub(crate) mask_frame: Vec<u8>,
    pub(crate) digit_frames: Vec<Vec<u8>>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) post_vga: Vec<u8>,
    pub(crate) post_work: Vec<u8>,
    pub(crate) digits: Vec<fdother::LMI1Entry>,
    pub(crate) targets: Vec<indexedmap::NativeCommandHealTailTarget>,
    pub(crate) palette: ColorPalette,
    pub(crate) frame: i64,
    pub(crate) phase: NativeCommandHealPresentationPhase,
    pub(crate) hold: i64,
    pub(crate) drawn: bool,
    pub(crate) transaction: Option<Box<dyn FnMut(&mut Game) -> Result<Vec<battle::NativeCommandHealResult>> + Send>>,
    pub(crate) results: Vec<battle::NativeCommandHealResult>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandHealResult>) + Send>>,
    pub(crate) map_stage: Option<NativeCh23AdapterState>,
    pub(crate) hd_before: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_signature: [u8; 32],
    pub(crate) hd_input: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_baseline: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) hd_passes: Vec<fdother::IndexedTransitionPass>,
    pub(crate) hd_luts: Vec<Vec<u8>>,
    pub(crate) effect_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) digit_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) effect_targets: Vec<indexedmap::NativeCommandHealTailTarget>,
    pub(crate) hd_frames: Option<ModernHDHealFrames>,
    pub(crate) hd_cached: Option<ModernHDHealFrames>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) hd_cached_theme: Option<ModernStoryPortraitSet>,
}

impl Game {
    pub(crate) fn execute_native_ai_command_heal(&mut self, action: &battle::AIPlan) -> Result<()> {
        if !self.current_native_system_options().full_presentation_enabled() || self.native_heal_presentation.is_some()
        {
            return Err(game_err!("原版回復指令演出不可用"));
        }
        let state = self.st.as_ref().ok_or_else(|| game_err!("回復指令缺少戰場"))?;
        let actor =
            action.u.filter(|&index| index < state.units.len()).ok_or_else(|| game_err!("回復指令缺少施放者"))?;
        let target =
            action.target.filter(|&index| index < state.units.len()).ok_or_else(|| game_err!("回復指令缺少目標"))?;
        let (x, y) = (state.units[actor].x, state.units[actor].y);
        let origin = action.native_action_destination;
        self.ai_focus_cursor(x, y);
        self.ai_focus_cursor(origin.x, origin.y);

        let id = action.native_command_id;
        let front = fdother::build_native_command_heal_presentation_schedule(id).map_err(|err| game_err!("{err}"))?;
        let tail = fdother::build_native_command_heal_tail_schedule(id).map_err(|err| game_err!("{err}"))?;
        let (hud, hud_ready) = self.native_map_hud_input();
        if !hud_ready {
            return Err(game_err!("回復指令缺少戰場資訊欄"));
        }
        let state = self.st.as_ref().unwrap();
        let assets = self
            .native_map_assets
            .as_ref()
            .filter(|a| native_map_assets_available(Some(a)))
            .ok_or_else(|| game_err!("回復指令缺少地圖素材"))?;
        let view = state.native_map_view_state;
        if !state.has_native_map_view_state
            || !view.visible_cursor_in_viewport()
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
        {
            return Err(game_err!("回復指令缺少有效的原版地圖畫面"));
        }
        let mut post = state.clone();
        let (results, rng) =
            post.execute_native_ai_command_heal_native(Some(actor), origin, id, self.native_rng_state)?;
        let mut targets = Vec::with_capacity(results.len());
        for result in &results {
            let index = result.target.ok_or_else(|| game_err!("回復指令缺少原生目標記錄"))?;
            let unit = &state.units[index];
            if !unit.has_native_map_presentation || !unit.has_map_selector_slot {
                return Err(game_err!("回復指令目標 {index} 缺少原生圖示"));
            }
            targets.push(indexedmap::NativeCommandHealTailTarget {
                record_index: index as i64,
                x: i64::from(unit.native_map_presentation.x),
                y: i64::from(unit.native_map_presentation.y),
                selector_slot: unit.map_selector_slot,
            });
        }
        let source = self.build_native_indexed_transition_input_for_state(Some(state))?;
        let source = source.input(assets);
        let (baseline_work, baseline_vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        let mut work = vec![0; baseline_work.len()];
        let stage = self.native_ch23_state.as_ref().filter(|_| assets.map_index == 23);
        if assets.map_index == 23 && stage.is_none() {
            return Err(game_err!("回復指令缺少第 23 號地圖底層"));
        }
        let mut front_frames = Vec::new();
        for spec in &front.frames {
            let lut = fdother::native_indexed_transition_lut(&assets.luts, spec.lut_index)?;
            let pass = fdother::build_native_indexed_transition_pass(
                24 * view.visible_cursor_x + 12,
                24 * view.visible_cursor_y + 16,
                spec.radius,
                0,
                fdother::NATIVE_TRANSITION_STAGE_HEIGHT,
            )?;
            work.fill(0);
            if let Some(stage) = stage {
                indexedmap::seed_native_ch23_staging(&mut work, &stage.staging)?;
            }
            let mut pixels = vec![0; baseline_vga.len()];
            indexedmap::compose_native_transition_frame(&mut work, &mut pixels, &source, pass, &lut)?;
            front_frames.push(pixels);
        }
        let effect_targets = targets
            .iter()
            .copied()
            .filter(|target| {
                (view.camera_x - 1..=view.camera_x + 12).contains(&target.x)
                    && (view.camera_y - 1..=view.camera_y + 8).contains(&target.y)
            })
            .collect::<Vec<_>>();
        let mut effect_frames = Vec::new();
        for index in tail.effect_start..tail.effect_start + tail.effect_frames {
            let effect = assets.fdother6.get(index as usize).ok_or_else(|| game_err!("缺少回復指令畫格 {index}"))?;
            work.copy_from_slice(&baseline_work);
            let mut pixels = baseline_vga.clone();
            indexedmap::compose_native_command_heal_effect_frame(
                &mut work,
                &mut pixels,
                &baseline_work,
                effect,
                &effect_targets,
                view.camera_x,
                view.camera_y,
            )?;
            effect_frames.push(pixels);
        }
        work.copy_from_slice(&baseline_work);
        let mut mask_frame = baseline_vga.clone();
        indexedmap::compose_native_command_heal_mask_frame(
            &mut work,
            &mut mask_frame,
            &baseline_work,
            assets.units.as_ref(),
            state.native_map_selector_cache.as_ref(),
            &effect_targets,
            view.camera_x,
            view.camera_y,
            state.native_map_frame_roster()?.cycles.idle,
            tail.mask_index as u8,
        )?;
        let mut post_input = build_native_map_frame_input(
            Some(assets),
            self.m.as_ref(),
            Some(&post),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )?;
        let (mut post_work, mut post_vga) = (baseline_work.clone(), baseline_vga.clone());
        if let Some(stage) = stage {
            indexedmap::seed_native_ch23_staging(&mut post_work, &stage.staging)?;
        }
        indexedmap::compose_native_frame(&mut post_work, &mut post_vga, &mut post_input)?;
        let mut queue = Vec::new();
        for (target, result) in targets.iter().zip(&results) {
            let visible = (view.camera_x..view.camera_x + 12).contains(&target.x)
                && (view.camera_y - 1..=view.camera_y + 7).contains(&target.y);
            queue = battle::append_native_presentation_digits(
                &queue,
                result.restore.actual,
                tail.digit_bias,
                target.record_index,
                visible,
            )?;
        }
        let mut digit_frames = Vec::new();
        for frame in 0..tail.digit_frames {
            work.copy_from_slice(&post_work);
            let mut pixels = post_vga.clone();
            indexedmap::compose_native_command_heal_digit_frame(
                &mut work,
                &mut pixels,
                &post_work,
                &assets.command_heal_digits,
                &queue,
                &targets,
                view.camera_x,
                view.camera_y,
                tail.digit_vertical,
                frame,
            )?;
            digit_frames.push(pixels);
        }
        let palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&self.native_map_dac)?);
        let sample = front.sample_index;
        let units = post.units;
        let mut ready = Some((units, results));
        let transaction = Box::new(move |g: &mut Game| {
            let state = g.st.as_mut().ok_or_else(|| game_err!("回復指令結算時缺少戰場"))?;
            let (units, results) = ready.take().ok_or_else(|| game_err!("回復指令已結算"))?;
            if state.units.len() != units.len() {
                return Err(game_err!("回復指令結算時戰場名冊已改變"));
            }
            state.units = units;
            g.native_rng_state = rng;
            Ok(results)
        });
        let then = Box::new(move |g: &mut Game, results: Vec<battle::NativeCommandHealResult>| {
            let state = g.st.as_mut().expect("回復指令結算保留戰場");
            let (a, t) = (&state.units[actor], &state.units[target]);
            let pose = dir_toward(a.x, a.y, t.x, t.y);
            state.units[actor].set_map_pose(pose);
            let total: i64 = results.iter().map(|result| result.restore.actual).sum();
            g.msg = format!("原始指令 {id}：回復 {total}");
            let after = g.native_ai_action_after.take();
            g.finish_successful_unit_action(Some(actor), after);
            g.check_result();
        });
        self.native_heal_presentation = Some(NativeCommandHealPresentationJob {
            command_id: id,
            front_schedule: front,
            tail_schedule: tail,
            front_frames,
            effect_frames,
            mask_frame,
            digit_frames,
            baseline_vga,
            baseline_work,
            post_vga,
            post_work,
            palette,
            transaction: Some(transaction),
            then: Some(then),
            ..Default::default()
        });
        self.st.as_mut().unwrap().native_experience_accumulator = 0;
        self.play_sfx(sample);
        Ok(())
    }

    pub(crate) fn present_native_command_heal(&mut self, screen: Option<&mut Picture>) {
        let job = self.native_heal_presentation.as_mut().unwrap();
        let index = job.frame as usize;
        let pixels = match job.phase {
            NATIVE_COMMAND_HEAL_FRAMES | NATIVE_COMMAND_HEAL_MID_HOLD => job.front_frames.get(index),
            NATIVE_COMMAND_HEAL_EFFECT_FRAMES => job.effect_frames.get(index),
            NATIVE_COMMAND_HEAL_MASK_FRAMES if job.frame % 2 == 1 => Some(&job.mask_frame),
            NATIVE_COMMAND_HEAL_DIGIT_FRAMES => job.digit_frames.get(index),
            _ => Some(&job.baseline_vga),
        };
        let Some(pixels) =
            pixels.filter(|p| p.len() as i64 == indexedmap::NATIVE_MAP_VGA_SIZE && job.palette.len() == 256)
        else {
            self.native_heal_presentation = None;
            self.load_err = "回復指令演出畫面無效".into();
            return;
        };
        if let Some(screen) = screen {
            let image = Picture::from_indexed(&IndexedFrame {
                width: 320,
                height: 200,
                pix: pixels.clone(),
                palette: job.palette.clone(),
            });
            let mut draw = DrawImageOptions::default();
            draw.geo_m.scale(2.0, 2.0);
            screen.draw_image(&image, &draw);
        }
        job.drawn = true;
    }

    /// Go：`native_command_heal_presentation.go:341`，`func (g *Game) stepNativeCommandHealPresentation()`
    pub(crate) fn step_native_command_heal_presentation(&mut self) {
        let Some(j) = self.native_heal_presentation.as_mut() else {
            return;
        };
        match j.phase {
            NATIVE_COMMAND_HEAL_FRAMES => {
                if !j.drawn {
                    return;
                }
                if j.frame + 1 == j.front_schedule.mid_frame {
                    (j.phase, j.hold) =
                        (NATIVE_COMMAND_HEAL_MID_HOLD, native_delay_ticks(j.front_schedule.mid_delay_ms));
                    return;
                }
                if j.frame + 1 >= j.front_frames.len() as i64 {
                    (j.phase, j.hold, j.drawn) =
                        (NATIVE_COMMAND_HEAL_FRONT_HOLD, native_delay_ticks(j.front_schedule.tail_delay_ms), false);
                    return;
                }
                j.frame += 1;
                j.drawn = false;
            }
            NATIVE_COMMAND_HEAL_MID_HOLD => {
                if j.hold > 0 {
                    j.hold -= 1;
                    return;
                }
                (j.frame, j.phase, j.drawn) = (j.front_schedule.mid_frame, NATIVE_COMMAND_HEAL_FRAMES, false);
            }
            NATIVE_COMMAND_HEAL_FRONT_HOLD => {
                if j.hold > 0 {
                    j.hold -= 1;
                    return;
                }
                (j.frame, j.phase, j.drawn) = (0, NATIVE_COMMAND_HEAL_EFFECT_FRAMES, false);
                let sample = j.tail_schedule.effect_sample;
                self.play_sfx(sample);
            }
            NATIVE_COMMAND_HEAL_EFFECT_FRAMES => {
                if !j.drawn {
                    return;
                }
                if j.frame + 1 < j.effect_frames.len() as i64 {
                    j.frame += 1;
                    j.drawn = false;
                    return;
                }
                (j.frame, j.phase, j.drawn) = (0, NATIVE_COMMAND_HEAL_MASK_FRAMES, false);
                let sample = j.tail_schedule.mask_sample;
                self.play_sfx(sample);
            }
            NATIVE_COMMAND_HEAL_MASK_FRAMES => {
                if !j.drawn {
                    return;
                }
                if j.frame < j.tail_schedule.mask_pairs * 2 {
                    j.frame += 1;
                    j.drawn = false;
                    return;
                }
                let rollback = self.native_heal_rollback();
                let mut transaction =
                    self.native_heal_presentation.as_mut().expect("Go：j 仍是同一個工作").transaction.take();
                // Go `j.transaction()`：nil func 解參考 panic。
                let results = (transaction.as_mut().expect("Go：j.transaction 為 nil 時呼叫 panic"))(self);
                if let Some(j) = self.native_heal_presentation.as_mut() {
                    j.transaction = transaction;
                }
                let results = match results {
                    Ok(results) => results,
                    Err(err) => {
                        rollback(self);
                        self.load_err = format!("native command heal post-presentation transaction: {}", err.message());
                        self.native_heal_presentation = None;
                        return;
                    }
                };
                let job = self.native_heal_presentation.as_mut().unwrap();
                job.results = results;
                job.baseline_work = std::mem::take(&mut job.post_work);
                job.baseline_vga = std::mem::take(&mut job.post_vga);
                (job.frame, job.phase, job.drawn) = (0, NATIVE_COMMAND_HEAL_DIGIT_FRAMES, false);
            }
            NATIVE_COMMAND_HEAL_DIGIT_FRAMES => {
                if !j.drawn {
                    return;
                }
                if j.frame + 1 < j.digit_frames.len() as i64 {
                    j.frame += 1;
                    j.drawn = false;
                    return;
                }
                (j.phase, j.hold, j.drawn) =
                    (NATIVE_COMMAND_HEAL_DIGIT_HOLD, native_delay_ticks(j.tail_schedule.digit_hold_ms), false);
            }
            NATIVE_COMMAND_HEAL_DIGIT_HOLD => {
                if j.hold > 0 {
                    j.hold -= 1;
                    return;
                }
                let mut j = self.native_heal_presentation.take().expect("Go：j 仍是同一個工作");
                let (then, results) = (j.then.take(), std::mem::take(&mut j.results));
                self.native_map_vga = j.baseline_vga;
                self.native_map_work = j.baseline_work;
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Both production callbacks are the native player/AI heal transactions.
    /// They write unit scalar/array fields and the native RNG, then defer action
    /// cleanup to the continuation. Keep unit pointers stable on a failed redraw.
    ///
    /// Go：`modern_hd_heal_source.go:76`，`func (g *Game) nativeHealRollback() func()`（`r8`，隨呼叫端
    /// `stepNativeCommandHealPresentation` 補移植）。Go 以單位指標為鍵存值複本；Rust 依 `st.units` 的索引。
    pub(crate) fn native_heal_rollback(&self) -> Box<dyn FnOnce(&mut Game) + Send> {
        let units = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic").units.clone();
        let rng = self.native_rng_state;
        let (work, vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        Box::new(move |g: &mut Game| {
            let current = presentation_units(&mut g.st);
            for (index, before) in units.into_iter().enumerate() {
                if let Some(unit) = current.get_mut(index) {
                    *unit = before;
                }
            }
            g.native_rng_state = rng;
            (g.native_map_work, g.native_map_vga) = (work, vga);
        })
    }
}
