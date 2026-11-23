//! Go 來源：`remake/cmd/fd2/native_defeat.go`（256 行）。

use super::*;

/// nativeDefeatJob 擁有 READY §5.2.1 的有限呈現，不讀鍵盤、不寫存檔。
/// 原始定位與分級：fd2_pending_code1_return_title_20261001.json。
///
/// Go：`native_defeat.go:18`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeDefeatJob {
    pub(crate) plan: Vec<fdother::PendingCode1PresentationStep>,
    pub(crate) pos: i64,
    pub(crate) frames: Vec<fdother::Frame>,
    pub(crate) vga: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) ramp: Option<NativePaletteRampJob>,
    pub(crate) drawn: bool,
    pub(crate) wait_start: GameInstant,
    pub(crate) title: Option<TitleAssets>,
}

impl Game {
    /// Go：`native_defeat.go:30`，`func (g *Game) beginNativeDefeatIfConfigured()`
    pub(crate) fn begin_native_defeat_if_configured(&mut self) {
        if self.result != "lose"
            || self.camp.as_ref().and_then(|camp| camp.node()).is_none_or(|n| n.native_defeat_return_title.is_none())
        {
            return;
        }
        if self.native_defeat.is_none()
            && let Err(err) = self.start_native_defeat()
        {
            self.load_err = format!("原生敗北呈現：{err}");
            self.startup_blocked = true;
        }
    }

    fn start_native_defeat(&mut self) -> Result<()> {
        let node = self.camp.as_ref().and_then(|runner| runner.node()).ok_or_else(|| game_err!("缺少敗北節點"))?;
        if node.r#type != "battle"
            || !node.on_lose.is_empty()
            || !node.native_defeat_return_title.as_ref().is_some_and(|spec| spec.is_recovered_contract())
        {
            return Err(game_err!("敗北返回標題契約無效"));
        }
        if !native_map_assets_available(self.native_map_assets.as_ref()) || self.st.is_none() || self.m.is_none() {
            return Err(game_err!("缺少原生戰場與色盤"));
        }
        let frames =
            fdother::load_separated_pending_code1_frames(separated_asset_path("animations/fdother_079_pending_code1"))?;
        let mut scratch = vec![0; indexedmap::NATIVE_MAP_VGA_SIZE as usize];
        for frame in &frames {
            frame.blit(&mut scratch, 320, -1)?;
        }
        let title = load_title_assets()?.ok_or_else(|| game_err!("缺少完整開場素材"))?;
        self.compose_native_map_frame()?;
        let palette = fdother::vga_palette_from_dac(&self.native_map_dac)?;
        self.native_defeat = Some(NativeDefeatJob {
            plan: fdother::native_pending_code1_presentation_plan(),
            pos: 1,
            frames,
            vga: self.native_map_vga.clone(),
            dac: self.native_map_dac.clone(),
            baseline: self.native_map_assets.as_ref().unwrap().palette_dac.clone(),
            palette: color_palette_from_nrgba(&palette),
            title: Some(title),
            ..Default::default()
        });
        self.stop_bgm();
        self.help_visible = false;
        Ok(())
    }

    pub(crate) fn step_native_defeat(&mut self, now: GameInstant) -> Result<()> {
        use fdother::PendingCode1PresentationKind::*;

        let Some(job) = self.native_defeat.as_mut() else { return Ok(()) };
        if let Some(ramp) = job.ramp.as_mut() {
            if !ramp.drawn {
                return Ok(());
            }
            ramp.step += 1;
            if ramp.step < ramp.deltas.len() as i64 {
                return ramp.apply_current();
            }
            let ramp = job.ramp.take().unwrap();
            (job.dac, job.palette) = (ramp.dac, ramp.palette);
            job.pos += 1;
        }
        while let Some(step) = job.plan.get(job.pos as usize) {
            match step.kind {
                WaitTick => {
                    if !job.drawn {
                        return Ok(());
                    }
                    if job.wait_start.is_zero() {
                        job.wait_start = now;
                        return Ok(());
                    }
                    if now - job.wait_start < GoDuration(step.count * NATIVE_BIOS_TICK_PERIOD.0) {
                        return Ok(());
                    }
                }
                PreparePalette | FadeIn => {
                    let (first, last) = if step.kind == PreparePalette { (0, 63) } else { (64, 0) };
                    let mut ramp = NativePaletteRampJob {
                        deltas: build_inclusive_palette_deltas(first, last)?,
                        delay_ms: 2,
                        dac: job.dac.clone(),
                        baseline: job.baseline.clone(),
                        ..Default::default()
                    };
                    ramp.apply_current()?;
                    job.ramp = Some(ramp);
                    return Ok(());
                }
                ClearScreen => job.vga.fill(0),
                DrawFrame => {
                    let frame = job.frames.get(step.frame as usize).ok_or_else(|| game_err!("缺少敗北畫格"))?;
                    frame.blit(&mut job.vga, 320, -1)?;
                }
                Release => {
                    let title = job.title.take().ok_or_else(|| game_err!("缺少敗北後開場素材"))?;
                    self.finish_native_defeat(title);
                    return Ok(());
                }
                StopBgm => return Err(game_err!("敗北開始步驟重複")),
            }
            job.pos += 1;
            job.drawn = false;
            job.wait_start = GameInstant::default();
        }
        Err(game_err!("敗北流程缺少返回步驟"))
    }

    fn clear_defeated_battle_state(&mut self) {
        self.reset_action_overlay_lifecycle();
        self.clear_native_player_movement();
        self.clear_shop_transient_state_for_load();
        self.clear_church_transient_state_for_load();
        self.result.clear();
        self.msg.clear();
        self.sel = None;
        self.reach.clear();
        self.dialog.clear();
        (self.atk, self.walk, self.native_player_focus) = (None, None, None);
        (self.native_command_open, self.spell_open, self.item_open) = (false, false, false);
        (self.ai_busy, self.ring, self.moved) = (false, false, false);
        (self.ai_phase_clear_bit7_pending, self.ai_phase_selector0_pending) = (false, false);
        (self.ai_phase_selector0_sweep_pend, self.ai_ally_phase_pending) = (false, false);
        (self.native_ai_action_plan, self.native_ai_idle_recovery, self.native_end_turn_recovery) = (None, None, None);
        self.pending_death_programs.clear();
        self.death_program_running = false;
        (self.death_program_killer, self.death_program_dead) = (None, None);
        self.death_rewarded.clear();
        (self.native_chapter_result, self.native_field_event_pending, self.native_field_event61) = (None, None, None);
        self.native_result_matched_rules.clear();
        (self.native_cmd0_presentation, self.native_cmd1_presentation, self.native_cmd2_presentation) =
            (None, None, None);
        (self.native_cmd3_presentation, self.native_cmd5_presentation, self.native_cmd6_presentation) =
            (None, None, None);
        (self.native_cmd7_presentation, self.native_cmd8_presentation, self.native_cmd9_player) = (None, None, None);
        (self.native_cmd9_ai_presentation, self.native_cmd1012, self.native_cmd24_presentation) = (None, None, None);
        self.native_cmd29_presentation = None;
        (self.native_heal_presentation, self.native_modifier_presentation) = (None, None);
        (self.battle_event, self.native_turn_staging, self.native_class_ui_job) = (None, None, None);
        (self.native_palette_ramp, self.native_palette_pulse) = (None, None);
        (self.indexed_transition, self.spawn_intro_transition, self.native_unit_present) = (None, None, None);
        (self.act_job, self.cam_pan, self.focus_job, self.fade, self.transition_reveal) =
            (None, None, None, None, None);
        self.story_walks.clear();
        self.beats.clear();
        self.camp_lines.clear();
        (self.story_auto_advance, self.beat_idx, self.beat_delay) = (0, -1, 0);
        (self.native_ch20_sky_key, self.native_ch23_loop, self.native2189_a) = (None, None, None);
        (self.native_ch28_post_present, self.native_ch22_reload) = (None, None);
        self.banner_t = 0;
        self.banner_frame.clear();
        (self.native_full_dac_white, self.native_full_dac_black) = (false, false);
        self.native_map_work.clear();
        self.native_map_vga.clear();
        self.native_map_dac.clear();
        self.native_map_clock.reset();
        (self.st, self.sc) = (None, None);
        self.story_actors.clear();
        self.story_roster.clear();
        self.story_spawned.clear();
        self.story_composition_event_bytes.clear();
        self.story_roster_path.clear();
        self.story_party_scenario.clear();
    }

    fn finish_native_defeat(&mut self, title: TitleAssets) {
        self.capture_native_map_hud_persistence();
        self.clear_defeated_battle_state();
        self.native_defeat = None;
        self.title_needs_new_campaign = true;
        self.title_assets = Some(title);
        self.title_phase = "cutscene".into();
        (self.title_sel, self.title_slot_sel, self.title_flash, self.title_tick) = (0, 0, 0, 0);
        (self.cut_idx, self.cut_frame, self.cut_tick) = (0, 0, 0);
        self.cut_cur.clear();
        self.play_bgm("FDMUS_018");
    }

    pub(crate) fn restart_campaign_after_defeat(&mut self) -> Result<()> {
        let campaign = self
            .camp
            .as_ref()
            .and_then(|runner| runner.c.as_ref())
            .filter(|campaign| campaign.nodes.contains_key(&campaign.start))
            .ok_or_else(|| game_err!("新戰役缺少起點"))?;
        let runner = campaign::new_runner(campaign);
        self.clear_defeated_battle_state();
        self.camp = Some(runner);
        self.party_members.clear();
        self.party_roster.clear();
        self.party_deploy.clear();
        self.party_join_order.clear();
        self.items.clear();
        (self.gold, self.handler_chapter) = (0, 0);
        self.native_current_save_plain.clear();
        self.native_chapter_slot_plain.clear();
        (self.native_chapter_slot_baseline, self.native_chapter_restore) = (None, None);
        self.native_chapter_slot_started = false;
        self.has_handler_inherited_map_view = false;
        self.native_continue_opening_confirm = false;
        self.load_err.clear();
        self.enter_node();
        if !self.load_err.is_empty() {
            return Err(game_err!("{}", self.load_err));
        }
        self.title_needs_new_campaign = false;
        Ok(())
    }
}

impl NativeDefeatJob {
    pub(crate) fn present(&mut self, screen: Option<&mut Picture>) {
        if let Some(screen) = screen {
            let palette = self.ramp.as_ref().map_or(&self.palette, |ramp| &ramp.palette);
            let frame = IndexedFrame { width: 320, height: 200, pix: self.vga.clone(), palette: palette.clone() };
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            screen.draw_image(&Picture::from_indexed(&frame), &options);
        }
        self.drawn = true;
        if let Some(ramp) = self.ramp.as_mut() {
            ramp.drawn = true;
        }
    }
}
