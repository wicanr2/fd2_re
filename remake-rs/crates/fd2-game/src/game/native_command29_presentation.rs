//! Go 來源：`remake/cmd/fd2/native_command29_presentation.go`（585 行）。
//!
//! 原版索引色路徑都已移植，指令 30 的入口與預檢（Go `native_command30_presentation.go`）也在這裡；
//! HD 分割打擊（`stageModernHDSplitStrikeTargetBanks`、`bindModernHDSplitStrike`、`modernFrame`）屬 R6。

use super::*;

/// Go：`native_command29_presentation.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand29TargetPresentation {
    pub(crate) unit: Option<UnitIdx>,
    pub(crate) base_before: Option<Arc<Picture>>,
    pub(crate) base_after: Option<Arc<Picture>>,
    pub(crate) idle_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) idle_positions: Vec<[i64; 2]>,
    pub(crate) idle_delays: Vec<i64>,
    pub(crate) transition_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) hp_before: i64,
    pub(crate) raw_byte5_before: u8,
    pub(crate) damage_published: bool,
}

/// Go：`native_command29_presentation.go:26`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand29PresentationJob {
    pub(crate) command28: Option<NativeCommand28Sequence>,
    pub(crate) hd: Option<ModernHDSplitStrike>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDerivedStrikePlan>,
    pub(crate) schedule: figani::NativeCommandDerivedStrikeSchedule,
    pub(crate) effect_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) effect_positions: Vec<[i64; 2]>,
    pub(crate) prelude_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) prelude_frame: i64,
    pub(crate) actor_base_before: Option<Arc<Picture>>,
    pub(crate) actor_base_after: Option<Arc<Picture>>,
    pub(crate) targets: Vec<NativeCommand29TargetPresentation>,
    pub(crate) target_index: i64,
    pub(crate) transition_frame: i64,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) idle_frame: i64,
    pub(crate) idle_repeat: i64,
    pub(crate) shake_counter: i64,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) sfx_actor: Vec<u8>,
    pub(crate) sfx_target: Vec<u8>,
    /// R4 D 組：Go `map[int][]byte` 的 nil 與空 map 在 `stepNativeCommand29Presentation`（Go 檔:436）行為不同
    /// （nil 才播 `sfxActor`），所以包 `Option`；Go 的 start（Go 檔:369）一律 `make`，對應 `Some`。
    pub(crate) actor_cues: Option<BTreeMap<i64, Vec<u8>>>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>>,
}

/// 目標畫面在特效最後一格時停在第幾個待機畫格；下一個目標的滑移底圖用它疊出上一個目標的收尾。
///
/// Go：`native_command29_presentation.go:51`，`func nativeCommand29FinalIdleFrame(delays []int, schedule figani.NativeCommandDerivedStrikeSchedule) (int, error)`
pub(crate) fn native_command29_final_idle_frame(
    delays: &[i64],
    schedule: &figani::NativeCommandDerivedStrikeSchedule,
) -> Result<usize> {
    if delays.is_empty() || schedule.target_start < 0 || schedule.target_start >= schedule.frame_delays.len() as i64 {
        return Err(game_err!("native command29 idle schedule unavailable"));
    }
    let (mut frame, mut repeat, mut last) = (0usize, 0i64, 0usize);
    for effect in schedule.target_start as usize..schedule.frame_delays.len() {
        for _ in 0..schedule.frame_delays[effect] {
            last = frame;
            repeat += 1;
            if repeat >= delays[frame] {
                repeat = 0;
                frame = (frame + 1) % delays.len();
            }
        }
    }
    Ok(last)
}

/// 指令 29／30 的目標來源：游標選格（指令 29 玩家路徑）、確認單位（一般選目標），或指令 30 的兩個游標。
#[derive(Debug, Clone, Copy)]
pub(crate) enum NativeDerivedStrikeTarget {
    Confirmed(UnitIdx),
    Cursor(battle::Cell),
    Command30 { saved: battle::Cell, confirmed: battle::Cell },
}

impl Game {
    /// Go：`native_command29_presentation.go:69`，`func (g *Game) startNativeCommand29Presentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommand24Damage)) error`
    pub(crate) fn start_native_command29_presentation(
        &mut self,
        actor: UnitIdx,
        confirmed: Option<UnitIdx>,
        then: Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>,
    ) -> Result<()> {
        let Some(confirmed) = confirmed else {
            return Err(game_err!("native command29 presentation context unavailable"));
        };
        self.start_native_derived_strike_presentation(actor, 29, NativeDerivedStrikeTarget::Confirmed(confirmed), then)
    }

    /// Go：`native_command29_presentation.go:73`，`func (g *Game) startNativeCommand29PresentationAtCursor(actor *battle.Unit, cursor battle.Cell, then func([]battle.NativeCommand24Damage)) error`
    pub(crate) fn start_native_command29_presentation_at_cursor(
        &mut self,
        actor: UnitIdx,
        cursor: battle::Cell,
        then: Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>,
    ) -> Result<()> {
        self.start_native_derived_strike_presentation(actor, 29, NativeDerivedStrikeTarget::Cursor(cursor), then)
    }

    /// Go：`native_command30_presentation.go:11`，`func (g *Game) startNativeCommand30PresentationAtCursor(actor *battle.Unit, saved, confirmed battle.Cell, then func([]battle.NativeCommand24Damage)) error`
    pub(crate) fn start_native_command30_presentation_at_cursor(
        &mut self,
        actor: UnitIdx,
        saved: battle::Cell,
        confirmed: battle::Cell,
        then: Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>,
    ) -> Result<()> {
        let Some(st) = self.st.as_ref() else {
            return Err(game_err!("native command30 context unavailable"));
        };
        if st.unit_at(confirmed.x, confirmed.y).is_none() {
            return Err(game_err!("native command30 confirmed cursor has no unit"));
        }
        self.start_native_derived_strike_presentation(
            actor,
            30,
            NativeDerivedStrikeTarget::Command30 { saved, confirmed },
            then,
        )
    }

    /// 0x276EC 多目標衍生打擊（指令 29、30）：前奏、施放者特效，之後每個目標各跑一次背景滑移與受擊。
    /// 靜態來源全部先驗過才消耗亂數；HD 分割打擊屬 R6，這裡只建原版索引色畫格。
    ///
    /// Go：`native_command29_presentation.go:81`，`func (g *Game) startNativeDerivedStrikePresentationTarget(actor, confirmed *battle.Unit, command int, savedCursor, confirmedCursor battle.Cell, cursor *battle.Cell, then func([]battle.NativeCommand24Damage)) error`
    pub(crate) fn start_native_derived_strike_presentation(
        &mut self,
        actor_index: UnitIdx,
        command: i64,
        selection: NativeDerivedStrikeTarget,
        then: Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>,
    ) -> Result<()> {
        if !self.current_native_system_options().full_presentation_enabled() {
            return Err(game_err!("native abbreviated presentation owner unavailable"));
        }
        let actor = self.st.as_ref().and_then(|st| st.units.get(actor_index)).cloned();
        let (Some(actor), false, true) = (actor, self.native_derived_strike_presentation_busy(), self.rng.is_some())
        else {
            return Err(game_err!("native command{command} presentation context unavailable"));
        };
        let (expected_fig, expected_audio, expected_actor_sample, expected_target_sample) = match command {
            29 => (34, 50, 1, 4),
            30 => (18, 49, 4, 3),
            _ => return Err(game_err!("native derived-strike presentation command unavailable")),
        };
        if !actor.has_battle_fig || actor.battle_fig != expected_fig || self.native_ui_palette.len() < 256 {
            return Err(game_err!(
                "native command{command} requires proven selector{expected_fig} actor and indexed palette"
            ));
        }
        let animations = separated_asset_path("animations");
        let effect = figani::load_separated_resource(&animations, actor.battle_fig * 3 + 2)?;
        let schedule = figani::build_native_command_derived_strike_schedule(Some(&effect), command)?;
        if schedule.command_id != command
            || schedule.audio_resource != expected_audio
            || schedule.actor_sample != expected_actor_sample
            || schedule.target_sample != expected_target_sample
            || schedule.prelude_mode != 1
            || !schedule.uses_bg_transition
            || schedule.uses_target_base
        {
            return Err(game_err!(
                "native command{command} resource{} presentation signature mismatch",
                actor.battle_fig * 3 + 2
            ));
        }
        let mut samples = vec![schedule.target_sample];
        samples.extend(schedule.actor_cues.iter().map(|cue| cue.sample));
        self.require_separated_command_sounds(schedule.audio_resource, &samples)?;
        let actor_idle = figani::load_separated_resource(&animations, actor.battle_fig * 3)
            .ok()
            .filter(|idle| !idle.frames.is_empty())
            .ok_or_else(|| game_err!("native command{command} actor idle FIGANI unavailable"))?;
        let st = self.st.as_ref().unwrap();
        let selected = match selection {
            NativeDerivedStrikeTarget::Command30 { saved, confirmed } => {
                st.native_command30_effect_targets(Some(actor_index), saved, confirmed)?
            }
            NativeDerivedStrikeTarget::Cursor(cursor) => {
                st.native_command29_effect_targets(Some(actor_index), cursor)?
            }
            NativeDerivedStrikeTarget::Confirmed(confirmed) => {
                let flags = st.native_command_base_flags()?;
                if st.native_command_book.len() as i64 != battle::NATIVE_COMMAND_RECORD_COUNT {
                    return Err(game_err!("native derived-strike command book unavailable"));
                }
                let record = st.native_command_book[command as usize];
                battle::native_command_effect_targets(
                    st.w,
                    st.h,
                    Some(actor_index),
                    Some(confirmed),
                    record.selection_mode,
                    record.effect_mode,
                    record.target_code,
                    &flags,
                    &st.units,
                )?
            }
        };
        if selected.is_empty() {
            return Err(if command == 30 {
                game_err!("native command30 final target list is empty")
            } else {
                game_err!("native derived-strike final target list is empty")
            });
        }
        if command == 30 {
            self.preflight_native_command30_sources(actor_index, &selected, &effect)?;
        } else {
            self.preflight_native_derived_strike_sources(actor_index, &selected, &effect, command)?;
        }
        let rng = self.rng.as_mut().map(|rng| rng as &mut dyn battle::GoRand);
        let st = self.st.as_ref().unwrap();
        let plan = match selection {
            NativeDerivedStrikeTarget::Command30 { saved, confirmed } => {
                st.plan_native_command30(Some(actor_index), saved, confirmed, rng)?
            }
            NativeDerivedStrikeTarget::Cursor(cursor) => {
                st.plan_native_command29_at_cursor(Some(actor_index), cursor, rng)?
            }
            NativeDerivedStrikeTarget::Confirmed(confirmed) => {
                st.plan_native_command_derived_strike(Some(actor_index), Some(confirmed), command, rng)?
            }
        };
        if plan.results.is_empty() {
            return Err(game_err!("native command{command} final target list is empty"));
        }
        let surfaces = separated_asset_path("surfaces");
        let bg_layers = [
            fdother::load_separated_single_frame(&surfaces, "BG.DAT", 0)?,
            fdother::load_separated_single_frame(&surfaces, "BG.DAT", 1)?,
            fdother::load_separated_single_frame(&surfaces, "BG.DAT", 2)?,
        ];
        let panel_assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let (palette_dac, _) = load_native_battle_palette()?;
        let actor_selector = native_command24_bg_selector(self.m.as_ref(), &actor)?;
        let actor_bg = fdother::load_separated_single_frame(&surfaces, "BG.DAT", actor_selector)?;
        let actor_unit_index = native_command24_runtime_unit_index(self.st.as_ref(), Some(actor_index))?;
        let actor_record_before = battle::native_battle_panel_record_for_unit(Some(&actor))?;
        let mut actor_after = actor.clone();
        actor_after.mp = plan.mp_after;
        let actor_record_after = battle::native_battle_panel_record_for_unit(Some(&actor_after))?;
        let platform = if actor.native_record_byte6 != 0 {
            Some(fdother::load_separated_single_frame(&surfaces, "TAI.DAT", actor_selector)?)
        } else {
            None
        };
        modern_split_strike_scenery(&actor_bg, actor_selector, platform.as_ref())?;
        let chapter = self.handler_chapter;
        let base = |g: &mut Game, bg: &fdother::Frame, record: &[u8], index: i64, platform: Option<&fdother::Frame>| {
            g.localized_native_command24_captured_base(bg, &panel_assets, record, index, chapter, platform, None, &[])
        };
        let actor_base_before = base(self, &actor_bg, &actor_record_before, actor_unit_index, platform.as_ref())?;
        let actor_base_after = base(self, &actor_bg, &actor_record_after, actor_unit_index, platform.as_ref())?;
        let prelude_base = base(self, &actor_bg, &actor_record_before, actor_unit_index, None)?;
        let prelude = battlepresent::build_native_command_prelude_frames(&battlepresent::NativeCommandPreludeInput {
            base: &prelude_base,
            actor_idle: actor_idle.frames[0].clone(),
            platform: platform.as_ref(),
            raw_side: actor.native_record_byte6,
            mode: schedule.prelude_mode,
            baseline_dac: &palette_dac,
            ..Default::default()
        })?;
        let prelude_frames = native_command24_prelude_images(&prelude)?
            .into_iter()
            .map(|frame| Some(Arc::new(Picture::from_indexed(&frame))))
            .collect();
        let palette = self.native_ui_palette.clone();
        let (effect_frames, effect_positions) = native_figani_images(&effect, &palette)?;
        let mut actor_bases =
            native_command24_indexed_images(&[actor_base_before, actor_base_after.clone()], &palette)?.into_iter();
        let mut source = actor_base_after;
        effect
            .frames
            .get((schedule.target_start - 1) as usize)
            .ok_or_else(|| game_err!("native command{command} source frame unavailable"))?
            .blit_at(&mut source, 320)?;
        let mut targets = Vec::with_capacity(plan.results.len());
        for (result_index, result) in plan.results.iter().enumerate() {
            let target = result
                .target
                .and_then(|t| self.st.as_ref().unwrap().units.get(t))
                .filter(|t| t.has_battle_fig)
                .cloned()
                .ok_or_else(|| game_err!("native command{command} target {result_index} BattleFig unavailable"))?;
            let idle = figani::load_separated_resource(&animations, target.battle_fig * 3)
                .ok()
                .filter(|idle| !idle.frames.is_empty())
                .ok_or_else(|| game_err!("native command{command} target {result_index} idle FIGANI unavailable"))?;
            let mut delays = Vec::with_capacity(idle.frames.len());
            for (i, frame) in idle.frames.iter().enumerate() {
                if frame.delay <= 0 {
                    return Err(game_err!(
                        "native command{command} target {result_index} idle frame {i} delay invalid"
                    ));
                }
                delays.push(frame.delay);
            }
            let selector = native_command24_bg_selector(self.m.as_ref(), &target)?;
            let background = fdother::load_separated_single_frame(&surfaces, "BG.DAT", selector)?;
            modern_split_strike_scenery(&background, selector, None)?;
            let unit_index = native_command24_runtime_unit_index(self.st.as_ref(), result.target)?;
            let record_before = battle::native_battle_panel_record_for_unit(Some(&target))?;
            let mut after = target.clone();
            after.hp = result.hp_after;
            let record_after = battle::native_battle_panel_record_for_unit(Some(&after))?;
            let base_before = base(self, &background, &record_before, unit_index, None)?;
            let base_after = base(self, &background, &record_after, unit_index, None)?;
            let transitions = battlepresent::build_native_command24_background_presents(
                &battlepresent::NativeCommand24BackgroundInputs {
                    layers: bg_layers.clone(),
                    source: &source,
                    target: &base_before,
                    target_idle: idle.frames[0].clone(),
                },
            )?;
            let transitions: Vec<Vec<u8>> = transitions.into_iter().map(|present| present.pixels).collect();
            let mut bases = native_command24_indexed_images(&[base_before, base_after.clone()], &palette)?.into_iter();
            let transition_frames = native_command24_indexed_images(&transitions, &palette)?;
            let (idle_frames, idle_positions) = native_figani_images(&idle, &palette)?;
            let last_idle = native_command29_final_idle_frame(&delays, &schedule)?;
            targets.push(NativeCommand29TargetPresentation {
                unit: result.target,
                base_before: bases.next().flatten(),
                base_after: bases.next().flatten(),
                idle_frames,
                idle_positions,
                idle_delays: delays,
                transition_frames,
                hp_before: target.hp,
                raw_byte5_before: target.native_record_byte5,
                damage_published: false,
            });
            source = base_after;
            idle.frames[last_idle].blit_at(&mut source, 320)?;
            effect.frames.last().expect("前面已驗過特效畫格").blit_at(&mut source, 320)?;
        }
        let sfx_actor = self.separated_command_sound(schedule.audio_resource, schedule.actor_sample);
        let sfx_target = self.separated_command_sound(schedule.audio_resource, schedule.target_sample);
        let mut actor_cues = BTreeMap::new();
        for cue in &schedule.actor_cues {
            let sample = self.separated_command_sound(schedule.audio_resource, cue.sample);
            actor_cues.insert(cue.frame, sample);
        }
        if !os_mute_or_shot(Some(self)) && (sfx_actor.is_empty() || sfx_target.is_empty()) {
            return Err(game_err!("native command{command} FDOTHER #{} samples unavailable", schedule.audio_resource));
        }
        self.native_cmd29_presentation = Some(NativeCommand29PresentationJob {
            actor: Some(actor_index),
            plan: Some(plan),
            schedule,
            effect_frames,
            effect_positions,
            prelude_frames,
            actor_base_before: actor_bases.next().flatten(),
            actor_base_after: actor_bases.next().flatten(),
            targets,
            transition_frame: -1,
            actor_mp_before: actor.mp,
            actor_acted_before: actor.acted,
            sfx_actor,
            sfx_target,
            actor_cues: Some(actor_cues),
            then: Some(then),
            ..Default::default()
        });
        Ok(())
    }

    /// 指令 30 先核對 resource56 的形狀與事件標記，再跑共用的靜態來源預檢。
    ///
    /// Go：`native_command30_presentation.go:24`，`func (g *Game) preflightNativeCommand30Sources(actor *battle.Unit, targets []*battle.Unit, effect *figani.Animation) error`
    pub(crate) fn preflight_native_command30_sources(
        &mut self,
        actor: UnitIdx,
        targets: &[UnitIdx],
        effect: &figani::Animation,
    ) -> Result<()> {
        if effect.frames.len() != 14 || effect.header_byte2 != 9 {
            return Err(game_err!("native command30 resource56 shape differs"));
        }
        let f = &effect.frames;
        if f[2].raw_byte5 != 2 || f[2].raw_byte4 != 0 || f[4].raw_byte4 != 1 || f[11].raw_byte4 != 1 {
            return Err(game_err!("native command30 effect events differ"));
        }
        self.preflight_native_derived_strike_sources(actor, targets, effect, 30)
    }

    /// 消耗亂數之前把每個靜態來源都解碼並試畫一次：調色盤、特效、BG／TAI、待機畫格與狀態框。
    ///
    /// Go：`native_command30_presentation.go:34`，`func (g *Game) preflightNativeDerivedStrikeSources(actor *battle.Unit, targets []*battle.Unit, effect *figani.Animation, command int) error`
    pub(crate) fn preflight_native_derived_strike_sources(
        &mut self,
        actor: UnitIdx,
        targets: &[UnitIdx],
        effect: &figani::Animation,
        command: i64,
    ) -> Result<()> {
        load_native_battle_palette()?;
        if effect.frames.is_empty() {
            return Err(game_err!("native command{command} effect source unavailable"));
        }
        for (i, frame) in effect.frames.iter().enumerate() {
            frame
                .blit_at(&mut vec![0; 64000], 320)
                .map_err(|err| game_err!("native command{command} effect{i}: {err}"))?;
        }
        let assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let surfaces = separated_asset_path("surfaces");
        if command != 28 {
            for i in 0..3 {
                fdother::load_separated_single_frame(&surfaces, "BG.DAT", i)?;
            }
        }
        let animations = separated_asset_path("animations");
        for &unit_index in std::iter::once(&actor).chain(targets) {
            let Some(unit) =
                self.st.as_ref().and_then(|st| st.units.get(unit_index)).filter(|u| u.has_battle_fig).cloned()
            else {
                return Err(game_err!("native command{command} unit source unavailable"));
            };
            let idle = match figani::load_separated_resource(&animations, unit.battle_fig * 3) {
                Ok(idle) if !idle.frames.is_empty() => idle,
                Ok(_) => return Err(game_err!("native command{command} idle unavailable: <nil>")),
                Err(err) => return Err(game_err!("native command{command} idle unavailable: {err}")),
            };
            for f in &idle.frames {
                if f.delay <= 0 {
                    return Err(game_err!("native command{command} idle delay invalid"));
                }
                f.blit_at(&mut vec![0; 64000], 320)?;
            }
            let selector = native_command24_bg_selector(self.m.as_ref(), &unit)?;
            fdother::load_separated_single_frame(&surfaces, "BG.DAT", selector)?;
            if unit_index == actor && (unit.native_record_byte6 != 0 || command == 28) {
                fdother::load_separated_single_frame(&surfaces, "TAI.DAT", selector)?;
            }
            let record = battle::native_battle_panel_record_for_unit(Some(&unit))?;
            let index = native_command24_runtime_unit_index(self.st.as_ref(), Some(unit_index))?;
            let mut text = Vec::new();
            let chapter = self.handler_chapter;
            self.render_battle_panel_text(&assets, &record, &mut vec![0; 64000], index, chapter, Some(&mut text))?;
        }
        Ok(())
    }

    /// 原版索引色分支；HD 複合畫格（`j.modernFrame()`）屬 R6。
    ///
    /// Go：`native_command29_presentation.go:506`，`func (g *Game) drawNativeCommand29Presentation(screen *ebiten.Image) bool`
    pub(crate) fn present_native_command29(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(j) = self.native_cmd29_presentation.as_mut() else {
            return false;
        };
        if j.command28.is_some() {
            return self.present_native_command28(screen);
        }
        if j.frame < 0
            || j.frame >= j.effect_frames.len() as i64
            || j.target_index < 0
            || j.target_index >= j.targets.len() as i64
        {
            return false;
        }
        let scaled = || {
            let mut op = DrawImageOptions::default();
            op.geo_m.scale(2.0, 2.0);
            op
        };
        if j.prelude_frame < j.prelude_frames.len() as i64 {
            let Some(image) = usize::try_from(j.prelude_frame).ok().and_then(|i| j.prelude_frames[i].clone()) else {
                return false;
            };
            if let Some(screen) = screen {
                screen.draw_image(&image, &scaled());
            }
            j.drawn = true;
            return true;
        }
        let target = &j.targets[j.target_index as usize];
        if j.transition_frame >= 0 {
            let Some(image) = target.transition_frames.get(j.transition_frame as usize).cloned().flatten() else {
                return false;
            };
            if let Some(screen) = screen {
                screen.draw_image(&image, &scaled());
            }
            j.drawn = true;
            return true;
        }
        let base = if j.frame < j.schedule.target_start {
            if j.frame >= j.schedule.actor_impact_frame { &j.actor_base_after } else { &j.actor_base_before }
        } else if target.damage_published || j.frame >= j.schedule.target_impact_frame {
            &target.base_after
        } else {
            &target.base_before
        };
        let Some(base) = base.clone() else {
            return false;
        };
        let mut idle = None;
        if j.frame >= j.schedule.target_start {
            if j.idle_frame < 0 || j.idle_frame >= target.idle_frames.len() as i64 {
                return false;
            }
            let mut shake = 0;
            if j.frame == j.schedule.target_impact_frame && !target.damage_published {
                shake = j.schedule.shake_offsets[5];
            } else if target.damage_published
                && j.shake_counter >= 0
                && j.shake_counter < j.schedule.shake_offsets.len() as i64
            {
                shake = j.schedule.shake_offsets[j.shake_counter as usize];
            }
            let [x, y] = target.idle_positions[j.idle_frame as usize];
            idle = Some((target.idle_frames[j.idle_frame as usize].clone(), x - shake, y));
        }
        if let Some(screen) = screen {
            screen.draw_image(&base, &scaled());
            if let Some((Some(image), x, y)) = idle {
                let mut op = scaled();
                op.geo_m.translate((x * 2) as f64, (y * 2) as f64);
                screen.draw_image(&image, &op);
            }
            let [x, y] = j.effect_positions[j.frame as usize];
            let mut op = scaled();
            op.geo_m.translate((x * 2) as f64, (y * 2) as f64);
            if let Some(image) = j.effect_frames[j.frame as usize].as_deref() {
                screen.draw_image(image, &op);
            }
        }
        j.drawn = true;
        true
    }
}

impl Game {
    /// Go：`native_command29_presentation.go:409`，`func (g *Game) stepNativeCommand29Presentation()`
    pub(crate) fn step_native_command29_presentation(&mut self) {
        let Some(j) = self.native_cmd29_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        if j.command28.is_some() {
            self.step_native_command28_presentation();
            return;
        }
        j.drawn = false;
        if j.prelude_frame < j.prelude_frames.len() as i64 {
            j.prelude_frame += 1;
            return;
        }
        if j.transition_frame >= 0 {
            j.transition_frame += 1;
            if j.transition_frame >= j.targets[j.target_index as usize].transition_frames.len() as i64 {
                (j.transition_frame, j.idle_frame, j.idle_repeat) = (-1, 0, 0);
            }
            return;
        }
        if j.frame == j.schedule.actor_impact_frame && !j.mp_published {
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::apply_native_command_derived_strike_mp(units, j.plan.as_ref()) {
                self.fail_native_command29_presentation(&err.into());
                return;
            }
            j.mp_published = true;
            if j.actor_cues.is_none() {
                let sfx = j.sfx_actor.clone();
                self.play_raw(&sfx);
            }
        }
        let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
        if j.frame < j.schedule.target_start
            && j.repeat == 0
            && let Some(cue) = j.actor_cues.as_ref().and_then(|cues| cues.get(&j.frame))
        {
            let cue = cue.clone();
            self.play_raw(&cue);
        }
        let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
        if j.frame >= j.schedule.target_start {
            let target_index = j.target_index;
            if j.targets[target_index as usize].damage_published && j.shake_counter >= 0 {
                j.shake_counter -= 1;
            }
            if j.frame == j.schedule.target_impact_frame && !j.targets[target_index as usize].damage_published {
                if !j.mp_published {
                    self.fail_native_command29_presentation(&game_err!("target marker preceded actor MP marker"));
                    return;
                }
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) =
                    battle::apply_native_command_derived_strike_target(units, j.plan.as_ref(), target_index)
                {
                    self.fail_native_command29_presentation(&err.into());
                    return;
                }
                j.targets[target_index as usize].damage_published = true;
                j.shake_counter = 5;
                let sfx = j.sfx_target.clone();
                self.play_raw(&sfx);
            }
            let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
            j.idle_repeat += 1;
            let target = &j.targets[target_index as usize];
            if j.idle_repeat >= target.idle_delays[j.idle_frame as usize] {
                let idle_len = target.idle_frames.len() as i64;
                j.idle_repeat = 0;
                // Go `%`：`len(target.idleFrames)` 為 0 時 panic，Rust 同樣 panic。
                j.idle_frame = (j.idle_frame + 1) % idle_len;
            }
        }
        let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.repeat += 1;
        if j.repeat < j.schedule.frame_delays[j.frame as usize] {
            return;
        }
        j.repeat = 0;
        j.frame += 1;
        if j.frame == j.schedule.target_start {
            j.transition_frame = 0;
            return;
        }
        if j.frame < j.effect_frames.len() as i64 {
            return;
        }
        if !j.targets[j.target_index as usize].damage_published {
            self.fail_native_command29_presentation(&game_err!("required target marker was not presented"));
            return;
        }
        if j.target_index + 1 < j.targets.len() as i64 {
            j.target_index += 1;
            (j.frame, j.repeat, j.idle_frame, j.idle_repeat, j.transition_frame) =
                (j.schedule.target_start, 0, 0, 0, 0);
            return;
        }
        if !j.mp_published {
            self.fail_native_command29_presentation(&game_err!("required actor marker was not presented"));
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

    /// Go：`native_command29_presentation.go:387`，`func (g *Game) failNativeCommand29Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand29Presentation` 補移植）。
    pub(crate) fn fail_native_command29_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd29_presentation.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        let actor = j.actor.expect("Go：j.actor 為 nil 時解參考 panic");
        (units[actor].mp, units[actor].acted) = (j.actor_mp_before, j.actor_acted_before);
        for target in &j.targets {
            let unit = &mut units[target.unit.expect("Go：target.unit 為 nil 時解參考 panic")];
            unit.hp = target.hp_before;
            if unit.has_native_record_byte5 {
                unit.native_record_byte5 = target.raw_byte5_before;
            }
        }
        battle::restore_native_command30_targets(units, j.plan.as_ref());
        let command = j.plan.as_ref().map_or(29, |plan| plan.command_id);
        self.load_err = format!("native command{command} presentation: {}", err.message());
    }
}

#[cfg(test)]
#[path = "native_command29_presentation/native_command29_presentation_test.rs"]
pub(crate) mod native_command29_presentation_test;
