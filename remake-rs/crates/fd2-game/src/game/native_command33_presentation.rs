//! Go 來源：`remake/cmd/fd2/native_command33_presentation.go`（487 行）。
//!
//! 原版索引色路徑都已移植；HD 綁定（`bindModernHDCompound*`、`stageModernHDCompoundMap`、`drawModernHDCompoundFrame`）屬 R6。

use super::*;

/// Go：`native_command33_presentation.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand33PresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCompoundCommand33Plan>,
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) publish_at: i64,
    pub(crate) published: bool,
    pub(crate) drawn: bool,
    pub(crate) holding: bool,
    pub(crate) hold: i64,
    pub(crate) rng_before: u16,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) post_transaction_work: Vec<u8>,
    pub(crate) post_vga: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, battle::NativeCompoundCommand33Result) + Send>>,
}

/// 交易後的戰場複本：目標的 HP 與暫態位元組換成結算值。
///
/// Go：`native_command33_presentation.go:29`，`func nativeCommand33ClonedState(st *battle.State, result battle.NativeCompoundCommand33Result) (*battle.State, error)`
pub(crate) fn native_command33_cloned_state(
    st: Option<&battle::State>,
    result: &battle::NativeCompoundCommand33Result,
) -> Result<battle::State> {
    let Some(st) = st.filter(|st| !st.units.is_empty() && !result.targets.is_empty()) else {
        return Err(game_err!("native command33 cloned state unavailable"));
    };
    let mut clone = st.clone();
    for (index, target) in result.targets.iter().enumerate() {
        let Some(copied) = target.target.and_then(|t| clone.units.get_mut(t)) else {
            return Err(game_err!("native command33 cloned target {index} unavailable"));
        };
        if target.hp_after < 0 || target.hp_after > copied.max_hp {
            return Err(game_err!("native command33 cloned target {index} unavailable"));
        }
        copied.hp = target.hp_after;
        copied.native_transient = target.transient_after;
    }
    Ok(clone)
}

impl Game {
    /// 玩家指令 33（0x27FC9 → 0x211A4）：前段與指令 32 相同（FDOTHER #66），地圖尾段是回復效果、
    /// 遮罩閃爍與回復數字。清除暫態與回復 HP 在數字段第一格才寫入。HD 綁定屬 R6。
    ///
    /// Go：`native_command33_presentation.go:55`，`func (g *Game) startNativeCommand33Presentation(actor, confirmed *battle.Unit, then func(battle.NativeCompoundCommand33Result)) error`
    pub(crate) fn start_native_command33_presentation(
        &mut self,
        actor_index: UnitIdx,
        confirmed: Option<UnitIdx>,
        then: Box<dyn FnOnce(&mut Game, battle::NativeCompoundCommand33Result) + Send>,
    ) -> Result<()> {
        let actor = self.st.as_ref().and_then(|st| st.units.get(actor_index)).cloned();
        let (true, Some(actor), Some(confirmed)) =
            (self.current_native_system_options().full_presentation_enabled(), actor, confirmed)
        else {
            return Err(game_err!("native command33 presentation context unavailable"));
        };
        self.require_separated_command_sounds(92, &[1])?;
        if self.native_cmd34_presentation.is_some()
            || self.native_cmd33_presentation.is_some()
            || self.native_command32_presentation_busy()
        {
            return Err(game_err!("native command33 presentation already active"));
        }
        let st = self.st.as_ref().unwrap();
        if !st.has_native_map_view_state
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || self.native_map_dac.len() != 256 * 3
            || !native_map_assets_available(self.native_map_assets.as_ref())
        {
            return Err(game_err!("native command33 indexed map state unavailable"));
        }
        let plan = st.plan_native_compound_command33(Some(actor_index), Some(confirmed), self.native_rng_state)?;
        if plan.result.targets.is_empty() || plan.result.restore.results.len() != plan.result.targets.len() {
            return Err(game_err!("native command33 final target array unavailable"));
        }
        let animations = separated_asset_path("animations");
        let actor_idle = figani::load_separated_resource(&animations, actor.battle_fig * 3)
            .ok()
            .filter(|idle| !idle.frames.is_empty())
            .ok_or_else(|| game_err!("native command33 actor idle FIGANI unavailable"))?;
        let actor_effect = figani::load_separated_resource(&animations, actor.battle_fig * 3 + 1)?;
        let first_target = plan.result.targets[0]
            .target
            .and_then(|t| st.units.get(t))
            .filter(|t| t.has_battle_fig)
            .ok_or_else(|| game_err!("native command33 first target BattleFig unavailable"))?;
        let target_idle = figani::load_separated_resource(&animations, first_target.battle_fig * 3)
            .ok()
            .filter(|idle| !idle.frames.is_empty())
            .ok_or_else(|| game_err!("native command33 target idle FIGANI unavailable"))?;
        let common_effect = figani::load_separated_archive_resource(&animations, "FDOTHER.DAT", 66)?;
        let common_schedule = figani::build_native_compound_presentation_schedule(33, Some(&common_effect))
            .ok()
            .filter(|s| !s.tail_enabled && s.sample1_frame == 6)
            .ok_or_else(|| game_err!("native command33 common schedule unavailable"))?;
        // Go 在這裡 stage HD 庫（`stageBattleHDBanks`，R6）。
        let tail_schedule = fdother::build_native_command33_tail_schedule()?;
        let a = self.native_map_assets.as_ref().unwrap();
        if (a.fdother6.len() as i64) < tail_schedule.effect_start + tail_schedule.effect_frames
            || a.command_heal_digits.len() as i64 <= tail_schedule.digit_bias + 9
        {
            return Err(game_err!("native command33 tail FDOTHER #5/#6 descriptors unavailable"));
        }
        let (palette_dac, battle_palette) = load_native_battle_palette()
            .ok()
            .filter(|(dac, _)| dac.len() == 256 * 3)
            .ok_or_else(|| game_err!("native command33 battle DAC unavailable"))?;
        let actor_selector = native_command24_bg_selector(self.m.as_ref(), &actor)?;
        let surfaces = separated_asset_path("surfaces");
        let background = fdother::load_separated_single_frame(&surfaces, "BG.DAT", actor_selector)?;
        let panel_assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let actor_record = battle::native_battle_panel_record_for_unit(Some(&actor))?;
        let actor_unit_index = native_command24_runtime_unit_index(self.st.as_ref(), Some(actor_index))?;
        let platform = if actor.native_record_byte6 != 0 {
            Some(fdother::load_separated_single_frame(&surfaces, "TAI.DAT", actor_selector)?)
        } else {
            None
        };
        let chapter = self.handler_chapter;
        let actor_base = self.localized_native_command24_captured_base(
            &background,
            &panel_assets,
            &actor_record,
            actor_unit_index,
            chapter,
            platform.as_ref(),
            None,
            &[],
        )?;
        let prelude_base = self.localized_native_command24_captured_base(
            &background,
            &panel_assets,
            &actor_record,
            actor_unit_index,
            chapter,
            None,
            None,
            &[],
        )?;
        let prelude = battlepresent::build_native_command_prelude_frames(&battlepresent::NativeCommandPreludeInput {
            base: &prelude_base,
            actor_idle: actor_idle.frames[0].clone(),
            platform: platform.as_ref(),
            raw_side: actor.native_record_byte6,
            mode: 1,
            baseline_dac: &palette_dac,
            ..Default::default()
        })?;
        let prelude_images: Vec<Option<Arc<Picture>>> = native_command24_prelude_images(&prelude)?
            .into_iter()
            .map(|frame| Some(Arc::new(Picture::from_indexed(&frame))))
            .collect();
        let actor_presents = battlepresent::build_native_compound_actor_presents(
            &actor_base,
            Some(&actor_effect),
            Some(&target_idle),
            actor.native_record_byte6,
        )?;
        let actor_pixels: Vec<Vec<u8>> = actor_presents.into_iter().map(|present| present.pixels).collect();
        let actor_images = native_command24_indexed_images(&actor_pixels, &battle_palette)?;
        let common = battlepresent::build_native_compound_common_frames(
            &actor_base,
            actor_effect.frames.first().ok_or_else(|| game_err!("native command33 actor effect is empty"))?,
            Some(&common_effect),
            common_schedule,
        )?;
        let actor_slide = native_command24_indexed_images(&common.actor_slide, &battle_palette)?;
        let effect_slide = native_command24_indexed_images(&common.effect_slide, &battle_palette)?;
        let main_images = native_command24_indexed_images(&common.main, &battle_palette)
            .ok()
            .filter(|images| images.len() as i64 >= common_schedule.sample1_frame)
            .ok_or_else(|| game_err!("native command33 main frames unavailable"))?;

        let st = self.st.as_ref().unwrap();
        let view = st.native_map_view_state;
        let target_units: Vec<Option<UnitIdx>> = plan.result.targets.iter().map(|target| target.target).collect();
        let tail_targets = native_command_heal_tail_targets(Some(st), &target_units)?;
        let effect_targets: Vec<indexedmap::NativeCommandHealTailTarget> = tail_targets
            .iter()
            .copied()
            .filter(|t| {
                (view.camera_x - 1..=view.camera_x + 12).contains(&t.x)
                    && (view.camera_y - 1..=view.camera_y + 8).contains(&t.y)
            })
            .collect();
        // Go 另凍結地圖輸入給 HD 複合地圖場景（`freezeNativeCommand9MapInput`，R6）。
        let (baseline_work, baseline_vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        let a = self.native_map_assets.as_ref().unwrap();
        let mut tail_effect_pixels = Vec::with_capacity(tail_schedule.effect_frames as usize);
        for frame in 0..tail_schedule.effect_frames {
            let (mut work, mut vga) = (baseline_work.clone(), baseline_vga.clone());
            indexedmap::compose_native_command_heal_effect_frame_with_sources(
                &mut work,
                &mut vga,
                &baseline_work,
                &a.fdother6[(tail_schedule.effect_start + frame) as usize],
                &effect_targets,
                view.camera_x,
                view.camera_y,
                6,
                tail_schedule.effect_start + frame,
            )?;
            tail_effect_pixels.push(vga);
        }
        let roster = st.native_map_frame_roster()?;
        let (mut mask_work, mut mask_vga) = (baseline_work.clone(), baseline_vga.clone());
        indexedmap::compose_native_command_heal_mask_frame(
            &mut mask_work,
            &mut mask_vga,
            &baseline_work,
            a.units.as_ref(),
            st.native_map_selector_cache.as_ref(),
            &effect_targets,
            view.camera_x,
            view.camera_y,
            roster.cycles.idle,
            tail_schedule.mask_index as u8,
        )?;
        let cloned_state = native_command33_cloned_state(Some(st), &plan.result)?;
        let (hud, hud_ok) = self.native_map_hud_input();
        if !hud_ok {
            return Err(game_err!("native command33 post-transaction HUD unavailable"));
        }
        let mut post_input = build_native_map_frame_input(
            self.native_map_assets.as_ref(),
            self.m.as_ref(),
            Some(&cloned_state),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )?;
        let (mut post_work, mut post_vga) = (baseline_work.clone(), baseline_vga.clone());
        indexedmap::compose_native_frame(&mut post_work, &mut post_vga, &mut post_input)?;
        let mut queue = Vec::with_capacity(plan.result.targets.len() * 4);
        for (index, result) in plan.result.restore.results.iter().enumerate() {
            let t = &tail_targets[index];
            let in_camera = (view.camera_x..view.camera_x + 12).contains(&t.x)
                && (view.camera_y - 1..=view.camera_y + 7).contains(&t.y);
            queue = battle::append_native_presentation_digits(
                &queue,
                result.actual,
                tail_schedule.digit_bias,
                t.record_index,
                in_camera,
            )?;
        }
        let a = self.native_map_assets.as_ref().unwrap();
        let mut digit_pixels = Vec::with_capacity(tail_schedule.digit_frames as usize);
        for frame in 0..tail_schedule.digit_frames {
            let (mut work, mut vga) = (post_work.clone(), post_vga.clone());
            indexedmap::compose_native_command_heal_digit_frame_with_sources(
                &mut work,
                &mut vga,
                &post_work,
                &a.command_heal_digits,
                &queue,
                &tail_targets,
                view.camera_x,
                view.camera_y,
                tail_schedule.digit_vertical,
                frame,
            )?;
            digit_pixels.push(vga);
        }
        let map_palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&self.native_map_dac)?);
        let tail_effect_images = native_command24_indexed_images(&tail_effect_pixels, &map_palette)?;
        let mask_images = native_command24_indexed_images(&[baseline_vga.clone(), mask_vga], &map_palette)?;
        let digit_images = native_command24_indexed_images(&digit_pixels, &map_palette)?;
        let ramp_pixels: Vec<Vec<u8>> = (0..=40).map(|_| baseline_vga.clone()).collect();
        let ramp_deltas: Vec<i64> = (0..=40).collect();
        let ramp_images = native_compound_images_with_dac(
            &ramp_pixels,
            &self.native_map_dac,
            &ramp_deltas,
            common_schedule.palette_rgb,
        )?;

        let mut frames = Vec::new();
        append_native_compound_frames(&mut frames, prelude_images, 1)?;
        append_native_compound_frames(&mut frames, actor_images, 1)?;
        frames.last_mut().unwrap().delay += common_schedule.prelude_hold_ticks;
        append_native_compound_frames(&mut frames, actor_slide, 1)?;
        append_native_compound_frames(&mut frames, effect_slide, 1)?;
        let main_start = frames.len();
        append_native_compound_frames(&mut frames, main_images, common_schedule.main_delay_ticks)?;
        let sample1_at = main_start + common_schedule.sample1_frame as usize - 1;
        frames[sample1_at].sound = self.separated_command_sound(92, 1);
        append_native_compound_frames(&mut frames, ramp_images, 6)?;
        let effect_start = frames.len();
        append_native_compound_frames(&mut frames, tail_effect_images, tail_schedule.effect_frame_delay_ticks)?;
        frames[effect_start].sound = self.separated_command_sound(80, 12);
        let mask_start = frames.len();
        for index in 0..tail_schedule.mask_pairs * 2 + 1 {
            let image = mask_images[index as usize % 2].clone();
            frames.push(NativeCompoundPresentedFrame { image, delay: 1, ..Default::default() });
        }
        frames[mask_start].sound = self.separated_command_sound(80, 1);
        let publish_at = frames.len() as i64;
        append_native_compound_frames(&mut frames, digit_images, 1)?;
        if !os_mute_or_shot(Some(self))
            && [&frames[sample1_at].sound, &frames[effect_start].sound, &frames[mask_start].sound]
                .iter()
                .any(|s| s.is_empty())
        {
            return Err(game_err!("native command33 required raw sample unavailable"));
        }
        // Go 在這裡綁 HD 背景、stage HD 地圖並綁狀態框文字（R6）。
        self.native_cmd33_presentation = Some(NativeCommand33PresentationJob {
            actor: Some(actor_index),
            plan: Some(plan),
            frames,
            publish_at,
            rng_before: self.native_rng_state,
            baseline_work,
            baseline_vga,
            post_transaction_work: post_work,
            post_vga,
            then: Some(then),
            ..Default::default()
        });
        Ok(())
    }

    /// 原版索引色分支；HD 複合畫格（`drawModernHDCompoundFrame`）屬 R6。
    ///
    /// Go：`native_command33_presentation.go:470`，`func (g *Game) drawNativeCommand33Presentation(screen *ebiten.Image) bool`
    pub(crate) fn present_native_command33(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(j) = self.native_cmd33_presentation.as_mut() else {
            return false;
        };
        let Some(image) = usize::try_from(j.frame).ok().and_then(|i| j.frames.get(i)).and_then(|f| f.image.clone())
        else {
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

impl Game {
    /// Go：`native_command33_presentation.go:396`，`func (g *Game) cancelNativeCommand33Presentation()`
    pub(crate) fn cancel_native_command33_presentation(&mut self) {
        let Some(mut j) = self.native_cmd33_presentation.take() else {
            return;
        };
        let _ = battle::abort_native_compound_command33(presentation_units(&mut self.st), j.plan.as_mut());
        self.native_rng_state = j.rng_before;
        self.native_map_work = j.baseline_work;
        self.native_map_vga = j.baseline_vga;
    }

    /// Go：`native_command33_presentation.go:408`，`func (g *Game) failNativeCommand33Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand33Presentation` 補移植）。
    pub(crate) fn fail_native_command33_presentation(&mut self, err: &GameError) {
        self.cancel_native_command33_presentation();
        self.load_err = format!("native command33 presentation: {}", err.message());
    }

    /// Go：`native_command33_presentation.go:413`，`func (g *Game) stepNativeCommand33Presentation()`
    pub(crate) fn step_native_command33_presentation(&mut self) {
        let Some(j) = self.native_cmd33_presentation.as_mut() else {
            return;
        };
        if j.holding {
            if j.hold > 0 {
                j.hold -= 1;
                return;
            }
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd33_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::complete_native_compound_command33(units, j.plan.as_mut()) {
                self.fail_native_command33_presentation(&err.into());
                return;
            }
            let mut j = self.native_cmd33_presentation.take().expect("Go：j 仍是同一個工作");
            let (then, result) =
                (j.then.take(), j.plan.take().expect("Go：CompleteNativeCompoundCommand33 已檢查 plan").result);
            if let Some(then) = then {
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
            self.play_raw(&sound);
            let j = self.native_cmd33_presentation.as_mut().expect("Go：j 仍是同一個工作");
            j.frames[j.frame as usize].sound_played = true;
        }
        let j = self.native_cmd33_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.repeat += 1;
        if j.repeat < j.frames[j.frame as usize].delay {
            return;
        }
        j.repeat = 0;
        let next = j.frame + 1;
        if next == j.publish_at && !j.published {
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd33_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::publish_native_compound_command33(units, j.plan.as_mut()) {
                self.fail_native_command33_presentation(&err.into());
                return;
            }
            self.native_rng_state = j.plan.as_ref().expect("Go：j.plan 已解參考").result.restore.rng_state;
            self.native_map_work = j.post_transaction_work.clone();
            self.native_map_vga = j.post_vga.clone();
            j.published = true;
        }
        let j = self.native_cmd33_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.frame = next;
        if j.frame >= j.frames.len() as i64 {
            if !j.published {
                self.fail_native_command33_presentation(&game_err!("required restore boundary was not presented"));
                return;
            }
            j.frame = j.frames.len() as i64 - 1;
            (j.holding, j.hold) = (true, native_delay_ticks(500));
        }
    }
}

#[cfg(test)]
#[path = "native_command33_presentation/native_command33_presentation_test.rs"]
mod native_command33_presentation_test;
