//! Go 來源：`remake/cmd/fd2/native_command6_presentation.go`（560 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command6_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand6PresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND6_COMMON_PRELUDE: NativeCommand6PresentationPhase = NativeCommand6PresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND6_ACTOR: NativeCommand6PresentationPhase = NativeCommand6PresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND6_HANDLER: NativeCommand6PresentationPhase = NativeCommand6PresentationPhase(2);

/// Go：`native_command6_presentation.go:23`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand6HandlerFrame {
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) target_index: i64,
    pub(crate) hp_stage: i64,
    pub(crate) play_target: bool,
    pub(crate) play_front: bool,
    pub(crate) play_tail: bool,
}

/// Go：`native_command6_presentation.go:32`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand6PresentationJob {
    pub(crate) hd: Option<ModernHDOrdinaryFrames>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) handler: Vec<NativeCommand6HandlerFrame>,
    pub(crate) phase: NativeCommand6PresentationPhase,
    pub(crate) frame: i64,
    pub(crate) pulse_black: bool,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: Vec<i64>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
}

impl Game {
    /// 敵方指令 6：前奏 → 施放者 → 前段、逐目標、目標間轉場、尾段。各段底圖以 LUT15 重畫背景與平台，
    /// 演出亂數與傷害規劃必須一致。原版索引色分支，HD 另期；玩家游標路徑另期。
    ///
    /// Go：`native_command6_presentation.go:68`，`func (g *Game) startNativeCommand6PresentationAtCursor(...)` 的敵方分支，
    /// 與 `nativeCommand6StageBases`（`:383`）。
    pub(crate) fn execute_native_ai_command6(&mut self, action: &battle::AIPlan) -> Result<()> {
        if !self.current_native_system_options().full_presentation_enabled()
            || self.native_cmd6_presentation.is_some()
            || self.native_command_scene.is_none()
            || self.native_command_palette_flash.is_none()
        {
            return Err(game_err!("native command6 presentation context unavailable"));
        }
        let state = self.st.as_ref().ok_or_else(|| game_err!("指令 6 缺少戰場"))?;
        let actor_index =
            action.u.filter(|&index| index < state.units.len()).ok_or_else(|| game_err!("指令 6 缺少施放者"))?;
        let confirmed =
            action.target.filter(|&index| index < state.units.len()).ok_or_else(|| game_err!("指令 6 缺少目標"))?;
        let actor = state.units[actor_index].clone();
        let luts = self
            .native_map_assets
            .as_ref()
            .filter(|assets| assets.luts.len() > 15 && assets.luts[15].len() == 256)
            .map(|assets| assets.luts.clone())
            .ok_or_else(|| game_err!("native command6 raw actor provenance unavailable"))?;
        if !actor.has_battle_fig
            || !actor.has_native_record_byte6
            || !(0..=255).contains(&actor.battle_fig)
            || self.native_ui_palette.len() != 256
            || actor.camp != battle::ENEMY
        {
            return Err(game_err!("native command6 raw actor provenance unavailable"));
        }
        let side = actor.native_record_byte6;
        let animations = separated_asset_path("animations");
        let effect_resource = if side == 0 { 33 } else { 32 };
        let effect = figani::load_separated_archive_resource(&animations, "FDOTHER.DAT", effect_resource)?;
        let schedule = figani::build_native_command6_presentation_schedule(side, Some(&effect))?;
        let rng = self.native_rng_state;
        let recorded = std::cell::RefCell::new((Vec::<bool>::new(), Vec::<u16>::new()));
        let origin = action.native_action_destination;
        let mut walk = |count: i64, resolve: &mut dyn FnMut(i64, u16) -> battle::Result<(u16, bool)>| {
            *recorded.borrow_mut() = (vec![false; count as usize], vec![0; count as usize]);
            let mut record = |index: i64, before: u16| {
                let (next, hit) = resolve(index, before).map_err(|err| Box::new(err) as figani::ResolveError)?;
                let mut r = recorded.borrow_mut();
                (r.0[index as usize], r.1[index as usize]) = (hit, next);
                Ok((next, hit))
            };
            figani::walk_native_command6_rng(rng, schedule, side, count, Some(&mut record))
                .map_err(|err| battle::BattleError::new(err.error.to_string()))
        };
        let plan = state.plan_native_ai_command_damage_walk(
            Some(actor_index),
            origin,
            6,
            &state.native_command_resistances,
            rng,
            Some(&mut walk),
        )?;
        let (target_hits, target_numeric_rng) = recorded.into_inner();
        if plan.results.is_empty()
            || plan.damage_stages != figani::NATIVE_COMMAND6_DAMAGE_STAGES
            || state.native_command_book.len() as i64 != battle::NATIVE_COMMAND_RECORD_COUNT
            || state.native_command_book[6].effect_mode != 2
        {
            return Err(game_err!("native command6 final targets unavailable"));
        }
        let (indices, targets): (Vec<_>, Vec<_>) = plan
            .results
            .iter()
            .map(|result| {
                let index =
                    result.target.ok_or_else(|| game_err!("native command6 raw target provenance unavailable"))?;
                let unit = &state.units[index];
                if !unit.has_battle_fig || !unit.has_native_record_byte6 {
                    return Err(game_err!("native command6 raw target provenance unavailable"));
                }
                Ok((index, unit.clone()))
            })
            .collect::<Result<Vec<_>>>()?
            .into_iter()
            .unzip();
        let (background, platform) = self.native_damage_command_scenery(&actor, &targets)?;
        let actor_idle = figani::load_separated_resource(&animations, actor.battle_fig * 3)?;
        let actor_effect =
            figani::load_separated_resource_with_zero_header_fallback(&animations, actor.battle_fig * 3 + 2)?;
        let target_idle = targets
            .iter()
            .map(|unit| figani::load_separated_resource(&animations, unit.battle_fig * 3))
            .collect::<std::result::Result<Vec<_>, _>>()?;
        if !os_mute_or_shot(Some(self))
            && (self.sfx_command6_actor.is_empty()
                || self.sfx_command6_target.is_empty()
                || self.sfx_command6_front.is_empty()
                || self.sfx_command6_tail.is_empty())
        {
            return Err(game_err!("native command6 converted #87 samples unavailable"));
        }
        let (Some(actor_idle_first), Some(target_idle_first)) =
            (actor_idle.frames.first(), target_idle[0].frames.first())
        else {
            return Err(game_err!("指令 6 缺少開場人物畫格"));
        };
        let panel_assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let (mut dac, _) = load_native_battle_palette()?;
        let actor_record = battle::native_battle_panel_record_for_unit(Some(&actor))?;
        let mut actor_after = actor.clone();
        actor_after.mp = plan.mp_after;
        let actor_after_record = battle::native_battle_panel_record_for_unit(Some(&actor_after))?;
        let stages = figani::NATIVE_COMMAND6_DAMAGE_STAGES;
        let mut bases = Vec::with_capacity(targets.len());
        for ((&index, target), result) in indices.iter().zip(&targets).zip(&plan.results) {
            let mut staged = target.clone();
            let mut target_bases = Vec::new();
            for stage in 0..=stages {
                staged.hp = result.hp_before - (result.hp_before - result.hp_after) * stage / stages;
                let record = battle::native_battle_panel_record_for_unit(Some(&staged))?;
                let mut base = self.native_damage_command_base(
                    &panel_assets,
                    [(actor_index, &actor_after_record), (index, &record)],
                    &background,
                    Some(&platform),
                )?;
                // sub_2B659：LUT15 重畫背景與平台。
                background.blit_lut_at(&mut base, 320, 0, &luts[15])?;
                platform.blit_lut_at(&mut base, 320, 0, &luts[15])?;
                target_bases.push(base);
            }
            bases.push(target_bases);
        }
        let last = stages as usize;
        let first = battle::native_battle_panel_record_for_unit(Some(&targets[0]))?;
        let before_panels = [(actor_index, actor_record.as_slice()), (indices[0], first.as_slice())];
        let prelude_base = self.native_damage_command_base(&panel_assets, before_panels, &background, None)?;
        let actor_base = self.native_damage_command_base(&panel_assets, before_panels, &background, Some(&platform))?;
        let prelude = battlepresent::build_native_command_prelude_frames(&battlepresent::NativeCommandPreludeInput {
            base: &prelude_base,
            actor_idle: actor_idle_first.clone(),
            first_target_idle: target_idle_first.clone(),
            platform: Some(&platform),
            raw_side: side,
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
                base_after: &bases[0][0],
                actor_effect: Some(&actor_effect),
                first_target_idle: Some(&target_idle[0]),
                raw_side: side,
                background: background.clone(),
                platform: platform.clone(),
                lut: &luts[15],
            })?;
        // Go 以 nativeCommandPaletteFlash.Entries[6] 做脈衝。
        let flash = self
            .native_command_palette_flash
            .as_ref()
            .filter(|table| table.entries.len() as i64 == battle::NATIVE_COMMAND_RECORD_COUNT)
            .map(|table| table.entries[6])
            .ok_or_else(|| game_err!("native command6 DAC inputs unavailable"))?;
        if dac.len() != 256 * 3 {
            return Err(game_err!("native command6 DAC inputs unavailable"));
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
        let sequence =
            battlepresent::build_native_command6_effect_sequence(&battlepresent::NativeCommand6EffectInput {
                target_hits,
                target_numeric_rng,
                front_base: &bases[0][0],
                tail_base: &bases[bases.len() - 1][last],
                target_bases: bases.iter().map(|frames| frames.iter().map(Vec::as_slice).collect()).collect(),
                transition_bases: bases[..bases.len() - 1].iter().map(|frames| frames[last].as_slice()).collect(),
                actor_effect: Some(&actor_effect),
                target_idle: target_idle.iter().map(Some).collect(),
                effect: Some(&effect),
                schedule,
                raw_side: side,
            })?;
        if sequence.rng_after != plan.rng_after {
            return Err(game_err!("native command6 display RNG disagrees with damage plan"));
        }
        let mut handler = Vec::new();
        let template = |target_index: i64, front: bool, target: bool, tail: bool| NativeCommand6HandlerFrame {
            target_index,
            play_front: front,
            play_target: target,
            play_tail: tail,
            ..Default::default()
        };
        let push =
            |handler: &mut Vec<NativeCommand6HandlerFrame>, frames: Vec<Vec<u8>>, t: NativeCommand6HandlerFrame| {
                handler.extend(
                    frames.into_iter().map(|pixels| NativeCommand6HandlerFrame { image: image(pixels), ..t.clone() }),
                );
            };
        push(&mut handler, sequence.front, template(-1, true, false, false));
        let mut transitions = sequence.transitions.into_iter();
        for (target_index, target_sequence) in sequence.targets.into_iter().enumerate() {
            if target_sequence.frames.len() != target_sequence.hp_stages.len() {
                return Err(game_err!("native command6 target source count differs"));
            }
            for (pixels, hp_stage) in target_sequence.frames.into_iter().zip(target_sequence.hp_stages) {
                handler.push(NativeCommand6HandlerFrame {
                    image: image(pixels),
                    target_index: target_index as i64,
                    hp_stage,
                    play_target: true,
                    ..Default::default()
                });
            }
            if let Some(frames) = transitions.next() {
                push(&mut handler, frames, template(-1, false, true, false));
            }
        }
        push(&mut handler, sequence.tail, template(-1, false, false, true));
        if handler.is_empty() {
            return Err(game_err!("native command6 handler frames unavailable"));
        }
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
            g.msg = format!("原始指令 6：命中 {hit}，傷害 {damage}");
            g.finish_successful_unit_action(Some(actor_index), None);
            g.check_result();
        });
        let job = NativeCommand6PresentationJob {
            actor: Some(actor_index),
            plan: Some(plan),
            prelude,
            actor_black,
            actor_pulse,
            actor_specs,
            handler,
            actor_mp_before: actor.mp,
            actor_acted_before: actor.acted,
            target_hp_before: targets.iter().map(|target| target.hp).collect(),
            then: Some(then),
            ..Default::default()
        };
        self.ai_focus_cursor(actor.x, actor.y);
        self.ai_focus_cursor(origin.x, origin.y);
        self.st.as_mut().unwrap().native_experience_accumulator = 0;
        self.native_cmd6_presentation = Some(job);
        Ok(())
    }

    /// 依階段取目前畫格畫到畫面並標記已呈現；步進由 `step_native_command6_presentation` 推進。
    ///
    /// Go：`native_command6_presentation.go:520`，`func (g *Game) drawNativeCommand6Presentation(screen *ebiten.Image) bool`
    /// 的原版索引色分支。
    pub(crate) fn present_native_command6(&mut self, screen: Option<&mut Picture>) {
        let job = self.native_cmd6_presentation.as_mut().unwrap();
        let index = job.frame as usize;
        let image = match job.phase {
            NATIVE_COMMAND6_COMMON_PRELUDE => job.prelude.get(index),
            NATIVE_COMMAND6_ACTOR => {
                let pulse = job.actor_specs.get(index).is_some_and(|spec| spec.pulse) && !job.pulse_black;
                (if pulse { &job.actor_pulse } else { &job.actor_black }).get(index)
            }
            NATIVE_COMMAND6_HANDLER => job.handler.get(index).map(|frame| &frame.image),
            _ => None,
        };
        let Some(image) = image.and_then(Option::as_deref) else {
            self.fail_native_command6_presentation(&game_err!("指令 6 演出畫格無效"));
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
    /// Go：`native_command6_presentation.go:443`，`func (g *Game) stepNativeCommand6Presentation()`
    pub(crate) fn step_native_command6_presentation(&mut self) {
        let Some(j) = self.native_cmd6_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        match j.phase {
            NATIVE_COMMAND6_COMMON_PRELUDE => {
                j.frame += 1;
                if j.frame >= j.prelude.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND6_ACTOR, 0);
                }
            }
            NATIVE_COMMAND6_ACTOR => {
                let spec = &j.actor_specs[j.frame as usize];
                let (spec_pulse, spec_publish_mp) = (spec.pulse, spec.publish_mp);
                if spec_pulse && !j.pulse_black {
                    if spec_publish_mp && !j.mp_published {
                        if let Err(err) = self.publish_native_command6_mp() {
                            self.fail_native_command6_presentation(&err);
                            return;
                        }
                        let sfx = self.sfx_command6_actor.clone();
                        self.play_raw(&sfx);
                    }
                    let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    j.pulse_black = true;
                    return;
                }
                if spec_publish_mp && !j.mp_published {
                    if let Err(err) = self.publish_native_command6_mp() {
                        self.fail_native_command6_presentation(&err);
                        return;
                    }
                    let sfx = self.sfx_command6_actor.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.pulse_black = false;
                j.frame += 1;
                if j.frame >= j.actor_black.len() as i64 {
                    if !j.mp_published {
                        self.fail_native_command6_presentation(&game_err!("actor MP marker was not presented"));
                        return;
                    }
                    (j.phase, j.frame) = (NATIVE_COMMAND6_HANDLER, 0);
                }
            }
            NATIVE_COMMAND6_HANDLER => {
                let frame = &j.handler[j.frame as usize];
                let (target_index, hp_stage) = (frame.target_index, frame.hp_stage);
                let play_front = frame.play_front && j.frame == 0;
                let play_target = frame.play_target;
                let play_tail = frame.play_tail && (j.frame == 0 || !j.handler[(j.frame - 1) as usize].play_tail);
                if hp_stage != 0 {
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) =
                        battle::apply_native_command_damage_stage(units, j.plan.as_mut(), target_index, hp_stage)
                    {
                        self.fail_native_command6_presentation(&err.into());
                        return;
                    }
                }
                if play_front {
                    let sfx = self.sfx_command6_front.clone();
                    self.play_raw(&sfx);
                }
                if play_target {
                    let sfx = self.sfx_command6_target.clone();
                    self.play_raw(&sfx);
                }
                if play_tail {
                    let sfx = self.sfx_command6_tail.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.frame += 1;
                if j.frame < j.handler.len() as i64 {
                    return;
                }
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
                    self.fail_native_command6_presentation(&err.into());
                    return;
                }
                let plan = j.plan.as_ref().expect("Go：CompleteNativeCommandDamage 已檢查 plan");
                self.native_rng_state = plan.rng_after;
                let results = plan.results.clone();
                let then = j.then.take();
                self.native_cmd6_presentation = None;
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Go `stepNativeCommand6Presentation` 兩處相同的 MP 發布：`ApplyNativeCommandDamageMP` 成功後設 `mpPublished`。
    fn publish_native_command6_mp(&mut self) -> Result<()> {
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
        battle::apply_native_command_damage_mp(units, j.plan.as_mut())?;
        j.mp_published = true;
        Ok(())
    }

    /// Go：`native_command6_presentation.go:428`，`func (g *Game) failNativeCommand6Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand6Presentation` 補移植）。
    pub(crate) fn fail_native_command6_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd6_presentation.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        if let Some(actor) = j.actor {
            (units[actor].mp, units[actor].acted) = (j.actor_mp_before, j.actor_acted_before);
        }
        if let Some(plan) = j.plan.as_ref() {
            for (index, result) in plan.results.iter().enumerate() {
                if let Some(target) = result.target
                    && index < j.target_hp_before.len()
                {
                    units[target].hp = j.target_hp_before[index];
                }
            }
        }
        self.load_err = format!("native command6 presentation: {}", err.message());
    }
}

#[cfg(test)]
#[path = "native_command6_presentation/native_command6_presentation_test.rs"]
mod native_command6_presentation_test;
