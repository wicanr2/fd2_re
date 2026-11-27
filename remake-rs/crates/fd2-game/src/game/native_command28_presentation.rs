//! Go 來源：`remake/cmd/fd2/native_command28_presentation.go`（413 行）。
//!
//! 四個函式都已移植；HD 繪製（`drawModernHDCompoundFrame`）與 HD 庫 stage 屬 R6。
//! `modernHDCommandFigure`／`modernHDCommandFrame`（Go `modern_hd_command0.go`）一併放在這裡。

use super::*;

/// Go：`native_command28_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand28Event {
    pub(crate) prelude: i64,
    pub(crate) effect: i64,
    pub(crate) tick: i64,
    pub(crate) target: i64,
    pub(crate) idle: i64,
    pub(crate) impact: i64,
    pub(crate) mp: bool,
}

/// The shared strike owner retains input, F2, rollback and action completion.
/// This sequence supplies command28's mode0, fixed scenery and eight impacts.
///
/// Go：`native_command28_presentation.go:22`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeCommand28Sequence {
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) events: Vec<NativeCommand28Event>,
    pub(crate) index: i64,
}

/// Go：`modern_hd_command0.go:37`，`func modernHDCommandFigure(animation *figani.Animation, resource, index int) modernHDBattleSource`
pub(crate) fn modern_hd_command_figure(
    animation: &figani::Animation,
    resource: i64,
    index: i64,
) -> ModernHDBattleSource {
    ModernHDBattleSource {
        frame: animation.frames[index as usize].clone(),
        resource,
        index,
        fill: -1,
        ..Default::default()
    }
}

/// Go：`modern_hd_command0.go:34`，`func modernHDCommandFrame(base []byte, layers []modernHDBattleSource, palette color.Palette, phase string) nativeCompoundPresentedFrame`
pub(crate) fn modern_hd_command_frame(
    base: &[u8],
    layers: Vec<ModernHDBattleSource>,
    palette: &ColorPalette,
    phase: &str,
) -> NativeCompoundPresentedFrame {
    NativeCompoundPresentedFrame {
        hd_sources: Some(ModernHDBattleSources { base: base.to_vec(), layers, ..Default::default() }),
        hd_palette: palette.clone(),
        hd_phase: phase.to_string(),
        ..Default::default()
    }
}

/// 以 960 寬的工作緩衝區疊圖層，兩側留白讓 320 寬的視窗自然裁切。
///
/// Go：`native_command28_presentation.go:28`，`func nativeCommand28Pixels(base []byte, layers []modernHDBattleSource) ([]byte, error)`
pub(crate) fn native_command28_pixels(base: &[u8], layers: &[ModernHDBattleSource]) -> Result<Vec<u8>> {
    if base.len() != 64000 {
        return Err(game_err!("native command28 base unavailable"));
    }
    const STRIDE: usize = 960;
    let mut work = vec![0u8; STRIDE * 200];
    for y in 0..200 {
        work[y * STRIDE + 320..y * STRIDE + 640].copy_from_slice(&base[y * 320..(y + 1) * 320]);
    }
    for layer in layers {
        layer.frame.blit_at_base(&mut work, STRIDE as i64, 320 + layer.dx)?;
    }
    let mut pixels = vec![0u8; 64000];
    for y in 0..200 {
        pixels[y * 320..(y + 1) * 320].copy_from_slice(&work[y * STRIDE + 320..y * STRIDE + 640]);
    }
    Ok(pixels)
}

/// Go `startNativeCommand28Presentation` 的 `appendFrame` closure。
#[allow(clippy::too_many_arguments)]
fn native_command28_append_frame(
    sequence: &mut NativeCommand28Sequence,
    base: &[u8],
    layers: Vec<ModernHDBattleSource>,
    surfaces: &[ModernHDBattleSource],
    palette: &ColorPalette,
    phase: &str,
    subtract: f32,
    event: NativeCommand28Event,
    sound: Vec<u8>,
    expected: Option<&[u8]>,
) -> Result<()> {
    let pixels = native_command28_pixels(base, &layers)?;
    if expected.is_some_and(|expected| expected != pixels.as_slice()) {
        return Err(game_err!("native command28 prelude layer reconstruction differs"));
    }
    let image = native_command24_indexed_images(&[pixels], palette)?.pop().flatten();
    let mut frame = modern_hd_command_frame(base, layers, palette, phase);
    bind_modern_physical_base_scenery(frame.hd_sources.as_mut(), &mut surfaces.to_vec())?;
    (frame.image, frame.delay, frame.sound, frame.hd_subtract) = (image, 1, sound, subtract);
    sequence.frames.push(frame);
    sequence.events.push(event);
    Ok(())
}

impl Game {
    /// 玩家指令 28（selector19，resource59）：mode0 前奏、施放者特效，之後每個目標八段受擊。
    /// 畫格整批預先合成，掛在指令 29 的工作上，由它保留輸入、回滾與行動完成。HD 繪製屬 R6；
    /// HD 來源仍照 Go 綁在畫格上，原版路徑只畫索引色畫格。
    ///
    /// Go：`native_command28_presentation.go:49`，`func (g *Game) startNativeCommand28Presentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommand24Damage)) error`
    pub(crate) fn start_native_command28_presentation(
        &mut self,
        actor_index: UnitIdx,
        confirmed: Option<UnitIdx>,
        then: Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>,
    ) -> Result<()> {
        let actor = self.st.as_ref().and_then(|st| st.units.get(actor_index)).cloned();
        let ok = self.current_native_system_options().full_presentation_enabled()
            && self.m.is_some()
            && self.rng.is_some()
            && self.native_ui_palette.len() == 256;
        let (true, Some(actor), Some(confirmed)) = (ok, actor, confirmed) else {
            return Err(game_err!("native command28 proven selector19 context unavailable"));
        };
        if !actor.has_battle_fig || actor.battle_fig != 19 {
            return Err(game_err!("native command28 proven selector19 context unavailable"));
        }
        if self.native_derived_strike_presentation_busy()
            || self.native_cmd9_player.is_some()
            || self.native_cmd1012.is_some()
            || self.native2189_a.is_some()
            || self.native_cmd32_presentation.is_some()
            || self.native_cmd33_presentation.is_some()
            || self.native_cmd34_presentation.is_some()
            || self.native_cmd35_presentation.is_some()
            || self.indexed_transition.is_some()
        {
            return Err(game_err!("native command28 presentation already active"));
        }
        let st = self.st.as_ref().unwrap();
        if st.native_command_book.len() as i64 != battle::NATIVE_COMMAND_RECORD_COUNT
            || st.native_command_book[28].id != 28
        {
            return Err(game_err!("native command28 record unavailable"));
        }
        let record = st.native_command_book[28];
        if !(0..=255).contains(&record.mp_cost) || actor.mp < record.mp_cost {
            return Err(game_err!("native command28 insufficient MP"));
        }
        let flags = st.native_command_base_flags()?;
        let targets = battle::native_command_effect_targets(
            st.w,
            st.h,
            Some(actor_index),
            Some(confirmed),
            record.selection_mode,
            record.effect_mode,
            record.target_code,
            &flags,
            &st.units,
        );
        let targets = match targets {
            Ok(targets) if !targets.is_empty() => targets,
            Ok(_) => return Err(game_err!("native command28 final targets unavailable: <nil>")),
            Err(err) => return Err(game_err!("native command28 final targets unavailable: {err}")),
        };
        let animations = separated_asset_path("animations");
        let effect = figani::load_separated_resource(&animations, 59)?;
        let schedule = figani::build_native_command_derived_strike_schedule(Some(&effect), 28)?;
        if effect.header_byte1 != 0
            || effect.frames.len() != 19
            || schedule.target_start != 7
            || schedule.actor_impact_frame != 6
            || schedule.audio_resource != 50
            || schedule.target_impact_frames.len() != 8
            || schedule.prelude_mode != 0
            || schedule.uses_bg_transition
            || !schedule.uses_target_base
        {
            return Err(game_err!("native command28 resource59 signature differs"));
        }
        if schedule.target_impact_frames.iter().enumerate().any(|(i, &impact)| impact != 7 + i as i64) {
            return Err(game_err!("native command28 resource59 impact order differs"));
        }
        let samples: Vec<i64> = schedule.actor_cues.iter().chain(&schedule.target_cues).map(|cue| cue.sample).collect();
        self.require_separated_command_sounds(schedule.audio_resource, &samples)?;
        self.preflight_native_derived_strike_sources(actor_index, &targets, &effect, 28)?;
        let actor_idle = figani::load_separated_resource(&animations, 57)?;
        let units = &self.st.as_ref().unwrap().units;
        let target_units: Vec<battle::Unit> = targets.iter().map(|&t| units[t].clone()).collect();
        let mut idles = Vec::with_capacity(targets.len());
        for target in &target_units {
            idles.push(figani::load_separated_resource(&animations, target.battle_fig * 3)?);
        }
        let actor_selector = native_command24_bg_selector(self.m.as_ref(), &actor)?;
        let target_selector = native_command24_bg_selector(self.m.as_ref(), &target_units[0])?;
        let surfaces = separated_asset_path("surfaces");
        let mut background = fdother::load_separated_single_frame(&surfaces, "BG.DAT", target_selector)?;
        let mut platform = fdother::load_separated_single_frame(&surfaces, "TAI.DAT", actor_selector)?;
        let scenery = modern_split_strike_scenery(&background, target_selector, None)?;
        let platform_source = modern_battle_platform_source(&platform, actor_selector)?;
        let mut all_scenery = scenery.clone();
        all_scenery.push(platform_source.clone());
        // Go 在 `g.modernStoryPortraits` 存在時預備 HD 背景來源並 stage HD 庫（R6）；原版路徑沒有它。
        let panel_assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let actor_unit_index = native_command24_runtime_unit_index(self.st.as_ref(), Some(actor_index))?;
        let chapter = self.handler_chapter;
        let panel = |g: &mut Game, base: &mut [u8], unit: &battle::Unit, index: i64| -> Result<()> {
            let raw = battle::native_battle_panel_record_for_unit(Some(unit))?;
            g.render_battle_panel_text(&panel_assets, &raw, base, index, chapter, None)
        };
        let mut initial = vec![0u8; 64000];
        panel(self, &mut initial, &actor, actor_unit_index)?;
        (background.x, background.y) = (0, 50);
        background.blit(&mut initial, 320, -1)?;
        let first_index = native_command24_runtime_unit_index(self.st.as_ref(), Some(targets[0]))?;
        panel(self, &mut initial, &target_units[0], first_index)?;
        let mut with_platform = initial.clone();
        (platform.x, platform.y) = (164, 157);
        platform.blit(&mut with_platform, 320, -1)?;
        let mut actor_after = actor.clone();
        actor_after.mp -= record.mp_cost;
        let mut after_mp = with_platform.clone();
        panel(self, &mut after_mp, &actor_after, actor_unit_index)?;
        let (dac, _) = load_native_battle_palette()?;
        let prelude = battlepresent::build_native_command_prelude_frames(&battlepresent::NativeCommandPreludeInput {
            base: &initial,
            actor_idle: actor_idle.frames[0].clone(),
            first_target_idle: idles[0].frames[0].clone(),
            platform: Some(&platform),
            raw_side: actor.native_record_byte6,
            mode: 0,
            baseline_dac: &dac,
        })?;
        let mut sequence = NativeCommand28Sequence::default();
        let first_resource = target_units[0].battle_fig * 3;
        for (i, present) in prelude.iter().enumerate() {
            let mut a = modern_hd_command_figure(&actor_idle, 57, 0);
            let b = modern_hd_command_figure(&idles[0], first_resource, 0);
            let (base, surfaces, layers) = if actor.native_record_byte6 == 0 {
                a.dx = -present.stage * 10;
                (&with_platform, &all_scenery, vec![a, b])
            } else {
                let mut moving = platform_source.clone();
                (moving.dx, a.dx) = (present.stage * 10, present.stage * 10);
                (&initial, &scenery, vec![b, moving, a])
            };
            let palette = color_palette_from_nrgba(&fdother::parse_vga_palette(&present.dac)?);
            native_command28_append_frame(
                &mut sequence,
                base,
                layers,
                surfaces,
                &palette,
                "prelude",
                (present.stage * 6) as f32 / 63.0,
                NativeCommand28Event { prelude: i as i64, ..Default::default() },
                Vec::new(),
                Some(&present.pixels),
            )?;
        }
        // All static input and HD checks precede the caller-owned damage RNG.
        let rng = self.rng.as_mut().map(|rng| rng as &mut dyn battle::GoRand);
        let plan = self.st.as_ref().unwrap().plan_native_command_derived_strike(
            Some(actor_index),
            Some(confirmed),
            28,
            rng,
        )?;
        if plan.target_impact_count != schedule.target_impact_frames.len() as i64 {
            return Err(game_err!("native command28 numeric/presentation impact count differs"));
        }
        let palette = self.native_ui_palette.clone();
        let (mut idle, mut idle_repeat) = (0i64, 0i64);
        for effect_index in 0..schedule.target_start {
            for tick in 0..schedule.frame_delays[effect_index as usize] {
                let base = if effect_index >= schedule.actor_impact_frame { &after_mp } else { &with_platform };
                let layers = vec![
                    modern_hd_command_figure(&idles[0], first_resource, idle),
                    modern_hd_command_figure(&effect, 59, effect_index),
                ];
                let raw5 = effect.frames[effect_index as usize].raw_byte5;
                let sound = if tick == 0 && raw5 != 0 {
                    self.separated_command_sound(schedule.audio_resource, i64::from(raw5))
                } else {
                    Vec::new()
                };
                let event = NativeCommand28Event {
                    prelude: 9,
                    effect: effect_index,
                    tick,
                    idle,
                    mp: effect_index == schedule.actor_impact_frame && tick == 0,
                    ..Default::default()
                };
                native_command28_append_frame(
                    &mut sequence,
                    base,
                    layers,
                    &all_scenery,
                    &palette,
                    "actor",
                    0.0,
                    event,
                    sound,
                    None,
                )?;
                idle_repeat += 1;
                if idle_repeat >= idles[0].frames[idle as usize].delay {
                    (idle_repeat, idle) = (0, (idle + 1) % idles[0].frames.len() as i64);
                }
            }
        }
        let mut base = after_mp;
        for (target_index, target) in target_units.iter().enumerate() {
            let index = native_command24_runtime_unit_index(self.st.as_ref(), Some(targets[target_index]))?;
            panel(self, &mut base, target, index)?;
            let resource = target.battle_fig * 3;
            let target_idle = &idles[target_index];
            let (mut idle, mut idle_repeat, mut impact, mut shake) = (0i64, 0i64, 0i64, -1i64);
            for effect_index in schedule.target_start..effect.frames.len() as i64 {
                let raw = &effect.frames[effect_index as usize];
                let is_impact = raw.raw_byte4 == 1;
                if is_impact {
                    (impact, shake) = (impact + 1, 5);
                    let units = &self.st.as_ref().unwrap().units;
                    let projection =
                        battle::native_command28_stage_result(units, Some(&plan), target_index as i64, impact)?;
                    let mut frozen = target.clone();
                    frozen.hp = projection.hp_after;
                    panel(self, &mut base, &frozen, index)?;
                }
                for tick in 0..schedule.frame_delays[effect_index as usize] {
                    let mut b = modern_hd_command_figure(target_idle, resource, idle);
                    if shake >= 0 {
                        b.dx = -schedule.shake_offsets[shake as usize];
                    }
                    let sound = if tick == 0 && raw.raw_byte5 != 0 {
                        self.separated_command_sound(schedule.audio_resource, i64::from(raw.raw_byte5))
                    } else {
                        Vec::new()
                    };
                    let stage = if tick == 0 && is_impact { impact } else { 0 };
                    let event = NativeCommand28Event {
                        prelude: 9,
                        effect: effect_index,
                        tick,
                        target: target_index as i64,
                        idle,
                        impact: stage,
                        mp: false,
                    };
                    native_command28_append_frame(
                        &mut sequence,
                        &base,
                        vec![b, modern_hd_command_figure(&effect, 59, effect_index)],
                        &all_scenery,
                        &palette,
                        &format!("target-{target_index}"),
                        0.0,
                        event,
                        sound,
                        None,
                    )?;
                    if shake >= 0 {
                        shake -= 1;
                    }
                    idle_repeat += 1;
                    if idle_repeat >= target_idle.frames[idle as usize].delay {
                        (idle_repeat, idle) = (0, (idle + 1) % target_idle.frames.len() as i64);
                    }
                }
            }
        }
        // Go `bindModernBattlePanelFrames(panelText, …)` 與 `publishHD()` 屬 R6。
        let job_targets = targets
            .iter()
            .zip(&target_units)
            .map(|(&unit, target)| NativeCommand29TargetPresentation {
                unit: Some(unit),
                hp_before: target.hp,
                raw_byte5_before: target.native_record_byte5,
                ..Default::default()
            })
            .collect();
        self.native_cmd29_presentation = Some(NativeCommand29PresentationJob {
            command28: Some(sequence),
            actor: Some(actor_index),
            plan: Some(plan),
            schedule,
            prelude_frames: vec![None; prelude.len()],
            targets: job_targets,
            transition_frame: -1,
            actor_mp_before: actor.mp,
            actor_acted_before: actor.acted,
            then: Some(then),
            ..Default::default()
        });
        Ok(())
    }

    /// Go：`native_command28_presentation.go:344`，`func (g *Game) stepNativeCommand28Presentation()`
    pub(crate) fn step_native_command28_presentation(&mut self) {
        let Some(j) = self.native_cmd29_presentation.as_mut().filter(|j| j.drawn) else {
            return;
        };
        let Some(event) = j.command28.as_ref().map(|sequence| sequence.events[sequence.index as usize]) else {
            return;
        };
        j.drawn = false;
        if event.mp {
            if j.mp_published {
                self.fail_native_command29_presentation(&game_err!("native command28 duplicate MP marker"));
                return;
            }
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::apply_native_command_derived_strike_mp(units, j.plan.as_ref()) {
                self.fail_native_command29_presentation(&err.into());
                return;
            }
            j.mp_published = true;
        }
        if event.impact > 0 {
            if !self.native_cmd29_presentation.as_ref().expect("Go：j 仍是同一個工作").mp_published {
                self.fail_native_command29_presentation(&game_err!("native command28 impact before MP marker"));
                return;
            }
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) =
                battle::apply_native_command28_target_stage(units, j.plan.as_mut(), event.target, event.impact)
            {
                self.fail_native_command29_presentation(&err.into());
                return;
            }
            j.targets[event.target as usize].damage_published = event.impact == 8;
        }
        let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
        let sequence = j.command28.as_mut().expect("Go：command28 仍是同一個序列");
        let sound = sequence.frames[sequence.index as usize].sound.clone();
        self.play_raw(&sound);
        let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
        let sequence = j.command28.as_mut().expect("Go：command28 仍是同一個序列");
        sequence.index += 1;
        if sequence.index < sequence.frames.len() as i64 {
            let next = sequence.events[sequence.index as usize];
            (j.prelude_frame, j.frame, j.repeat, j.target_index, j.idle_frame) =
                (next.prelude, next.effect, next.tick, next.target, next.idle);
            return;
        }
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
        if let Err(err) = battle::complete_native_command_derived_strike(units, j.plan.as_ref()) {
            self.fail_native_command29_presentation(&err.into());
            return;
        }
        let mut j = self.native_cmd29_presentation.take().expect("Go：j 仍是同一個工作");
        let (then, results) =
            (j.then.take(), j.plan.take().expect("Go：CompleteNativeCommandDerivedStrike 已檢查 plan").results);
        if let Some(then) = then {
            then(self, results);
        }
    }

    /// 原版索引色分支；HD 複合畫格（`drawModernHDCompoundFrame`）屬 R6。
    ///
    /// Go：`native_command28_presentation.go:392`，`func (g *Game) drawNativeCommand28Presentation(screen *ebiten.Image) bool`
    pub(crate) fn present_native_command28(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(j) = self.native_cmd29_presentation.as_mut() else {
            return false;
        };
        let Some(sequence) = j.command28.as_ref() else {
            return false;
        };
        let Some(frame) = usize::try_from(sequence.index).ok().and_then(|i| sequence.frames.get(i)) else {
            return false;
        };
        let Some(image) = frame.image.clone() else {
            return false;
        };
        if let Some(screen) = screen {
            let mut op = DrawImageOptions::default();
            op.geo_m.scale(2.0, 2.0);
            screen.draw_image(&image, &op);
        }
        j.drawn = true;
        true
    }
}

#[cfg(test)]
#[path = "native_command28_presentation/native_command28_presentation_test.rs"]
mod native_command28_presentation_test;
