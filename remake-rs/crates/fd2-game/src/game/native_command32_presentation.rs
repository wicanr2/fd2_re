//! Go 來源：`remake/cmd/fd2/native_command32_presentation.go`（522 行）。
//!
//! 原版索引色路徑都已移植；HD 綁定（`bindModernHDCompound*`、`stageModernHDCompoundMap`、`drawModernHDCompoundFrame`）屬 R6。
//! `nativeCommandHealTailTargets`（Go `native_command_heal_presentation.go:60`）也放在這裡，指令 32～35 共用。

use super::*;

/// Go：`native_command32_presentation.go:16`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeCompoundPresentedFrame {
    pub(crate) hd_map: Option<ModernHDCompoundMapSource>,
    pub(crate) hd_sources: Option<ModernHDBattleSources>,
    pub(crate) hd_palette: ColorPalette,
    pub(crate) hd_resource: i64,
    pub(crate) hd_mix: f32,
    pub(crate) hd_target: [f32; 3],
    pub(crate) hd_subtract: f32,
    pub(crate) hd_phase: String,
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) delay: i64,
    pub(crate) sound: Vec<u8>,
    pub(crate) sound_played: bool,
}

/// Go：`native_command32_presentation.go:31`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand32PresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCompoundCommand32Plan>,
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
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, battle::NativeCompoundCommand32Result) + Send>>,
}

/// 交易後的戰場複本：每個目標的 HP 換成結算值，用來預先合成扣血後的地圖畫面。
///
/// Go：`native_command32_presentation.go:45`，`func nativeCommand32ClonedState(st *battle.State, result battle.NativeCompoundCommand32Result) (*battle.State, error)`
pub(crate) fn native_command32_cloned_state(
    st: Option<&battle::State>,
    result: &battle::NativeCompoundCommand32Result,
) -> Result<battle::State> {
    let Some(st) = st.filter(|st| !st.units.is_empty()) else {
        return Err(game_err!("native command32 cloned state unavailable"));
    };
    let mut clone = st.clone();
    for (index, target) in result.targets.iter().enumerate() {
        let Some(copied) = target.target.and_then(|t| clone.units.get_mut(t)) else {
            return Err(game_err!("native command32 cloned target {index} unavailable"));
        };
        if target.hp_after < 0 || target.hp_after > copied.max_hp {
            return Err(game_err!("native command32 cloned target {index} unavailable"));
        }
        copied.hp = target.hp_after;
    }
    Ok(clone)
}

/// 每一格各自以 `InterpolateNativeCompoundDAC(baseline, 0, 0xff, delta, raw)` 的調色盤轉成畫面。
///
/// Go：`native_command32_presentation.go:70`，`func nativeCompoundImagesWithDAC(frames [][]byte, baselineDAC []byte, deltas []int, raw [3]byte) ([]*ebiten.Image, error)`
pub(crate) fn native_compound_images_with_dac(
    frames: &[Vec<u8>],
    baseline_dac: &[u8],
    deltas: &[i64],
    raw: [u8; 3],
) -> Result<Vec<Option<Arc<Picture>>>> {
    if frames.len() != deltas.len() {
        return Err(game_err!("native command32 frame/DAC schedule mismatch"));
    }
    let mut out = Vec::with_capacity(frames.len());
    for (pixels, &delta) in frames.iter().zip(deltas) {
        let dac = fdother::interpolate_native_compound_dac(baseline_dac, 0, 0xff, delta, raw)?;
        let palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&dac)?);
        out.extend(native_command24_indexed_images(std::slice::from_ref(pixels), &palette)?);
    }
    Ok(out)
}

/// Go：`native_command32_presentation.go:93`，`func appendNativeCompoundFrames(dst []nativeCompoundPresentedFrame, images []*ebiten.Image, delay int) ([]nativeCompoundPresentedFrame, error)`
///
/// Go 回傳新切片；Rust 直接接在 `dst` 後面。
pub(crate) fn append_native_compound_frames(
    dst: &mut Vec<NativeCompoundPresentedFrame>,
    images: Vec<Option<Arc<Picture>>>,
    delay: i64,
) -> Result<()> {
    if images.is_empty() || delay <= 0 {
        return Err(game_err!("native command32 presented frames unavailable"));
    }
    for image in images {
        if image.is_none() {
            return Err(game_err!("native command32 nil presented frame"));
        }
        dst.push(NativeCompoundPresentedFrame { image, delay, ..Default::default() });
    }
    Ok(())
}

/// Go：`native_command_heal_presentation.go:60`，`func nativeCommandHealTailTargets(st *battle.State, targets []*battle.Unit) ([]indexedmap.NativeCommandHealTailTarget, error)`
pub(crate) fn native_command_heal_tail_targets(
    st: Option<&battle::State>,
    targets: &[Option<UnitIdx>],
) -> Result<Vec<indexedmap::NativeCommandHealTailTarget>> {
    let Some(st) = st.filter(|_| !targets.is_empty()) else {
        return Err(game_err!("native command heal tail target array unavailable"));
    };
    let mut out = Vec::with_capacity(targets.len());
    for &target in targets {
        let Some((index, unit)) = target.and_then(|t| st.units.get(t).map(|u| (t, u))) else {
            return Err(game_err!("native command heal tail target lacks raw record presentation"));
        };
        if !unit.has_native_map_presentation || !unit.has_map_selector_slot {
            return Err(game_err!("native command heal tail target lacks raw record presentation"));
        }
        out.push(indexedmap::NativeCommandHealTailTarget {
            record_index: index as i64,
            x: i64::from(unit.native_map_presentation.x),
            y: i64::from(unit.native_map_presentation.y),
            selector_slot: unit.map_selector_slot,
        });
    }
    Ok(out)
}

impl Game {
    /// 指令 32～35 建立前要求沒有其他指令演出在跑（Go 各建立函式的同一串 nil 檢查，指令 32 的版本）。
    pub(crate) fn native_command32_presentation_busy(&self) -> bool {
        self.native_cmd32_presentation.is_some()
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
            || self.native_cmd9_player.is_some()
            || self.native_cmd9_ai_presentation.is_some()
            || self.native_cmd1012.is_some()
            || self.native_cmd24_presentation.is_some()
            || self.native_cmd29_presentation.is_some()
            || self.indexed_transition.is_some()
            || self.atk.is_some()
    }

    /// 玩家指令 32（0x27FC9 複合演出）：前奏、施放者特效、FDOTHER #65 共用段、地圖調色盤漸變、
    /// 地圖上的受擊與傷害數字。扣血與地圖畫面在數字段第一格才寫入。HD 綁定（`bindModernHDCompound*`、
    /// `stageModernHDCompoundMap`）屬 R6，這裡只建原版索引色畫格。
    ///
    /// Go：`native_command32_presentation.go:106`，`func (g *Game) startNativeCommand32Presentation(actor, confirmed *battle.Unit, then func(battle.NativeCompoundCommand32Result)) error`
    pub(crate) fn start_native_command32_presentation(
        &mut self,
        actor_index: UnitIdx,
        confirmed: Option<UnitIdx>,
        then: Box<dyn FnOnce(&mut Game, battle::NativeCompoundCommand32Result) + Send>,
    ) -> Result<()> {
        let actor = self.st.as_ref().and_then(|st| st.units.get(actor_index)).cloned();
        let (true, Some(actor), Some(confirmed)) =
            (self.current_native_system_options().full_presentation_enabled(), actor, confirmed)
        else {
            return Err(game_err!("native command32 presentation context unavailable"));
        };
        self.require_separated_command_sounds(91, &[1, 2])?;
        if self.native_command32_presentation_busy() {
            return Err(game_err!("native command32 presentation already active"));
        }
        let st = self.st.as_ref().unwrap();
        if !st.has_native_map_view_state
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || self.native_map_dac.len() != 256 * 3
            || !native_map_assets_available(self.native_map_assets.as_ref())
        {
            return Err(game_err!("native command32 indexed map state unavailable"));
        }
        let plan = st.plan_native_compound_command32(Some(actor_index), Some(confirmed), self.native_rng_state)?;
        if plan.result.targets.is_empty() {
            return Err(game_err!("native command32 final target array is empty"));
        }
        let animations = separated_asset_path("animations");
        let actor_idle = figani::load_separated_resource(&animations, actor.battle_fig * 3)
            .ok()
            .filter(|idle| !idle.frames.is_empty())
            .ok_or_else(|| game_err!("native command32 actor idle FIGANI unavailable"))?;
        let actor_effect = figani::load_separated_resource(&animations, actor.battle_fig * 3 + 1)?;
        let first_target = plan.result.targets[0]
            .target
            .and_then(|t| st.units.get(t))
            .filter(|t| t.has_battle_fig)
            .ok_or_else(|| game_err!("native command32 first target BattleFig unavailable"))?;
        let target_idle = figani::load_separated_resource(&animations, first_target.battle_fig * 3)
            .ok()
            .filter(|idle| !idle.frames.is_empty())
            .ok_or_else(|| game_err!("native command32 target idle FIGANI unavailable"))?;
        let common_effect = figani::load_separated_archive_resource(&animations, "FDOTHER.DAT", 65)?;
        let common_schedule = figani::build_native_compound_presentation_schedule(32, Some(&common_effect))?;
        // Go 在這裡 stage HD 庫（`stageBattleHDBanks`，R6）。
        let assets = self.native_map_assets.as_ref().unwrap();
        let tail_schedule =
            fdother::build_native_command32_tail_schedule(&assets.fdother6, &assets.command_heal_digits)?;
        let (palette_dac, battle_palette) = load_native_battle_palette()
            .ok()
            .filter(|(dac, _)| dac.len() == 256 * 3)
            .ok_or_else(|| game_err!("native command32 battle DAC unavailable"))?;
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
            actor_effect.frames.first().ok_or_else(|| game_err!("native command32 actor effect is empty"))?,
            Some(&common_effect),
            common_schedule,
        )?;
        let actor_slide = native_command24_indexed_images(&common.actor_slide, &battle_palette)?;
        let effect_slide = native_command24_indexed_images(&common.effect_slide, &battle_palette)?;
        let main_images = native_command24_indexed_images(&common.main, &battle_palette)?;
        let tail_deltas: Vec<i64> = (0..common.tail.len() as i64).map(|index| 40 - 4 * index).collect();
        let common_tail_images =
            native_compound_images_with_dac(&common.tail, &palette_dac, &tail_deltas, common_schedule.palette_rgb)?;

        let st = self.st.as_ref().unwrap();
        let mut target_units = Vec::with_capacity(plan.result.targets.len());
        let mut results = Vec::with_capacity(plan.result.targets.len());
        for target in &plan.result.targets {
            target_units.push(target.target);
            let record_index = native_command24_runtime_unit_index(Some(st), target.target)?;
            results.push(battlepresent::NativeCommand32TailResult {
                target_record: record_index,
                hit: target.damage.hit,
                damage: target.damage.damage,
            });
        }
        let tail_targets = native_command_heal_tail_targets(Some(st), &target_units)?;
        let cloned_state = native_command32_cloned_state(Some(st), &plan.result)?;
        let (hud, hud_ok) = self.native_map_hud_input();
        if !hud_ok {
            return Err(game_err!("native command32 post-transaction HUD unavailable"));
        }
        let mut post_input = build_native_map_frame_input(
            self.native_map_assets.as_ref(),
            self.m.as_ref(),
            Some(&cloned_state),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )?;
        let mut post_work = self.native_map_work.clone();
        let mut post_vga = self.native_map_vga.clone();
        indexedmap::compose_native_frame(&mut post_work, &mut post_vga, &mut post_input)?;
        let view = self.st.as_ref().unwrap().native_map_view_state;
        let assets = self.native_map_assets.as_ref().unwrap();
        let tail = battlepresent::build_native_command32_tail_frames(
            &self.native_map_work,
            &self.native_map_vga,
            &post_work,
            &post_vga,
            &assets.fdother6,
            &assets.command_heal_digits,
            &tail_targets,
            &results,
            view.camera_x,
            view.camera_y,
            &tail_schedule,
        )?;
        // Go 另凍結地圖輸入給 HD 複合地圖場景（`freezeNativeCommand9MapInput`、`newModernHDCompoundMapScene`，R6）。
        let map_palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&self.native_map_dac)?);
        let tail_effect_images = native_command24_indexed_images(&tail.effect, &map_palette)?;
        let toggle_images = native_command24_indexed_images(&tail.toggle, &map_palette)?;
        let result_images = native_command24_indexed_images(&tail.result, &map_palette)?;
        let ramp_pixels: Vec<Vec<u8>> = (0..=40).map(|_| self.native_map_vga.clone()).collect();
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
        frames[main_start].sound = self.separated_command_sound(91, 2);
        let tail_start = frames.len();
        let common_tail_len = common_tail_images.len();
        append_native_compound_frames(&mut frames, common_tail_images, common_schedule.main_delay_ticks)?;
        let common_sample1 = self.separated_command_sound(91, 1);
        for index in (0..common_tail_len).step_by(2) {
            frames[tail_start + index].sound = common_sample1.clone();
        }
        append_native_compound_frames(&mut frames, ramp_images, 6)?;
        let effect_start = frames.len();
        append_native_compound_frames(&mut frames, tail_effect_images, tail_schedule.effect_delay_ticks)?;
        frames[effect_start].sound = self.separated_command_sound(80, 9);
        append_native_compound_frames(&mut frames, toggle_images, native_delay_ticks(tail_schedule.toggle_delay_ms))?;
        let publish_at = frames.len() as i64;
        append_native_compound_frames(&mut frames, result_images, 1)?;
        if !os_mute_or_shot(Some(self))
            && [&frames[main_start].sound, &common_sample1, &frames[effect_start].sound].iter().any(|s| s.is_empty())
        {
            return Err(game_err!("native command32 required raw sample unavailable"));
        }
        // Go 在這裡綁 HD 背景、stage HD 地圖並綁狀態框文字（R6）。
        self.native_cmd32_presentation = Some(NativeCommand32PresentationJob {
            actor: Some(actor_index),
            plan: Some(plan),
            frames,
            publish_at,
            rng_before: self.native_rng_state,
            baseline_work: self.native_map_work.clone(),
            baseline_vga: self.native_map_vga.clone(),
            post_transaction_work: post_work,
            post_vga,
            then: Some(then),
            ..Default::default()
        });
        Ok(())
    }

    /// 原版索引色分支；HD 複合畫格（`drawModernHDCompoundFrame`）屬 R6。
    ///
    /// Go：`native_command32_presentation.go:505`，`func (g *Game) drawNativeCommand32Presentation(screen *ebiten.Image) bool`
    pub(crate) fn present_native_command32(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(j) = self.native_cmd32_presentation.as_mut() else {
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
    /// Go：`native_command32_presentation.go:429`，`func (g *Game) cancelNativeCommand32Presentation()`
    pub(crate) fn cancel_native_command32_presentation(&mut self) {
        let Some(mut j) = self.native_cmd32_presentation.take() else {
            return;
        };
        let _ = battle::abort_native_compound_command32(presentation_units(&mut self.st), j.plan.as_mut());
        self.native_rng_state = j.rng_before;
        self.native_map_work = j.baseline_work;
        self.native_map_vga = j.baseline_vga;
    }

    /// Go：`native_command32_presentation.go:441`，`func (g *Game) failNativeCommand32Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand32Presentation` 補移植）。
    pub(crate) fn fail_native_command32_presentation(&mut self, err: &GameError) {
        self.cancel_native_command32_presentation();
        self.load_err = format!("native command32 presentation: {}", err.message());
    }

    /// Go：`native_command32_presentation.go:446`，`func (g *Game) stepNativeCommand32Presentation()`
    pub(crate) fn step_native_command32_presentation(&mut self) {
        let Some(j) = self.native_cmd32_presentation.as_mut() else {
            return;
        };
        if j.holding {
            if j.hold > 0 {
                j.hold -= 1;
                return;
            }
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::complete_native_compound_command32(units, j.plan.as_mut()) {
                self.fail_native_command32_presentation(&err.into());
                return;
            }
            let mut j = self.native_cmd32_presentation.take().expect("Go：j 仍是同一個工作");
            let (then, result) =
                (j.then.take(), j.plan.take().expect("Go：CompleteNativeCompoundCommand32 已檢查 plan").result);
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
            let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
            j.frames[j.frame as usize].sound_played = true;
        }
        let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.repeat += 1;
        if j.repeat < j.frames[j.frame as usize].delay {
            return;
        }
        j.repeat = 0;
        let next = j.frame + 1;
        if next == j.publish_at && !j.published {
            let targets = j.plan.as_ref().map_or(0, |plan| plan.result.targets.len());
            for index in 0..targets {
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::apply_native_compound_command32_target(units, j.plan.as_mut(), index as i64) {
                    self.fail_native_command32_presentation(&err.into());
                    return;
                }
            }
            let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
            self.native_rng_state = j.plan.as_ref().expect("Go：j.plan 已解參考").result.rng_state;
            self.native_map_work = j.post_transaction_work.clone();
            self.native_map_vga = j.post_vga.clone();
            j.published = true;
        }
        let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.frame = next;
        if j.frame >= j.frames.len() as i64 {
            if !j.published {
                self.fail_native_command32_presentation(&game_err!("required damage boundary was not presented"));
                return;
            }
            j.frame = j.frames.len() as i64 - 1;
            (j.holding, j.hold) = (true, native_delay_ticks(500));
        }
    }
}

#[cfg(test)]
#[path = "native_command32_presentation/native_command32_presentation_test.rs"]
mod native_command32_presentation_test;
