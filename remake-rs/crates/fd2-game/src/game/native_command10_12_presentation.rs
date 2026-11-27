//! Go 來源：`remake/cmd/fd2/native_command10_12_presentation.go`（346 行）。
//!
//! 原版索引色路徑都已移植；HD 畫格（`prepareModernHDCommand1012`、`drawPreparedModernHDMapFrame`）屬 R6。

use super::*;

/// Go：`native_command10_12_presentation.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand1012Phase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND1012_PRELUDE: NativeCommand1012Phase = NativeCommand1012Phase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND1012_MAIN: NativeCommand1012Phase = NativeCommand1012Phase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND1012_RESULT: NativeCommand1012Phase = NativeCommand1012Phase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND1012_HOLD: NativeCommand1012Phase = NativeCommand1012Phase(3);

/// Go：`native_command10_12_presentation.go:25`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand1012Job {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) schedule: fdother::NativeCommand10To12Schedule,
    pub(crate) main_frames: Vec<Vec<u8>>,
    pub(crate) result_frames: Vec<Vec<u8>>,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) phase: NativeCommand1012Phase,
    pub(crate) frame: i64,
    pub(crate) hold: i64,
    pub(crate) drawn: bool,
    pub(crate) prelude_done: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: Vec<i64>,
    pub(crate) rng_before: u16,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
    pub(crate) result_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) hd_input: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_baseline: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_signature: [u8; 32],
    pub(crate) hd_frames: Option<ModernHDCommand1012Frames>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
}

impl Game {
    /// Go：`native_command10_12_presentation.go:53`，`func (g *Game) startNativeCommand1012PresentationAtCursor(actor *battle.Unit, cursor battle.Cell, commandID int, then func([]battle.NativeCommandDamageResult)) error`
    pub(crate) fn start_native_command1012_presentation_at_cursor(
        &mut self,
        actor: UnitIdx,
        cursor: battle::Cell,
        command_id: i64,
        then: Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>,
    ) -> Result<()> {
        self.start_native_command1012_presentation_target(actor, None, Some(cursor), command_id, then)
    }

    /// 玩家指令 10～12：地圖畫面三種取樣面輪播，再播傷害數字。指令 11、12 先跑 0x2189A 的十次半徑前奏，
    /// 前奏收尾才進主段並播主音效。扣血在結果段才寫入。HD 綁定（`prepareModernHDCommand1012` 等）屬 R6。
    ///
    /// Go：`native_command10_12_presentation.go:57`，`func (g *Game) startNativeCommand1012PresentationTarget(actor, confirmed *battle.Unit, cursor *battle.Cell, commandID int, then func([]battle.NativeCommandDamageResult)) error`
    pub(crate) fn start_native_command1012_presentation_target(
        &mut self,
        actor_index: UnitIdx,
        confirmed: Option<UnitIdx>,
        cursor: Option<battle::Cell>,
        command_id: i64,
        then: Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>,
    ) -> Result<()> {
        let actor = self.st.as_ref().and_then(|st| st.units.get(actor_index)).cloned();
        let (true, Some(actor), true, false, false) = (
            self.current_native_system_options().full_presentation_enabled(),
            actor,
            confirmed.is_some() || cursor.is_some(),
            self.native_cmd1012.is_some(),
            self.native2189_a.is_some(),
        ) else {
            return Err(game_err!("native command10-12 presentation context unavailable"));
        };
        if self.native_heal_presentation.is_some()
            || self.native_modifier_presentation.is_some()
            || self.native_cmd0_presentation.is_some()
            || self.native_cmd1_presentation.is_some()
            || self.native_cmd2_presentation.is_some()
            || self.native_cmd3_presentation.is_some()
            || self.native_cmd5_presentation.is_some()
            || self.native_cmd6_presentation.is_some()
            || self.native_cmd7_presentation.is_some()
            || self.native_cmd8_presentation.is_some()
            || self.native_cmd9_player.is_some()
            || self.native_cmd9_ai_presentation.is_some()
            || self.native_cmd24_presentation.is_some()
            || self.native_cmd29_presentation.is_some()
            || self.native_cmd32_presentation.is_some()
            || self.native_cmd33_presentation.is_some()
            || self.native_cmd34_presentation.is_some()
            || self.native_cmd35_presentation.is_some()
            || self.indexed_transition.is_some()
        {
            return Err(game_err!("native command10-12 presentation already active"));
        }
        let schedule = fdother::build_native_command10_to12_schedule(command_id)?;
        let st = self.st.as_ref().unwrap();
        if self.m.is_none()
            || !native_map_assets_available(self.native_map_assets.as_ref())
            || !st.has_native_map_view_state
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || self.native_map_assets.as_ref().unwrap().command_heal_digits.len() <= 118
            || (!os_mute_or_shot(Some(self))
                && (self.sfx_command1012_main.is_empty()
                    || schedule.prelude.enabled && self.sfx_command1012_prelude.is_empty()))
        {
            return Err(game_err!("native command10-12 map/result/audio assets unavailable"));
        }
        let resistances = &st.native_command_resistances;
        let plan = match cursor {
            Some(cursor) => st.plan_native_command_damage_at_cursor(
                Some(actor_index),
                cursor,
                command_id,
                resistances,
                self.native_rng_state,
                None,
            )?,
            None => st.plan_native_command_damage(
                Some(actor_index),
                confirmed,
                command_id,
                resistances,
                self.native_rng_state,
            )?,
        };
        if plan.results.is_empty() || plan.damage_stages != 1 {
            return Err(game_err!("native command10-12 final targets unavailable"));
        }
        let target_units: Vec<Option<UnitIdx>> = plan.results.iter().map(|result| result.target).collect();
        let targets = native_command_heal_tail_targets(Some(st), &target_units)?;
        let (hud, hud_ok) = self.native_map_hud_input();
        if !hud_ok {
            return Err(game_err!("native command10-12 HUD provenance unavailable"));
        }
        let st = self.st.as_ref().unwrap();
        let input = build_native_map_frame_input(
            self.native_map_assets.as_ref(),
            self.m.as_ref(),
            Some(st),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )?;
        let transition = native_transition_input(&input);
        let (baseline_work, baseline_vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        let mut surfaces = vec![baseline_vga.clone()];
        for phase in 0..3 {
            let (mut work, mut vga) = (baseline_work.clone(), baseline_vga.clone());
            indexedmap::compose_native_command10_to12_surface(
                &mut work,
                &mut vga,
                &transition,
                schedule.x_origins[phase],
                schedule.y_origins[phase],
                schedule.sampling_increments[phase],
            )
            .map_err(|err| game_err!("native command10-12 sampled surface {phase}: {err}"))?;
            surfaces.push(vga);
        }
        let cycle = &schedule.surface_cycle;
        let main_frames: Vec<Vec<u8>> =
            (0..schedule.frames as usize).map(|frame| surfaces[cycle[frame % cycle.len()] as usize].clone()).collect();
        let view = st.native_map_view_state;
        let mut queue = Vec::with_capacity(plan.results.len() * 4);
        for (index, result) in plan.results.iter().enumerate() {
            let target = &targets[index];
            let in_camera = (view.camera_x..view.camera_x + 12).contains(&target.x)
                && (view.camera_y - 1..=view.camera_y + 7).contains(&target.y);
            if result.native_command_damage.hit {
                queue = battle::append_native_presentation_digits(
                    &queue,
                    result.native_command_damage.damage,
                    schedule.result_digit_bias,
                    target.record_index,
                    in_camera,
                )?;
            } else if in_camera {
                for (slot, &descriptor) in schedule.result_miss_descriptors.iter().enumerate() {
                    queue.push(battle::NativePresentationDigit {
                        position_code: [2, 8, 12, 17][slot],
                        target: target.record_index,
                        digit: descriptor,
                    });
                }
            }
        }
        let assets = self.native_map_assets.as_ref().unwrap();
        let mut result_frames = Vec::with_capacity(schedule.result_frames as usize);
        let mut result_sources = Vec::with_capacity(schedule.result_frames as usize);
        for frame in 0..schedule.result_frames {
            let (mut work, mut vga) = (baseline_work.clone(), baseline_vga.clone());
            let sources = indexedmap::compose_native_command_heal_digit_frame_with_sources(
                &mut work,
                &mut vga,
                &baseline_work,
                &assets.command_heal_digits,
                &queue,
                &targets,
                view.camera_x,
                view.camera_y,
                schedule.result_vertical,
                frame,
            )
            .map_err(|err| game_err!("native command10-12 result frame {frame}: {err}"))?;
            result_frames.push(vga);
            result_sources.push(sources);
        }
        let palette = if self.native_map_dac.len() == 256 * 3 {
            color_palette_from_nrgba(&fdother::vga_palette_from_dac(&self.native_map_dac)?)
        } else {
            assets.palette.clone()
        };
        let target_hp_before = plan.results.iter().map(|result| result.target.map_or(0, |t| st.units[t].hp)).collect();
        // Go 另凍結 HD 地圖輸入、場景與簽章，主題存在時預備 HD 畫格（`prepareModernHDCommand1012`，R6）。
        let prelude = schedule.prelude;
        self.native_cmd1012 = Some(NativeCommand1012Job {
            actor: Some(actor_index),
            plan: Some(plan),
            schedule,
            main_frames,
            result_frames,
            baseline_work,
            baseline_vga,
            palette,
            phase: NATIVE_COMMAND1012_MAIN,
            actor_mp_before: actor.mp,
            actor_acted_before: actor.acted,
            target_hp_before,
            rng_before: self.native_rng_state,
            then: Some(then),
            result_sources,
            ..Default::default()
        });
        if prelude.enabled {
            // Go 以指標比對找施放者的原始欄位；Rust 的單位就是 `st.units` 的索引。
            self.native_cmd1012.as_mut().unwrap().phase = NATIVE_COMMAND1012_PRELUDE;
            let lp = campaign::Native2189ALoop {
                slot: actor_index as i64,
                initial_radius: prelude.initial_radius,
                radius_step: prelude.radius_step,
                repeat: prelude.repeat,
                work_offset: 0x8088,
                work_stride: 456,
                map_rows: 8,
                map_columns: 13,
                clip_width: 312,
                clip_height: 192,
                present_stride: 320,
                ..Default::default()
            };
            let done = Box::new(|g: &mut Game| {
                if let Some(job) = g.native_cmd1012.as_mut() {
                    job.prelude_done = true;
                    (job.phase, job.frame) = (NATIVE_COMMAND1012_MAIN, 0);
                }
                let sfx = g.sfx_command1012_main.clone();
                g.play_raw(&sfx);
            });
            if let Err(err) = self.start_native_2189a(&lp, done) {
                self.native_cmd1012 = None;
                return Err(err);
            }
            let sfx = self.sfx_command1012_prelude.clone();
            self.play_raw(&sfx);
        } else {
            let sfx = self.sfx_command1012_main.clone();
            self.play_raw(&sfx);
        }
        Ok(())
    }

    /// 原版索引色分支；HD 畫格（`drawPreparedModernHDMapFrame`）屬 R6。
    ///
    /// Go：`native_command10_12_presentation.go:299`，`func (g *Game) drawNativeCommand1012Presentation(screen *ebiten.Image) bool`
    pub(crate) fn present_native_command1012(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(j) = self.native_cmd1012.as_mut() else {
            return false;
        };
        if j.palette.len() != 256 || j.phase == NATIVE_COMMAND1012_PRELUDE {
            return false;
        }
        let frames = match j.phase {
            NATIVE_COMMAND1012_MAIN => Some(&j.main_frames),
            NATIVE_COMMAND1012_RESULT => Some(&j.result_frames),
            _ => None,
        };
        let pixels = match frames {
            Some(frames) => match usize::try_from(j.frame).ok().and_then(|i| frames.get(i)) {
                Some(pixels) => pixels,
                None => return false,
            },
            None => &j.baseline_vga,
        };
        if pixels.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE {
            return false;
        }
        if let Some(screen) = screen {
            let image = Picture::from_indexed(&IndexedFrame {
                width: 320,
                height: 200,
                pix: pixels.clone(),
                palette: j.palette.clone(),
            });
            let mut op = DrawImageOptions::default();
            op.geo_m.scale(2.0, 2.0);
            screen.draw_image(&image, &op);
        }
        j.drawn = true;
        true
    }
}

impl Game {
    /// Go：`native_command10_12_presentation.go:231`，`func (g *Game) stepNativeCommand1012Presentation()`
    pub(crate) fn step_native_command1012_presentation(&mut self) {
        let Some(j) = self.native_cmd1012.as_mut() else {
            return;
        };
        if j.phase == NATIVE_COMMAND1012_PRELUDE {
            if self.native2189_a.is_none() && !j.prelude_done {
                self.fail_native_command1012_presentation(&game_err!("prelude ended without completion"));
            }
            return;
        }
        if j.phase == NATIVE_COMMAND1012_MAIN {
            if !j.drawn {
                return;
            }
            j.drawn = false;
            if j.frame == 0 {
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::apply_native_command_damage_mp(units, j.plan.as_mut()) {
                    self.fail_native_command1012_presentation(&err.into());
                    return;
                }
            }
            let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
            j.frame += 1;
            let frame = j.frame;
            let markers = j.schedule.main_sample_frames;
            for marker in markers {
                if frame == marker {
                    let sfx = self.sfx_command1012_main.clone();
                    self.play_raw(&sfx);
                }
            }
            let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
            if j.frame < j.main_frames.len() as i64 {
                return;
            }
            let (targets, damage_stages) =
                j.plan.as_ref().map_or((0, 0), |plan| (plan.results.len(), plan.damage_stages));
            for index in 0..targets {
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) =
                    battle::apply_native_command_damage_stage(units, j.plan.as_mut(), index as i64, damage_stages)
                {
                    self.fail_native_command1012_presentation(&err.into());
                    return;
                }
            }
            let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
            self.native_rng_state = j.plan.as_ref().expect("Go：j.plan 已解參考").rng_after;
            (j.phase, j.frame) = (NATIVE_COMMAND1012_RESULT, 0);
            return;
        }
        if j.phase == NATIVE_COMMAND1012_RESULT {
            if !j.drawn {
                return;
            }
            j.drawn = false;
            j.frame += 1;
            if j.frame >= j.result_frames.len() as i64 {
                (j.phase, j.hold) = (NATIVE_COMMAND1012_HOLD, native_delay_ticks(j.schedule.result_hold_ms));
            }
            return;
        }
        if j.hold > 0 {
            j.hold -= 1;
            return;
        }
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
        if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
            self.fail_native_command1012_presentation(&err.into());
            return;
        }
        let mut j = self.native_cmd1012.take().expect("Go：j 仍是同一個工作");
        let (then, results) =
            (j.then.take(), j.plan.take().expect("Go：CompleteNativeCommandDamage 已檢查 plan").results);
        (self.native_map_work, self.native_map_vga) = (j.baseline_work, j.baseline_vga);
        if let Some(then) = then {
            then(self, results);
        }
    }

    /// Go：`native_command10_12_presentation.go:216`，`func (g *Game) failNativeCommand1012Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand1012Presentation` 補移植）。
    pub(crate) fn fail_native_command1012_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd1012.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        let actor = j.actor.expect("Go：j.actor 為 nil 時解參考 panic");
        (units[actor].mp, units[actor].acted) = (j.actor_mp_before, j.actor_acted_before);
        for (i, result) in j.plan.as_ref().expect("Go：j.plan 為 nil 時解參考 panic").results.iter().enumerate() {
            units[result.target.expect("Go：result.Target 為 nil 時解參考 panic")].hp = j.target_hp_before[i];
        }
        self.native_rng_state = j.rng_before;
        (self.native_map_work, self.native_map_vga) = (j.baseline_work, j.baseline_vga);
        self.load_err = format!("native command10-12 presentation: {}", err.message());
    }
}

#[cfg(test)]
#[path = "native_command10_12_presentation/native_command10_12_presentation_test.rs"]
mod native_command10_12_presentation_test;
