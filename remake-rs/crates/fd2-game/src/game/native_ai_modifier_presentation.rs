//! Go 來源：`remake/cmd/fd2/native_ai_modifier_presentation.go`（604 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_ai_modifier_presentation.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeAICommandModifierPresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeAICommandModifierPlan>,
    pub(crate) plan2022: Option<battle::NativeAICommand2022Plan>,
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) publish_at: i64,
    pub(crate) end_at: i64,
    pub(crate) drawn: bool,
    pub(crate) holding: bool,
    pub(crate) published: bool,
    pub(crate) skip_hold: bool,
    pub(crate) hold: i64,
    pub(crate) rng_before: u16,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) stage_work: Vec<u8>,
    pub(crate) stage_vga: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, battle::NativeCommandModifierResult) + Send>>,
    pub(crate) then2022: Option<Box<dyn FnOnce(&mut Game, &battle::NativeAICommand2022Plan) + Send>>,
}

impl Game {
    pub(crate) fn execute_native_ai_command_modifier(&mut self, action: &battle::AIPlan) -> Result<()> {
        if !self.current_native_system_options().full_presentation_enabled()
            || self.native_ai_command_modifier.is_some()
        {
            return Err(game_err!("原版狀態變更演出不可用"));
        }
        let state = self.st.as_ref().ok_or_else(|| game_err!("狀態變更指令缺少戰場"))?;
        let actor = action.u.filter(|&i| i < state.units.len()).ok_or_else(|| game_err!("指令缺少施放者"))?;
        let target = action.target.filter(|&i| i < state.units.len()).ok_or_else(|| game_err!("指令缺少目標"))?;
        let origin = action.native_action_destination;
        let id = action.native_command_id;
        let plan = state.plan_native_ai_command_modifier(Some(actor), origin, id, self.native_rng_state)?;
        let (x, y) = (state.units[actor].x, state.units[actor].y);
        self.ai_focus_cursor(x, y);
        self.ai_focus_cursor(origin.x, origin.y);
        let (hud, hud_ready) = self.native_map_hud_input();
        let state = self.st.as_ref().unwrap();
        let assets = self
            .native_map_assets
            .as_ref()
            .filter(|a| native_map_assets_available(Some(a)))
            .ok_or_else(|| game_err!("狀態變更指令缺少地圖素材"))?;
        if !hud_ready
            || !state.has_native_map_view_state
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
        {
            return Err(game_err!("狀態變更指令缺少原版地圖畫面"));
        }
        let schedule = fdother::build_native_command34_tail_schedule()
            .map_err(|err| game_err!("{err}"))?
            .into_iter()
            .find(|schedule| schedule.command_id == id)
            .ok_or_else(|| game_err!("缺少指令 {id} 的原版演出排程"))?;
        let targets = plan
            .targets
            .iter()
            .map(|&index| {
                let unit = &state.units[index];
                if !unit.has_native_map_presentation || !unit.has_map_selector_slot {
                    return Err(game_err!("指令目標 {index} 缺少原生圖示"));
                }
                Ok(indexedmap::NativeCommandHealTailTarget {
                    record_index: index as i64,
                    x: i64::from(unit.native_map_presentation.x),
                    y: i64::from(unit.native_map_presentation.y),
                    selector_slot: unit.map_selector_slot,
                })
            })
            .collect::<Result<Vec<_>>>()?;
        let view = state.native_map_view_state;
        let visible = targets
            .iter()
            .copied()
            .filter(|target| {
                (view.camera_x - 1..=view.camera_x + 12).contains(&target.x)
                    && (view.camera_y - 1..=view.camera_y + 8).contains(&target.y)
            })
            .collect::<Vec<_>>();
        let palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&self.native_map_dac)?);
        let image = |pix| {
            Arc::new(Picture::from_indexed(&IndexedFrame { width: 320, height: 200, pix, palette: palette.clone() }))
        };
        let mut frames = Vec::new();
        let (baseline_work, baseline_vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        let mut work = baseline_work.clone();
        for index in schedule.effect_start..schedule.effect_start + schedule.effect_frames {
            let effect =
                assets.fdother6.get(index as usize).ok_or_else(|| game_err!("缺少指令 {id} 的畫格 {index}"))?;
            work.copy_from_slice(&baseline_work);
            let mut pixels = baseline_vga.clone();
            indexedmap::compose_native_command_heal_effect_frame(
                &mut work,
                &mut pixels,
                &baseline_work,
                effect,
                &visible,
                view.camera_x,
                view.camera_y,
            )?;
            frames.push(NativeCompoundPresentedFrame { image: Some(image(pixels)), delay: 1, ..Default::default() });
        }
        let sample = |selector| {
            self.separated_command_sfx.get(&80).and_then(|bank| bank.get(&selector)).cloned().unwrap_or_default()
        };
        let effect_sound = sample(schedule.effect_sample);
        let mask_sound = sample(1);
        if !os_mute_or_shot(Some(self)) && (effect_sound.is_empty() || mask_sound.is_empty()) {
            return Err(game_err!("指令 {id} 缺少原版音效"));
        }
        for index in std::iter::once(0).chain(schedule.extra_sample_frame_indices) {
            frames.get_mut(index as usize).ok_or_else(|| game_err!("指令 {id} 的音效畫格無效"))?.sound =
                effect_sound.clone();
        }
        work.copy_from_slice(&baseline_work);
        let mut mask = baseline_vga.clone();
        indexedmap::compose_native_command_heal_mask_frame(
            &mut work,
            &mut mask,
            &baseline_work,
            assets.units.as_ref(),
            state.native_map_selector_cache.as_ref(),
            &visible,
            view.camera_x,
            view.camera_y,
            state.native_map_frame_roster()?.cycles.idle,
            schedule.mask_index as u8,
        )?;
        let masks = [image(baseline_vga.clone()), image(mask)];
        for index in 0..schedule.mask_pairs * 2 + 1 {
            frames.push(NativeCompoundPresentedFrame {
                image: Some(masks[index as usize % 2].clone()),
                delay: 1,
                sound: if index == 0 { mask_sound.clone() } else { Vec::new() },
                ..Default::default()
            });
        }
        let publish_at = frames.len() as i64;
        let mut post = state.clone();
        battle::publish_native_ai_command_modifier(&mut post.units, Some(&mut plan.clone()))?;
        let mut input = build_native_map_frame_input(
            Some(assets),
            self.m.as_ref(),
            Some(&post),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )?;
        let (mut stage_work, mut stage_vga) = (baseline_work.clone(), baseline_vga.clone());
        indexedmap::compose_native_frame(&mut stage_work, &mut stage_vga, &mut input)?;
        let mut queue = Vec::new();
        for (index, target) in targets.iter().enumerate() {
            let visible = (view.camera_x..view.camera_x + 12).contains(&target.x)
                && (view.camera_y - 1..=view.camera_y + 7).contains(&target.y);
            let (processed, value) = if id < 19 {
                let step = &plan.result.word_steps[index];
                (step.processed, i64::from(step.delta))
            } else {
                (plan.result.pair_steps[index].processed, 0x0f)
            };
            if processed {
                queue = battle::append_native_presentation_digits(
                    &queue,
                    value,
                    schedule.digit_bias,
                    target.record_index,
                    visible,
                )?;
            } else if visible {
                queue = battle::append_native_presentation_miss(&queue, target.record_index);
            }
        }
        for index in 0..schedule.digit_frames {
            work.copy_from_slice(&stage_work);
            let mut pixels = stage_vga.clone();
            indexedmap::compose_native_command_heal_digit_frame(
                &mut work,
                &mut pixels,
                &stage_work,
                &assets.command_heal_digits,
                &queue,
                &targets,
                view.camera_x,
                view.camera_y,
                schedule.digit_vertical,
                index,
            )?;
            frames.push(NativeCompoundPresentedFrame { image: Some(image(pixels)), delay: 1, ..Default::default() });
        }
        let then = Box::new(move |g: &mut Game, result: battle::NativeCommandModifierResult| {
            let units = &mut g.st.as_mut().unwrap().units;
            let (a, t) = (&units[actor], &units[target]);
            let pose = dir_toward(a.x, a.y, t.x, t.y);
            units[actor].set_map_pose(pose);
            let count = if id < 19 { result.word_steps.len() } else { result.pair_steps.len() };
            g.msg = format!("原始指令 {id}：影響 {count} 個目標");
            g.finish_successful_unit_action(Some(actor), None);
            g.check_result();
        });
        self.native_ai_command_modifier = Some(NativeAICommandModifierPresentationJob {
            actor: Some(actor),
            plan: Some(plan),
            end_at: frames.len() as i64,
            frames,
            publish_at,
            rng_before: self.native_rng_state,
            baseline_work,
            baseline_vga,
            stage_work,
            stage_vga,
            then: Some(then),
            ..Default::default()
        });
        self.st.as_mut().unwrap().native_experience_accumulator = 0;
        Ok(())
    }

    pub(crate) fn present_native_ai_command_modifier(&mut self, screen: Option<&mut Picture>) {
        let job = self.native_ai_command_modifier.as_mut().unwrap();
        let Some(image) = job.frames.get(job.frame as usize).and_then(|frame| frame.image.as_ref()) else {
            self.fail_native_ai_command_modifier_presentation(&game_err!("狀態變更指令的畫格無效"));
            return;
        };
        if let Some(screen) = screen {
            let mut draw = DrawImageOptions::default();
            draw.geo_m.scale(2.0, 2.0);
            screen.draw_image(image, &draw);
        }
        job.drawn = true;
    }

    /// Go：`native_ai_modifier_presentation.go:483`，`func (g *Game) cancelNativeAICommandModifierPresentation()`
    pub(crate) fn cancel_native_ai_command_modifier_presentation(&mut self) {
        let Some(mut j) = self.native_ai_command_modifier.take() else {
            return;
        };
        // Go 的計畫以指標指向 `g.st.Units`；Rust 交給規則層的是同一個 `units` 陣列（沒有戰場時是空陣列）。
        let mut empty = Vec::new();
        let units = self.st.as_mut().map_or(&mut empty, |st| &mut st.units);
        if j.plan2022.is_some() {
            let _ = battle::abort_native_ai_command2022(units, j.plan2022.as_mut());
        } else {
            let _ = battle::abort_native_ai_command_modifier(units, j.plan.as_mut());
        }
        self.native_rng_state = j.rng_before;
        self.native_map_work = j.baseline_work.clone();
        self.native_map_vga = j.baseline_vga.clone();
        self.native_ai_command_modifier = None;
    }

    /// Go：`native_ai_modifier_presentation.go:499`，`func (g *Game) failNativeAICommandModifierPresentation(err error)`
    /// （`r8`，隨 `stepNativeAICommandModifierPresentation` 補移植）。
    pub(crate) fn fail_native_ai_command_modifier_presentation(&mut self, err: &GameError) {
        self.cancel_native_ai_command_modifier_presentation();
        self.load_err = format!("native AI command modifier presentation: {}", err.message());
    }

    /// Go：`native_ai_modifier_presentation.go:504`，`func (g *Game) stepNativeAICommandModifierPresentation()`
    pub(crate) fn step_native_ai_command_modifier_presentation(&mut self) {
        let Some(j) = self.native_ai_command_modifier.as_mut() else {
            return;
        };
        if j.holding {
            if j.hold > 0 {
                j.hold -= 1;
                return;
            }
            let mut empty = Vec::new();
            let units = self.st.as_mut().map_or(&mut empty, |st| &mut st.units);
            let result = if j.plan2022.is_some() {
                battle::complete_native_ai_command2022(units, j.plan2022.as_mut())
            } else {
                battle::complete_native_ai_command_modifier(units, j.plan.as_mut())
            };
            if let Err(err) = result {
                self.fail_native_ai_command_modifier_presentation(&GameError::from(err));
                return;
            }
            let mut j = self.native_ai_command_modifier.take().unwrap();
            let (then, then2022) = (j.then.take(), j.then2022.take());
            let result = j.plan.as_ref().map(|plan| plan.result.clone()).unwrap_or_default();
            if let Some(then2022) = then2022 {
                // Go 傳 `plan2022` 指標（可為 nil）；Rust 的延續收 `&NativeAICommand2022Plan`，nil 時以零值代入。
                let plan2022 = j.plan2022.take().unwrap_or_default();
                then2022(self, &plan2022);
            } else if let Some(then) = then {
                then(self, result);
            }
            return;
        }
        if !j.drawn || j.frame < 0 || j.frame >= j.frames.len() as i64 {
            return;
        }
        j.drawn = false;
        let frame = &mut j.frames[j.frame as usize];
        if !frame.sound_played && !frame.sound.is_empty() {
            let sound = frame.sound.clone();
            frame.sound_played = true;
            self.play_raw(&sound);
        }
        let j = self.native_ai_command_modifier.as_mut().unwrap();
        j.repeat += 1;
        if j.repeat < j.frames[j.frame as usize].delay {
            return;
        }
        j.repeat = 0;
        let next = j.frame + 1;
        if next == j.publish_at && !j.published {
            let mut empty = Vec::new();
            let units = self.st.as_mut().map_or(&mut empty, |st| &mut st.units);
            let result = if j.plan2022.is_some() {
                battle::publish_native_ai_command2022(units, j.plan2022.as_mut())
            } else {
                battle::publish_native_ai_command_modifier(units, j.plan.as_mut())
            };
            if let Err(err) = result {
                self.fail_native_ai_command_modifier_presentation(&GameError::from(err));
                return;
            }
            let j = self.native_ai_command_modifier.as_mut().unwrap();
            self.native_rng_state = match (&j.plan2022, &j.plan) {
                (Some(plan2022), _) => plan2022.rng_state,
                (None, Some(plan)) => plan.result.rng_state,
                // Go 解參考 nil 的 `j.plan` 會 panic；Publish 已先回報 nil 計畫的錯誤，這裡到不了。
                (None, None) => unreachable!("Go：j.plan 為 nil 時 Publish 已回錯"),
            };
            self.native_map_work = j.stage_work.clone();
            self.native_map_vga = j.stage_vga.clone();
            j.published = true;
        }
        let j = self.native_ai_command_modifier.as_mut().unwrap();
        if next == j.end_at {
            if !j.published {
                self.fail_native_ai_command_modifier_presentation(&game_err!(
                    "required modifier boundary was not presented"
                ));
                return;
            }
            j.frame = next - 1;
            j.holding = true;
            if !j.skip_hold {
                j.hold = native_delay_ticks(500);
            }
            return;
        }
        j.frame = next;
    }
}
