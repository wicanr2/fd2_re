//! Go 來源：`remake/cmd/fd2/native_command9_player_presentation.go`（300 行）。
//!
//! Go 檔的 4 個函式都已移植；HD 的地圖凍結、複合場景與 HD 幀在 R6（PORTING.md「D 組：HD 預凍結延後」）。

use super::*;

/// Go：`native_command9_player_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand9PlayerPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND9_PLAYER_PALETTE: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_PLAYER_EFFECT: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_PLAYER_RESULT: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_PLAYER_HOLD: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(3);

/// Go：`native_command9_player_presentation.go:24`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand9PlayerJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) target: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) palettes: Vec<ColorPalette>,
    pub(crate) effect_frames: Vec<Vec<u8>>,
    pub(crate) result_frames: Vec<Vec<u8>>,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) phase: NativeCommand9PlayerPhase,
    pub(crate) frame: i64,
    pub(crate) hold: i64,
    pub(crate) drawn: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) target_hp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) rng_before: u16,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
    pub(crate) effect_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) result_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) hd_input: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_frames: Option<ModernHDPlayer9Frames>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
}

impl Game {
    /// 玩家指令 9：先播 8 段指令調色盤閃爍，再在目標身上疊 FDOTHER#6 的效果格，扣 MP 與傷害在效果段結束時寫入，
    /// 接著播傷害數字並停 500 ms。所有畫格在發布工作前都合成好；計畫與素材任一不合就不開演出。
    ///
    /// Go：`native_command9_player_presentation.go:45`，`func (g *Game) startNativeCommand9PlayerPresentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommandDamageResult)) error`
    /// 所有權：`actor`、`confirmed` 是 `st.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn start_native_command9_player_presentation(
        &mut self,
        actor: UnitIdx,
        confirmed: Option<UnitIdx>,
        then: Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>,
    ) -> Result<()> {
        let (true, true, Some(confirmed)) =
            (self.current_native_system_options().full_presentation_enabled(), self.st.is_some(), confirmed)
        else {
            return Err(game_err!("native command9 player presentation context unavailable"));
        };
        if self.native_cmd9_player.is_some()
            || self.native_heal_presentation.is_some()
            || self.native_modifier_presentation.is_some()
            || self.native_cmd0_presentation.is_some()
            || self.native_cmd1_presentation.is_some()
            || self.native_cmd2_presentation.is_some()
            || self.native_cmd3_presentation.is_some()
            || self.native_cmd5_presentation.is_some()
            || self.native_cmd6_presentation.is_some()
            || self.native_cmd7_presentation.is_some()
            || self.native_cmd8_presentation.is_some()
            || self.native_cmd9_ai_presentation.is_some()
            || self.native_cmd24_presentation.is_some()
            || self.native_cmd29_presentation.is_some()
            || self.indexed_transition.is_some()
        {
            return Err(game_err!("native command9 player presentation already active"));
        }
        let st = self.st.as_ref().unwrap();
        if !st.has_native_map_view_state
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || self.native_map_dac.len() != 256 * 3
            || self.native_command_palette_flash.is_none()
            || !native_map_assets_available(self.native_map_assets.as_ref())
        {
            return Err(game_err!("native command9 player map/palette assets unavailable"));
        }
        let a = self.native_map_assets.as_ref().unwrap();
        let schedule = fdother::build_native_command9_player_schedule(&a.fdother6, &a.command_heal_digits)?;
        let sounds_missing = !os_mute_or_shot(Some(self))
            && (self.sfx_command9_player_palette.is_empty()
                || self.sfx_command9_player_initial.is_empty()
                || self.sfx_command9_player_repeat.is_empty());
        if schedule.sound_resource != 80
            || schedule.initial_sample != 14
            || schedule.repeat_sample != 15
            || sounds_missing
        {
            return Err(game_err!("native command9 player FDOTHER #80 selectors unavailable"));
        }
        let plan = st.plan_native_command_damage(
            Some(actor),
            Some(confirmed),
            9,
            &st.native_command_resistances,
            self.native_rng_state,
        )?;
        if plan.results.len() != 1 || plan.results[0].target != Some(confirmed) {
            return Err(game_err!("native command9 player single target contract unavailable"));
        }
        let targets = native_command_heal_tail_targets(Some(st), &[Some(confirmed)])?;
        let view = st.native_map_view_state;
        let target = targets[0];
        if target.x < view.camera_x - 1
            || target.x > view.camera_x + 12
            || target.y < view.camera_y - 1
            || target.y > view.camera_y + 8
        {
            return Err(game_err!("native command9 player target outside recovered compositor bounds"));
        }
        let (baseline_work, baseline_vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        let mut effects = Vec::with_capacity(schedule.effect_frames as usize);
        let mut effect_sources = Vec::with_capacity(schedule.effect_frames as usize);
        for frame in 0..schedule.effect_frames {
            let (mut work, mut vga) = (baseline_work.clone(), baseline_vga.clone());
            let index = schedule.effect_start + frame;
            let sources = indexedmap::compose_native_command_heal_effect_frame_with_sources(
                &mut work,
                &mut vga,
                &baseline_work,
                &a.fdother6[index as usize],
                &targets,
                view.camera_x,
                view.camera_y,
                6,
                index,
            )
            .map_err(|err| game_err!("native command9 player effect {frame}: {err}"))?;
            effects.push(vga);
            effect_sources.push(sources);
        }
        let record = target.record_index;
        let in_camera = (view.camera_x..view.camera_x + 12).contains(&target.x)
            && (view.camera_y - 1..=view.camera_y + 7).contains(&target.y);
        let mut queue = Vec::with_capacity(4);
        let damage = plan.results[0].native_command_damage;
        if damage.hit {
            queue = battle::append_native_presentation_digits(
                &queue,
                damage.damage,
                schedule.result_digit_bias,
                record,
                in_camera,
            )?;
        } else if in_camera {
            for (slot, &descriptor) in schedule.result_miss_descriptors.iter().enumerate() {
                queue.push(battle::NativePresentationDigit {
                    position_code: [2, 8, 12, 17][slot],
                    target: record,
                    digit: descriptor,
                });
            }
        }
        let mut results = Vec::with_capacity(schedule.result_frames as usize);
        let mut result_sources = Vec::with_capacity(schedule.result_frames as usize);
        for frame in 0..schedule.result_frames {
            let (mut work, mut vga) = (baseline_work.clone(), baseline_vga.clone());
            let sources = indexedmap::compose_native_command_heal_digit_frame_with_sources(
                &mut work,
                &mut vga,
                &baseline_work,
                &a.command_heal_digits,
                &queue,
                &targets,
                view.camera_x,
                view.camera_y,
                schedule.result_vertical,
                frame,
            )
            .map_err(|err| game_err!("native command9 player result {frame}: {err}"))?;
            results.push(vga);
            result_sources.push(sources);
        }
        let phases = self.native_command_palette_flash.as_ref().unwrap().native_command_palette_flash_phases(9)?;
        let mut palettes = Vec::with_capacity(phases.len());
        for rgb in phases {
            let mut dac = self.native_map_dac.clone();
            dac[..3].copy_from_slice(&rgb);
            palettes.push(color_palette_from_nrgba(&fdother::vga_palette_from_dac(&dac)?));
        }
        let units = &st.units;
        let job = NativeCommand9PlayerJob {
            actor: Some(actor),
            target: Some(confirmed),
            actor_mp_before: units[actor].mp,
            actor_acted_before: units[actor].acted,
            target_hp_before: units[confirmed].hp,
            plan: Some(plan),
            palettes,
            effect_frames: effects,
            result_frames: results,
            baseline_work,
            baseline_vga,
            rng_before: self.native_rng_state,
            then: Some(then),
            effect_sources,
            result_sources,
            ..Default::default()
        };
        // Go 在這裡以 `freezeNativeCommand9MapInput`／`newModernHDCompoundMapScene` 凍結 HD 地圖場景（錯誤只存進
        // `hdScene`）；Rust 留到 R6，`hd_scene` 是 `None`。只做地圖調色盤的錯誤檢查，與 Go 同一個 `return err`。
        fdother::vga_palette_from_dac(&self.native_map_dac)?;
        if self.modern_story_portraits.is_some() {
            todo!("R6: native_command9_player_presentation.go:149 prepareModernHDPlayer9Frames（HD 主題）")
        }
        self.native_cmd9_player = Some(job);
        let sfx = self.sfx_command9_player_palette.clone();
        self.play_raw(&sfx);
        Ok(())
    }

    /// 原版索引色分支；HD 幀（`hdFrames`）屬 R6。
    ///
    /// Go：`native_command9_player_presentation.go:236`，`func (g *Game) drawNativeCommand9PlayerPresentation(screen *ebiten.Image) bool`
    pub(crate) fn present_native_command9_player(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(j) = self.native_cmd9_player.as_ref() else {
            return false;
        };
        if j.hd_frames.is_some() && self.modern_story_portraits.is_some() {
            todo!("r6:native_command9_player_presentation.go:259 drawPreparedModernHDMapFrame（HD 指令 9 幀）")
        }
        let frame = j.frame;
        let (pixels, palette) = match j.phase {
            NATIVE_COMMAND9_PLAYER_PALETTE => {
                let Some(palette) = usize::try_from(frame).ok().and_then(|f| j.palettes.get(f)) else {
                    return false;
                };
                (&j.baseline_vga, Some(palette.clone()))
            }
            NATIVE_COMMAND9_PLAYER_EFFECT => {
                let Some(pixels) = usize::try_from(frame).ok().and_then(|f| j.effect_frames.get(f)) else {
                    return false;
                };
                (pixels, None)
            }
            NATIVE_COMMAND9_PLAYER_RESULT => {
                let Some(pixels) = usize::try_from(frame).ok().and_then(|f| j.result_frames.get(f)) else {
                    return false;
                };
                (pixels, None)
            }
            _ => (&j.baseline_vga, None),
        };
        let palette = match palette {
            Some(palette) => palette,
            None => match fdother::vga_palette_from_dac(&self.native_map_dac) {
                Ok(palette) => color_palette_from_nrgba(&palette),
                Err(_) => return false,
            },
        };
        if pixels.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE || palette.len() != 256 {
            return false;
        }
        if let Some(screen) = screen {
            let img = indexed_frame(320, 200, pixels, palette);
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            screen.draw_image(&Picture::from_indexed(&img), &options);
        }
        self.native_cmd9_player.as_mut().unwrap().drawn = true;
        true
    }

    /// Go：`native_command9_player_presentation.go:172`，`func (g *Game) stepNativeCommand9PlayerPresentation()`
    pub(crate) fn step_native_command9_player_presentation(&mut self) {
        let Some(j) = self.native_cmd9_player.as_mut() else {
            return;
        };
        match j.phase {
            NATIVE_COMMAND9_PLAYER_PALETTE => {
                if !j.drawn {
                    return;
                }
                j.drawn = false;
                j.frame += 1;
                if j.frame >= j.palettes.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND9_PLAYER_EFFECT, 0);
                    let sfx = self.sfx_command9_player_initial.clone();
                    self.play_raw(&sfx);
                }
            }
            NATIVE_COMMAND9_PLAYER_EFFECT => {
                if !j.drawn {
                    return;
                }
                j.drawn = false;
                j.frame += 1;
                if j.frame == 15 || j.frame == 19 {
                    let sfx = self.sfx_command9_player_repeat.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd9_player.as_mut().expect("Go：j 仍是同一個工作");
                if j.frame >= j.effect_frames.len() as i64 {
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd9_player.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) = battle::apply_native_command_damage_mp(units, j.plan.as_mut()) {
                        self.fail_native_command9_player_presentation(&err.into());
                        return;
                    }
                    let damage_stages = j.plan.as_ref().expect("Go：j.plan 已解參考").damage_stages;
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd9_player.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) =
                        battle::apply_native_command_damage_stage(units, j.plan.as_mut(), 0, damage_stages)
                    {
                        self.fail_native_command9_player_presentation(&err.into());
                        return;
                    }
                    self.native_rng_state = j.plan.as_ref().expect("Go：j.plan 已解參考").rng_after;
                    (j.phase, j.frame) = (NATIVE_COMMAND9_PLAYER_RESULT, 0);
                }
            }
            NATIVE_COMMAND9_PLAYER_RESULT => {
                if !j.drawn {
                    return;
                }
                j.drawn = false;
                j.frame += 1;
                if j.frame >= j.result_frames.len() as i64 {
                    (j.phase, j.hold) = (NATIVE_COMMAND9_PLAYER_HOLD, native_delay_ticks(500));
                }
            }
            NATIVE_COMMAND9_PLAYER_HOLD => {
                if j.hold > 0 {
                    j.hold -= 1;
                    return;
                }
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd9_player.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
                    self.fail_native_command9_player_presentation(&err.into());
                    return;
                }
                let mut j = self.native_cmd9_player.take().expect("Go：j 仍是同一個工作");
                let (then, results) =
                    (j.then.take(), j.plan.take().expect("Go：CompleteNativeCommandDamage 已檢查 plan").results);
                (self.native_map_work, self.native_map_vga) = (j.baseline_work, j.baseline_vga);
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Go：`native_command9_player_presentation.go:160`，`func (g *Game) failNativeCommand9PlayerPresentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand9PlayerPresentation` 補移植）。
    pub(crate) fn fail_native_command9_player_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd9_player.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        let actor = j.actor.expect("Go：j.actor 為 nil 時解參考 panic");
        let target = j.target.expect("Go：j.target 為 nil 時解參考 panic");
        (units[actor].mp, units[actor].acted, units[target].hp) =
            (j.actor_mp_before, j.actor_acted_before, j.target_hp_before);
        self.native_rng_state = j.rng_before;
        (self.native_map_work, self.native_map_vga) = (j.baseline_work, j.baseline_vga);
        self.load_err = format!("native command9 player presentation: {}", err.message());
    }
}

#[cfg(test)]
#[path = "native_command9_player_presentation/native_command9_player_presentation_test.rs"]
mod native_command9_player_presentation_test;
