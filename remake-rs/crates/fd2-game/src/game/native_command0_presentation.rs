//! Go 來源：`remake/cmd/fd2/native_command0_presentation.go`（546 行）。
//!
//! 骨架：Go 檔的 9 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command0_presentation.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand0PresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND0_PRELUDE: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND0_ACTOR: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND0_TARGET: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND0_TAIL: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(3);

/// Go：`native_command0_presentation.go:26`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand0PresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) target: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) target_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) target_impact: Vec<bool>,
    pub(crate) tail: Vec<Option<Arc<Picture>>>,
    pub(crate) hd: Option<ModernHDCommand0Frames>,
    pub(crate) phase: NativeCommand0PresentationPhase,
    pub(crate) frame: i64,
    pub(crate) pulse_black: bool,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) damage_stages: i64,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
}

impl NativeCommand0PresentationJob {
    fn current_image(&self) -> Option<&Picture> {
        let index = usize::try_from(self.frame).ok()?;
        let images = match self.phase {
            NATIVE_COMMAND0_PRELUDE => &self.prelude,
            NATIVE_COMMAND0_ACTOR => {
                if self.actor_specs.get(index)?.pulse && !self.pulse_black {
                    &self.actor_pulse
                } else {
                    &self.actor_black
                }
            }
            NATIVE_COMMAND0_TARGET => &self.target_frames,
            NATIVE_COMMAND0_TAIL => &self.tail,
            _ => return None,
        };
        images.get(index)?.as_deref()
    }

    pub(crate) fn present(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(image) = self.current_image() else { return false };
        if let Some(screen) = screen {
            let mut draw = DrawImageOptions::default();
            draw.geo_m.scale(2.0, 2.0);
            screen.draw_image(image, &draw);
        }
        self.drawn = true;
        true
    }
}

fn command0_terrain_control(map: &MapData, unit: &battle::Unit) -> Result<[u8; 4]> {
    if !(0..map.w).contains(&unit.x)
        || !(0..map.h).contains(&unit.y)
        || map.w.checked_mul(map.h) != Some(map.tiles.len() as i64)
    {
        return Err(game_err!("指令演出缺少原版地形格"));
    }
    let tile = map.tiles[(unit.y * map.w + unit.x) as usize];
    let (controls, remainder) = map.native_terrain_control.as_chunks::<4>();
    if !(0..=0x3ff).contains(&tile) || !remainder.is_empty() {
        return Err(game_err!("指令演出的地形控制表無效"));
    }
    controls.get(tile as usize).copied().ok_or_else(|| game_err!("指令演出缺少地形控制列 {tile}"))
}

pub(super) fn command0_image(pixels: Vec<u8>, palette: &ColorPalette) -> Option<Arc<Picture>> {
    Some(Arc::new(Picture::from_indexed(&IndexedFrame {
        width: 320,
        height: 200,
        pix: pixels,
        palette: palette.clone(),
    })))
}

impl Game {
    pub(super) fn native_damage_command_scenery(
        &self,
        actor: &battle::Unit,
        targets: &[battle::Unit],
    ) -> Result<(fdother::Frame, fdother::Frame)> {
        let initial = self
            .native_command_scene
            .as_ref()
            .ok_or_else(|| game_err!("指令演出缺少場景表"))?
            .initial_background(self.handler_chapter)?;
        let map = self.m.as_ref().ok_or_else(|| game_err!("指令演出缺少地圖"))?;
        let actor_control = command0_terrain_control(map, actor)?;
        let actor_gate = battle::native_command_background_gate(Some(actor))?;
        let actor_selector = if actor_gate && initial != 0 { initial } else { actor_control[2] };
        let target_controls = targets
            .iter()
            .map(|target| {
                Ok(fdicon::NativeCommandBackgroundTarget {
                    gate: battle::native_command_background_gate(Some(target))?,
                    control: command0_terrain_control(map, target)?,
                })
            })
            .collect::<Result<Vec<_>>>()?;
        let target_selector = fdicon::native_command_background_selector(initial, &target_controls);
        let (bg, tai) = if actor.native_record_byte6 == 0 {
            (actor_selector, target_selector)
        } else {
            (target_selector, actor_selector)
        };
        let surfaces = separated_asset_path("surfaces");
        let mut background = fdother::load_separated_single_frame(&surfaces, "BG.DAT", i64::from(bg))?;
        let mut platform = fdother::load_separated_single_frame(&surfaces, "TAI.DAT", i64::from(tai))?;
        (background.x, background.y) = (0, 50);
        (platform.x, platform.y) = (164, 157);
        Ok((background, platform))
    }

    pub(super) fn native_damage_command_base(
        &mut self,
        panel_assets: &battle::NativeItemPanelDataAssets,
        panels: [(UnitIdx, &[u8]); 2],
        background: &fdother::Frame,
        platform: Option<&fdother::Frame>,
    ) -> Result<Vec<u8>> {
        let mut pixels = vec![0; 320 * 200];
        for (index, record) in panels {
            self.render_localized_native_battle_panel(
                panel_assets,
                record,
                &mut pixels,
                index as i64,
                self.handler_chapter,
            )?;
        }
        background.blit(&mut pixels, 320, -1)?;
        if let Some(platform) = platform {
            platform.blit(&mut pixels, 320, -1)?;
        }
        Ok(pixels)
    }

    pub(crate) fn execute_native_ai_command0(&mut self, action: &battle::AIPlan) -> Result<()> {
        let actor = action.u.ok_or_else(|| game_err!("指令 0 缺少施放者"))?;
        let target = action.target.ok_or_else(|| game_err!("指令 0 缺少目標"))?;
        let job = self.build_native_command0_presentation(actor, target, Some(action.native_action_destination))?;
        let state = self.st.as_ref().expect("已驗證指令 0 戰場");
        let (x, y) = (state.units[actor].x, state.units[actor].y);
        self.ai_focus_cursor(x, y);
        self.ai_focus_cursor(action.native_action_destination.x, action.native_action_destination.y);
        self.st.as_mut().expect("已驗證指令 0 戰場").native_experience_accumulator = 0;
        self.native_cmd0_presentation = Some(job);
        Ok(())
    }

    fn build_native_command0_presentation(
        &mut self,
        actor_index: UnitIdx,
        target_index: UnitIdx,
        ai_origin: Option<battle::Cell>,
    ) -> Result<NativeCommand0PresentationJob> {
        if !self.current_native_system_options().full_presentation_enabled() {
            return Err(game_err!("指令 0 的簡略演出尚無執行入口"));
        }
        if self.native_cmd0_presentation.is_some() || self.atk.is_some() {
            return Err(game_err!("另一個戰鬥演出尚未完成"));
        }
        let state = self.st.as_ref().ok_or_else(|| game_err!("指令 0 缺少戰場"))?;
        let actor = state.units.get(actor_index).ok_or_else(|| game_err!("指令 0 的施放者不在名冊"))?.clone();
        let target = state.units.get(target_index).ok_or_else(|| game_err!("指令 0 的目標不在名冊"))?.clone();
        if !actor.has_battle_fig
            || !target.has_battle_fig
            || !actor.has_native_record_byte6
            || !target.has_native_record_byte6
            || self.native_ui_palette.len() != 256
        {
            return Err(game_err!("指令 0 缺少原版人物或調色盤"));
        }
        let luts = self
            .native_map_assets
            .as_ref()
            .and_then(|assets| assets.luts.get(11..15))
            .ok_or_else(|| game_err!("指令 0 缺少地形調色表"))?
            .to_vec();
        if !os_mute_or_shot(Some(self)) && (self.sfx_command0_actor.is_empty() || self.sfx_command0_target.is_empty()) {
            return Err(game_err!("指令 0 缺少原版音效 82/0 或 82/1"));
        }

        let animations = separated_asset_path("animations");
        let side = actor.native_record_byte6;
        let effect =
            figani::load_separated_archive_resource(&animations, "FDOTHER.DAT", if side == 0 { 20 } else { 18 })?;
        let schedule = figani::build_native_command0_presentation_schedule(side, Some(&effect))?;
        let rng_state = self.native_rng_state;
        let mut walk = |count: i64, resolve: &mut dyn FnMut(i64, u16) -> battle::Result<(u16, bool)>| {
            let mut bridge = |index, rng| resolve(index, rng).map_err(|err| Box::new(err) as figani::ResolveError);
            figani::walk_native_command0_rng(rng_state, schedule, count, Some(&mut bridge))
                .map_err(|err| battle::BattleError::new(err.error.to_string()))
        };
        let resistances = &state.native_command_resistances;
        let plan = if actor.camp == battle::ENEMY {
            let origin = ai_origin.ok_or_else(|| game_err!("指令 0 缺少敵方選定的目標格"))?;
            state.plan_native_ai_command_damage_walk(
                Some(actor_index),
                origin,
                0,
                resistances,
                rng_state,
                Some(&mut walk),
            )?
        } else {
            if resistances.is_empty() {
                return Err(game_err!("指令 0 缺少抗性表"));
            }
            state.plan_native_command_damage_walk(
                Some(actor_index),
                Some(target_index),
                0,
                resistances,
                rng_state,
                Some(&mut walk),
            )?
        };
        let [result] = plan.results.as_slice() else {
            return Err(game_err!("指令 0 需要單一原版目標"));
        };
        if result.target != Some(target_index) || state.native_command_book[0].effect_mode != 0 {
            return Err(game_err!("指令 0 的效果模式或目標不符"));
        }

        let (background, platform) = self.native_damage_command_scenery(&actor, std::slice::from_ref(&target))?;
        let actor_idle = figani::load_separated_resource(&animations, actor.battle_fig * 3)?;
        let actor_effect =
            figani::load_separated_resource_with_zero_header_fallback(&animations, actor.battle_fig * 3 + 2)?;
        let target_idle = figani::load_separated_resource(&animations, target.battle_fig * 3)?;
        let (Some(actor_idle_first), Some(target_idle_first), Some(actor_effect_first)) =
            (actor_idle.frames.first(), target_idle.frames.first(), actor_effect.frames.first())
        else {
            return Err(game_err!("指令 0 缺少開場人物畫格"));
        };
        let (mut dac, _) = load_native_battle_palette()?;
        let panel_assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let mut actor_after = actor.clone();
        actor_after.mp = plan.mp_after;
        let actor_before_record = battle::native_battle_panel_record_for_unit(Some(&actor))?;
        let actor_after_record = battle::native_battle_panel_record_for_unit(Some(&actor_after))?;
        let target_before_record = battle::native_battle_panel_record_for_unit(Some(&target))?;
        let mut base = |actor_record: &[u8], target_record: &[u8], include_platform: bool| {
            self.native_damage_command_base(
                &panel_assets,
                [(actor_index, actor_record), (target_index, target_record)],
                &background,
                include_platform.then_some(&platform),
            )
        };
        let prelude_base = base(&actor_before_record, &target_before_record, false)?;
        let actor_base = base(&actor_before_record, &target_before_record, true)?;
        let mut target_bases = Vec::new();
        let mut staged = target.clone();
        for stage in 0..=schedule.damage_stages {
            staged.hp = result.hp_before - (result.hp_before - result.hp_after) * stage / schedule.damage_stages;
            let target_record = battle::native_battle_panel_record_for_unit(Some(&staged))?;
            target_bases.push(base(&actor_after_record, &target_record, true)?);
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
                base_after: &target_bases[0],
                actor_effect: Some(&actor_effect),
                first_target_idle: Some(&target_idle),
                raw_side: side,
                background: background.clone(),
                platform: platform.clone(),
                lut: &luts[0],
            })?;
        let flash = self
            .native_command_palette_flash
            .as_ref()
            .filter(|table| table.entries.len() as i64 == battle::NATIVE_COMMAND_RECORD_COUNT)
            .ok_or_else(|| game_err!("指令 0 缺少閃光調色表"))?;
        dac[..3].copy_from_slice(&flash.entries[0]);
        let pulse = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&dac)?);
        let palette = &self.native_ui_palette;
        let actor_black = actor_specs.iter().map(|spec| command0_image(spec.pixels.clone(), palette)).collect();
        let actor_pulse = actor_specs.iter().map(|spec| command0_image(spec.pixels.clone(), &pulse)).collect();
        let mut idle_index = 0;
        let mut idle_repeat = 0;
        let mut next_idle = || {
            let frame = &target_idle.frames[idle_index];
            idle_repeat += 1;
            if idle_repeat == frame.delay {
                idle_repeat = 0;
                idle_index = (idle_index + 1) % target_idle.frames.len();
            }
            frame
        };
        let mut stage = 0;
        let mut target_frames = Vec::new();
        let mut target_impact = Vec::new();
        for step in 0..schedule.frames {
            let pixels = battlepresent::compose_native_command0_target_frame(
                &target_bases[stage],
                next_idle(),
                Some(&effect),
                schedule,
                step,
            )?;
            let impact = (0..7).any(|layer| figani::native_command0_impact(step, layer));
            stage += usize::from(impact);
            target_frames.push(command0_image(pixels, palette));
            target_impact.push(impact);
        }
        if stage as i64 != schedule.damage_stages {
            return Err(game_err!("指令 0 的分段命中標記不足"));
        }
        let mut tail = Vec::new();
        for lut in &luts[1..] {
            let mut pixels = target_bases[stage].clone();
            background.blit_lut_at(&mut pixels, 320, 0, lut)?;
            platform.blit_lut_at(&mut pixels, 320, 0, lut)?;
            actor_effect_first.blit_at(&mut pixels, 320)?;
            next_idle().blit_at(&mut pixels, 320)?;
            tail.push(command0_image(pixels, palette));
        }
        let mut pixels = target_bases[stage].clone();
        background.blit(&mut pixels, 320, -1)?;
        platform.blit(&mut pixels, 320, -1)?;
        actor_effect_first.blit_at(&mut pixels, 320)?;
        next_idle().blit_at(&mut pixels, 320)?;
        tail.push(command0_image(pixels, palette));
        let then = Box::new(move |g: &mut Game, results: Vec<battle::NativeCommandDamageResult>| {
            for result in &results {
                g.award_death_reward(result.target, Some(actor_index));
            }
            let state = g.st.as_mut().expect("指令 0 結算保留戰場");
            let target = &state.units[target_index];
            let actor = &state.units[actor_index];
            let pose = dir_toward(actor.x, actor.y, target.x, target.y);
            state.units[actor_index].set_map_pose(pose);
            let hit = results.iter().any(|result| result.native_command_damage.hit);
            let damage: i64 = results.iter().map(|result| result.native_command_damage.damage).sum();
            g.msg = format!("原始指令 0：命中 {}，傷害 {damage}", i64::from(hit));
            g.finish_successful_unit_action(Some(actor_index), None);
            g.check_result();
        });
        Ok(NativeCommand0PresentationJob {
            actor: Some(actor_index),
            target: Some(target_index),
            plan: Some(plan),
            prelude,
            actor_black,
            actor_pulse,
            actor_specs,
            target_frames,
            target_impact,
            tail,
            actor_mp_before: actor.mp,
            actor_acted_before: actor.acted,
            target_hp_before: target.hp,
            then: Some(then),
            ..Default::default()
        })
    }
}

impl Game {
    /// Go：`native_command0_presentation.go:420`，`func (g *Game) stepNativeCommand0Presentation()`
    pub(crate) fn step_native_command0_presentation(&mut self) {
        let Some(j) = self.native_cmd0_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        match j.phase {
            NATIVE_COMMAND0_PRELUDE => {
                j.frame += 1;
                if j.frame >= j.prelude.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND0_ACTOR, 0);
                }
            }
            NATIVE_COMMAND0_ACTOR => {
                let spec = &j.actor_specs[j.frame as usize];
                let (spec_pulse, spec_publish_mp) = (spec.pulse, spec.publish_mp);
                if spec_pulse && !j.pulse_black {
                    if spec_publish_mp && !j.mp_published {
                        if let Err(err) = self.publish_native_command0_mp() {
                            self.fail_native_command0_presentation(&err);
                            return;
                        }
                        let sfx = self.sfx_command0_actor.clone();
                        self.play_raw(&sfx);
                    }
                    let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    j.pulse_black = true;
                    return;
                }
                if spec_publish_mp && !j.mp_published {
                    if let Err(err) = self.publish_native_command0_mp() {
                        self.fail_native_command0_presentation(&err);
                        return;
                    }
                    let sfx = self.sfx_command0_actor.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.pulse_black = false;
                j.frame += 1;
                if j.frame >= j.actor_black.len() as i64 {
                    if !j.mp_published {
                        self.fail_native_command0_presentation(&game_err!("actor MP marker was not presented"));
                        return;
                    }
                    (j.phase, j.frame) = (NATIVE_COMMAND0_TARGET, 0);
                }
            }
            NATIVE_COMMAND0_TARGET => {
                if j.target_impact[j.frame as usize] {
                    j.damage_stages += 1;
                    let stage = j.damage_stages;
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) = battle::apply_native_command_damage_stage(units, j.plan.as_mut(), 0, stage) {
                        self.fail_native_command0_presentation(&err.into());
                        return;
                    }
                    let sfx = self.sfx_command0_target.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.frame += 1;
                if j.frame >= j.target_frames.len() as i64 {
                    if j.damage_stages != figani::NATIVE_COMMAND0_DAMAGE_STAGES {
                        self.fail_native_command0_presentation(&game_err!("target impact stages incomplete"));
                        return;
                    }
                    (j.phase, j.frame) = (NATIVE_COMMAND0_TAIL, 0);
                }
            }
            NATIVE_COMMAND0_TAIL => {
                j.frame += 1;
                if j.frame < j.tail.len() as i64 {
                    return;
                }
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
                    self.fail_native_command0_presentation(&err.into());
                    return;
                }
                let plan = j.plan.as_ref().expect("Go：CompleteNativeCommandDamage 已檢查 plan");
                self.native_rng_state = plan.rng_after;
                let results = plan.results.clone();
                let then = j.then.take();
                self.native_cmd0_presentation = None;
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Go `stepNativeCommand0Presentation` 兩處相同的 MP 發布（`native_command0_presentation.go:438`、`447`）：
    /// `ApplyNativeCommandDamageMP` 成功後設 `mpPublished`。
    fn publish_native_command0_mp(&mut self) -> Result<()> {
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
        battle::apply_native_command_damage_mp(units, j.plan.as_mut())?;
        j.mp_published = true;
        Ok(())
    }

    /// Go：`native_command0_presentation.go:410`，`func (g *Game) failNativeCommand0Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand0Presentation` 補移植）。
    pub(crate) fn fail_native_command0_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd0_presentation.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        if let Some(actor) = j.actor {
            (units[actor].mp, units[actor].acted) = (j.actor_mp_before, j.actor_acted_before);
        }
        if let Some(target) = j.target {
            units[target].hp = j.target_hp_before;
        }
        self.load_err = format!("native command0 presentation: {}", err.message());
    }
}

/// R4 D 組新增，沒有 Go 對應：演出工作與計畫裡的 Go `*battle.Unit` 都指向 `g.st.Units`，Rust 是
/// `UnitIdx`，`fd2_rules` 的發布函式另收單位切片。Go 的演出工作只在戰場上建立，`g.st` 不會是 nil。
pub(crate) fn presentation_units(st: &mut Option<battle::State>) -> &mut [battle::Unit] {
    &mut st.as_mut().expect("Go：演出工作的單位指標指向 g.st.Units").units
}
