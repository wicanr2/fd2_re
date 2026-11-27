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
            let after = g.native_ai_action_after.take();
            g.finish_successful_unit_action(Some(actor), after);
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

    /// 敵方指令 20／21／22／26／27 的 handler 尾段：與玩家表共用的 0x1541F 路線，敵方不播玩家專屬的 0x1D6C8
    /// 調色盤前奏。效果畫格 → 遮罩交替 → 發布 → 數字尾段，與指令 17～19 共用同一個演出擁有者。
    /// 只做原版索引色畫面，HD 綁定與指令 17～19 的 Rust 版相同另期。
    ///
    /// Go：`native_ai_modifier_presentation.go:70`，`func (g *Game) startNativeAICommand2022Presentation(...)`
    /// 與 `:74` `startNativeCommand2022Presentation` 的敵方分支（`player == false`）。
    pub(crate) fn execute_native_ai_command2022(&mut self, action: &battle::AIPlan) -> Result<()> {
        let id = action.native_command_id;
        if !matches!(id, 20 | 21 | 22 | 26 | 27) {
            return Err(game_err!("原版指令尾段演出不可用"));
        }
        let state = self.st.as_ref().ok_or_else(|| game_err!("指令尾段缺少戰場"))?;
        let actor = action.u.filter(|&i| i < state.units.len()).ok_or_else(|| game_err!("指令缺少施放者"))?;
        let target = action.target.filter(|&i| i < state.units.len()).ok_or_else(|| game_err!("指令缺少目標"))?;
        let origin = action.native_action_destination;
        let then2022 = Box::new(move |g: &mut Game, result: &battle::NativeAICommand2022Plan| {
            if matches!(id, 22 | 26 | 27) {
                for entry in &result.results {
                    if entry.apply.is_some_and(|apply| apply.damage.actual > 0) {
                        g.award_death_reward(entry.target, Some(actor));
                    }
                }
            }
            let units = &mut g.st.as_mut().unwrap().units;
            let (a, t) = (&units[actor], &units[target]);
            let pose = dir_toward(a.x, a.y, t.x, t.y);
            units[actor].set_map_pose(pose);
            g.msg = format!("原始指令 {id}：完成敵方 indexed 尾段 ({} targets)", result.results.len());
            let after = g.native_ai_action_after.take();
            g.finish_successful_unit_action(Some(actor), after);
            g.check_result();
        });
        self.start_native_command2022_presentation(actor, None, id, Some(origin), then2022)?;
        self.st.as_mut().unwrap().native_experience_accumulator = 0;
        Ok(())
    }

    /// 指令 20～22、25～27 的 handler 尾段。玩家（`ai_origin` 為 `None`）先播 0x1D6C8 調色盤前奏的八個階段，
    /// 再接與敵方相同的效果、遮罩、發布與數字尾段；敵方的游標聚焦在計畫成立之後、擷取地圖畫面之前。
    ///
    /// Go：`native_ai_modifier_presentation.go:74`，`func (g *Game) startNativeCommand2022Presentation(actor, confirmed *battle.Unit, commandID int, player bool, then func(*battle.NativeAICommand2022Plan)) error`
    pub(crate) fn start_native_command2022_presentation(
        &mut self,
        actor: UnitIdx,
        confirmed: Option<UnitIdx>,
        id: i64,
        ai_origin: Option<battle::Cell>,
        then2022: Box<dyn FnOnce(&mut Game, &battle::NativeAICommand2022Plan) + Send>,
    ) -> Result<()> {
        let player = ai_origin.is_none();
        if !self.current_native_system_options().full_presentation_enabled()
            || !(20..=27).contains(&id)
            || id == 23
            || id == 24
            || (!player && id == 25)
            || self.st.is_none()
        {
            return Err(game_err!("native command tail presentation context unavailable"));
        }
        if player && confirmed.is_none() {
            return Err(game_err!("native player command tail confirmed target unavailable"));
        }
        if self.native_ai_command_modifier.is_some()
            || self.native_modifier_presentation.is_some()
            || self.native_heal_presentation.is_some()
        {
            return Err(game_err!("native command tail presentation already active"));
        }
        let state = self.st.as_ref().unwrap();
        let rng = self.native_rng_state;
        let plan = match ai_origin {
            Some(origin) => state.plan_native_ai_command2022(Some(actor), origin, id, rng)?,
            None => state.plan_native_player_command2022(Some(actor), confirmed, id, rng)?,
        };
        if let Some(origin) = ai_origin {
            let (x, y) = (state.units[actor].x, state.units[actor].y);
            self.ai_focus_cursor(x, y);
            self.ai_focus_cursor(origin.x, origin.y);
        }
        let (hud, hud_ready) = self.native_map_hud_input();
        let state = self.st.as_ref().unwrap();
        let assets = self
            .native_map_assets
            .as_ref()
            .filter(|a| native_map_assets_available(Some(a)))
            .ok_or_else(|| game_err!("指令尾段缺少地圖素材"))?;
        if !hud_ready
            || !state.has_native_map_view_state
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
        {
            return Err(game_err!("指令尾段缺少原版地圖畫面"));
        }
        let schedule = fdother::build_native_command2022_tail_schedule()
            .map_err(|err| game_err!("{err}"))?
            .into_iter()
            .find(|schedule| schedule.command_id == id)
            .ok_or_else(|| game_err!("缺少指令 {id} 的原版演出排程"))?;
        if (assets.fdother6.len() as i64) < schedule.effect_start + schedule.effect_frames
            || assets.command_heal_digits.len() as i64 <= schedule.digit_bias + 9
        {
            return Err(game_err!("指令 {id} 的尾段描述不可用"));
        }
        let tail_targets = plan
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
        let effect_targets = tail_targets
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
        if player {
            let flash = self.native_command_palette_flash.as_ref();
            if flash.is_none() || (!os_mute_or_shot(Some(self)) && self.sfx_command_modifier.is_empty()) {
                return Err(game_err!("native player command tail palette assets unavailable"));
            }
            let phases = flash
                .unwrap()
                .native_command_palette_flash_phases(id)
                .ok()
                .filter(|phases| phases.len() == 8)
                .ok_or_else(|| game_err!("native player command tail palette phases unavailable"))?;
            for rgb in phases {
                let mut dac = self.native_map_dac.clone();
                dac[..3].copy_from_slice(&rgb);
                let phase_palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&dac)?);
                let indexed =
                    IndexedFrame { width: 320, height: 200, pix: baseline_vga.clone(), palette: phase_palette };
                frames.push(NativeCompoundPresentedFrame {
                    image: Some(Arc::new(Picture::from_indexed(&indexed))),
                    delay: 1,
                    ..Default::default()
                });
            }
            frames[0].sound = self.sfx_command_modifier.clone();
        }
        let effect_start = frames.len();
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
                &effect_targets,
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
        for &raw_frame in &schedule.extra_sample_frame_indices {
            if raw_frame <= 0 || raw_frame >= schedule.effect_frames {
                return Err(game_err!("指令 {id} 的額外音效畫格無效"));
            }
        }
        for index in std::iter::once(0).chain(schedule.extra_sample_frame_indices.iter().copied()) {
            frames.get_mut(effect_start + index as usize).ok_or_else(|| game_err!("指令 {id} 的音效畫格無效"))?.sound =
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
            &effect_targets,
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
        // Go `nativeCommand2022ClonedState`：只投影計畫結果到複本，畫出發布後的地圖。
        let mut post = state.clone();
        for (index, result) in plan.results.iter().enumerate() {
            let Some(unit) = result.target.and_then(|t| post.units.get_mut(t)) else {
                return Err(game_err!("native command tail cloned target {index} unavailable"));
            };
            if !result.command25 && !(0x22..=0x27).contains(&result.offset) {
                return Err(game_err!("native command tail cloned target {index} unavailable"));
            }
            if let Some(restore) = result.restore.filter(|r| r.applied) {
                unit.hp += restore.restore.actual;
                unit.set_native_transient_duration(result.offset, 0);
            }
            if let Some(apply) = result.apply.filter(|a| a.applied) {
                unit.hp -= apply.damage.actual;
                unit.set_native_transient_duration(result.offset, apply.marker);
            }
            if result.command25 && result.cleared_byte5 {
                unit.native_record_byte5 &= !0x80;
            }
        }
        let mut input = build_native_map_frame_input(
            Some(assets),
            self.m.as_ref(),
            Some(&post),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )?;
        let (mut stage_work, mut stage_vga) = (baseline_work.clone(), baseline_vga.clone());
        indexedmap::compose_native_frame(&mut stage_work, &mut stage_vga, &mut input)?;
        let mut queue = Vec::new();
        for result in &plan.results {
            let record = result.target.ok_or_else(|| game_err!("native command tail result target unavailable"))?;
            let unit = &state.units[record];
            let in_camera = (view.camera_x..view.camera_x + 12).contains(&unit.x)
                && (view.camera_y - 1..=view.camera_y + 7).contains(&unit.y);
            let (processed, value) = if let Some(restore) = result.restore {
                (restore.applied, restore.restore.actual)
            } else if let Some(apply) = result.apply {
                (apply.applied, apply.damage.actual)
            } else if result.command25 {
                // 0x22C04 沒有成功數字；只有沒清掉的目標進 sub_1E1DC 而排入數字尾段。
                if result.cleared_byte5 {
                    continue;
                }
                (false, 0)
            } else {
                (false, 0)
            };
            if processed {
                queue = battle::append_native_presentation_digits(
                    &queue,
                    value,
                    schedule.digit_bias,
                    record as i64,
                    in_camera,
                )?;
            } else if in_camera {
                queue = battle::append_native_presentation_miss(&queue, record as i64);
            }
        }
        let has_digits = !queue.is_empty();
        if has_digits {
            for index in 0..schedule.digit_frames {
                work.copy_from_slice(&stage_work);
                let mut pixels = stage_vga.clone();
                indexedmap::compose_native_command_heal_digit_frame(
                    &mut work,
                    &mut pixels,
                    &stage_work,
                    &assets.command_heal_digits,
                    &queue,
                    &tail_targets,
                    view.camera_x,
                    view.camera_y,
                    schedule.digit_vertical,
                    index,
                )?;
                frames.push(NativeCompoundPresentedFrame {
                    image: Some(image(pixels)),
                    delay: 1,
                    ..Default::default()
                });
            }
        }
        self.native_ai_command_modifier = Some(NativeAICommandModifierPresentationJob {
            actor: Some(actor),
            plan2022: Some(plan),
            end_at: frames.len() as i64,
            frames,
            publish_at,
            skip_hold: !has_digits,
            rng_before: self.native_rng_state,
            baseline_work,
            baseline_vga,
            stage_work,
            stage_vga,
            then2022: Some(then2022),
            ..Default::default()
        });
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
