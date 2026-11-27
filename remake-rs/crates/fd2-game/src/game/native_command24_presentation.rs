//! Go 來源：`remake/cmd/fd2/native_command24_presentation.go`（537 行）。
//!
//! 原版索引色路徑的函式都已移植；HD 分割打擊（`stageModernHDSplitStrikeBanks`、`bindModernHDSplitStrike`、
//! `modernFrame`）屬 R6。

use super::*;

/// Go：`native_command24_presentation.go:16`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand24PresentationJob {
    pub(crate) hd: Option<ModernHDSplitStrike>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) target: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDerivedStrikePlan>,
    pub(crate) schedule: figani::NativeCommand24Schedule,
    pub(crate) effect_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) effect_positions: Vec<[i64; 2]>,
    pub(crate) target_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) target_positions: Vec<[i64; 2]>,
    pub(crate) target_delays: Vec<i64>,
    pub(crate) prelude_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) prelude_frame: i64,
    pub(crate) actor_base_before: Option<Arc<Picture>>,
    pub(crate) actor_base_after: Option<Arc<Picture>>,
    pub(crate) target_base_before: Option<Arc<Picture>>,
    pub(crate) target_base_after: Option<Arc<Picture>>,
    pub(crate) transition_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) transition_frame: i64,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) target_frame: i64,
    pub(crate) target_repeat: i64,
    pub(crate) shake_counter: i64,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) damage_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: i64,
    pub(crate) target_raw_byte5_before: u8,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>>,
}

/// Go：`native_command24_presentation.go:48`，`func nativeCommand24BGSelector(m *MapData, unit *battle.Unit) (int, error)`
pub(crate) fn native_command24_bg_selector(m: Option<&MapData>, unit: &battle::Unit) -> Result<i64> {
    let Some(m) = m.filter(|m| {
        m.w > 0
            && m.h > 0
            && m.tiles.len() as i64 == m.w * m.h
            && !m.native_terrain_control.is_empty()
            && m.native_terrain_control.len() % 4 == 0
            && (0..m.w).contains(&unit.x)
            && (0..m.h).contains(&unit.y)
    }) else {
        return Err(game_err!("native command24 terrain control unavailable"));
    };
    let tile = m.tiles[(unit.y * m.w + unit.x) as usize];
    if !(0..=0x3ff).contains(&tile) || tile >= m.native_terrain_control.len() as i64 / 4 {
        return Err(game_err!("native command24 terrain selector out of range"));
    }
    Ok(i64::from(m.native_terrain_control[tile as usize * 4 + 2]))
}

/// Go：`native_command24_presentation.go:73`，`func nativeCommand24IndexedImages(frames [][]byte, palette color.Palette) ([]*ebiten.Image, error)`
pub(crate) fn native_command24_indexed_images(
    frames: &[Vec<u8>],
    palette: &ColorPalette,
) -> Result<Vec<Option<Arc<Picture>>>> {
    if frames.is_empty() || palette.len() < 256 {
        return Err(game_err!("native command24 indexed frames unavailable"));
    }
    let mut out = Vec::with_capacity(frames.len());
    for (i, pixels) in frames.iter().enumerate() {
        if pixels.len() != 320 * 200 {
            return Err(game_err!("native command24 indexed frame {i} is malformed"));
        }
        out.push(Some(Arc::new(Picture::from_indexed(&IndexedFrame {
            width: 320,
            height: 200,
            pix: pixels.clone(),
            palette: palette.clone(),
        }))));
    }
    Ok(out)
}

/// Go：`native_command24_presentation.go:61`，`func nativeCommand24RuntimeUnitIndex(st *battle.State, unit *battle.Unit) (int, error)`
pub(crate) fn native_command24_runtime_unit_index(st: Option<&battle::State>, unit: Option<UnitIdx>) -> Result<i64> {
    let (Some(st), Some(unit)) = (st, unit) else {
        return Err(game_err!("native command24 runtime unit unavailable"));
    };
    // Go 以指標相等在 `st.Units` 裡找；Rust 的單位就是 `st.units` 的索引，只要在範圍內。
    if unit < st.units.len() {
        return Ok(unit as i64);
    }
    Err(game_err!("native command24 unit is not in the runtime array"))
}

/// Go：`native_command24_presentation.go:89`，`func nativeCommand24PreludeImages(frames []battlepresent.NativeCommand24PreludeFrame) ([]*ebiten.Image, error)`
///
/// R4 D 組：回傳改成 `Vec<IndexedFrame>`（骨架是 `Vec<Option<Arc<Picture>>>`），畫格保存索引色像素與 DAC 轉出的調色盤。
pub(crate) fn native_command24_prelude_images(
    frames: &[battlepresent::NativeCommand24PreludeFrame],
) -> Result<Vec<IndexedFrame>> {
    if frames.len() != 9 {
        return Err(game_err!("native command24 prelude must contain nine indexed frames"));
    }
    let mut out = Vec::with_capacity(frames.len());
    for (i, frame) in frames.iter().enumerate() {
        if frame.stage != 8 - i as i64 || frame.pixels.len() != 320 * 200 {
            return Err(game_err!("native command24 prelude frame {i} is malformed"));
        }
        let palette = fdother::vga_palette_from_dac(&frame.dac)?;
        // Go `image.NewPaletted`＋`ebiten.NewImageFromImage`：Rust 保存索引色畫格（120「狀態只保存索引色畫面與調色盤」），
        // GPU 貼圖由 `fd2-app` 建立（R5）。
        out.push(IndexedFrame {
            width: 320,
            height: 200,
            pix: frame.pixels.clone(),
            palette: color_palette_from_nrgba(&palette),
        });
    }
    Ok(out)
}

impl Game {
    /// 指令 24／28～30 建立前要求沒有其他戰鬥演出在跑（Go 兩個建立函式的同一串 nil 檢查）。
    pub(crate) fn native_derived_strike_presentation_busy(&self) -> bool {
        self.native_cmd24_presentation.is_some()
            || self.native_cmd29_presentation.is_some()
            || self.native_cmd0_presentation.is_some()
            || self.native_cmd1_presentation.is_some()
            || self.native_cmd2_presentation.is_some()
            || self.native_cmd3_presentation.is_some()
            || self.native_cmd5_presentation.is_some()
            || self.native_cmd6_presentation.is_some()
            || self.native_cmd7_presentation.is_some()
            || self.native_cmd8_presentation.is_some()
            || self.native_cmd9_ai_presentation.is_some()
            || self.native_heal_presentation.is_some()
            || self.native_modifier_presentation.is_some()
            || self.atk.is_some()
    }

    /// 玩家指令 24（0x276EC 單體衍生打擊）：九格淡入前奏、施放者特效、二十格背景滑移、目標受擊。
    /// 計畫先消耗亂數但不改狀態；MP 與傷害等演出畫到對應原版標記才寫入。
    /// HD 分割打擊（`stageModernHDSplitStrikeBanks`／`bindModernHDSplitStrike`）屬 R6，這裡只建原版索引色畫格。
    ///
    /// Go：`native_command24_presentation.go:133`，`func (g *Game) startNativeCommand24Presentation(actor, target *battle.Unit, then func([]battle.NativeCommand24Damage)) error`
    pub(crate) fn start_native_command24_presentation(
        &mut self,
        actor_index: UnitIdx,
        target_index: Option<UnitIdx>,
        then: Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>,
    ) -> Result<()> {
        if !self.current_native_system_options().full_presentation_enabled() {
            return Err(game_err!("native abbreviated presentation owner unavailable"));
        }
        let busy = self.native_derived_strike_presentation_busy();
        let units = self.st.as_ref().map(|st| &st.units);
        let (Some(units), Some(target_index), false, true) = (units, target_index, busy, self.rng.is_some()) else {
            return Err(game_err!("native command24 presentation context unavailable"));
        };
        let (Some(actor), Some(target)) = (units.get(actor_index).cloned(), units.get(target_index).cloned()) else {
            return Err(game_err!("native command24 presentation context unavailable"));
        };
        if !actor.has_battle_fig || actor.battle_fig != 32 || !target.has_battle_fig {
            return Err(game_err!("native command24 requires proven selector32 actor and target BattleFig"));
        }
        if self.native_ui_palette.len() < 256 {
            return Err(game_err!("native command24 palette unavailable"));
        }
        let animations = separated_asset_path("animations");
        let effect = figani::load_separated_resource(&animations, actor.battle_fig * 3 + 2)?;
        let schedule = figani::build_native_command24_schedule(Some(&effect))?;
        self.require_separated_command_sounds(
            schedule.audio_resource,
            &[schedule.actor_sample, schedule.target_sample],
        )?;
        let actor_idle = figani::load_separated_resource(&animations, actor.battle_fig * 3)?;
        let Some(actor_idle_first) = actor_idle.frames.first() else {
            return Err(game_err!("native command24 actor idle FIGANI is empty"));
        };
        let target_idle = figani::load_separated_resource(&animations, target.battle_fig * 3)?;
        let Some(target_idle_first) = target_idle.frames.first() else {
            return Err(game_err!("native command24 target idle FIGANI is empty"));
        };
        let rng = self.rng.as_mut().map(|rng| rng as &mut dyn battle::GoRand);
        let plan = self.st.as_ref().unwrap().plan_native_command_derived_strike(
            Some(actor_index),
            Some(target_index),
            24,
            rng,
        )?;
        let [result] = plan.results.as_slice() else {
            return Err(game_err!("native command24 normal player path is not a single target"));
        };
        if result.target != Some(target_index) {
            return Err(game_err!("native command24 normal player path is not a single target"));
        }
        let target_hp_after = result.hp_after;
        let actor_bg_selector = native_command24_bg_selector(self.m.as_ref(), &actor)?;
        let target_bg_selector = native_command24_bg_selector(self.m.as_ref(), &target)?;
        let surfaces = separated_asset_path("surfaces");
        let bg_layers = [
            fdother::load_separated_single_frame(&surfaces, "BG.DAT", 0)?,
            fdother::load_separated_single_frame(&surfaces, "BG.DAT", 1)?,
            fdother::load_separated_single_frame(&surfaces, "BG.DAT", 2)?,
        ];
        let actor_bg = fdother::load_separated_single_frame(&surfaces, "BG.DAT", actor_bg_selector)?;
        let target_bg = fdother::load_separated_single_frame(&surfaces, "BG.DAT", target_bg_selector)?;
        let panel_assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let (palette_dac, _) = load_native_battle_palette()?;
        let actor_record_before = battle::native_battle_panel_record_for_unit(Some(&actor))?;
        let mut actor_after_mp = actor.clone();
        actor_after_mp.mp = plan.mp_after;
        let actor_record_after = battle::native_battle_panel_record_for_unit(Some(&actor_after_mp))?;
        let target_record_before = battle::native_battle_panel_record_for_unit(Some(&target))?;
        let mut target_after_damage = target.clone();
        target_after_damage.hp = target_hp_after;
        let target_record_after = battle::native_battle_panel_record_for_unit(Some(&target_after_damage))?;
        let actor_unit_index = native_command24_runtime_unit_index(self.st.as_ref(), Some(actor_index))?;
        let target_unit_index = native_command24_runtime_unit_index(self.st.as_ref(), Some(target_index))?;
        let actor_platform = if actor.native_record_byte6 != 0 {
            Some(fdother::load_separated_single_frame(&surfaces, "TAI.DAT", actor_bg_selector)?)
        } else {
            None
        };
        modern_split_strike_scenery(&actor_bg, actor_bg_selector, actor_platform.as_ref())?;
        modern_split_strike_scenery(&target_bg, target_bg_selector, None)?;
        let chapter = self.handler_chapter;
        let base = |g: &mut Game, bg: &fdother::Frame, record: &[u8], index: i64, platform: Option<&fdother::Frame>| {
            g.localized_native_command24_captured_base(bg, &panel_assets, record, index, chapter, platform, None, &[])
        };
        let actor_base_before = base(self, &actor_bg, &actor_record_before, actor_unit_index, actor_platform.as_ref())?;
        let actor_prelude_base = base(self, &actor_bg, &actor_record_before, actor_unit_index, None)?;
        let prelude =
            battlepresent::build_native_command24_prelude_frames(&battlepresent::NativeCommand24PreludeInput {
                base: &actor_prelude_base,
                actor_idle: actor_idle_first.clone(),
                platform: actor_platform.as_ref(),
                raw_side: actor.native_record_byte6,
                baseline_dac: &palette_dac,
            })?;
        let prelude_frames = native_command24_prelude_images(&prelude)?
            .into_iter()
            .map(|frame| Some(Arc::new(Picture::from_indexed(&frame))))
            .collect();
        let actor_base_after = base(self, &actor_bg, &actor_record_after, actor_unit_index, actor_platform.as_ref())?;
        let mut source_base = actor_base_after.clone();
        let source_frame = (schedule.target_start - 1) as usize;
        effect
            .frames
            .get(source_frame)
            .ok_or_else(|| game_err!("native command24 source frame unavailable"))?
            .blit_at(&mut source_base, 320)?;
        let target_base_before = base(self, &target_bg, &target_record_before, target_unit_index, None)?;
        let target_base_after = base(self, &target_bg, &target_record_after, target_unit_index, None)?;
        let transition_presents = battlepresent::build_native_command24_background_presents(
            &battlepresent::NativeCommand24BackgroundInputs {
                layers: bg_layers,
                source: &source_base,
                target: &target_base_before,
                target_idle: target_idle_first.clone(),
            },
        )?;
        let transition_pixels: Vec<Vec<u8>> = transition_presents.into_iter().map(|present| present.pixels).collect();
        let palette = self.native_ui_palette.clone();
        let mut base_images = native_command24_indexed_images(
            &[actor_base_before, actor_base_after, target_base_before, target_base_after],
            &palette,
        )?
        .into_iter();
        let transition_frames = native_command24_indexed_images(&transition_pixels, &palette)?;
        let (effect_frames, effect_positions) = native_figani_images(&effect, &palette)?;
        let (target_frames, target_positions) = native_figani_images(&target_idle, &palette)?;
        let mut target_delays = Vec::with_capacity(target_idle.frames.len());
        for (i, frame) in target_idle.frames.iter().enumerate() {
            if frame.delay <= 0 {
                return Err(game_err!("native command24 target idle frame {i} has invalid delay"));
            }
            target_delays.push(frame.delay);
        }
        if !os_mute_or_shot(Some(self)) && (self.sfx_command24_actor.is_empty() || self.sfx_command24_target.is_empty())
        {
            return Err(game_err!("native command24 FDOTHER #53 samples3/2 unavailable"));
        }
        self.native_cmd24_presentation = Some(NativeCommand24PresentationJob {
            actor: Some(actor_index),
            target: Some(target_index),
            plan: Some(plan),
            schedule,
            effect_frames,
            effect_positions,
            target_frames,
            target_positions,
            target_delays,
            prelude_frames,
            actor_base_before: base_images.next().flatten(),
            actor_base_after: base_images.next().flatten(),
            target_base_before: base_images.next().flatten(),
            target_base_after: base_images.next().flatten(),
            transition_frames,
            transition_frame: -1,
            actor_mp_before: actor.mp,
            actor_acted_before: actor.acted,
            target_hp_before: target.hp,
            target_raw_byte5_before: target.native_record_byte5,
            then: Some(then),
            ..Default::default()
        });
        Ok(())
    }

    /// 原版索引色分支；HD 複合畫格（`j.modernFrame()`）屬 R6。
    ///
    /// Go：`native_command24_presentation.go:465`，`func (g *Game) drawNativeCommand24Presentation(screen *ebiten.Image) bool`
    pub(crate) fn present_native_command24(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(j) = self.native_cmd24_presentation.as_mut() else {
            return false;
        };
        if j.frame < 0
            || j.frame >= j.effect_frames.len() as i64
            || j.target_frame < 0
            || j.target_frame >= j.target_frames.len() as i64
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
        if j.transition_frame >= 0 {
            let Some(image) = j.transition_frames.get(j.transition_frame as usize).cloned().flatten() else {
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
        } else if j.frame >= j.schedule.target_impact_frame {
            &j.target_base_after
        } else {
            &j.target_base_before
        };
        let Some(base) = base.clone() else {
            return false;
        };
        let mut shake = 0;
        if j.frame == j.schedule.target_impact_frame && !j.damage_published {
            shake = j.schedule.shake_offsets[5];
        } else if j.damage_published && j.shake_counter >= 0 && j.shake_counter < j.schedule.shake_offsets.len() as i64
        {
            shake = j.schedule.shake_offsets[j.shake_counter as usize];
        }
        if let Some(screen) = screen {
            screen.draw_image(&base, &scaled());
            if j.frame >= j.schedule.target_start {
                let [x, y] = j.target_positions[j.target_frame as usize];
                let mut op = scaled();
                op.geo_m.translate(((x - shake) * 2) as f64, (y * 2) as f64);
                if let Some(image) = j.target_frames[j.target_frame as usize].as_deref() {
                    screen.draw_image(image, &op);
                }
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

    /// Go：`native_command24_presentation.go:378`，`func (g *Game) stepNativeCommand24Presentation()`
    pub(crate) fn step_native_command24_presentation(&mut self) {
        let Some(j) = self.native_cmd24_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        if j.prelude_frame < j.prelude_frames.len() as i64 {
            j.prelude_frame += 1;
            return;
        }
        if j.transition_frame >= 0 {
            j.transition_frame += 1;
            if j.transition_frame >= j.transition_frames.len() as i64 {
                j.transition_frame = -1;
                (j.target_frame, j.target_repeat) = (0, 0);
            }
            return;
        }
        let frame = j.frame;
        if j.damage_published && j.shake_counter >= 0 {
            j.shake_counter -= 1;
        }
        if frame == j.schedule.actor_impact_frame && !j.mp_published {
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd24_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::apply_native_command_derived_strike_mp(units, j.plan.as_ref()) {
                self.fail_native_command24_presentation(&err.into());
                return;
            }
            j.mp_published = true;
            let sfx = self.sfx_command24_actor.clone();
            self.play_raw(&sfx);
        }
        let j = self.native_cmd24_presentation.as_mut().expect("Go：j 仍是同一個工作");
        if frame == j.schedule.target_impact_frame && !j.damage_published {
            if !j.mp_published {
                self.fail_native_command24_presentation(&game_err!("target marker preceded actor MP marker"));
                return;
            }
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd24_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::apply_native_command_derived_strike_target(units, j.plan.as_ref(), 0) {
                self.fail_native_command24_presentation(&err.into());
                return;
            }
            j.damage_published = true;
            j.shake_counter = 5;
            let sfx = self.sfx_command24_target.clone();
            self.play_raw(&sfx);
        }
        let j = self.native_cmd24_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.target_repeat += 1;
        if j.target_repeat >= j.target_delays[j.target_frame as usize] {
            j.target_repeat = 0;
            // Go `%`：`len(j.targetFrames)` 為 0 時 panic，Rust 的 `%` 同樣 panic。
            j.target_frame = (j.target_frame + 1) % j.target_frames.len() as i64;
        }
        j.repeat += 1;
        if j.repeat < native_command24_raw_frames_delay(Some(&*j)) {
            return;
        }
        j.repeat = 0;
        j.frame += 1;
        if j.frame == j.schedule.target_start {
            j.transition_frame = 0;
        }
        if j.frame < j.effect_frames.len() as i64 {
            return;
        }
        if !j.mp_published || !j.damage_published {
            self.fail_native_command24_presentation(&game_err!("required raw marker was not presented"));
            return;
        }
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd24_presentation.as_mut().expect("Go：j 仍是同一個工作");
        if let Err(err) = battle::complete_native_command_derived_strike(units, j.plan.as_ref()) {
            self.fail_native_command24_presentation(&err.into());
            return;
        }
        let mut j = self.native_cmd24_presentation.take().expect("Go：j 仍是同一個工作");
        let (then, results) =
            (j.then.take(), j.plan.take().expect("Go：CompleteNativeCommandDerivedStrike 已檢查 plan").results);
        if let Some(then) = then {
            then(self, results);
        }
    }

    /// Go：`native_command24_presentation.go:365`，`func (g *Game) failNativeCommand24Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand24Presentation` 補移植）。
    pub(crate) fn fail_native_command24_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd24_presentation.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        let actor = j.actor.expect("Go：j.actor 為 nil 時解參考 panic");
        let target = j.target.expect("Go：j.target 為 nil 時解參考 panic");
        (units[actor].mp, units[actor].acted, units[target].hp) =
            (j.actor_mp_before, j.actor_acted_before, j.target_hp_before);
        if units[target].has_native_record_byte5 {
            units[target].native_record_byte5 = j.target_raw_byte5_before;
        }
        self.load_err = format!("native command24 presentation: {}", err.message());
    }
}

/// Go：`native_command24_presentation.go:453`，`func nativeCommand24RawFramesDelay(j *nativeCommand24PresentationJob) int`
/// （`r8`，隨呼叫端 `stepNativeCommand24Presentation` 補移植）。
pub(crate) fn native_command24_raw_frames_delay(j: Option<&NativeCommand24PresentationJob>) -> i64 {
    let Some(j) = j.filter(|j| j.frame >= 0 && j.frame < NATIVE_COMMAND24_RAW_DELAYS.len() as i64) else {
        return 0;
    };
    // BuildNativeCommand24Schedule already fixed the complete resource98 raw
    // signature; the explicit table keeps Draw timing independent of PNG count.
    NATIVE_COMMAND24_RAW_DELAYS[j.frame as usize] as i64
}

/// Go：`native_command24_presentation.go:462`，`var nativeCommand24RawDelays = [15]byte{…}`（`r8` 函式用到，隨呼叫端補上）。
pub(crate) const NATIVE_COMMAND24_RAW_DELAYS: [u8; 15] = [2, 2, 2, 8, 2, 2, 2, 2, 8, 2, 2, 6, 2, 2, 4];

#[cfg(test)]
#[path = "native_command24_presentation/native_command24_presentation_test.rs"]
mod native_command24_presentation_test;
