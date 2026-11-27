//! Go 來源：`remake/cmd/fd2/native_command7_presentation.go`（511 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command7_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand7PresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND7_COMMON_PRELUDE: NativeCommand7PresentationPhase = NativeCommand7PresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND7_ACTOR: NativeCommand7PresentationPhase = NativeCommand7PresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND7_HANDLER: NativeCommand7PresentationPhase = NativeCommand7PresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND7_COMMON_TAIL: NativeCommand7PresentationPhase = NativeCommand7PresentationPhase(3);

/// Go：`native_command7_presentation.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand7HandlerFrame {
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) target_index: i64,
    pub(crate) hp_stage: i64,
    pub(crate) play_handle_a: bool,
    pub(crate) play_handle_b: bool,
}

/// Go：`native_command7_presentation.go:32`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand7PresentationJob {
    pub(crate) hd: Option<ModernHDOrdinaryFrames>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) handler: Vec<NativeCommand7HandlerFrame>,
    pub(crate) tail: Vec<Option<Arc<Picture>>>,
    pub(crate) phase: NativeCommand7PresentationPhase,
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
    /// 敵方指令 7：前奏 → 施放者 → 前段、逐目標、目標間轉場、尾段 → 共用尾段。原版索引色分支，HD 另期。
    ///
    /// Go：`native_command7_presentation.go:51`，`func (g *Game) startNativeCommand7Presentation(...)`；AI 與玩家入口共用下方的建立函式。
    pub(crate) fn execute_native_ai_command7(&mut self, action: &battle::AIPlan) -> Result<()> {
        let actor = action.u.ok_or_else(|| game_err!("指令 7缺少施放者"))?;
        let target = action.target.ok_or_else(|| game_err!("指令 7缺少目標"))?;
        let origin = action.native_action_destination;

        let then = native_ai_damage_command_then(7, actor, target);
        let job = self.build_native_command7_presentation(actor, target, Some(origin), then)?;
        self.begin_native_ai_command_job(actor, origin);
        self.native_cmd7_presentation = Some(job);
        Ok(())
    }

    /// 玩家確認後的入口：以確認的目標建計畫，完成回呼由呼叫端決定。
    pub(crate) fn start_native_command7_presentation(
        &mut self,
        actor: UnitIdx,
        confirmed: UnitIdx,
        then: NativeCommandDamageThen,
    ) -> Result<()> {
        let job = self.build_native_command7_presentation(actor, confirmed, None, then)?;
        self.native_cmd7_presentation = Some(job);
        Ok(())
    }

    fn build_native_command7_presentation(
        &mut self,
        actor_index: UnitIdx,
        confirmed: UnitIdx,
        ai_origin: Option<battle::Cell>,
        then: NativeCommandDamageThen,
    ) -> Result<NativeCommand7PresentationJob> {
        if !self.current_native_system_options().full_presentation_enabled()
            || self.native_cmd7_presentation.is_some()
            || self.native_command_scene.is_none()
            || self.native_command_palette_flash.is_none()
        {
            return Err(game_err!("native command7 presentation context unavailable"));
        }
        let state = self.st.as_ref().ok_or_else(|| game_err!("指令 7 缺少戰場"))?;
        let actor_index = Some(actor_index)
            .filter(|&index| index < state.units.len())
            .ok_or_else(|| game_err!("指令 7 缺少施放者"))?;
        let confirmed =
            Some(confirmed).filter(|&index| index < state.units.len()).ok_or_else(|| game_err!("指令 7 缺少目標"))?;
        let actor = state.units[actor_index].clone();
        let luts = self
            .native_map_assets
            .as_ref()
            .filter(|assets| assets.luts.len() > 14)
            .map(|assets| assets.luts.clone())
            .ok_or_else(|| game_err!("native command7 raw actor provenance unavailable"))?;
        if !actor.has_battle_fig
            || !actor.has_native_record_byte6
            || !(0..=255).contains(&actor.battle_fig)
            || self.native_ui_palette.len() != 256
        {
            return Err(game_err!("native command7 raw actor provenance unavailable"));
        }
        let side = actor.native_record_byte6;
        let animations = separated_asset_path("animations");
        let effect_resource = if side == 0 { 38 } else { 37 };
        let effect = figani::load_separated_archive_resource(&animations, "FDOTHER.DAT", effect_resource)?;
        let schedule = figani::build_native_command7_presentation_schedule(side, Some(&effect))?;
        let rng = self.native_rng_state;
        let origin = native_command_plan_origin(&actor, ai_origin)?;
        let mut walk = |count: i64, resolve: &mut dyn FnMut(i64, u16) -> battle::Result<(u16, bool)>| {
            let mut record =
                |index: i64, before: u16| resolve(index, before).map_err(|err| Box::new(err) as figani::ResolveError);
            figani::walk_native_command7_rng(rng, schedule, side, count, Some(&mut record))
                .map_err(|err| battle::BattleError::new(err.error.to_string()))
        };
        let resistances = &state.native_command_resistances;
        let plan = if actor.camp == battle::ENEMY {
            state.plan_native_ai_command_damage_walk(Some(actor_index), origin, 7, resistances, rng, Some(&mut walk))?
        } else {
            state.plan_native_command_damage_walk(
                Some(actor_index),
                Some(confirmed),
                7,
                resistances,
                rng,
                Some(&mut walk),
            )?
        };
        if plan.results.is_empty()
            || plan.damage_stages != figani::NATIVE_COMMAND7_DAMAGE_STAGES
            || state.native_command_book.len() as i64 != battle::NATIVE_COMMAND_RECORD_COUNT
            || state.native_command_book[7].effect_mode != 2
        {
            return Err(game_err!("native command7 final targets unavailable"));
        }
        let (indices, targets): (Vec<_>, Vec<_>) = plan
            .results
            .iter()
            .map(|result| {
                let index =
                    result.target.ok_or_else(|| game_err!("native command7 raw target provenance unavailable"))?;
                let unit = &state.units[index];
                if !unit.has_battle_fig || !unit.has_native_record_byte6 {
                    return Err(game_err!("native command7 raw target provenance unavailable"));
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
        if !os_mute_or_shot(Some(self)) && (self.sfx_command7_actor.is_empty() || self.sfx_command7_target.is_empty()) {
            return Err(game_err!("native command7 converted #88 samples unavailable"));
        }
        let (Some(actor_idle_first), Some(target_idle_first)) =
            (actor_idle.frames.first(), target_idle[0].frames.first())
        else {
            return Err(game_err!("指令 7 缺少開場人物畫格"));
        };
        let panel_assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let (mut dac, _) = load_native_battle_palette()?;
        let actor_record = battle::native_battle_panel_record_for_unit(Some(&actor))?;
        let mut actor_after = actor.clone();
        actor_after.mp = plan.mp_after;
        let actor_after_record = battle::native_battle_panel_record_for_unit(Some(&actor_after))?;
        let stages = figani::NATIVE_COMMAND7_DAMAGE_STAGES;
        let mut bases = Vec::with_capacity(targets.len());
        for ((&index, target), result) in indices.iter().zip(&targets).zip(&plan.results) {
            let mut staged = target.clone();
            let mut target_bases = Vec::new();
            for stage in 0..=stages {
                staged.hp = result.hp_before - (result.hp_before - result.hp_after) * stage / stages;
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
                lut: &luts[11],
            })?;
        // Go 以 nativeCommandPaletteFlash.Entries[7] 做脈衝。
        let flash = self
            .native_command_palette_flash
            .as_ref()
            .filter(|table| table.entries.len() as i64 == battle::NATIVE_COMMAND_RECORD_COUNT)
            .map(|table| table.entries[7])
            .ok_or_else(|| game_err!("native command7 DAC inputs unavailable"))?;
        if dac.len() != 256 * 3 {
            return Err(game_err!("native command7 DAC inputs unavailable"));
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
        let tail_base = &bases[bases.len() - 1][last];
        let sequence =
            battlepresent::build_native_command7_effect_sequence(&battlepresent::NativeCommand7EffectInput {
                front_base: &bases[0][0],
                tail_base,
                target_bases: bases.iter().map(|frames| frames.iter().map(Vec::as_slice).collect()).collect(),
                transition_bases: bases[..bases.len() - 1].iter().map(|frames| frames[last].as_slice()).collect(),
                actor_effect: Some(&actor_effect),
                target_idle: target_idle.iter().map(Some).collect(),
                effect: Some(&effect),
                schedule,
                raw_side: side,
            })?;
        let (next_frame, next_repeat) = (sequence.next_idle_frame, sequence.next_idle_repeat);
        let mut handler = Vec::new();
        let mut push = |frames: Vec<battlepresent::NativeCommand7RenderedFrame>, target_index: i64| {
            handler.extend(frames.into_iter().map(|rendered| NativeCommand7HandlerFrame {
                image: image(rendered.pixels),
                target_index,
                hp_stage: rendered.hp_stage,
                play_handle_a: rendered.play_handle_a,
                play_handle_b: rendered.play_handle_b,
            }));
        };
        push(sequence.front, -1);
        let mut transitions = sequence.transitions.into_iter();
        for (target_index, frames) in sequence.targets.into_iter().enumerate() {
            push(frames, target_index as i64);
            if let Some(frames) = transitions.next() {
                push(frames, -1);
            }
        }
        push(sequence.tail, -1);
        if handler.is_empty() {
            return Err(game_err!("native command7 handler frames unavailable"));
        }
        let last_idle = &target_idle[target_idle.len() - 1];
        let next_frame =
            usize::try_from(next_frame).map_err(|_| game_err!("native command7 tail inputs unavailable"))?;
        let tail = super::native_command2_presentation::native_command2_tail_pixels(
            tail_base,
            &background,
            &platform,
            &actor_effect,
            last_idle,
            next_frame,
            next_repeat,
            &luts,
        )?
        .into_iter()
        .map(image)
        .collect();
        let job = NativeCommand7PresentationJob {
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
            then: Some(then),
            ..Default::default()
        };
        Ok(job)
    }

    /// 依階段取目前畫格畫到畫面並標記已呈現；步進由 `step_native_command7_presentation` 推進。
    ///
    /// Go：`native_command7_presentation.go:467`，`func (g *Game) drawNativeCommand7Presentation(screen *ebiten.Image) bool`
    /// 的原版索引色分支。
    pub(crate) fn present_native_command7(&mut self, screen: Option<&mut Picture>) {
        let job = self.native_cmd7_presentation.as_mut().unwrap();
        let index = job.frame as usize;
        let image = match job.phase {
            NATIVE_COMMAND7_COMMON_PRELUDE => job.prelude.get(index),
            NATIVE_COMMAND7_ACTOR => {
                let pulse = job.actor_specs.get(index).is_some_and(|spec| spec.pulse) && !job.pulse_black;
                (if pulse { &job.actor_pulse } else { &job.actor_black }).get(index)
            }
            NATIVE_COMMAND7_HANDLER => job.handler.get(index).map(|frame| &frame.image),
            NATIVE_COMMAND7_COMMON_TAIL => job.tail.get(index),
            _ => None,
        };
        let Some(image) = image.and_then(Option::as_deref) else {
            self.fail_native_command7_presentation(&game_err!("指令 7 演出畫格無效"));
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
    /// Go：`native_command7_presentation.go:386`，`func (g *Game) stepNativeCommand7Presentation()`
    pub(crate) fn step_native_command7_presentation(&mut self) {
        let Some(j) = self.native_cmd7_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        match j.phase {
            NATIVE_COMMAND7_COMMON_PRELUDE => {
                j.frame += 1;
                if j.frame >= j.prelude.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND7_ACTOR, 0);
                }
            }
            NATIVE_COMMAND7_ACTOR => {
                let spec = &j.actor_specs[j.frame as usize];
                let (spec_pulse, spec_publish_mp) = (spec.pulse, spec.publish_mp);
                if spec_pulse && !j.pulse_black {
                    if spec_publish_mp && !j.mp_published {
                        if let Err(err) = self.publish_native_command7_mp() {
                            self.fail_native_command7_presentation(&err);
                            return;
                        }
                        let sfx = self.sfx_command7_actor.clone();
                        self.play_raw(&sfx);
                    }
                    let j = self.native_cmd7_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    j.pulse_black = true;
                    return;
                }
                if spec_publish_mp && !j.mp_published {
                    if let Err(err) = self.publish_native_command7_mp() {
                        self.fail_native_command7_presentation(&err);
                        return;
                    }
                    let sfx = self.sfx_command7_actor.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd7_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.pulse_black = false;
                j.frame += 1;
                if j.frame >= j.actor_black.len() as i64 {
                    if !j.mp_published {
                        self.fail_native_command7_presentation(&game_err!("actor MP marker was not presented"));
                        return;
                    }
                    (j.phase, j.frame) = (NATIVE_COMMAND7_HANDLER, 0);
                }
            }
            NATIVE_COMMAND7_HANDLER => {
                let frame = &j.handler[j.frame as usize];
                let (target_index, hp_stage) = (frame.target_index, frame.hp_stage);
                let (play1, play2) = (frame.play_handle_a, frame.play_handle_b);
                if hp_stage != 0 {
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd7_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) =
                        battle::apply_native_command_damage_stage(units, j.plan.as_mut(), target_index, hp_stage)
                    {
                        self.fail_native_command7_presentation(&err.into());
                        return;
                    }
                }
                if play1 {
                    let sfx = self.sfx_command7_target.clone();
                    self.play_raw(&sfx);
                }
                if play2 {
                    let sfx = self.sfx_command7_target.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd7_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.frame += 1;
                if j.frame < j.handler.len() as i64 {
                    return;
                }
                (j.phase, j.frame) = (NATIVE_COMMAND7_COMMON_TAIL, 0);
            }
            NATIVE_COMMAND7_COMMON_TAIL => {
                j.frame += 1;
                if j.frame < j.tail.len() as i64 {
                    return;
                }
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd7_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
                    self.fail_native_command7_presentation(&err.into());
                    return;
                }
                let plan = j.plan.as_ref().expect("Go：CompleteNativeCommandDamage 已檢查 plan");
                self.native_rng_state = plan.rng_after;
                let results = plan.results.clone();
                let then = j.then.take();
                self.native_cmd7_presentation = None;
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Go `stepNativeCommand7Presentation` 兩處相同的 MP 發布：`ApplyNativeCommandDamageMP` 成功後設 `mpPublished`。
    fn publish_native_command7_mp(&mut self) -> Result<()> {
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd7_presentation.as_mut().expect("Go：j 仍是同一個工作");
        battle::apply_native_command_damage_mp(units, j.plan.as_mut())?;
        j.mp_published = true;
        Ok(())
    }

    /// Go：`native_command7_presentation.go:371`，`func (g *Game) failNativeCommand7Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand7Presentation` 補移植）。
    pub(crate) fn fail_native_command7_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd7_presentation.take() else {
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
        self.load_err = format!("native command7 presentation: {}", err.message());
    }
}

#[cfg(test)]
#[path = "native_command7_presentation/native_command7_presentation_test.rs"]
mod native_command7_presentation_test;
