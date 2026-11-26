//! Go 來源：`remake/cmd/fd2/native_command9_ai_presentation.go`（456 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command9_ai_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand9AIPresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND9_AI_COMMON_PRELUDE: NativeCommand9AIPresentationPhase =
    NativeCommand9AIPresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_AI_ACTOR: NativeCommand9AIPresentationPhase = NativeCommand9AIPresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_AI_HANDLER: NativeCommand9AIPresentationPhase = NativeCommand9AIPresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_AI_COMMON_TAIL: NativeCommand9AIPresentationPhase =
    NativeCommand9AIPresentationPhase(3);

/// Go：`native_command9_ai_presentation.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand9AIHandlerFrame {
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) hp_stage: i64,
    pub(crate) play_sub1: bool,
    pub(crate) play_sub2: bool,
    pub(crate) hold_after: i64,
}

/// Go：`native_command9_ai_presentation.go:31`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand9AIPresentationJob {
    pub(crate) hd: Option<ModernHDOrdinaryFrames>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) handler: Vec<NativeCommand9AIHandlerFrame>,
    pub(crate) tail: Vec<Option<Arc<Picture>>>,
    pub(crate) phase: NativeCommand9AIPresentationPhase,
    pub(crate) frame: i64,
    pub(crate) hold: i64,
    pub(crate) pulse_black: bool,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
}

impl Game {
    /// 敵方指令 9（只限 raw side 0 的施放者）：前奏 → 施放者 → 滑入、前段、目標段、尾段、滑出 119 格 → 共用尾段。
    /// 背景取目標側、平台取施放者側，不依 side 交換。原版索引色分支，HD 另期。
    ///
    /// Go：`native_command9_ai_presentation.go:53`，`func (g *Game) startNativeCommand9AIPresentation(...)`
    pub(crate) fn execute_native_ai_command9(&mut self, action: &battle::AIPlan) -> Result<()> {
        if !self.current_native_system_options().full_presentation_enabled()
            || self.native_cmd9_ai_presentation.is_some()
            || self.native_command_scene.is_none()
            || self.native_command_palette_flash.is_none()
        {
            return Err(game_err!("native command9 AI presentation context unavailable"));
        }
        let state = self.st.as_ref().ok_or_else(|| game_err!("指令 9 缺少戰場"))?;
        let actor_index =
            action.u.filter(|&index| index < state.units.len()).ok_or_else(|| game_err!("指令 9 缺少施放者"))?;
        let confirmed =
            action.target.filter(|&index| index < state.units.len()).ok_or_else(|| game_err!("指令 9 缺少目標"))?;
        let actor = state.units[actor_index].clone();
        let luts = self
            .native_map_assets
            .as_ref()
            .filter(|assets| assets.luts.len() > 14)
            .map(|assets| assets.luts.clone())
            .ok_or_else(|| game_err!("native command9 AI raw-side-zero actor provenance unavailable"))?;
        if !actor.has_battle_fig
            || !actor.has_native_record_byte6
            || actor.native_record_byte6 != 0
            || !(0..=255).contains(&actor.battle_fig)
            || self.native_ui_palette.len() != 256
        {
            return Err(game_err!("native command9 AI raw-side-zero actor provenance unavailable"));
        }
        let plan = state.plan_native_ai_command_damage_single_target(
            Some(actor_index),
            Some(confirmed),
            9,
            &state.native_command_resistances,
            self.native_rng_state,
        )?;
        if plan.results.len() != 1
            || plan.results[0].target != Some(confirmed)
            || plan.damage_stages != figani::NATIVE_COMMAND9_AI_DAMAGE_STAGES
            || state.native_command_book.len() as i64 != battle::NATIVE_COMMAND_RECORD_COUNT
            || state.native_command_book[9].effect_mode != 0
        {
            return Err(game_err!("native command9 AI single final target unavailable"));
        }
        let target = state.units[confirmed].clone();
        if !target.has_battle_fig || !target.has_native_record_byte6 {
            return Err(game_err!("native command9 AI raw target provenance unavailable"));
        }
        let (background, platform) =
            self.native_damage_command_scenery_with(&actor, std::slice::from_ref(&target), false)?;
        let animations = separated_asset_path("animations");
        let actor_idle = figani::load_separated_resource(&animations, actor.battle_fig * 3)?;
        let actor_effect =
            figani::load_separated_resource_with_zero_header_fallback(&animations, actor.battle_fig * 3 + 2)?;
        let target_idle = figani::load_separated_resource(&animations, target.battle_fig * 3)?;
        let effect = figani::load_separated_archive_resource(
            &animations,
            "FDOTHER.DAT",
            figani::NATIVE_COMMAND9_AI_EFFECT_RESOURCE,
        )?;
        let schedule = figani::build_native_command9_ai_schedule(0, Some(&effect))?;
        if !os_mute_or_shot(Some(self))
            && (self.sfx_command8_actor.is_empty()
                || self.sfx_command8_sub1.is_empty()
                || self.sfx_command8_sub2.is_empty())
        {
            return Err(game_err!("native command9 AI converted #90 samples unavailable"));
        }
        let (Some(actor_idle_first), Some(target_idle_first)) = (actor_idle.frames.first(), target_idle.frames.first())
        else {
            return Err(game_err!("指令 9 缺少開場人物畫格"));
        };
        let panel_assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let (mut dac, _) = load_native_battle_palette()?;
        let actor_record = battle::native_battle_panel_record_for_unit(Some(&actor))?;
        let mut actor_after = actor.clone();
        actor_after.mp = plan.mp_after;
        let actor_after_record = battle::native_battle_panel_record_for_unit(Some(&actor_after))?;
        let stages = figani::NATIVE_COMMAND9_AI_DAMAGE_STAGES;
        let result = plan.results[0];
        let mut staged = target.clone();
        let mut bases = Vec::with_capacity(stages as usize + 1);
        for stage in 0..=stages {
            staged.hp = result.hp_before - (result.hp_before - result.hp_after) * stage / stages;
            let record = battle::native_battle_panel_record_for_unit(Some(&staged))?;
            bases.push(self.native_damage_command_base(
                &panel_assets,
                [(actor_index, &actor_after_record), (confirmed, &record)],
                &background,
                Some(&platform),
            )?);
        }
        let target_record = battle::native_battle_panel_record_for_unit(Some(&target))?;
        let before_panels = [(actor_index, actor_record.as_slice()), (confirmed, target_record.as_slice())];
        let prelude_base = self.native_damage_command_base(&panel_assets, before_panels, &background, None)?;
        let actor_base = self.native_damage_command_base(&panel_assets, before_panels, &background, Some(&platform))?;
        let prelude = battlepresent::build_native_command_prelude_frames(&battlepresent::NativeCommandPreludeInput {
            base: &prelude_base,
            actor_idle: actor_idle_first.clone(),
            first_target_idle: target_idle_first.clone(),
            platform: Some(&platform),
            raw_side: 0,
            mode: 0,
            baseline_dac: &dac,
        })?;
        let prelude = native_command24_prelude_images(&prelude)?
            .into_iter()
            .map(|frame| Some(Arc::new(Picture::from_indexed(&frame))))
            .collect();
        let actor_specs =
            battlepresent::build_native_command0_actor_frames(&battlepresent::NativeCommand0ActorInput {
                base_before: &actor_base,
                base_after: &bases[0],
                actor_effect: Some(&actor_effect),
                first_target_idle: Some(&target_idle),
                raw_side: 0,
                background: background.clone(),
                platform: platform.clone(),
                lut: &luts[11],
            })?;
        // Go 以 nativeCommandPaletteFlash.Entries[9] 做脈衝。
        let flash = self
            .native_command_palette_flash
            .as_ref()
            .filter(|table| table.entries.len() as i64 == battle::NATIVE_COMMAND_RECORD_COUNT)
            .map(|table| table.entries[9])
            .ok_or_else(|| game_err!("native command9 AI DAC inputs unavailable"))?;
        if dac.len() != 256 * 3 {
            return Err(game_err!("native command9 AI DAC inputs unavailable"));
        }
        dac[..3].copy_from_slice(&flash);
        let pulse = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&dac)?);
        let palette = self.native_ui_palette.clone();
        let image = |pixels: Vec<u8>| super::native_command0_presentation::command0_image(pixels, &palette);
        let actor_black = actor_specs.iter().map(|spec| image(spec.pixels.clone())).collect();
        let actor_pulse = actor_specs
            .iter()
            .map(|spec| super::native_command0_presentation::command0_image(spec.pixels.clone(), &pulse))
            .collect();
        let last = stages as usize;
        let sequence = battlepresent::build_native_command9_ai_sequence(&battlepresent::NativeCommand9AIInput {
            before_base: &bases[0],
            after_base: &bases[last],
            target_bases: bases.iter().map(Vec::as_slice).collect(),
            actor: Some(&actor_idle),
            target: Some(&target_idle),
            effect: Some(&effect),
            schedule,
        })?;
        let (next_frame, next_repeat) = (sequence.next_idle_frame, sequence.next_idle_repeat);
        let frame_of = |rendered: battlepresent::NativeCommand9RenderedFrame| NativeCommand9AIHandlerFrame {
            image: image(rendered.pixels),
            hp_stage: rendered.hp_stage,
            play_sub1: rendered.play_sub1,
            play_sub2: rendered.play_sub2,
            hold_after: 0,
        };
        let mut handler: Vec<NativeCommand9AIHandlerFrame> = sequence.slide_in.into_iter().map(frame_of).collect();
        let Some(last_slide) = handler.last_mut() else {
            return Err(game_err!("native command9 AI slide-in unavailable"));
        };
        last_slide.hold_after = native_delay_ticks(schedule.slide_in_hold_ms);
        for frames in [sequence.front, sequence.target, sequence.tail, sequence.slide_out] {
            handler.extend(frames.into_iter().map(frame_of));
        }
        if handler.len() != 119 {
            return Err(game_err!("native command9 AI handler frame count unavailable"));
        }
        let next_frame =
            usize::try_from(next_frame).map_err(|_| game_err!("native command9 AI tail inputs unavailable"))?;
        let tail = super::native_command2_presentation::native_command2_tail_pixels(
            &bases[last],
            &background,
            &platform,
            &actor_effect,
            &target_idle,
            next_frame,
            next_repeat,
            &luts,
        )?
        .into_iter()
        .map(image)
        .collect();
        let then = Box::new(move |g: &mut Game, results: Vec<battle::NativeCommandDamageResult>| {
            let (mut hit, mut damage) = (0, 0);
            for result in results {
                if result.native_command_damage.hit {
                    hit += 1;
                    damage += result.native_command_damage.damage;
                }
                g.award_death_reward(result.target, Some(actor_index));
            }
            let units = &mut g.st.as_mut().unwrap().units;
            let (a, t) = (&units[actor_index], &units[confirmed]);
            let pose = dir_toward(a.x, a.y, t.x, t.y);
            units[actor_index].set_map_pose(pose);
            g.msg = format!("原始指令 9：命中 {hit}，傷害 {damage}");
            g.finish_successful_unit_action(Some(actor_index), None);
            g.check_result();
        });
        let job = NativeCommand9AIPresentationJob {
            actor: Some(actor_index),
            plan: Some(plan),
            prelude,
            actor_black,
            actor_pulse,
            actor_specs,
            handler,
            tail,
            actor_mp_before: actor.mp,
            actor_acted_before: actor.acted,
            target_hp_before: target.hp,
            then: Some(then),
            ..Default::default()
        };
        self.ai_focus_cursor(actor.x, actor.y);
        self.ai_focus_cursor(action.native_action_destination.x, action.native_action_destination.y);
        self.st.as_mut().unwrap().native_experience_accumulator = 0;
        self.native_cmd9_ai_presentation = Some(job);
        Ok(())
    }

    /// 依階段取目前畫格畫到畫面並標記已呈現；步進由 `step_native_command9_ai_presentation` 推進。
    ///
    /// Go：`native_command9_ai_presentation.go:412`，`func (g *Game) drawNativeCommand9AIPresentation(screen *ebiten.Image) bool`
    /// 的原版索引色分支。
    pub(crate) fn present_native_command9_ai(&mut self, screen: Option<&mut Picture>) {
        let job = self.native_cmd9_ai_presentation.as_mut().unwrap();
        let index = job.frame as usize;
        let image = match job.phase {
            NATIVE_COMMAND9_AI_COMMON_PRELUDE => job.prelude.get(index),
            NATIVE_COMMAND9_AI_ACTOR => {
                let pulse = job.actor_specs.get(index).is_some_and(|spec| spec.pulse) && !job.pulse_black;
                (if pulse { &job.actor_pulse } else { &job.actor_black }).get(index)
            }
            NATIVE_COMMAND9_AI_HANDLER => job.handler.get(index).map(|frame| &frame.image),
            NATIVE_COMMAND9_AI_COMMON_TAIL => job.tail.get(index),
            _ => None,
        };
        let Some(image) = image.and_then(Option::as_deref) else {
            self.fail_native_command9_ai_presentation(&game_err!("指令 9 演出畫格無效"));
            return;
        };
        if let Some(screen) = screen {
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            screen.draw_image(image, &options);
        }
        job.drawn = true;
    }
}

impl Game {
    /// Go：`native_command9_ai_presentation.go:319`，`func (g *Game) stepNativeCommand9AIPresentation()`
    pub(crate) fn step_native_command9_ai_presentation(&mut self) {
        let Some(j) = self.native_cmd9_ai_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        match j.phase {
            NATIVE_COMMAND9_AI_COMMON_PRELUDE => {
                j.frame += 1;
                if j.frame >= j.prelude.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND9_AI_ACTOR, 0);
                }
            }
            NATIVE_COMMAND9_AI_ACTOR => {
                let spec = &j.actor_specs[j.frame as usize];
                let (spec_pulse, spec_publish_mp) = (spec.pulse, spec.publish_mp);
                if spec_pulse && !j.pulse_black {
                    if spec_publish_mp && !j.mp_published {
                        if let Err(err) = self.publish_native_command9_ai_mp() {
                            self.fail_native_command9_ai_presentation(&err);
                            return;
                        }
                        let sfx = self.sfx_command8_actor.clone();
                        self.play_raw(&sfx);
                    }
                    let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    j.pulse_black = true;
                    return;
                }
                if spec_publish_mp && !j.mp_published {
                    if let Err(err) = self.publish_native_command9_ai_mp() {
                        self.fail_native_command9_ai_presentation(&err);
                        return;
                    }
                    let sfx = self.sfx_command8_actor.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.pulse_black = false;
                j.frame += 1;
                if j.frame >= j.actor_black.len() as i64 {
                    if !j.mp_published {
                        self.fail_native_command9_ai_presentation(&game_err!("actor MP marker was not presented"));
                        return;
                    }
                    (j.phase, j.frame) = (NATIVE_COMMAND9_AI_HANDLER, 0);
                }
            }
            NATIVE_COMMAND9_AI_HANDLER => {
                if j.hold > 0 {
                    j.hold -= 1;
                    if j.hold == 0 {
                        j.frame += 1;
                        if j.frame >= j.handler.len() as i64 {
                            (j.phase, j.frame) = (NATIVE_COMMAND9_AI_COMMON_TAIL, 0);
                        }
                    }
                    return;
                }
                let frame = &j.handler[j.frame as usize];
                let hp_stage = frame.hp_stage;
                let (play1, play2, hold_after) = (frame.play_sub1, frame.play_sub2, frame.hold_after);
                if hp_stage != 0 {
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) = battle::apply_native_command_damage_stage(units, j.plan.as_mut(), 0, hp_stage) {
                        self.fail_native_command9_ai_presentation(&err.into());
                        return;
                    }
                }
                if play1 {
                    let sfx = self.sfx_command8_sub1.clone();
                    self.play_raw(&sfx);
                }
                if play2 {
                    let sfx = self.sfx_command8_sub2.clone();
                    self.play_raw(&sfx);
                }
                if hold_after > 0 {
                    self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作").hold = hold_after;
                    return;
                }
                let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.frame += 1;
                if j.frame >= j.handler.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND9_AI_COMMON_TAIL, 0);
                }
            }
            NATIVE_COMMAND9_AI_COMMON_TAIL => {
                j.frame += 1;
                if j.frame < j.tail.len() as i64 {
                    return;
                }
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
                    self.fail_native_command9_ai_presentation(&err.into());
                    return;
                }
                let plan = j.plan.as_ref().expect("Go：CompleteNativeCommandDamage 已檢查 plan");
                self.native_rng_state = plan.rng_after;
                let results = plan.results.clone();
                let then = j.then.take();
                self.native_cmd9_ai_presentation = None;
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Go `stepNativeCommand9AIPresentation` 兩處相同的 MP 發布：`ApplyNativeCommandDamageMP` 成功後設 `mpPublished`。
    fn publish_native_command9_ai_mp(&mut self) -> Result<()> {
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
        battle::apply_native_command_damage_mp(units, j.plan.as_mut())?;
        j.mp_published = true;
        Ok(())
    }

    /// Go：`native_command9_ai_presentation.go:306`，`func (g *Game) failNativeCommand9AIPresentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand9AIPresentation` 補移植）。
    pub(crate) fn fail_native_command9_ai_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd9_ai_presentation.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        if let Some(actor) = j.actor {
            (units[actor].mp, units[actor].acted) = (j.actor_mp_before, j.actor_acted_before);
        }
        if let Some(plan) = j.plan.as_ref()
            && plan.results.len() == 1
            && let Some(target) = plan.results[0].target
        {
            units[target].hp = j.target_hp_before;
        }
        self.load_err = format!("native command9 AI presentation: {}", err.message());
    }
}

#[cfg(test)]
#[path = "native_command9_ai_presentation/native_command9_ai_presentation_test.rs"]
mod native_command9_ai_presentation_test;
