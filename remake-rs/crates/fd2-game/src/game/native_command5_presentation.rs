//! Go 來源：`remake/cmd/fd2/native_command5_presentation.go`（577 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::native_command0_presentation::command0_image;
use super::*;

/// Go：`native_command5_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand5PresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND5_COMMON_PRELUDE: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND5_ACTOR: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND5_HANDLER: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND5_COMMON_TAIL: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(3);

/// Go：`native_command5_presentation.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand5HandlerFrame {
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) target_index: i64,
    pub(crate) hp_stage: i64,
    pub(crate) play_primary: bool,
    pub(crate) play_secondary: bool,
}

/// Go：`native_command5_presentation.go:32`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand5PresentationJob {
    pub(crate) command_id: i64,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) handler: Vec<NativeCommand5HandlerFrame>,
    pub(crate) tail: Vec<Option<Arc<Picture>>>,
    pub(crate) hd: Option<ModernHDCommand5Frames>,
    pub(crate) phase: NativeCommand5PresentationPhase,
    pub(crate) frame: i64,
    pub(crate) pulse_black: bool,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: Vec<i64>,
    pub(crate) actor_sfx: Vec<u8>,
    pub(crate) target_sfx: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
}

impl Game {
    pub(crate) fn execute_native_ai_indexed_damage_command(&mut self, action: &battle::AIPlan) -> Result<()> {
        let id = action.native_command_id;
        if !self.current_native_system_options().full_presentation_enabled() || self.native_cmd5_presentation.is_some()
        {
            return Err(game_err!("指令 {id} 的原版演出不可用"));
        }
        let state = self.st.as_ref().ok_or_else(|| game_err!("指令缺少戰場"))?;
        let actor_index =
            action.u.filter(|&index| index < state.units.len()).ok_or_else(|| game_err!("指令缺少施放者"))?;
        let confirmed =
            action.target.filter(|&index| index < state.units.len()).ok_or_else(|| game_err!("指令缺少目標"))?;
        let actor = state.units[actor_index].clone();
        if !actor.has_battle_fig
            || !actor.has_native_record_byte6
            || !(0..=255).contains(&actor.battle_fig)
            || self.native_ui_palette.len() != 256
        {
            return Err(game_err!("指令 {id} 缺少人物編號或調色盤"));
        }
        let luts = self
            .native_map_assets
            .as_ref()
            .and_then(|assets| assets.luts.get(11..15))
            .ok_or_else(|| game_err!("指令 {id} 缺少場景調色表"))?
            .to_vec();
        let animations = separated_asset_path("animations");
        let side = actor.native_record_byte6;
        let resource = match id {
            1 => {
                if side == 0 {
                    21
                } else {
                    19
                }
            }
            4 => 22 + i64::from(side == 0),
            5 => 24 + i64::from(side == 0),
            _ => return Err(game_err!("未知的指令演出 {id}")),
        };
        let effect = figani::load_separated_archive_resource(&animations, "FDOTHER.DAT", resource)?;
        let schedule = match id {
            4 => Some(figani::build_native_command4_presentation_schedule(side, Some(&effect))?),
            5 => Some(figani::build_native_command5_presentation_schedule(side, Some(&effect))?),
            _ => None,
        };
        let rng = self.native_rng_state;
        let mut rolls = BTreeMap::new();
        let origin = action.native_action_destination;
        let resistances = &state.native_command_resistances;
        let plan = if let Some(schedule) = schedule {
            let mut walk = |count: i64, resolve: &mut dyn FnMut(i64, u16) -> battle::Result<(u16, bool)>| {
                let mut record = |index, before| {
                    let (after, hit) = resolve(index, before).map_err(|err| Box::new(err) as figani::ResolveError)?;
                    rolls.insert(index, (before, after, hit));
                    Ok((after, hit))
                };
                figani::walk_native_command5_rng(rng, schedule, side, count, Some(&mut record))
                    .map_err(|err| battle::BattleError::new(err.error.to_string()))
            };
            if actor.camp == battle::ENEMY {
                state.plan_native_ai_command_damage_walk(
                    Some(actor_index),
                    origin,
                    id,
                    resistances,
                    rng,
                    Some(&mut walk),
                )?
            } else {
                state.plan_native_command_damage_walk(
                    Some(actor_index),
                    Some(confirmed),
                    id,
                    resistances,
                    rng,
                    Some(&mut walk),
                )?
            }
        } else if actor.camp == battle::ENEMY {
            state.plan_native_ai_command_damage(Some(actor_index), origin, id, resistances, rng)?
        } else {
            state.plan_native_command_damage(Some(actor_index), Some(confirmed), id, resistances, rng)?
        };
        if plan.results.is_empty()
            || plan.damage_stages != if id == 1 { 8 } else { figani::NATIVE_COMMAND5_DAMAGE_STAGES }
            || (schedule.is_some() && state.native_command_book[id as usize].effect_mode != 1)
        {
            return Err(game_err!("指令 {id} 缺少原版範圍或傷害排程"));
        }
        let (indices, targets): (Vec<_>, Vec<_>) = plan
            .results
            .iter()
            .map(|result| {
                let index = result.target.ok_or_else(|| game_err!("指令 {id} 缺少傷害目標"))?;
                let unit = &state.units[index];
                if !unit.has_battle_fig || !unit.has_native_record_byte6 {
                    return Err(game_err!("指令 {id} 目標 {index} 缺少原版人物編號"));
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
        let (Some(actor_idle_first), Some(target_idle_first), Some(actor_effect_first)) =
            (actor_idle.frames.first(), target_idle[0].frames.first(), actor_effect.frames.first())
        else {
            return Err(game_err!("指令 {id} 缺少開場人物畫格"));
        };
        let (actor_sfx, target_sfx) = match id {
            1 => (&self.sfx_command0_actor, &self.sfx_command0_target),
            4 => (&self.sfx_command4_actor, &self.sfx_command4_target),
            _ => (&self.sfx_command5_actor, &self.sfx_command5_target),
        };
        if !os_mute_or_shot(Some(self)) && (actor_sfx.is_empty() || target_sfx.is_empty()) {
            return Err(game_err!("指令 {id} 缺少原版音效"));
        }
        let (actor_sfx, target_sfx) = (actor_sfx.clone(), target_sfx.clone());
        let panel_assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let (mut dac, _) = load_native_battle_palette()?;
        let actor_record = battle::native_battle_panel_record_for_unit(Some(&actor))?;
        let mut actor_after = actor.clone();
        actor_after.mp = plan.mp_after;
        let actor_after_record = battle::native_battle_panel_record_for_unit(Some(&actor_after))?;
        let first = battle::native_battle_panel_record_for_unit(Some(&targets[0]))?;
        let before_panels = [(actor_index, actor_record.as_slice()), (indices[0], first.as_slice())];
        let prelude_base = self.native_damage_command_base(&panel_assets, before_panels, &background, None)?;
        let actor_base = self.native_damage_command_base(&panel_assets, before_panels, &background, Some(&platform))?;
        let mut bases = Vec::with_capacity(targets.len());
        for ((&index, target), result) in indices.iter().zip(&targets).zip(&plan.results) {
            let mut staged = target.clone();
            let mut target_bases = Vec::new();
            for stage in 0..=plan.damage_stages {
                staged.hp = result.hp_before - (result.hp_before - result.hp_after) * stage / plan.damage_stages;
                let record = battle::native_battle_panel_record_for_unit(Some(&staged))?;
                target_bases.push(self.native_damage_command_base(
                    &panel_assets,
                    [(actor_index, &actor_after_record), (index, &record)],
                    &background,
                    Some(&platform),
                )?);
            }
            bases.push(target_bases);
        }
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
                lut: &luts[0],
            })?;
        let flash = self
            .native_command_palette_flash
            .as_ref()
            .and_then(|table| table.entries.get(id as usize))
            .ok_or_else(|| game_err!("指令 {id} 缺少閃光調色表"))?;
        dac[..3].copy_from_slice(flash);
        let pulse = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&dac)?);
        let palette = &self.native_ui_palette;
        let actor_black = actor_specs.iter().map(|spec| command0_image(spec.pixels.clone(), palette)).collect();
        let actor_pulse = actor_specs.iter().map(|spec| command0_image(spec.pixels.clone(), &pulse)).collect();
        let mut replay = |index, before| {
            let Some(&(_, after, hit)) = rolls.get(&index).filter(|(seed, _, _)| *seed == before) else {
                return Err(battle::BattleError::new(format!("指令 {id} 的目標 {index} 亂數序列不符")));
            };
            Ok((after, hit))
        };
        let last_stage = plan.damage_stages as usize;
        let tail_base = &bases[bases.len() - 1][last_stage];
        let (handler, next_idle_frame, next_idle_repeat) = if let Some(schedule) = schedule {
            let sequence =
                battlepresent::build_native_command5_effect_sequence(battlepresent::NativeCommand5EffectInput {
                    front_base: &bases[0][0],
                    tail_base,
                    target_bases: bases.iter().map(|frames| frames.iter().map(Vec::as_slice).collect()).collect(),
                    transition_bases: bases[..bases.len() - 1]
                        .iter()
                        .map(|frames| frames[last_stage].as_slice())
                        .collect(),
                    actor_effect: Some(&actor_effect),
                    target_idle: target_idle.iter().map(Some).collect(),
                    effect: Some(&effect),
                    schedule,
                    raw_side: side,
                    visual_rng: rng,
                    resolve_target: Some(&mut replay),
                })?;
            if sequence.visual_rng != plan.rng_after {
                return Err(game_err!("指令 {id} 的演出與傷害亂數序列不一致"));
            }
            let mut handler = Vec::new();
            let mut append = |frames: Vec<battlepresent::NativeCommand5RenderedFrame>, target_index| {
                handler.extend(frames.into_iter().map(|frame| NativeCommand5HandlerFrame {
                    image: command0_image(frame.pixels, palette),
                    target_index,
                    hp_stage: frame.hp_stage,
                    play_primary: frame.play_primary,
                    play_secondary: frame.play_secondary,
                }));
            };
            append(sequence.front, -1);
            let mut transitions = sequence.transitions.into_iter();
            for (index, frames) in sequence.targets.into_iter().enumerate() {
                append(frames, index as i64);
                if let Some(frames) = transitions.next() {
                    append(frames, -1);
                }
            }
            append(sequence.tail, -1);
            (handler, sequence.next_idle_frame, sequence.next_idle_repeat)
        } else {
            let schedule = figani::build_native_command1_presentation_schedule(side, Some(&effect))?;
            let mut handler = Vec::new();
            let mut next_idle = (0, 0);
            for (index, (base, idle)) in bases.iter().zip(&target_idle).enumerate() {
                let refs = base.iter().map(Vec::as_slice).collect::<Vec<_>>();
                let sequence =
                    battlepresent::build_native_command1_target_sequence(&refs, Some(idle), Some(&effect), schedule)?;
                next_idle = (sequence.next_idle_frame, sequence.next_idle_repeat);
                handler.extend(sequence.frames.into_iter().zip(sequence.hp_stages).zip(sequence.sample_markers).map(
                    |((pixels, hp_stage), play_primary)| NativeCommand5HandlerFrame {
                        image: command0_image(pixels, palette),
                        target_index: index as i64,
                        hp_stage,
                        play_primary,
                        play_secondary: false,
                    },
                ));
                if let Some(next) = target_idle.get(index + 1) {
                    let frames = battlepresent::build_native_command1_transition_frames(
                        &base[last_stage],
                        Some(&actor_effect),
                        Some(idle),
                        Some(next),
                        side,
                    )?;
                    handler.extend(frames.into_iter().map(|pixels| NativeCommand5HandlerFrame {
                        image: command0_image(pixels, palette),
                        ..Default::default()
                    }));
                }
            }
            (handler, next_idle.0, next_idle.1)
        };
        let idle = &target_idle[target_idle.len() - 1].frames;
        let mut idle_index = next_idle_frame as usize;
        let mut idle_repeat = next_idle_repeat;
        let mut tail = Vec::new();
        for lut in luts[1..].iter().map(Some).chain(std::iter::once(None)) {
            let mut pixels = tail_base.clone();
            if let Some(lut) = lut {
                background.blit_lut_at(&mut pixels, 320, 0, lut)?;
                platform.blit_lut_at(&mut pixels, 320, 0, lut)?;
            } else {
                background.blit(&mut pixels, 320, -1)?;
                platform.blit(&mut pixels, 320, -1)?;
            }
            actor_effect_first.blit_at(&mut pixels, 320)?;
            let frame = idle.get(idle_index).ok_or_else(|| game_err!("指令 {id} 缺少尾段人物畫格"))?;
            frame.blit_at(&mut pixels, 320)?;
            idle_repeat += 1;
            if idle_repeat >= frame.delay {
                idle_repeat = 0;
                idle_index = (idle_index + 1) % idle.len();
            }
            tail.push(command0_image(pixels, palette));
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
            let (actor, target) = (&units[actor_index], &units[confirmed]);
            let pose = dir_toward(actor.x, actor.y, target.x, target.y);
            units[actor_index].set_map_pose(pose);
            g.msg = format!("原始指令 {id}：命中 {hit}，傷害 {damage}");
            let after = g.native_ai_action_after.take();
            g.finish_successful_unit_action(Some(actor_index), after);
            g.check_result();
        });
        let job = NativeCommand5PresentationJob {
            command_id: id,
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
            target_hp_before: targets.iter().map(|target| target.hp).collect(),
            actor_sfx,
            target_sfx,
            then: Some(then),
            ..Default::default()
        };
        self.ai_focus_cursor(actor.x, actor.y);
        self.ai_focus_cursor(origin.x, origin.y);
        self.st.as_mut().unwrap().native_experience_accumulator = 0;
        self.native_cmd5_presentation = Some(job);
        Ok(())
    }

    pub(crate) fn present_native_indexed_damage_command(&mut self, screen: Option<&mut Picture>) {
        let job = self.native_cmd5_presentation.as_mut().unwrap();
        let index = job.frame as usize;
        let image = match job.phase {
            NATIVE_COMMAND5_COMMON_PRELUDE => job.prelude.get(index),
            NATIVE_COMMAND5_ACTOR => {
                let pulse = job.actor_specs.get(index).is_some_and(|spec| spec.pulse) && !job.pulse_black;
                (if pulse { &job.actor_pulse } else { &job.actor_black }).get(index)
            }
            NATIVE_COMMAND5_HANDLER => job.handler.get(index).map(|frame| &frame.image),
            NATIVE_COMMAND5_COMMON_TAIL => job.tail.get(index),
            _ => None,
        };
        let Some(image) = image.and_then(Option::as_deref) else {
            self.fail_native_command5_presentation(&game_err!("指令演出畫格無效"));
            return;
        };
        if let Some(screen) = screen {
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            screen.draw_image(image, &options);
        }
        job.drawn = true;
    }

    /// Go：`native_command5_presentation.go:441`，`func (g *Game) stepNativeCommand5Presentation()`
    pub(crate) fn step_native_command5_presentation(&mut self) {
        let Some(j) = self.native_cmd5_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        match j.phase {
            NATIVE_COMMAND5_COMMON_PRELUDE => {
                j.frame += 1;
                if j.frame >= j.prelude.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND5_ACTOR, 0);
                }
            }
            NATIVE_COMMAND5_ACTOR => {
                let spec = &j.actor_specs[j.frame as usize];
                let (spec_pulse, spec_publish_mp) = (spec.pulse, spec.publish_mp);
                if spec_pulse && !j.pulse_black {
                    if spec_publish_mp && !j.mp_published {
                        if let Err(err) = self.publish_native_command5_mp() {
                            self.fail_native_command5_presentation(&err);
                            return;
                        }
                        let sfx =
                            self.native_cmd5_presentation.as_ref().expect("Go：j 仍是同一個工作").actor_sfx.clone();
                        self.play_raw(&sfx);
                    }
                    let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    j.pulse_black = true;
                    return;
                }
                if spec_publish_mp && !j.mp_published {
                    if let Err(err) = self.publish_native_command5_mp() {
                        self.fail_native_command5_presentation(&err);
                        return;
                    }
                    let sfx = self.native_cmd5_presentation.as_ref().expect("Go：j 仍是同一個工作").actor_sfx.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.pulse_black = false;
                j.frame += 1;
                if j.frame >= j.actor_black.len() as i64 {
                    if !j.mp_published {
                        self.fail_native_command5_presentation(&game_err!("actor MP marker was not presented"));
                        return;
                    }
                    (j.phase, j.frame) = (NATIVE_COMMAND5_HANDLER, 0);
                }
            }
            NATIVE_COMMAND5_HANDLER => {
                let frame = &j.handler[j.frame as usize];
                let (target_index, hp_stage) = (frame.target_index, frame.hp_stage);
                let (play1, play2) = (frame.play_primary, frame.play_secondary);
                if hp_stage != 0 {
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) =
                        battle::apply_native_command_damage_stage(units, j.plan.as_mut(), target_index, hp_stage)
                    {
                        self.fail_native_command5_presentation(&err.into());
                        return;
                    }
                }
                if play1 {
                    let sfx = self.native_cmd5_presentation.as_ref().expect("Go：j 仍是同一個工作").target_sfx.clone();
                    self.play_raw(&sfx);
                }
                if play2 {
                    let sfx = self.native_cmd5_presentation.as_ref().expect("Go：j 仍是同一個工作").target_sfx.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.frame += 1;
                if j.frame < j.handler.len() as i64 {
                    return;
                }
                (j.phase, j.frame) = (NATIVE_COMMAND5_COMMON_TAIL, 0);
            }
            NATIVE_COMMAND5_COMMON_TAIL => {
                j.frame += 1;
                if j.frame < j.tail.len() as i64 {
                    return;
                }
                // 未命中的目標走 0x2AF61 分支，沒有 HP 分段畫格；數值上 HP 不變，直接收到終段。
                let (misses, damage_stages) = j.plan.as_ref().map_or((Vec::new(), 0), |plan| {
                    let misses = plan
                        .results
                        .iter()
                        .enumerate()
                        .filter(|(_, result)| j.command_id != 1 && !result.native_command_damage.hit)
                        .map(|(index, _)| index as i64)
                        .collect::<Vec<_>>();
                    (misses, plan.damage_stages)
                });
                for index in misses {
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) =
                        battle::apply_native_command_damage_stage(units, j.plan.as_mut(), index, damage_stages)
                    {
                        self.fail_native_command5_presentation(&err.into());
                        return;
                    }
                }
                // plan.RNGAfter 已含 handler 的 mode 0／重置／marker 抖動與每個目標的
                // 0x1C75E 擲骰（r4 收據 seq 1451：18 步），是整段演出結束時的全域亂數。
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
                    self.fail_native_command5_presentation(&err.into());
                    return;
                }
                let plan = j.plan.as_ref().expect("Go：CompleteNativeCommandDamage 已檢查 plan");
                self.native_rng_state = plan.rng_after;
                let results = plan.results.clone();
                let then = j.then.take();
                self.native_cmd5_presentation = None;
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Go `stepNativeCommand5Presentation` 兩處相同的 MP 發布：`ApplyNativeCommandDamageMP` 成功後設 `mpPublished`。
    fn publish_native_command5_mp(&mut self) -> Result<()> {
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
        battle::apply_native_command_damage_mp(units, j.plan.as_mut())?;
        j.mp_published = true;
        Ok(())
    }

    /// Go：`native_command5_presentation.go:426`，`func (g *Game) failNativeCommand5Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand5Presentation` 補移植）。
    pub(crate) fn fail_native_command5_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd5_presentation.take() else {
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
        self.load_err = format!("native command{} presentation: {}", j.command_id, err.message());
    }
}

#[cfg(test)]
#[path = "native_command5_presentation/native_command5_presentation_test.rs"]
mod native_command5_presentation_test;
